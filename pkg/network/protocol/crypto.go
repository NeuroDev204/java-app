// Package protocol provides network packet structures, serialization, and stream cipher
// implementations for the TeaMobi Knight Online (HSO) client protocol.
package protocol

import (
	"sync"
)

// Crypto handles symmetric stream encryption and decryption using the TeaMobi HSO dynamic XOR algorithm.
// It maintains independent pointers for reading (decryption) and sending (encryption), matching
// the JVM client implementation.
type Crypto struct {
	key []byte

	// r is the pointer index for the decryption stream.
	r int
	// s is the pointer index for the encryption stream.
	s int

	// decMu protects the decryption stream pointer r.
	decMu sync.Mutex
	// encMu protects the encryption stream pointer s.
	encMu sync.Mutex
}

// NewCrypto initializes a new Crypto instance with a dynamic cipher key.
// The key bytes are copied to prevent external modifications.
func NewCrypto(key []byte) *Crypto {
	var keyCopy []byte
	if len(key) > 0 {
		keyCopy = make([]byte, len(key))
		copy(keyCopy, key)
	}
	return &Crypto{
		key: keyCopy,
		r:   0,
		s:   0,
	}
}

// Key returns a copy of the dynamic cipher key bytes.
func (c *Crypto) Key() []byte {
	if c == nil || len(c.key) == 0 {
		return nil
	}
	keyCopy := make([]byte, len(c.key))
	copy(keyCopy, c.key)
	return keyCopy
}


// DecryptByte decrypts a single byte in the stream and advances the read pointer (r).
// Formula: res = (key[r] ^ b); r = (r + 1) % len(key)
func (c *Crypto) DecryptByte(b byte) byte {
	if len(c.key) == 0 {
		return b
	}

	c.decMu.Lock()
	res := c.key[c.r] ^ b
	c.r = (c.r + 1) % len(c.key)
	c.decMu.Unlock()

	return res
}

// EncryptByte encrypts a single byte in the stream and advances the send pointer (s).
// Formula: res = (key[s] ^ b); s = (s + 1) % len(key)
func (c *Crypto) EncryptByte(b byte) byte {
	if len(c.key) == 0 {
		return b
	}

	c.encMu.Lock()
	res := c.key[c.s] ^ b
	c.s = (c.s + 1) % len(c.key)
	c.encMu.Unlock()

	return res
}

// Decrypt decrypts an entire byte slice in stream order and advances the read pointer (r).
func (c *Crypto) Decrypt(src []byte) []byte {
	if len(c.key) == 0 || len(src) == 0 {
		dst := make([]byte, len(src))
		copy(dst, src)
		return dst
	}

	dst := make([]byte, len(src))
	keyLen := len(c.key)

	c.decMu.Lock()
	for i := 0; i < len(src); i++ {
		dst[i] = c.key[c.r] ^ src[i]
		c.r = (c.r + 1) % keyLen
	}
	c.decMu.Unlock()

	return dst
}

// Encrypt encrypts an entire byte slice in stream order and advances the send pointer (s).
func (c *Crypto) Encrypt(src []byte) []byte {
	if len(c.key) == 0 || len(src) == 0 {
		dst := make([]byte, len(src))
		copy(dst, src)
		return dst
	}

	dst := make([]byte, len(src))
	keyLen := len(c.key)

	c.encMu.Lock()
	for i := 0; i < len(src); i++ {
		dst[i] = c.key[c.s] ^ src[i]
		c.s = (c.s + 1) % keyLen
	}
	c.encMu.Unlock()

	return dst
}

// Reset resets both the decryption (r) and encryption (s) pointers back to index 0.
func (c *Crypto) Reset() {
	c.decMu.Lock()
	defer c.decMu.Unlock()
	c.encMu.Lock()
	defer c.encMu.Unlock()

	c.r = 0
	c.s = 0
}
