package vm

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"java-app/pkg/classfile"
	"java-app/pkg/midp"
)

func TestConnectorOpenHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "xChiến Thần:hs1.teamobi.com:19129")
	}))
	t.Cleanup(srv.Close)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(srv.URL)
	ctx.HandleNativeCall("javax/microedition/io/Connector", "open", "(Ljava/lang/String;)Ljavax/microedition/io/Connection;", frame, OpInvokestatic)
	conn := frame.OperandStack.PopRef()
	if httpResultFromRef(conn) == nil {
		t.Fatal("expected HTTPResult connection")
	}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/HttpConnection", "getResponseCode", "()I", frame, OpInvokeinterface)
	if frame.OperandStack.PopInt() != 200 {
		t.Fatal("expected HTTP 200")
	}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/InputConnection", "openInputStream", "()Ljava/io/InputStream;", frame, OpInvokeinterface)
	stream := frame.OperandStack.PopRef()
	obj, _ := stream.(*Object)
	reader, _ := obj.Extra.(*DataStreamReader)
	if reader == nil || string(reader.Data) != "xChiến Thần:hs1.teamobi.com:19129" {
		t.Fatalf("body=%v", obj)
	}
}

func TestConnectorOpenStoresSocketTap(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	orig := midp.Dial
	midp.Dial = func(host string, port int) (net.Conn, error) {
		if host != "hs1.teamobi.com" || port != 19129 {
			t.Fatalf("dial %s:%d", host, port)
		}
		return client, nil
	}
	t.Cleanup(func() { midp.Dial = orig })

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("socket://hs1.teamobi.com:19129")
	ctx.HandleNativeCall("javax/microedition/io/Connector", "open", "(Ljava/lang/String;)Ljavax/microedition/io/Connection;", frame, OpInvokestatic)
	ref := frame.OperandStack.PopRef()
	obj, ok := ref.(*Object)
	if !ok || obj == nil {
		t.Fatal("expected connection object")
	}
	tap, ok := obj.Extra.(*midp.SocketTap)
	if !ok || tap == nil {
		t.Fatal("expected SocketTap extra")
	}
	if ctx.ActiveSocket != tap {
		t.Fatal("ActiveSocket not set")
	}
	if ctx.ActiveTap() != tap {
		t.Fatal("ActiveTap not set")
	}
}

func stubDial(t *testing.T, client net.Conn) {
	t.Helper()
	orig := midp.Dial
	midp.Dial = func(host string, port int) (net.Conn, error) {
		return client, nil
	}
	t.Cleanup(func() { midp.Dial = orig })
}

func openTestConnection(t *testing.T, ctx *VMContext) *Object {
	t.Helper()
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("socket://hs1.teamobi.com:19129")
	ctx.HandleNativeCall("javax/microedition/io/Connector", "open", "(Ljava/lang/String;)Ljavax/microedition/io/Connection;", frame, OpInvokestatic)
	obj, ok := frame.OperandStack.PopRef().(*Object)
	if !ok || obj == nil {
		t.Fatal("expected connection object")
	}
	return obj
}

func TestConnectorOpenInputOutputStreamsReuseTap(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	conn := openTestConnection(t, ctx)
	tap := conn.Extra.(*midp.SocketTap)

	for _, tc := range []struct {
		cls, name, desc string
	}{
		{"javax/microedition/io/SocketConnection", "openInputStream", "()Ljava/io/InputStream;"},
		{"javax/microedition/io/SocketConnection", "openOutputStream", "()Ljava/io/OutputStream;"},
		{"javax/microedition/io/InputConnection", "openDataInputStream", "()Ljava/io/DataInputStream;"},
		{"javax/microedition/io/OutputConnection", "openDataOutputStream", "()Ljava/io/DataOutputStream;"},
	} {
		frame := &Frame{OperandStack: NewOperandStack(8)}
		frame.OperandStack.PushRef(conn)
		ctx.HandleNativeCall(tc.cls, tc.name, tc.desc, frame, OpInvokevirtual)
		stream, ok := frame.OperandStack.PopRef().(*Object)
		if !ok || stream == nil {
			t.Fatalf("%s.%s: expected stream object", tc.cls, tc.name)
		}
		got, ok := stream.Extra.(*midp.SocketTap)
		if !ok || got != tap {
			t.Fatalf("%s.%s: expected same SocketTap extra", tc.cls, tc.name)
		}
	}
}

