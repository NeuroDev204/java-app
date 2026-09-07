// Package protocol provides network packet structures, serialization, and stream cipher
// implementations for the TeaMobi Knight Online (HSO) client protocol.
package protocol

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Session manages the TCP socket connection, stream cipher state, and message framing
// for network communication with the HSO game server.
type Session struct {
	conn   net.Conn
	crypto *Crypto

	sendMu sync.Mutex
	readMu sync.Mutex
	mu     sync.RWMutex

	closed bool
}

// NewSession creates a new Session wrapping an existing network connection.
func NewSession(conn net.Conn) *Session {
	return &Session{
		conn:   conn,
		closed: conn == nil,
	}
}

// NewEmptySession creates a disconnected Session instance.
func NewEmptySession() *Session {
	return &Session{
		conn:   nil,
		closed: true,
	}
}

// Connect establishes a TCP connection to the specified host and port with a 10-second default timeout.
func (s *Session) Connect(host string, port int) error {
	return s.ConnectWithTimeout(host, port, 10*time.Second)
}

// ConnectWithTimeout establishes a TCP connection to the specified host and port with a custom timeout.
func (s *Session) ConnectWithTimeout(host string, port int, timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil && !s.closed {
		_ = s.conn.Close()
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	s.conn = conn
	s.closed = false
	s.crypto = nil
	return nil
}

// SetCrypto sets or updates the stream cipher used for encryption and decryption.
func (s *Session) SetCrypto(c *Crypto) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.crypto = c
}

// GetCrypto returns the current stream cipher or nil if unencrypted.
func (s *Session) GetCrypto() *Crypto {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.crypto
}

// IsConnected returns true if the session has an open, active network connection.
func (s *Session) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conn != nil && !s.closed
}

// Close closes the underlying network connection and marks the session as closed.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.conn == nil {
		s.closed = true
		return nil
	}

	s.closed = true
	err := s.conn.Close()
	s.conn = nil
	return err
}

// SetReadDeadline sets a deadline for the next read operation.
func (s *Session) SetReadDeadline(t time.Time) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.conn == nil {
		return fmt.Errorf("connection is nil")
	}
	return s.conn.SetReadDeadline(t)
}

// PerformHandshake reads the server's initial handshake key and initializes the dynamic XOR cipher.
// Handshake protocol: 1 byte key length + raw key bytes.
func (s *Session) PerformHandshake() error {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	s.mu.RLock()
	conn := s.conn
	closed := s.closed
	s.mu.RUnlock()

	if conn == nil || closed {
		return fmt.Errorf("session is not connected")
	}

	var lenBuf [1]byte
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return fmt.Errorf("failed to read handshake key length: %w", err)
	}

	keyLen := int(lenBuf[0])
	if keyLen <= 0 {
		return fmt.Errorf("invalid handshake key length: %d", keyLen)
	}

	key := make([]byte, keyLen)
	if _, err := io.ReadFull(conn, key); err != nil {
		return fmt.Errorf("failed to read handshake key bytes: %w", err)
	}

	s.SetCrypto(NewCrypto(key))
	return nil
}

// SendMessage encrypts and writes a Message packet to the network stream.
func (s *Session) SendMessage(msg *Message) error {
	if msg == nil {
		return fmt.Errorf("cannot send nil message")
	}

	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	s.mu.RLock()
	conn := s.conn
	closed := s.closed
	crypto := s.crypto
	s.mu.RUnlock()

	if conn == nil || closed {
		return fmt.Errorf("session is not connected")
	}

	return WriteMessage(conn, crypto, msg)
}

// ReadMessage reads and decrypts the next incoming Message packet from the network stream.
func (s *Session) ReadMessage() (*Message, error) {
	s.readMu.Lock()
	defer s.readMu.Unlock()

	s.mu.RLock()
	conn := s.conn
	closed := s.closed
	crypto := s.crypto
	s.mu.RUnlock()

	if conn == nil || closed {
		return nil, fmt.Errorf("session is not connected")
	}

	var cmdBuf [1]byte
	if _, err := io.ReadFull(conn, cmdBuf[:]); err != nil {
		return nil, fmt.Errorf("failed to read message command byte: %w", err)
	}

	var cmd int8
	var payloadLen int

	if crypto != nil {
		decCmdByte := crypto.DecryptByte(cmdBuf[0])
		cmd = int8(decCmdByte)

		if isLargePayloadOpcode(cmd) {
			var lenBuf [4]byte
			if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
				return nil, fmt.Errorf("failed to read 4-byte message length (cmd=%d): %w", cmd, err)
			}
			b0 := int(crypto.DecryptByte(lenBuf[0]))
			b1 := int(crypto.DecryptByte(lenBuf[1]))
			b2 := int(crypto.DecryptByte(lenBuf[2]))
			b3 := int(crypto.DecryptByte(lenBuf[3]))
			payloadLen = (b0 << 24) | (b1 << 16) | (b2 << 8) | b3
		} else {
			var lenBuf [2]byte
			if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
				return nil, fmt.Errorf("failed to read 2-byte message length (cmd=%d): %w", cmd, err)
			}
			b0 := int(crypto.DecryptByte(lenBuf[0]))
			b1 := int(crypto.DecryptByte(lenBuf[1]))
			payloadLen = (b0 << 8) | b1
		}

		if payloadLen < 0 {
			return nil, fmt.Errorf("invalid negative payload length (%d) for opcode %d", payloadLen, cmd)
		}

		payload := make([]byte, payloadLen)
		if payloadLen > 0 {
			if _, err := io.ReadFull(conn, payload); err != nil {
				return nil, fmt.Errorf("failed to read message payload of %d bytes (cmd=%d): %w", payloadLen, cmd, err)
			}
		}

		decryptedPayload := crypto.Decrypt(payload)
		return NewMessageWithData(cmd, decryptedPayload), nil
	}

	// Plain (unencrypted) message reading fallback
	cmd = int8(cmdBuf[0])
	if isLargePayloadOpcode(cmd) {
		var lenBuf [4]byte
		if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
			return nil, fmt.Errorf("failed to read 4-byte message length (cmd=%d): %w", cmd, err)
		}
		b0 := int(lenBuf[0])
		b1 := int(lenBuf[1])
		b2 := int(lenBuf[2])
		b3 := int(lenBuf[3])
		payloadLen = (b0 << 24) | (b1 << 16) | (b2 << 8) | b3
	} else {
		var lenBuf [2]byte
		if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
			return nil, fmt.Errorf("failed to read 2-byte message length (cmd=%d): %w", cmd, err)
		}
		b0 := int(lenBuf[0])
		b1 := int(lenBuf[1])
		payloadLen = (b0 << 8) | b1
	}

	if payloadLen < 0 {
		return nil, fmt.Errorf("invalid negative payload length (%d) for opcode %d", payloadLen, cmd)
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			return nil, fmt.Errorf("failed to read message payload of %d bytes (cmd=%d): %w", payloadLen, cmd, err)
		}
	}

	return NewMessageWithData(cmd, payload), nil
}
