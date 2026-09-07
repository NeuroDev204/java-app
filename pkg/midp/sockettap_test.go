package midp

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"java-app/pkg/network/protocol"
)

type oneByteConn struct {
	net.Conn
}

func (c *oneByteConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return c.Conn.Write(p[:1])
}

type failAfterConn struct {
	net.Conn
	remain int
}

func (c *failAfterConn) Write(p []byte) (int, error) {
	if c.remain <= 0 || len(p) == 0 {
		return 0, errors.New("write failed")
	}
	c.remain--
	return 1, errors.New("write failed")
}

func TestSocketTapHandshakeThenXORInjectSharesSendPointer(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)

	key := []byte{0x11, 0x22, 0x33}
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Handshake: 1 byte length + key (unencrypted).
		if _, err := server.Write(append([]byte{byte(len(key))}, key...)); err != nil {
			t.Errorf("handshake write: %v", err)
		}
		// Read one injected encrypted frame from the client side of the pipe.
		buf := make([]byte, 64)
		_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := server.Read(buf)
		if err != nil {
			t.Errorf("server read: %v", err)
			return
		}
		peer := protocol.NewCrypto(key)
		dec := make([]byte, n)
		for i := 0; i < n; i++ {
			dec[i] = peer.DecryptByte(buf[i])
		}
		if dec[0] != 7 {
			t.Errorf("injected command = %d, want 7", int8(dec[0]))
		}
	}()

	hs := make([]byte, 1+len(key))
	n, err := tap.ReadPlain(hs)
	if err != nil || n != len(hs) {
		t.Fatalf("ReadPlain handshake n=%d err=%v", n, err)
	}
	if tap.Crypto() == nil {
		t.Fatal("expected crypto after handshake bytes")
	}

	msg := protocol.NewMessage(7)
	msg.WriteUTF("Smiley")
	if err := tap.Inject(msg); err != nil {
		t.Fatal(err)
	}
	<-done
}

func TestSocketTapWritePlainAndInjectSerialized(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)
	key := []byte{0xAA, 0xBB}
	go func() {
		_, _ = server.Write(append([]byte{byte(len(key))}, key...))
	}()
	hs := make([]byte, 1+len(key))
	_, _ = tap.ReadPlain(hs)

	got := make(chan []byte, 2)
	go func() {
		buf := make([]byte, 256)
		for i := 0; i < 2; i++ {
			_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := server.Read(buf)
			if err != nil {
				return
			}
			cp := append([]byte(nil), buf[:n]...)
			got <- cp
		}
	}()

	if _, err := tap.WritePlain([]byte{0x01, 0x00, 0x01, 0xFF}); err != nil {
		t.Fatal(err)
	}
	inj := protocol.NewMessage(2)
	inj.WriteByte(0)
	if err := tap.Inject(inj); err != nil {
		t.Fatal(err)
	}

	a := <-got
	b := <-got
	if len(a) == 0 || len(b) == 0 {
		t.Fatal("empty write")
	}
}

func TestSocketTapCloseReturnsEOF(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	tap := NewSocketTap(client)
	if err := tap.Close(); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8)
	_, err := tap.ReadPlain(buf)
	if err != io.EOF {
		t.Fatalf("ReadPlain after Close: %v", err)
	}
	_, err = tap.WritePlain([]byte{1})
	if err == nil {
		t.Fatal("WritePlain after Close should fail")
	}
	if err := tap.Inject(protocol.NewMessage(1)); err == nil {
		t.Fatal("Inject after Close should fail")
	}
}

func TestSocketTapDoesNotSwapConn(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	tap := NewSocketTap(client)
	if err := tap.Close(); err != nil {
		t.Fatal(err)
	}
	replacement, _ := net.Pipe()
	defer replacement.Close()
	// There is no SetConn API. Replacing the connection requires a new SocketTap.
	other := NewSocketTap(replacement)
	if other == tap {
		t.Fatal("new tap must be a distinct object")
	}
}

func TestSocketTapHandshakeAndCiphertextInOneRead(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)
	key := []byte{0x11, 0x22, 0x33}

	msg := protocol.NewMessage(7)
	msg.WriteByte(0xAB)
	var plainBuf bytes.Buffer
	if err := protocol.WriteMessage(&plainBuf, nil, msg); err != nil {
		t.Fatal(err)
	}
	plainFrame := plainBuf.Bytes()

	encMsg := protocol.NewMessage(7)
	encMsg.WriteByte(0xAB)
	var cipherBuf bytes.Buffer
	if err := protocol.WriteMessage(&cipherBuf, protocol.NewCrypto(key), encMsg); err != nil {
		t.Fatal(err)
	}
	cipherFrame := cipherBuf.Bytes()

	hs := append([]byte{byte(len(key))}, key...)
	payload := append(append([]byte{}, hs...), cipherFrame...)

	go func() {
		_, _ = server.Write(payload)
	}()

	buf := make([]byte, len(payload))
	n, err := tap.ReadPlain(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(payload) {
		t.Fatalf("n=%d want %d", n, len(payload))
	}
	if !bytes.Equal(buf[:len(hs)], hs) {
		t.Fatalf("handshake mutated: got %v want %v", buf[:len(hs)], hs)
	}
	gotTail := buf[len(hs):n]
	if !bytes.Equal(gotTail, plainFrame) {
		t.Fatalf("tail=%v want plaintext %v (cipher was %v)", gotTail, plainFrame, cipherFrame)
	}
}

