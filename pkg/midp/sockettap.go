package midp

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"java-app/pkg/network/protocol"
)

// maxPlainPayloadLen caps inbound frame payloads parsed by tryReadPlain.
// Prevents huge allocations from a forged length header.
const maxPlainPayloadLen = 1 << 20 // 1 MiB

// SocketTap owns the JAR's single TCP connection.
type SocketTap struct {
	mu        sync.Mutex
	conn      net.Conn
	crypto    *protocol.Crypto
	closed    bool
	readBuf   []byte
	hsDone    bool
	hsNeed    int
	hsPending []byte
	onPlain   func(*protocol.Message)
	plainAcc  []byte
}

func NewSocketTap(conn net.Conn) *SocketTap {
	return &SocketTap{conn: conn, hsNeed: 1}
}

var Dial = func(host string, port int) (net.Conn, error) {
	return net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 10*time.Second)
}

func OpenSocket(host string, port int) (*SocketTap, error) {
	conn, err := Dial(host, port)
	if err != nil {
		return nil, err
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(15 * time.Second)
	}
	return NewSocketTap(conn), nil
}

func (t *SocketTap) Crypto() *protocol.Crypto {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.crypto
}

func (t *SocketTap) SetOnPlainMessage(fn func(*protocol.Message)) {
	t.mu.Lock()
	t.onPlain = fn
	t.mu.Unlock()
}

func (t *SocketTap) ReadPlain(p []byte) (int, error) {
	t.mu.Lock()
	if t.closed || t.conn == nil {
		t.mu.Unlock()
		return 0, io.EOF
	}
	conn := t.conn
	t.mu.Unlock()

	n, err := conn.Read(p)
	if n > 0 {
		log.Printf("[SOCK-READ] read n=%d hsDone=%v hex=% x", n, t.hsDone, p[:n])
	} else if err != nil {
		t.mu.Lock()
		alreadyClosed := t.closed
		t.closed = true
		t.mu.Unlock()
		if !alreadyClosed {
			log.Printf("[SOCK-READ-ERR] closed connection on err=%v", err)
		}
	}
	var (
		msgs []*protocol.Message
		cb   func(*protocol.Message)
	)
	if n > 0 {
		t.mu.Lock()
		if !t.hsDone {
			// Post-key bytes in this read are decrypted into p's trailing suffix;
			// the handshake prefix stays raw for the JAR.
			plainTail, assembled := t.ingestHandshakeLocked(p[:n])
			if len(plainTail) > 0 {
				copy(p[n-len(plainTail):n], plainTail)
			}
			msgs = assembled
		} else if t.crypto != nil {
			plain := t.crypto.Decrypt(p[:n])
			copy(p[:n], plain)
			msgs = t.feedPlain(plain)
		}
		cb = t.onPlain
		t.mu.Unlock()
	}
	// Invoke outside the lock so callbacks may re-enter Inject/WritePlain/Crypto.
	if cb != nil {
		for _, msg := range msgs {
			cb(msg)
		}
	}
	return n, err
}

