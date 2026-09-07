package vm

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"java-app/pkg/classfile"
	"java-app/pkg/midp"
)

func TestCallSerializedExcludesConcurrent(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	var inside int32
	var wg sync.WaitGroup
	wg.Add(2)
	run := func() {
		defer wg.Done()
		ctx.CallSerialized(func() {
			if atomic.AddInt32(&inside, 1) != 1 {
				t.Error("CallSerialized overlapped")
			}
			time.Sleep(30 * time.Millisecond)
			atomic.AddInt32(&inside, -1)
		})
	}
	go run()
	go run()
	wg.Wait()
}

func TestExecuteAdd(t *testing.T) {
	// Bytecode: iconst_5 (0x08), bipush 10 (0x10 0x0A), iadd (0x60), ireturn (0xAC)
	code := []byte{0x08, 0x10, 0x0A, 0x60, 0xAC}
	frame := &Frame{
		LocalVars:    NewLocalVars(2),
		OperandStack: NewOperandStack(5),
		PC:           0,
	}

	vmCtx := NewVMContext(nil, nil)
	result, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if result.Num != 15 {
		t.Errorf("expected 15, got %d", result.Num)
	}
}

func TestNewarrayIntRoundTrip(t *testing.T) {
	// iconst_2, newarray T_INT, dup, iconst_1, bipush 42, iastore, iconst_1, iaload, ireturn
	code := []byte{OpIconst2, OpNewarray, 10, OpDup, OpIconst1, OpBipush, 42, OpIastore, OpIconst1, OpIaload, OpIreturn}
	frame := &Frame{
		LocalVars:    NewLocalVars(2),
		OperandStack: NewOperandStack(8),
		PC:           0,
	}
	vmCtx := NewVMContext(nil, nil)
	result, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	if result.Num != 42 {
		t.Fatalf("int[] iastore/iaload must round-trip 42, got %d", result.Num)
	}
}

func TestStringCompareTo(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("png")
	frame.OperandStack.PushRef("img")
	vmCtx.handleNativeCall("java/lang/String", "compareTo", "(Ljava/lang/String;)I", frame, OpInvokevirtual)
	got := frame.OperandStack.PopInt()
	if got == 0 {
		t.Fatal("\"png\".compareTo(\"img\") must be non-zero so aq.a does not treat every image as .img")
	}
	frame.OperandStack.PushRef("img")
	frame.OperandStack.PushRef("img")
	vmCtx.handleNativeCall("java/lang/String", "compareTo", "(Ljava/lang/String;)I", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 0 {
		t.Fatal("\"img\".compareTo(\"img\") must be 0")
	}
}

func TestStringLastIndexOfChar(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("/mfont/tahoma_7b_white.png")
	frame.OperandStack.PushInt(int32('.'))
	vmCtx.handleNativeCall("java/lang/String", "lastIndexOf", "(I)I", frame, OpInvokevirtual)
	got := frame.OperandStack.PopInt()
	want := int32(strings.LastIndex("/mfont/tahoma_7b_white.png", "."))
	if got != want {
		t.Fatalf("lastIndexOf('.') on tahoma path: got %d want %d", got, want)
	}
}

func TestStringLengthAndIndexOfUseRunes(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}
	charset := "aábc"
	frame.OperandStack.PushRef(charset)
	vmCtx.handleNativeCall("java/lang/String", "length", "()I", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 4 {
		t.Fatalf("length of %q must be 4 runes, not UTF-8 bytes", charset)
	}
	frame.OperandStack.PushRef(charset)
	frame.OperandStack.PushInt(int32('b'))
	vmCtx.handleNativeCall("java/lang/String", "indexOf", "(I)I", frame, OpInvokevirtual)
	got := frame.OperandStack.PopInt()
	if got != 2 {
		t.Fatalf("indexOf('b') in %q must be rune index 2, got %d (byte index would be 3)", charset, got)
	}
}

func TestStringSubstringFrom(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("xChiến Thần:hs1.teamobi.com:19129")
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("java/lang/String", "substring", "(I)Ljava/lang/String;", frame, OpInvokevirtual)
	got := extractString(frame.OperandStack.PopRef())
	if got != "Chiến Thần:hs1.teamobi.com:19129" {
		t.Fatalf("substring(1) got %q", got)
	}
}

func TestStringInitFromUTF8Bytes(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}
	obj := &Object{}
	frame.OperandStack.PushRef(obj)
	frame.OperandStack.PushRef([]byte("hello"))
	frame.OperandStack.PushRef("UTF-8")
	vmCtx.handleNativeCall("java/lang/String", "<init>", "([BLjava/lang/String;)V", frame, OpInvokespecial)
	if extractString(obj) != "hello" {
		t.Fatalf("String.<init>([B,UTF-8) Extra=%v", obj.Extra)
	}
}

func TestStringBufferAppendInt(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	sb := &Object{Extra: "/x"}
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(sb)
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("java/lang/StringBuffer", "append", "(I)Ljava/lang/StringBuffer;", frame, OpInvokevirtual)
	got, _ := sb.Extra.(string)
	if got != "/x1" {
		t.Fatalf("append(I) must append decimal int, got %q", got)
	}
}