func TestConnectorInputStreamReadPlain(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	conn := openTestConnection(t, ctx)

	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/SocketConnection", "openInputStream", "()Ljava/io/InputStream;", frame, OpInvokevirtual)
	in, ok := frame.OperandStack.PopRef().(*Object)
	if !ok || in == nil {
		t.Fatal("expected input stream")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = server.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, _ = server.Write([]byte{0xAB, 0x01, 0x02, 0x03})
	}()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })

	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(in)
	ctx.HandleNativeCall("java/io/InputStream", "read", "()I", frame, OpInvokevirtual)
	if got := frame.OperandStack.PopInt(); got != 0xAB {
		t.Fatalf("read() = %d, want 0xAB", got)
	}

	dst := make([]byte, 8)
	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(in)
	frame.OperandStack.PushRef(dst)
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(3)
	ctx.HandleNativeCall("java/io/InputStream", "read", "([BII)I", frame, OpInvokevirtual)
	if n := frame.OperandStack.PopInt(); n != 3 {
		t.Fatalf("read([BII) = %d, want 3", n)
	}
	if string(dst[:3]) != "\x01\x02\x03" {
		t.Fatalf("read buffer = %v", dst[:3])
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server write did not finish")
	}
}

func TestConnectorInputStreamReadArrayReturnsAvailable(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	conn := openTestConnection(t, ctx)

	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/SocketConnection", "openInputStream", "()Ljava/io/InputStream;", frame, OpInvokevirtual)
	in, ok := frame.OperandStack.PopRef().(*Object)
	if !ok || in == nil {
		t.Fatal("expected input stream")
	}

	go func() {
		_ = server.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, _ = server.Write([]byte{0x01, 0x02})
	}()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })

	done := make(chan int32, 1)
	go func() {
		f := &Frame{OperandStack: NewOperandStack(8)}
		dst := make([]byte, 8)
		f.OperandStack.PushRef(in)
		f.OperandStack.PushRef(dst)
		ctx.HandleNativeCall("java/io/InputStream", "read", "([B)I", f, OpInvokevirtual)
		done <- f.OperandStack.PopInt()
	}()
	select {
	case n := <-done:
		if n != 2 {
			t.Fatalf("read([B) = %d, want 2", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("read([B) hung waiting to fill dest")
	}
}

func TestConnectorOutputStreamWritePlain(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	conn := openTestConnection(t, ctx)

	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/SocketConnection", "openOutputStream", "()Ljava/io/OutputStream;", frame, OpInvokevirtual)
	out, ok := frame.OperandStack.PopRef().(*Object)
	if !ok || out == nil {
		t.Fatal("expected output stream")
	}

	got := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 4)
		off := 0
		_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
		for off < 4 {
			n, err := server.Read(buf[off:])
			if n > 0 {
				off += n
			}
			if err != nil {
				got <- buf[:off]
				return
			}
		}
		got <- buf[:off]
	}()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })

	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(out)
	frame.OperandStack.PushInt(0xAA)
	ctx.HandleNativeCall("java/io/OutputStream", "write", "(I)V", frame, OpInvokevirtual)

	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(out)
	frame.OperandStack.PushRef([]byte{0x01, 0x02, 0x03})
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(3)
	ctx.HandleNativeCall("java/io/OutputStream", "write", "([BII)V", frame, OpInvokevirtual)

	select {
	case buf := <-got:
		if string(buf) != "\xAA\x01\x02\x03" {
			t.Fatalf("wrote %v, want AA 01 02 03", buf)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for written bytes")
	}
}

