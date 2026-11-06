package vm

import (
	"testing"
)

func TestByteArrayOutputStreamViaDataOutputStream(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	baos := &Object{}
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "<init>", "()V", frame, OpInvokespecial)

	dos := &Object{}
	frame.OperandStack.PushRef(dos)
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/DataOutputStream", "<init>", "(Ljava/io/OutputStream;)V", frame, OpInvokespecial)

	frame.OperandStack.PushRef(dos)
	frame.OperandStack.PushInt(0xD8) // handshake cmd -40 as unsigned
	ctx.handleNativeCall("java/io/DataOutputStream", "writeByte", "(I)V", frame, OpInvokevirtual)

	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "toByteArray", "()[B", frame, OpInvokevirtual)
	got := extractBytes(frame.OperandStack.PopRef())
	if len(got) != 1 || got[0] != 0xD8 {
		t.Fatalf("BAOS via DataOutputStream must keep written bytes, got %v", got)
	}
}

func TestVectorQueue(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	v := &Object{}
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(v)
	ctx.handleNativeCall("java/util/Vector", "<init>", "()V", frame, OpInvokespecial)
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("pkt")
	ctx.handleNativeCall("java/util/Vector", "addElement", "(Ljava/lang/Object;)V", frame, OpInvokevirtual)
	frame.OperandStack.PushRef(v)
	ctx.handleNativeCall("java/util/Vector", "size", "()I", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 1 {
		t.Fatal("Vector.size after addElement")
	}
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushInt(0)
	ctx.handleNativeCall("java/util/Vector", "elementAt", "(I)Ljava/lang/Object;", frame, OpInvokevirtual)
	if extractString(frame.OperandStack.PopRef()) != "pkt" {
		t.Fatal("Vector.elementAt(0)")
	}
}

func TestByteArrayOutputStreamFullLifecycle(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	baos := &Object{}
	frame := &Frame{OperandStack: NewOperandStack(16)}

	// <init>(I)V with initial capacity
	frame.OperandStack.PushRef(baos)
	frame.OperandStack.PushInt(32)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "<init>", "(I)V", frame, OpInvokespecial)

	// write(I)V
	frame.OperandStack.PushRef(baos)
	frame.OperandStack.PushInt(0x41) // 'A'
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "write", "(I)V", frame, OpInvokevirtual)

	// write([B)V
	frame.OperandStack.PushRef(baos)
	frame.OperandStack.PushRef([]byte{0x42, 0x43}) // 'B', 'C'
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "write", "([B)V", frame, OpInvokevirtual)

	// write([BII)V
	frame.OperandStack.PushRef(baos)
	frame.OperandStack.PushRef([]byte{0x00, 0x44, 0x45, 0x00})
	frame.OperandStack.PushInt(1) // offset
	frame.OperandStack.PushInt(2) // length ("DE")
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "write", "([BII)V", frame, OpInvokevirtual)

	// size()I
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "size", "()I", frame, OpInvokevirtual)
	if sz := frame.OperandStack.PopInt(); sz != 5 {
		t.Fatalf("expected size 5, got %d", sz)
	}

	// toString()Ljava/lang/String;
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "toString", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if s := extractString(frame.OperandStack.PopRef()); s != "ABCDE" {
		t.Fatalf("expected ABCDE, got %s", s)
	}

	// toByteArray()[B
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "toByteArray", "()[B", frame, OpInvokevirtual)
	data := extractBytes(frame.OperandStack.PopRef())
	if string(data) != "ABCDE" {
		t.Fatalf("expected bytes ABCDE, got %v", data)
	}

	// writeTo another BAOS
	baos2 := &Object{}
	frame.OperandStack.PushRef(baos2)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "<init>", "()V", frame, OpInvokespecial)

	frame.OperandStack.PushRef(baos)
	frame.OperandStack.PushRef(baos2)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "writeTo", "(Ljava/io/OutputStream;)V", frame, OpInvokevirtual)

	frame.OperandStack.PushRef(baos2)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "size", "()I", frame, OpInvokevirtual)
	if sz := frame.OperandStack.PopInt(); sz != 5 {
		t.Fatalf("expected baos2 size 5, got %d", sz)
	}

	// reset()V
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "reset", "()V", frame, OpInvokevirtual)
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "size", "()I", frame, OpInvokevirtual)
	if sz := frame.OperandStack.PopInt(); sz != 0 {
		t.Fatalf("expected size 0 after reset, got %d", sz)
	}

	// close()V
	frame.OperandStack.PushRef(baos)
	ctx.handleNativeCall("java/io/ByteArrayOutputStream", "close", "()V", frame, OpInvokevirtual)
	if frame.OperandStack.size != 0 {
		t.Fatalf("expected empty stack after close, got %d", frame.OperandStack.size)
	}
}

