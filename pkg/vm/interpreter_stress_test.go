package vm

import (
	"java-app/pkg/midp"
	"testing"
)

// setupTestVM Context helper
func setupTestVM() (*VMContext, *Frame) {
	g := midp.NewGraphics(240, 320)
	ctx := NewVMContext(nil, g)
	frame := &Frame{
		LocalVars:    NewLocalVars(10),
		OperandStack: NewOperandStack(32),
		PC:           0,
	}
	return ctx, frame
}

// TestVMFontMetricsStubs stress tests all native javax/microedition/lcdui/Font methods.
func TestVMFontMetricsStubs(t *testing.T) {
	ctx, frame := setupTestVM()

	// 1. Font.getFont(int face, int style, int size)
	frame.OperandStack.PushInt(0) // face
	frame.OperandStack.PushInt(0) // style
	frame.OperandStack.PushInt(0) // size
	ctx.handleNativeCall("javax/microedition/lcdui/Font", "getFont", "(III)Ljavax/microedition/lcdui/Font;", frame, OpInvokestatic)
	if frame.OperandStack.size != 1 {
		t.Fatalf("getFont expected 1 item on stack, got %d", frame.OperandStack.size)
	}
	fontObj := frame.OperandStack.PopRef()
	if fontObj == nil {
		t.Fatalf("getFont returned nil reference")
	}

	// 2. Font.getDefaultFont()
	ctx.handleNativeCall("javax/microedition/lcdui/Font", "getDefaultFont", "()Ljavax/microedition/lcdui/Font;", frame, OpInvokestatic)
	if frame.OperandStack.size != 1 {
		t.Fatalf("getDefaultFont expected 1 item on stack, got %d", frame.OperandStack.size)
	}
	defaultFontObj := frame.OperandStack.PopRef()
	if defaultFontObj == nil {
		t.Fatalf("getDefaultFont returned nil reference")
	}

	// 3. Font.stringWidth(String str)
	testStrings := []struct {
		input    interface{}
		expected int32
	}{
		{"Hello World", 88},    // 11 chars * 8 = 88
		{"", 0},               // 0 chars * 8 = 0
		{"ñ€✓", 24},           // 3 runes * 8 = 24
		{nil, 0},              // nil -> 0
		{"Line1\nLine2", 88},  // 11 runes * 8 = 88
	}

	for _, ts := range testStrings {
		frame.OperandStack.PushRef(defaultFontObj) // this
		frame.OperandStack.PushRef(ts.input)        // str
		ctx.handleNativeCall("javax/microedition/lcdui/Font", "stringWidth", "(Ljava/lang/String;)I", frame, OpInvokevirtual)
		w := frame.OperandStack.PopInt()
		if w != ts.expected {
			t.Errorf("stringWidth(%v) expected %d, got %d", ts.input, ts.expected, w)
		}
	}

	// 4. Font.substringWidth(String str, int offset, int len)
	frame.OperandStack.PushRef(defaultFontObj) // this
	frame.OperandStack.PushRef("Test String")  // str
	frame.OperandStack.PushInt(2)               // offset
	frame.OperandStack.PushInt(4)               // len
	ctx.handleNativeCall("javax/microedition/lcdui/Font", "substringWidth", "(Ljava/lang/String;II)I", frame, OpInvokevirtual)
	sw := frame.OperandStack.PopInt()
	if sw != 32 { // 4 * 8 = 32
		t.Errorf("substringWidth expected 32, got %d", sw)
	}

	// 5. Font.charWidth(char ch)
	frame.OperandStack.PushRef(defaultFontObj) // this
	frame.OperandStack.PushInt('A')             // ch
	ctx.handleNativeCall("javax/microedition/lcdui/Font", "charWidth", "(C)I", frame, OpInvokevirtual)
	cw := frame.OperandStack.PopInt()
	if cw != 8 {
		t.Errorf("charWidth expected 8, got %d", cw)
	}

	// 6. Font.getHeight()
	frame.OperandStack.PushRef(defaultFontObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Font", "getHeight", "()I", frame, OpInvokevirtual)
	fh := frame.OperandStack.PopInt()
	if fh != 13 {
		t.Errorf("getHeight expected 13, got %d", fh)
	}

	// 7. Font.getBaselinePosition()
	frame.OperandStack.PushRef(defaultFontObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Font", "getBaselinePosition", "()I", frame, OpInvokevirtual)
	bp := frame.OperandStack.PopInt()
	if bp != 10 {
		t.Errorf("getBaselinePosition expected 10, got %d", bp)
	}

	// 8. Font.getStyle(), getSize(), getFace()
	for _, m := range []string{"getStyle", "getSize", "getFace"} {
		frame.OperandStack.PushRef(defaultFontObj)
		ctx.handleNativeCall("javax/microedition/lcdui/Font", m, "()I", frame, OpInvokevirtual)
		v := frame.OperandStack.PopInt()
		if v != 0 {
			t.Errorf("%s expected 0, got %d", m, v)
		}
	}
}

// TestVMGraphicsClipNativeBindings stress tests setClip, clipRect, getClipX/Y/Width/Height via VM engine.
func TestVMGraphicsClipNativeBindings(t *testing.T) {
	ctx, frame := setupTestVM()
	gfxObj := &Object{}

	// Step 1: Initial clip check (default 0, 0, 240, 320)
	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipX", "()I", frame, OpInvokevirtual)
	cx := frame.OperandStack.PopInt()
	if cx != 0 {
		t.Errorf("initial getClipX expected 0, got %d", cx)
	}

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipWidth", "()I", frame, OpInvokevirtual)
	cw := frame.OperandStack.PopInt()
	if cw != 240 {
		t.Errorf("initial getClipWidth expected 240, got %d", cw)
	}

	// Step 2: Translate(20, 30) then setClip(10, 15, 100, 150)
	frame.OperandStack.PushRef(gfxObj)
	frame.OperandStack.PushInt(20) // dx
	frame.OperandStack.PushInt(30) // dy
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "translate", "(II)V", frame, OpInvokevirtual)

	frame.OperandStack.PushRef(gfxObj)
	frame.OperandStack.PushInt(10)  // x
	frame.OperandStack.PushInt(15)  // y
	frame.OperandStack.PushInt(100) // w
	frame.OperandStack.PushInt(150) // h
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "setClip", "(IIII)V", frame, OpInvokevirtual)

	// Verify getClipX, getClipY, getClipWidth, getClipHeight relative to current translation
	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipX", "()I", frame, OpInvokevirtual)
	cx = frame.OperandStack.PopInt()

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipY", "()I", frame, OpInvokevirtual)
	cy := frame.OperandStack.PopInt()

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipWidth", "()I", frame, OpInvokevirtual)
	cw = frame.OperandStack.PopInt()

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipHeight", "()I", frame, OpInvokevirtual)
	ch := frame.OperandStack.PopInt()

	if cx != 10 || cy != 15 || cw != 100 || ch != 150 {
		t.Errorf("setClip after translate expected (10,15,100,150), got (%d,%d,%d,%d)", cx, cy, cw, ch)
	}

	// Step 3: Call clipRect(15, 20, 50, 50)
	frame.OperandStack.PushRef(gfxObj)
	frame.OperandStack.PushInt(15) // x
	frame.OperandStack.PushInt(20) // y
	frame.OperandStack.PushInt(50) // w
	frame.OperandStack.PushInt(50) // h
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "clipRect", "(IIII)V", frame, OpInvokevirtual)

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipX", "()I", frame, OpInvokevirtual)
	cx = frame.OperandStack.PopInt()

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipY", "()I", frame, OpInvokevirtual)
	cy = frame.OperandStack.PopInt()

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipWidth", "()I", frame, OpInvokevirtual)
	cw = frame.OperandStack.PopInt()

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipHeight", "()I", frame, OpInvokevirtual)
	ch = frame.OperandStack.PopInt()

	// Intersection of (10,15,100,150) and (15,20,50,50) is (15,20,50,50)
	if cx != 15 || cy != 20 || cw != 50 || ch != 50 {
		t.Errorf("clipRect intersection expected (15,20,50,50), got (%d,%d,%d,%d)", cx, cy, cw, ch)
	}

	// Step 4: setClip with empty/zero dimensions
	frame.OperandStack.PushRef(gfxObj)
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(0)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "setClip", "(IIII)V", frame, OpInvokevirtual)

	frame.OperandStack.PushRef(gfxObj)
	ctx.handleNativeCall("javax/microedition/lcdui/Graphics", "getClipWidth", "()I", frame, OpInvokevirtual)
	cw = frame.OperandStack.PopInt()
	if cw != 240 {
		t.Errorf("empty setClip expected getClipWidth=240 fallback, got %d", cw)
	}
}
