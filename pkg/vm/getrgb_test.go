package vm

import (
	"testing"

	"java-app/pkg/classfile"
	"java-app/pkg/midp"
)

func i32(v uint32) int32 { return int32(v) }

// makeRGBImage builds an Image object from opaque ARGB pixels via createRGBImage.
func makeRGBImage(t *testing.T, ctx *VMContext, w, h int32, argb []int32) interface{} {
	t.Helper()
	f := &Frame{OperandStack: NewOperandStack(8)}
	f.OperandStack.PushRef(argb)
	f.OperandStack.PushInt(w)
	f.OperandStack.PushInt(h)
	f.OperandStack.PushInt(1) // processAlpha
	ctx.HandleNativeCall("javax/microedition/lcdui/Image", "createRGBImage", "([IIIZ)Ljavax/microedition/lcdui/Image;", f, OpInvokestatic)
	return f.OperandStack.PopRef()
}

func getRGB(ctx *VMContext, img interface{}, dst []int32, offset, scan, x, y, w, h int32) {
	f := &Frame{OperandStack: NewOperandStack(16)}
	f.OperandStack.PushRef(img)
	f.OperandStack.PushRef(dst)
	f.OperandStack.PushInt(offset)
	f.OperandStack.PushInt(scan)
	f.OperandStack.PushInt(x)
	f.OperandStack.PushInt(y)
	f.OperandStack.PushInt(w)
	f.OperandStack.PushInt(h)
	ctx.HandleNativeCall("javax/microedition/lcdui/Image", "getRGB", "([IIIIIII)V", f, OpInvokevirtual)
}

func TestGetRGBRoundTripOpaque(t *testing.T) {
	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	// 2x2 opaque pixels, distinct channels (transparent/character + icon colors).
	src := []int32{
		i32(0xFF112233), i32(0xFF445566),
		i32(0xFF778899), i32(0xFFAABBCC),
	}
	img := makeRGBImage(t, ctx, 2, 2, src)

	dst := make([]int32, 4)
	getRGB(ctx, img, dst, 0, 2, 0, 0, 2, 2)
	for i := range src {
		if dst[i] != src[i] {
			t.Fatalf("pixel %d: got %#08x want %#08x", i, uint32(dst[i]), uint32(src[i]))
		}
	}
}

// A fully transparent source pixel must round-trip to alpha 0 (the "character invisible"
// case) and an opaque one to alpha 0xFF — not swapped or dropped.
func TestGetRGBAlphaExtremes(t *testing.T) {
	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	src := []int32{i32(0x00000000), i32(0xFFDEAD00)}
	img := makeRGBImage(t, ctx, 2, 1, src)
	dst := make([]int32, 2)
	getRGB(ctx, img, dst, 0, 2, 0, 0, 2, 1)
	if a := uint32(dst[0]) >> 24; a != 0 {
		t.Fatalf("transparent pixel alpha=%d want 0", a)
	}
	if a := uint32(dst[1]) >> 24; a != 0xFF {
		t.Fatalf("opaque pixel alpha=%d want 255", a)
	}
}

// Sub-region with offset/scanlength: pulling one icon cell out of a sheet must land at
// the right place in a larger destination and leave the rest untouched.
func TestGetRGBSubRegionScanline(t *testing.T) {
	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	// 3x3 sheet, each pixel = 0xFF0000<row*3+col>
	src := make([]int32, 9)
	for i := range src {
		src[i] = int32(uint32(0xFF000000) | uint32(i))
	}
	img := makeRGBImage(t, ctx, 3, 3, src)

	// Extract the 2x2 bottom-right cell (x=1,y=1) into a 4-wide dst at offset 5, scan 4.
	dst := make([]int32, 16)
	getRGB(ctx, img, dst, 5, 4, 1, 1, 2, 2)
	want := map[int]uint32{5: 4, 6: 5, 9: 7, 10: 8} // (1,1)(2,1)(1,2)(2,2)
	for i, v := range dst {
		exp, sub := want[i]
		if sub {
			if uint32(v)&0xFF != exp {
				t.Fatalf("dst[%d]=%#08x want low byte %#x", i, uint32(v), exp)
			}
		} else if v != 0 {
			t.Fatalf("dst[%d]=%#08x should be untouched (0)", i, uint32(v))
		}
	}
}