// ingestHandshakeLocked accumulates raw handshake bytes. When complete,
// installs crypto and returns plaintext for any post-handshake suffix of
// raw (same length as that suffix) plus any frames assembled from that suffix.
// Handshake bytes themselves are never decrypted.
//
// Supports two formats:
// 1. TeaMobi standard: opcode 0xD8 (216 / -40), 2-byte big-endian payload length,
//    followed by payload (1-byte keyLen, key bytes).
// 2. Legacy/test format: 1-byte keyLen, followed by key bytes.
func (t *SocketTap) ingestHandshakeLocked(raw []byte) (plainTail []byte, msgs []*protocol.Message) {
	t.hsPending = append(t.hsPending, raw...)
	if len(t.hsPending) < 1 {
		return nil, nil
	}

	if t.hsPending[0] == 0xD8 {
		if len(t.hsPending) < 3 {
			return nil, nil
		}
		payloadLen := int(binary.BigEndian.Uint16(t.hsPending[1:3]))
		totalHandshake := 3 + payloadLen
		if len(t.hsPending) < totalHandshake {
			return nil, nil
		}
		if payloadLen < 1 {
			return nil, nil
		}
		keyLen := int(t.hsPending[3])
		if keyLen <= 0 || 4+keyLen > totalHandshake {
			return nil, nil
		}
		key := append([]byte(nil), t.hsPending[4:4+keyLen]...)
		for i := 0; i < len(key)-1; i++ {
			key[i+1] ^= key[i]
		}
		t.crypto = protocol.NewCrypto(key)
		t.hsDone = true
		log.Printf("[SOCK-HANDSHAKE-DONE] key=% x", key)
		rest := t.hsPending[totalHandshake:]
		t.hsPending = nil
		if len(rest) == 0 {
			return nil, nil
		}
		plain := t.crypto.Decrypt(rest)
		return plain, t.feedPlain(plain)
	}

	keyLen := int(t.hsPending[0])
	if keyLen <= 0 || len(t.hsPending) < 1+keyLen {
		return nil, nil
	}
	key := append([]byte(nil), t.hsPending[1:1+keyLen]...)
	t.crypto = protocol.NewCrypto(key)
	t.hsDone = true
	rest := t.hsPending[1+keyLen:]
	t.hsPending = nil
	if len(rest) == 0 {
		return nil, nil
	}
	plain := t.crypto.Decrypt(rest)
	return plain, t.feedPlain(plain)
}


// feedPlain appends plaintext and assembles complete HSO frames.
// Does not invoke onPlain — caller must dispatch outside t.mu.
func (t *SocketTap) feedPlain(plain []byte) []*protocol.Message {
	t.plainAcc = append(t.plainAcc, plain...)
	var out []*protocol.Message
	for {
		msg, rest, ok := tryReadPlain(t.plainAcc)
		if !ok {
			// rest==nil means absurd length — discard accumulator.
			if rest == nil {
				t.plainAcc = nil
			}
			return out
		}
		t.plainAcc = rest
		out = append(out, msg)
	}
}

func plainFrameHeaderLen(cmd int8) int {
	if cmd == -51 || cmd == -52 || cmd == -54 || cmd == 126 {
		return 5
	}
	return 3
}

// tryReadPlain peeks header+length and only succeeds when the full frame is buffered.
// Absurd lengths are rejected with rest=nil (caller discards accumulator).
func tryReadPlain(buf []byte) (*protocol.Message, []byte, bool) {
	if len(buf) < 1 {
		return nil, buf, false
	}
	cmd := int8(buf[0])
	headerLen := plainFrameHeaderLen(cmd)
	if len(buf) < headerLen {
		return nil, buf, false
	}
	var payloadLen int
	if headerLen == 5 {
		payloadLen = int(binary.BigEndian.Uint32(buf[1:5]))
	} else {
		payloadLen = int(binary.BigEndian.Uint16(buf[1:3]))
	}
	if payloadLen < 0 || payloadLen > maxPlainPayloadLen {
		return nil, nil, false
	}
	total := headerLen + payloadLen
	if len(buf) < total {
		return nil, buf, false
	}
	msg := protocol.NewMessageWithData(cmd, buf[headerLen:total])
	return msg, buf[total:], true
}

func writeFull(conn net.Conn, p []byte) error {
	for len(p) > 0 {
		n, err := conn.Write(p)
		if n > 0 {
			p = p[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func (t *SocketTap) WritePlain(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.conn == nil {
		return 0, fmt.Errorf("socket closed")
	}
	out := p
	if t.crypto != nil {
		out = t.crypto.Encrypt(p)
	}
	log.Printf("[SOCK-WRITE] len=%d hsDone=%v crypto=%v hex=% x", len(p), t.hsDone, t.crypto != nil, out)
	if err := writeFull(t.conn, out); err != nil {
		_ = t.closeLocked()
		return 0, err
	}
	return len(p), nil
}

func (t *SocketTap) Inject(msg *protocol.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.conn == nil {
		return fmt.Errorf("socket closed")
	}
	if err := protocol.WriteMessage(t.conn, t.crypto, msg); err != nil {
		_ = t.closeLocked()
		return err
	}
	return nil
}

func (t *SocketTap) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closeLocked()
}

func (t *SocketTap) closeLocked() error {
	t.closed = true
	if t.conn == nil {
		return nil
	}
	err := t.conn.Close()
	t.conn = nil
	return err
}