func TestVectorFullOperations(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	v := &Object{}
	frame := &Frame{OperandStack: NewOperandStack(16)}

	// <init>(II)V
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushInt(10)
	frame.OperandStack.PushInt(5)
	ctx.handleNativeCall("java/util/Vector", "<init>", "(II)V", frame, OpInvokespecial)

	// isEmpty() initially true
	frame.OperandStack.PushRef(v)
	ctx.handleNativeCall("java/util/Vector", "isEmpty", "()Z", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 1 {
		t.Fatal("expected empty vector")
	}

	// addElement("A"), addElement("B")
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("A")
	ctx.handleNativeCall("java/util/Vector", "addElement", "(Ljava/lang/Object;)V", frame, OpInvokevirtual)

	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("B")
	ctx.handleNativeCall("java/util/Vector", "addElement", "(Ljava/lang/Object;)V", frame, OpInvokevirtual)

	// insertElementAt("X", 0) -> ["X", "A", "B"]
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("X")
	frame.OperandStack.PushInt(0)
	ctx.handleNativeCall("java/util/Vector", "insertElementAt", "(Ljava/lang/Object;I)V", frame, OpInvokevirtual)

	// firstElement -> "X"
	frame.OperandStack.PushRef(v)
	ctx.handleNativeCall("java/util/Vector", "firstElement", "()Ljava/lang/Object;", frame, OpInvokevirtual)
	if extractString(frame.OperandStack.PopRef()) != "X" {
		t.Fatal("expected firstElement X")
	}

	// lastElement -> "B"
	frame.OperandStack.PushRef(v)
	ctx.handleNativeCall("java/util/Vector", "lastElement", "()Ljava/lang/Object;", frame, OpInvokevirtual)
	if extractString(frame.OperandStack.PopRef()) != "B" {
		t.Fatal("expected lastElement B")
	}

	// contains("A") -> true
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("A")
	ctx.handleNativeCall("java/util/Vector", "contains", "(Ljava/lang/Object;)Z", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 1 {
		t.Fatal("expected contains A == true")
	}

	// indexOf("A") -> 1
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("A")
	ctx.handleNativeCall("java/util/Vector", "indexOf", "(Ljava/lang/Object;)I", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 1 {
		t.Fatal("expected indexOf A == 1")
	}

	// setElementAt("Z", 1) -> ["X", "Z", "B"]
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("Z")
	frame.OperandStack.PushInt(1)
	ctx.handleNativeCall("java/util/Vector", "setElementAt", "(Ljava/lang/Object;I)V", frame, OpInvokevirtual)

	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushInt(1)
	ctx.handleNativeCall("java/util/Vector", "elementAt", "(I)Ljava/lang/Object;", frame, OpInvokevirtual)
	if extractString(frame.OperandStack.PopRef()) != "Z" {
		t.Fatal("expected elementAt 1 to be Z")
	}

	// removeElementAt(0) -> ["Z", "B"]
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushInt(0)
	ctx.handleNativeCall("java/util/Vector", "removeElementAt", "(I)V", frame, OpInvokevirtual)

	// removeElement("B") -> true, leaves ["Z"]
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef("B")
	ctx.handleNativeCall("java/util/Vector", "removeElement", "(Ljava/lang/Object;)Z", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 1 {
		t.Fatal("expected removeElement B == true")
	}

	// copyInto
	arr := make([]Slot, 1)
	frame.OperandStack.PushRef(v)
	frame.OperandStack.PushRef(arr)
	ctx.handleNativeCall("java/util/Vector", "copyInto", "([Ljava/lang/Object;)V", frame, OpInvokevirtual)
	if extractString(arr[0].Ref) != "Z" {
		t.Fatal("expected copyInto to place Z")
	}

	// removeAllElements
	frame.OperandStack.PushRef(v)
	ctx.handleNativeCall("java/util/Vector", "removeAllElements", "()V", frame, OpInvokevirtual)
	frame.OperandStack.PushRef(v)
	ctx.handleNativeCall("java/util/Vector", "size", "()I", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 0 {
		t.Fatal("expected size 0 after removeAllElements")
	}
}

func TestVectorThreadSafety(t *testing.T) {
	vData := NewVectorData()
	const goroutines = 20
	const iterations = 100

	var done [goroutines]chan bool
	for i := 0; i < goroutines; i++ {
		done[i] = make(chan bool)
		go func(id int) {
			for j := 0; j < iterations; j++ {
				vData.Add(id*iterations + j)
				_ = vData.Size()
				_ = vData.Get(0)
			}
			done[id] <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done[i]
	}

	if sz := vData.Size(); sz != goroutines*iterations {
		t.Fatalf("expected %d elements, got %d", goroutines*iterations, sz)
	}
}

func TestPrintStreamLogging(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	ps := ctx.GetStaticField("java/lang/System.out").Ref
	if ps == nil {
		t.Fatal("expected System.out to be initialized")
	}

	frame := &Frame{OperandStack: NewOperandStack(16)}

	// println(Ljava/lang/String;)V
	frame.OperandStack.PushRef(ps)
	frame.OperandStack.PushRef("Test log output from J2ME")
	ctx.handleNativeCall("java/io/PrintStream", "println", "(Ljava/lang/String;)V", frame, OpInvokevirtual)
	if frame.OperandStack.size != 0 {
		t.Fatalf("stack imbalance after println(String): size=%d", frame.OperandStack.size)
	}

	// println(I)V
	frame.OperandStack.PushRef(ps)
	frame.OperandStack.PushInt(42)
	ctx.handleNativeCall("java/io/PrintStream", "println", "(I)V", frame, OpInvokevirtual)
	if frame.OperandStack.size != 0 {
		t.Fatalf("stack imbalance after println(int): size=%d", frame.OperandStack.size)
	}

	// println()V
	frame.OperandStack.PushRef(ps)
	ctx.handleNativeCall("java/io/PrintStream", "println", "()V", frame, OpInvokevirtual)
	if frame.OperandStack.size != 0 {
		t.Fatalf("stack imbalance after println(): size=%d", frame.OperandStack.size)
	}

	// print(Ljava/lang/String;)V
	frame.OperandStack.PushRef(ps)
	frame.OperandStack.PushRef("Inline text")
	ctx.handleNativeCall("java/io/PrintStream", "print", "(Ljava/lang/String;)V", frame, OpInvokevirtual)
	if frame.OperandStack.size != 0 {
		t.Fatalf("stack imbalance after print(String): size=%d", frame.OperandStack.size)
	}

	// write([B)V
	frame.OperandStack.PushRef(ps)
	frame.OperandStack.PushRef([]byte("Raw bytes"))
	ctx.handleNativeCall("java/io/PrintStream", "write", "([B)V", frame, OpInvokevirtual)
	if frame.OperandStack.size != 0 {
		t.Fatalf("stack imbalance after write([]byte): size=%d", frame.OperandStack.size)
	}
}


// aq.c wraps getResourceAsStream's null in a DataInputStream and relies on the
// NPE from available() to fall back to the server icon download.
func TestDataInputStreamOverNullThrowsNPE(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	dis := &Object{}
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(dis)
	frame.OperandStack.PushRef(nil)
	ctx.handleNativeCall("java/io/DataInputStream", "<init>", "(Ljava/io/InputStream;)V", frame, OpInvokespecial)
	if frame.PendingException != nil {
		t.Fatal("constructor must not throw")
	}
	frame.OperandStack.PushRef(dis)
	ctx.handleNativeCall("java/io/DataInputStream", "available", "()I", frame, OpInvokevirtual)
	if frame.PendingException == nil {
		t.Fatal("available() on a null stream must throw NullPointerException")
	}
	if frame.OperandStack.size != 0 {
		t.Fatalf("operand stack not cleaned, size=%d", frame.OperandStack.size)
	}
}

// Bag equipment rows build new j(..., Long.parseLong(expiry)); a 1-slot result
// shifts every constructor argument and the items never render.
func TestLongParseLongPushesTwoSlots(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("-1")
	ctx.handleNativeCall("java/lang/Long", "parseLong", "(Ljava/lang/String;)J", frame, OpInvokestatic)
	if frame.OperandStack.size != 2 {
		t.Fatalf("long result must take 2 slots, got %d", frame.OperandStack.size)
	}
	lo := uint32(frame.OperandStack.PopInt())
	hi := uint32(frame.OperandStack.PopInt())
	if v := int64(uint64(hi)<<32 | uint64(lo)); v != -1 {
		t.Fatalf("parseLong(\"-1\") = %d", v)
	}
}

func TestUnknownNativeLongReturnPushesTwoSlots(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}
	ctx.handleNativeCall("java/lang/Nope", "nope", "()J", frame, OpInvokestatic)
	if frame.OperandStack.size != 2 {
		t.Fatalf("fallback long result must take 2 slots, got %d", frame.OperandStack.size)
	}
}
