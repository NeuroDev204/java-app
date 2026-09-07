package protocol

import (
	"bytes"
	"testing"
)

func TestWriteMessagePlainRoundtrip(t *testing.T) {
	msg := NewMessage(10)
	msg.WriteByte(42)
	msg.WriteShort(1234)

	var buf bytes.Buffer
	if err := WriteMessage(&buf, nil, msg); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPlainMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != 10 {
		t.Fatalf("command=%d", got.Command)
	}
	b, _ := got.ReadByte()
	if b != 42 {
		t.Fatalf("byte=%d", b)
	}
	s, _ := got.ReadShort()
	if s != 1234 {
		t.Fatalf("short=%d", s)
	}
}

func TestWriteMessageLargeOpcodeUses4ByteLength(t *testing.T) {
	msg := NewMessage(-51)
	msg.WriteBytes(make([]byte, 3))
	var buf bytes.Buffer
	if err := WriteMessage(&buf, nil, msg); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	if len(raw) != 1+4+3 {
		t.Fatalf("want 8 byte frame, got %d", len(raw))
	}
}

func TestWriteMessageNilRejected(t *testing.T) {
	if err := WriteMessage(bytes.NewBuffer(nil), nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

type oneByteWriter struct {
	buf bytes.Buffer
}

func (w *oneByteWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return w.buf.Write(p[:1])
}

func TestWriteMessageLoopsUntilAllBytesSent(t *testing.T) {
	msg := NewMessage(10)
	msg.WriteByte(42)
	msg.WriteShort(1234)
	w := &oneByteWriter{}
	if err := WriteMessage(w, nil, msg); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPlainMessage(&w.buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != 10 {
		t.Fatalf("command=%d", got.Command)
	}
	b, _ := got.ReadByte()
	if b != 42 {
		t.Fatalf("byte=%d", b)
	}
}
