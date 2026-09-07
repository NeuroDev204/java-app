package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

func isLargePayloadOpcode(cmd int8) bool {
	return cmd == -51 || cmd == -52 || cmd == -54 || cmd == 126
}

// WriteMessage writes one HSO frame to w. If crypto is non-nil, command,
// length, and payload are XOR-stream encrypted with crypto's send pointer.
func WriteMessage(w io.Writer, crypto *Crypto, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("cannot send nil message")
	}
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	data := msg.GetData()
	size := len(data)
	cmd := msg.Command

	var header []byte
	var payload []byte
	if crypto != nil {
		encCmd := crypto.EncryptByte(byte(cmd))
		if isLargePayloadOpcode(cmd) {
			header = []byte{
				encCmd,
				crypto.EncryptByte(byte(size >> 24)),
				crypto.EncryptByte(byte(size >> 16)),
				crypto.EncryptByte(byte(size >> 8)),
				crypto.EncryptByte(byte(size & 0xFF)),
			}
		} else {
			header = []byte{
				encCmd,
				crypto.EncryptByte(byte(size >> 8)),
				crypto.EncryptByte(byte(size & 0xFF)),
			}
		}
		payload = crypto.Encrypt(data)
	} else {
		if isLargePayloadOpcode(cmd) {
			header = []byte{byte(cmd), byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size & 0xFF)}
		} else {
			header = []byte{byte(cmd), byte(size >> 8), byte(size & 0xFF)}
		}
		payload = data
	}
	packet := append(header, payload...)
	return writeFull(w, packet)
}

func writeFull(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
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

// ReadPlainMessage reads one unencrypted HSO frame from r.
func ReadPlainMessage(r io.Reader) (*Message, error) {
	var cmdBuf [1]byte
	if _, err := io.ReadFull(r, cmdBuf[:]); err != nil {
		return nil, err
	}
	cmd := int8(cmdBuf[0])
	var payloadLen int
	if isLargePayloadOpcode(cmd) {
		var lenBuf [4]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return nil, err
		}
		payloadLen = int(binary.BigEndian.Uint32(lenBuf[:]))
	} else {
		var lenBuf [2]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return nil, err
		}
		payloadLen = int(binary.BigEndian.Uint16(lenBuf[:]))
	}
	if payloadLen < 0 {
		return nil, fmt.Errorf("negative payload length %d", payloadLen)
	}
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}
	return NewMessageWithData(cmd, payload), nil
}