func TestSocketTapOnPlainMessage(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)
	key := []byte{0x11, 0x22, 0x33}

	got := make(chan *protocol.Message, 1)
	tap.SetOnPlainMessage(func(msg *protocol.Message) {
		// Must not hold t.mu — Crypto also takes it (Task 6 re-entry).
		if tap.Crypto() == nil {
			t.Error("Crypto nil inside onPlain")
		}
		got <- msg
	})

	encMsg := protocol.NewMessage(7)
	encMsg.WriteByte(0xAB)
	var cipherBuf bytes.Buffer
	if err := protocol.WriteMessage(&cipherBuf, protocol.NewCrypto(key), encMsg); err != nil {
		t.Fatal(err)
	}

	hs := append([]byte{byte(len(key))}, key...)
	payload := append(append([]byte{}, hs...), cipherBuf.Bytes()...)
	go func() {
		_, _ = server.Write(payload)
	}()

	buf := make([]byte, len(payload))
	if _, err := tap.ReadPlain(buf); err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-got:
		if msg.Command != 7 {
			t.Fatalf("command=%d want 7", msg.Command)
		}
		b, err := msg.ReadByte()
		if err != nil || b != 0xAB {
			t.Fatalf("payload byte=%d err=%v want 0xAB", b, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("onPlain not called")
	}
}

func TestSocketTapWritePlainLoopsShortWrites(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	tap := NewSocketTap(&oneByteConn{Conn: client})
	key := []byte{0x11, 0x22, 0x33}
	go func() {
		_, _ = server.Write(append([]byte{byte(len(key))}, key...))
	}()
	hs := make([]byte, 1+len(key))
	if _, err := tap.ReadPlain(hs); err != nil {
		t.Fatal(err)
	}

	plain := []byte{0x01, 0x00, 0x01, 0xFF}
	done := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 64)
		off := 0
		_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
		for off < len(plain) {
			n, err := server.Read(buf[off:])
			if n > 0 {
				off += n
			}
			if err != nil {
				done <- buf[:off]
				return
			}
		}
		done <- buf[:off]
	}()
	if _, err := tap.WritePlain(plain); err != nil {
		t.Fatal(err)
	}
	got := <-done
	if len(got) != len(plain) {
		t.Fatalf("short-write loop sent %d want %d", len(got), len(plain))
	}
	peer := protocol.NewCrypto(key)
	dec := peer.Decrypt(got)
	if !bytes.Equal(dec, plain) {
		t.Fatalf("decrypted %v want %v", dec, plain)
	}
}

func TestSocketTapWritePlainClosesOnError(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	tap := NewSocketTap(&failAfterConn{Conn: client, remain: 1})
	key := []byte{0xAA, 0xBB}
	go func() {
		_, _ = server.Write(append([]byte{byte(len(key))}, key...))
	}()
	hs := make([]byte, 1+len(key))
	_, _ = tap.ReadPlain(hs)

	if _, err := tap.WritePlain([]byte{0x01, 0x02, 0x03, 0x04}); err == nil {
		t.Fatal("expected write error")
	}
	if _, err := tap.WritePlain([]byte{0x05}); err == nil {
		t.Fatal("tap should be closed after short-write error")
	}
}

func TestTryReadPlainIncompleteFrame(t *testing.T) {
	// cmd=10, length=5, only 2 payload bytes present
	buf := []byte{10, 0, 5, 1, 2}
	msg, rest, ok := tryReadPlain(buf)
	if ok || msg != nil {
		t.Fatal("expected incomplete frame")
	}
	if len(rest) != len(buf) {
		t.Fatalf("buf should stay intact, rest len=%d", len(rest))
	}
}

func TestTryReadPlainAbsurdLengthRejected(t *testing.T) {
	// Large opcode -51 with 16MiB claimed length; only header present.
	buf := []byte{0xCD, 0x01, 0x00, 0x00, 0x00} // int8(-51)
	msg, rest, ok := tryReadPlain(buf)
	if ok || msg != nil {
		t.Fatal("expected reject")
	}
	if len(rest) != 0 {
		t.Fatalf("absurd length should discard buffer, rest len=%d", len(rest))
	}
}

