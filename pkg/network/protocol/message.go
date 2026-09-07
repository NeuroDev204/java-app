// Package protocol provides network packet structures, serialization, and stream cipher
// implementations for the TeaMobi Knight Online (HSO) client protocol.
package protocol

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Message represents an HSO protocol network packet containing a 1-byte opcode Command
// and a binary payload Buffer formatted in big-endian byte order matching the JVM client.
type Message struct {
	Command int8
	Buffer  *bytes.Buffer
}

// NewMessage creates an empty Message instance for the specified command opcode.
func NewMessage(command int8) *Message {
	return &Message{
		Command: command,
		Buffer:  new(bytes.Buffer),
	}
}

// NewMessageWithData creates a Message with existing binary payload data.
// A copy of data is made to avoid aliasing and race conditions.
func NewMessageWithData(command int8, data []byte) *Message {
	var buf *bytes.Buffer
	if len(data) > 0 {
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		buf = bytes.NewBuffer(dataCopy)
	} else {
		buf = new(bytes.Buffer)
	}

	return &Message{
		Command: command,
		Buffer:  buf,
	}
}

// GetData returns the unread slice of bytes currently in the Message payload buffer.
func (m *Message) GetData() []byte {
	if m.Buffer == nil {
		return []byte{}
	}
	return m.Buffer.Bytes()
}

// Len returns the number of unread bytes in the payload buffer.
func (m *Message) Len() int {
	if m.Buffer == nil {
		return 0
	}
	return m.Buffer.Len()
}

// Reset resets the payload buffer to be empty.
func (m *Message) Reset() {
	if m.Buffer != nil {
		m.Buffer.Reset()
	}
}

// WriteByte implements io.ByteWriter by writing a single unsigned byte into the payload buffer.
func (m *Message) WriteByte(v byte) error {
	if m.Buffer == nil {
		m.Buffer = new(bytes.Buffer)
	}
	return m.Buffer.WriteByte(v)
}

// WriteInt8 writes a single signed 8-bit integer into the payload buffer.
func (m *Message) WriteInt8(v int8) {
	_ = m.WriteByte(byte(v))
}

// WriteShort writes a 16-bit signed integer in big-endian order into the payload buffer.
func (m *Message) WriteShort(v int16) {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(v))
	m.Buffer.Write(b[:])
}

// WriteInt writes a 32-bit signed integer in big-endian order into the payload buffer.
func (m *Message) WriteInt(v int32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(v))
	m.Buffer.Write(b[:])
}

// WriteLong writes a 64-bit signed integer in big-endian order into the payload buffer.
func (m *Message) WriteLong(v int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(v))
	m.Buffer.Write(b[:])
}

// WriteBool writes a boolean value as a 1-byte value (1 for true, 0 for false).
func (m *Message) WriteBool(v bool) {
	if v {
		_ = m.WriteByte(1)
	} else {
		_ = m.WriteByte(0)
	}
}

// WriteUTF writes a UTF-8 string prefixed by its 2-byte big-endian unsigned length.
func (m *Message) WriteUTF(v string) {
	data := []byte(v)
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(len(data)))
	m.Buffer.Write(b[:])
	m.Buffer.Write(data)
}

// WriteBytes appends raw bytes directly to the payload buffer.
func (m *Message) WriteBytes(v []byte) {
	if len(v) > 0 {
		m.Buffer.Write(v)
	}
}

// ReadByte implements io.ByteReader by reading a single unsigned byte from the payload buffer.
// Returns an error if the buffer is empty.
func (m *Message) ReadByte() (byte, error) {
	if m.Buffer == nil {
		return 0, io.EOF
	}
	return m.Buffer.ReadByte()
}

// ReadInt8 reads a single signed 8-bit integer (-128 to 127) from the payload buffer.
// Returns an error if the buffer is empty.
func (m *Message) ReadInt8() (int8, error) {
	b, err := m.ReadByte()
	if err != nil {
		return 0, err
	}
	return int8(b), nil
}

// ReadUnsignedByte reads a single unsigned 8-bit integer (0-255) from the payload buffer.
func (m *Message) ReadUnsignedByte() (uint8, error) {
	return m.ReadByte()
}

// ReadShort reads a 16-bit signed integer in big-endian order from the payload buffer.
func (m *Message) ReadShort() (int16, error) {
	if m.Buffer == nil {
		return 0, io.EOF
	}
	var b [2]byte
	if _, err := io.ReadFull(m.Buffer, b[:]); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(b[:])), nil
}

// ReadUnsignedShort reads a 16-bit unsigned integer (0-65535) in big-endian order.
func (m *Message) ReadUnsignedShort() (uint16, error) {
	if m.Buffer == nil {
		return 0, io.EOF
	}
	var b [2]byte
	if _, err := io.ReadFull(m.Buffer, b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b[:]), nil
}

// ReadInt reads a 32-bit signed integer in big-endian order from the payload buffer.
func (m *Message) ReadInt() (int32, error) {
	if m.Buffer == nil {
		return 0, io.EOF
	}
	var b [4]byte
	if _, err := io.ReadFull(m.Buffer, b[:]); err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(b[:])), nil
}

// ReadLong reads a 64-bit signed integer in big-endian order from the payload buffer.
func (m *Message) ReadLong() (int64, error) {
	if m.Buffer == nil {
		return 0, io.EOF
	}
	var b [8]byte
	if _, err := io.ReadFull(m.Buffer, b[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(b[:])), nil
}

// ReadBool reads a boolean value from the payload buffer (non-zero is true).
func (m *Message) ReadBool() (bool, error) {
	b, err := m.ReadByte()
	if err != nil {
		return false, err
	}
	return b != 0, nil
}

// ReadUTF reads a UTF-8 string prefixed by a 2-byte big-endian unsigned length.
func (m *Message) ReadUTF() (string, error) {
	length, err := m.ReadUnsignedShort()
	if err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(m.Buffer, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// ReadBytes reads exactly length bytes from the payload buffer into a slice.
func (m *Message) ReadBytes(length int) ([]byte, error) {
	if length <= 0 {
		return []byte{}, nil
	}
	if m.Buffer == nil {
		return nil, io.EOF
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(m.Buffer, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