func TestStringBufferInitWithJavaString(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// 1. Init with Java String Object
	sbObj := &Object{}
	strObj := &Object{Extra: "image"}
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(sbObj)
	frame.OperandStack.PushRef(strObj)
	vmCtx.handleNativeCall("java/lang/StringBuffer", "<init>", "(Ljava/lang/String;)V", frame, OpInvokespecial)

	got, ok := sbObj.Extra.(string)
	if !ok || got != "image" {
		t.Fatalf("StringBuffer.<init>(String) expected 'image', got %v (type %T)", sbObj.Extra, sbObj.Extra)
	}

	// 2. Append int 4000
	frame.OperandStack.PushRef(sbObj)
	frame.OperandStack.PushInt(4000)
	vmCtx.handleNativeCall("java/lang/StringBuffer", "append", "(I)Ljava/lang/StringBuffer;", frame, OpInvokevirtual)
	_ = frame.OperandStack.PopRef() // pops returned sbObj

	// 3. toString()
	frame.OperandStack.PushRef(sbObj)
	vmCtx.handleNativeCall("java/lang/StringBuffer", "toString", "()Ljava/lang/String;", frame, OpInvokevirtual)
	result := frame.OperandStack.PopRef()
	if extractString(result) != "image4000" {
		t.Fatalf("expected 'image4000', got %v", result)
	}
}

func TestStringBufferAppendTypes(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// Long append
	sb := &Object{Extra: "val="}
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(sb)
	// JVM long: high then low
	var longVal int64 = 1234567890123
	frame.OperandStack.PushInt(int32(longVal >> 32))
	frame.OperandStack.PushInt(int32(longVal))
	vmCtx.handleNativeCall("java/lang/StringBuffer", "append", "(J)Ljava/lang/StringBuffer;", frame, OpInvokevirtual)
	_ = frame.OperandStack.PopRef()
	if sb.Extra != "val=1234567890123" {
		t.Fatalf("expected 'val=1234567890123', got %v", sb.Extra)
	}

	// Boolean append
	sb = &Object{Extra: "flag="}
	frame.OperandStack.PushRef(sb)
	frame.OperandStack.PushInt(1) // true
	vmCtx.handleNativeCall("java/lang/StringBuffer", "append", "(Z)Ljava/lang/StringBuffer;", frame, OpInvokevirtual)
	_ = frame.OperandStack.PopRef()
	if sb.Extra != "flag=true" {
		t.Fatalf("expected 'flag=true', got %v", sb.Extra)
	}

	// Char append
	sb = &Object{Extra: "char="}
	frame.OperandStack.PushRef(sb)
	frame.OperandStack.PushInt(int32(':'))
	vmCtx.handleNativeCall("java/lang/StringBuffer", "append", "(C)Ljava/lang/StringBuffer;", frame, OpInvokevirtual)
	_ = frame.OperandStack.PopRef()
	if sb.Extra != "char=:" {
		t.Fatalf("expected 'char=:', got %v", sb.Extra)
	}
}


func TestNativeClipMethods(t *testing.T) {
	g := midp.NewGraphics(240, 320)
	vmCtx := NewVMContext(nil, g)

	frame := &Frame{
		OperandStack: NewOperandStack(10),
	}

	// Test setClip(10, 20, 100, 150)
	frame.OperandStack.PushRef(g)
	frame.OperandStack.PushInt(10)  // x
	frame.OperandStack.PushInt(20)  // y
	frame.OperandStack.PushInt(100) // w
	frame.OperandStack.PushInt(150) // h

	vmCtx.handleNativeCall("javax/microedition/lcdui/Graphics", "setClip", "(IIII)V", frame, OpInvokevirtual)

	if g.ClipX != 10 || g.ClipY != 20 || g.ClipW != 100 || g.ClipH != 150 {
		t.Errorf("setClip failed: got (%d,%d,%d,%d)", g.ClipX, g.ClipY, g.ClipW, g.ClipH)
	}

	// Test getClipWidth
	frame.OperandStack.PushRef(g)
	vmCtx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipWidth", "()I", frame, OpInvokevirtual)
	w := frame.OperandStack.PopInt()
	if w != 100 {
		t.Errorf("expected clip width 100, got %d", w)
	}

	// Test getClipHeight
	frame.OperandStack.PushRef(g)
	vmCtx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipHeight", "()I", frame, OpInvokevirtual)
	h := frame.OperandStack.PopInt()
	if h != 150 {
		t.Errorf("expected clip height 150, got %d", h)
	}
}

func TestNativeFontMethods(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{
		OperandStack: NewOperandStack(10),
	}

	// Test Font.getDefaultFont()
	vmCtx.handleNativeCall("javax/microedition/lcdui/Font", "getDefaultFont", "()Ljavax/microedition/lcdui/Font;", frame, OpInvokestatic)
	fontRef := frame.OperandStack.PopRef()
	if fontRef == nil {
		t.Fatalf("getDefaultFont returned nil")
	}

	// Test font.stringWidth("TEST")
	frame.OperandStack.PushRef(fontRef)
	// Push String object ref
	strObj := &Object{Class: &classfile.ClassFile{ConstantPool: make(classfile.ConstantPool, 10)}, Extra: "TEST"}
	frame.OperandStack.PushRef(strObj)

	vmCtx.handleNativeCall("javax/microedition/lcdui/Font", "stringWidth", "(Ljava/lang/String;)I", frame, OpInvokevirtual)
	sw := frame.OperandStack.PopInt()
	if sw != 32 { // 4 chars * 8 = 32
		t.Errorf("expected stringWidth 32 for 'TEST', got %d", sw)
	}

	// Test font.getHeight()
	frame.OperandStack.PushRef(fontRef)
	vmCtx.handleNativeCall("javax/microedition/lcdui/Font", "getHeight", "()I", frame, OpInvokevirtual)
	fh := frame.OperandStack.PopInt()
	if fh != 13 {
		t.Errorf("expected font height 13, got %d", fh)
	}
}