func TestSocketTapRealTeaMobiHandshake(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)

	// Real TeaMobi server response to handshake request [0xD8, 0x00, 0x00]:
	// Byte 0: 0xD8 (opcode 216 / -40)
	// Bytes 1-2: 0x00, 0x02 (big-endian uint16 length = 2)
	// Byte 3: 0x01 (key length = 1)
	// Byte 4: 0xA8 (the XOR key byte)
	serverHS := []byte{0xD8, 0x00, 0x02, 0x01, 0xA8}

	go func() {
		_, _ = server.Write(serverHS)
	}()

	buf := make([]byte, len(serverHS))
	n, err := tap.ReadPlain(buf)
	if err != nil {
		t.Fatalf("ReadPlain failed: %v", err)
	}
	if n != len(serverHS) {
		t.Fatalf("ReadPlain n=%d, want %d", n, len(serverHS))
	}
	if !bytes.Equal(buf[:n], serverHS) {
		t.Fatalf("raw handshake corrupted: got %v, want %v", buf[:n], serverHS)
	}

	cr := tap.Crypto()
	if cr == nil {
		t.Fatal("tap.Crypto() is nil after handshake")
	}
	wantKey := []byte{0xA8}
	if !bytes.Equal(cr.Key(), wantKey) {
		t.Fatalf("tap.Crypto().Key() = %v, want %v", cr.Key(), wantKey)
	}
}

func TestSocketTapRealTeaMobiHandshakeAndCiphertextInOneRead(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)
	serverHS := []byte{0xD8, 0x00, 0x02, 0x01, 0xA8}
	key := []byte{0xA8}

	msg := protocol.NewMessage(10)
	msg.WriteByte(0x42)
	var cipherBuf bytes.Buffer
	if err := protocol.WriteMessage(&cipherBuf, protocol.NewCrypto(key), msg); err != nil {
		t.Fatal(err)
	}
	cipherFrame := cipherBuf.Bytes()

	var plainBuf bytes.Buffer
	if err := protocol.WriteMessage(&plainBuf, nil, msg); err != nil {
		t.Fatal(err)
	}
	plainFrame := plainBuf.Bytes()

	payload := append(append([]byte{}, serverHS...), cipherFrame...)
	go func() {
		_, _ = server.Write(payload)
	}()

	buf := make([]byte, len(payload))
	n, err := tap.ReadPlain(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(payload) {
		t.Fatalf("n=%d, want %d", n, len(payload))
	}
	if !bytes.Equal(buf[:len(serverHS)], serverHS) {
		t.Fatalf("handshake corrupted: got %v, want %v", buf[:len(serverHS)], serverHS)
	}
	gotTail := buf[len(serverHS):n]
	if !bytes.Equal(gotTail, plainFrame) {
		t.Fatalf("tail=%v want plaintext %v", gotTail, plainFrame)
	}
	if tap.Crypto() == nil || !bytes.Equal(tap.Crypto().Key(), key) {
		t.Fatalf("crypto key mismatch: got %v want %v", tap.Crypto().Key(), key)
	}
}

func TestSocketTapRealTeaMobiHandshakeFragmented(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)
	serverHS := []byte{0xD8, 0x00, 0x02, 0x01, 0xA8}

	go func() {
		_, _ = server.Write(serverHS[:2])
		time.Sleep(10 * time.Millisecond)
		_, _ = server.Write(serverHS[2:])
	}()

	// First read: 2 bytes
	buf1 := make([]byte, 2)
	n1, err := tap.ReadPlain(buf1)
	if err != nil {
		t.Fatalf("first ReadPlain err: %v", err)
	}
	if n1 != 2 || !bytes.Equal(buf1[:n1], serverHS[:2]) {
		t.Fatalf("first chunk got %v want %v", buf1[:n1], serverHS[:2])
	}
	if tap.Crypto() != nil {
		t.Fatal("crypto should not be initialized after incomplete handshake")
	}

	// Second read: 3 bytes
	buf2 := make([]byte, 3)
	n2, err := tap.ReadPlain(buf2)
	if err != nil {
		t.Fatalf("second ReadPlain err: %v", err)
	}
	if n2 != 3 || !bytes.Equal(buf2[:n2], serverHS[2:]) {
		t.Fatalf("second chunk got %v want %v", buf2[:n2], serverHS[2:])
	}
	if tap.Crypto() == nil {
		t.Fatal("crypto should be initialized after complete handshake")
	}
	if !bytes.Equal(tap.Crypto().Key(), []byte{0xA8}) {
		t.Fatalf("key got %v want [0xA8]", tap.Crypto().Key())
	}
}