func TestConnectorOutputStreamWriteNegativeLengthDoesNotPanic(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	conn := openTestConnection(t, ctx)

	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/SocketConnection", "openOutputStream", "()Ljava/io/OutputStream;", frame, OpInvokevirtual)
	out, ok := frame.OperandStack.PopRef().(*Object)
	if !ok || out == nil {
		t.Fatal("expected output stream")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("write([BII) panicked: %v", r)
		}
	}()
	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(out)
	frame.OperandStack.PushRef([]byte{0x01, 0x02, 0x03})
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(-1)
	ctx.HandleNativeCall("java/io/OutputStream", "write", "([BII)V", frame, OpInvokevirtual)
}

func TestConnectorCloseNilsActiveSocket(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	conn := openTestConnection(t, ctx)
	if ctx.ActiveSocket == nil {
		t.Fatal("ActiveSocket not set")
	}

	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/SocketConnection", "close", "()V", frame, OpInvokevirtual)
	if ctx.ActiveSocket != nil {
		t.Fatal("ActiveSocket should be nil after close")
	}
}

func TestConnectorOpenReplacesActiveSocket(t *testing.T) {
	client1, server1 := net.Pipe()
	defer server1.Close()
	client2, server2 := net.Pipe()
	defer server2.Close()

	orig := midp.Dial
	n := 0
	midp.Dial = func(host string, port int) (net.Conn, error) {
		n++
		if n == 1 {
			return client1, nil
		}
		return client2, nil
	}
	t.Cleanup(func() { midp.Dial = orig })

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	first := openTestConnection(t, ctx)
	firstTap := first.Extra.(*midp.SocketTap)
	second := openTestConnection(t, ctx)
	secondTap := second.Extra.(*midp.SocketTap)
	if firstTap == secondTap {
		t.Fatal("reconnect should create a new SocketTap")
	}
	if ctx.ActiveSocket != secondTap {
		t.Fatal("ActiveSocket should be the later tap")
	}
	if _, err := firstTap.WritePlain([]byte{1}); err == nil {
		t.Fatal("previous ActiveSocket must be closed before replace")
	}
	_ = server1.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 1)
	if _, err := server1.Read(buf); err == nil {
		t.Fatal("previous TCP session should be closed")
	}
}

func TestConnectorOpenWithModeAndTimeouts(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("socket://hs1.teamobi.com:19129")
	frame.OperandStack.PushInt(3)
	frame.OperandStack.PushInt(1)
	ctx.HandleNativeCall("javax/microedition/io/Connector", "open", "(Ljava/lang/String;IZ)Ljavax/microedition/io/Connection;", frame, OpInvokestatic)
	obj, ok := frame.OperandStack.PopRef().(*Object)
	if !ok || obj == nil {
		t.Fatal("expected connection object")
	}
	if _, ok := obj.Extra.(*midp.SocketTap); !ok {
		t.Fatal("expected SocketTap extra")
	}
}

func TestConnectorDataInputStreamWrapsTap(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	stubDial(t, client)

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	conn := openTestConnection(t, ctx)
	tap := conn.Extra.(*midp.SocketTap)

	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(conn)
	ctx.HandleNativeCall("javax/microedition/io/SocketConnection", "openInputStream", "()Ljava/io/InputStream;", frame, OpInvokevirtual)
	in := frame.OperandStack.PopRef()

	dis := &Object{}
	frame = &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(dis)
	frame.OperandStack.PushRef(in)
	ctx.HandleNativeCall("java/io/DataInputStream", "<init>", "(Ljava/io/InputStream;)V", frame, OpInvokespecial)
	if dis.Extra != tap {
		t.Fatal("DataInputStream should reuse SocketTap extra")
	}
}

func TestParseSocketURL(t *testing.T) {
	host, port, err := parseSocketURL("socket://hs1.teamobi.com:19129")
	if err != nil {
		t.Fatal(err)
	}
	if host != "hs1.teamobi.com" || port != 19129 {
		t.Fatalf("got %s:%d", host, port)
	}
	if _, _, err := parseSocketURL("http://example.com"); err == nil {
		t.Fatal("expected error for non-socket URL")
	}
	if _, _, err := parseSocketURL("socket://noport"); err == nil {
		t.Fatal("expected error for missing port")
	}
}
