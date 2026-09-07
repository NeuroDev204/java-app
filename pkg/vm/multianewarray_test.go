package vm

import (
	"image/color"
	"testing"

	"java-app/pkg/classfile"
)

func TestCreateMultiArray(t *testing.T) {
	// Test 3D reference array: [[[Lap; with dimensions [4, 5, 3]
	arr3D := createMultiArray("[[[Lap;", []int{4, 5, 3})
	slots, ok := arr3D.([]Slot)
	if !ok || len(slots) != 4 {
		t.Fatalf("expected []Slot of len 4, got %T with len %v", arr3D, len(slots))
	}
	for i := 0; i < 4; i++ {
		dim2, ok := slots[i].Ref.([]Slot)
		if !ok || len(dim2) != 5 {
			t.Fatalf("dim2[%d] expected []Slot of len 5, got %T", i, slots[i].Ref)
		}
		for j := 0; j < 5; j++ {
			dim3, ok := dim2[j].Ref.([]Slot)
			if !ok || len(dim3) != 3 {
				t.Fatalf("dim3[%d][%d] expected []Slot of len 3, got %T", i, j, dim2[j].Ref)
			}
			for k := 0; k < 3; k++ {
				if dim3[k].Ref != nil {
					t.Fatalf("dim3[%d][%d][%d] expected nil, got %v", i, j, k, dim3[k].Ref)
				}
			}
		}
	}

	// Test 2D int array: [[I with dimensions [4, 8]
	arr2DInt := createMultiArray("[[I", []int{4, 8})
	slotsInt, ok := arr2DInt.([]Slot)
	if !ok || len(slotsInt) != 4 {
		t.Fatalf("expected []Slot of len 4, got %T", arr2DInt)
	}
	for i := 0; i < 4; i++ {
		row, ok := slotsInt[i].Ref.([]int32)
		if !ok || len(row) != 8 {
			t.Fatalf("row[%d] expected []int32 of len 8, got %T", i, slotsInt[i].Ref)
		}
	}

	// Test 2D byte array: [[B with dimensions [3, 6]
	arr2DByte := createMultiArray("[[B", []int{3, 6})
	slotsByte, ok := arr2DByte.([]Slot)
	if !ok || len(slotsByte) != 3 {
		t.Fatalf("expected []Slot of len 3, got %T", arr2DByte)
	}
	for i := 0; i < 3; i++ {
		row, ok := slotsByte[i].Ref.([]byte)
		if !ok || len(row) != 6 {
			t.Fatalf("row[%d] expected []byte of len 6, got %T", i, slotsByte[i].Ref)
		}
	}
}

func TestOpMultianewarrayBytecode(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	cf := &classfile.ClassFile{
		ConstantPool: classfile.ConstantPool{
			nil,
			classfile.ConstantUtf8Info{Str: "[[[Lap;"},
			classfile.ConstantClassInfo{NameIndex: 1},
		},
	}

	// Bytecode sequence:
	// iconst_4 (4)
	// iconst_5 (5)
	// iconst_3 (3)
	// multianewarray #2, 3
	// dup
	// arraylength (should be 4)
	// return
	code := []byte{
		OpIconst4,
		OpIconst5,
		OpIconst3,
		OpMultianewarray, 0x00, 0x02, 0x03,
		OpDup,
		OpArraylength,
		OpReturn,
	}

	frame := &Frame{
		OperandStack: NewOperandStack(16),
		PC:           0,
	}
	cfFrame := codeWithFrame{
		code:  code,
		frame: frame,
	}

	_, err := vmCtx.ExecuteBytecode(cf, cfFrame)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	length := frame.OperandStack.PopInt()
	if length != 4 {
		t.Fatalf("expected outer array length 4, got %d", length)
	}

	arrRef := frame.OperandStack.PopRef()
	slots, ok := arrRef.([]Slot)
	if !ok || len(slots) != 4 {
		t.Fatalf("expected []Slot len 4, got %T", arrRef)
	}
}

func TestMakeFallbackImageTransparent(t *testing.T) {
	obj := makeFallbackImage(24, 24)
	img, ok := obj.Extra.(interface {
		At(x, y int) color.Color
	})
	if !ok || img == nil {
		t.Fatalf("expected image with At method")
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if a != 0 || r != 0 || g != 0 || b != 0 {
		t.Fatalf("expected transparent fallback image, got RGBA(%d,%d,%d,%d)", r, g, b, a)
	}
}

func TestCreateImageThrowsIOExceptionOnNotFound(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{
		OperandStack: NewOperandStack(8),
		PC:           0,
	}
	frame.OperandStack.PushRef("non_existent_file.png")

	vmCtx.handleNativeCall("javax/microedition/lcdui/Image", "createImage", "(Ljava/lang/String;)Ljavax/microedition/lcdui/Image;", frame, OpInvokestatic)

	if frame.PendingException == nil {
		t.Fatalf("expected PendingException for non-existent image resource")
	}
	if frame.PendingException.Class == nil || frame.PendingException.Class.GetClassName() != "java/io/IOException" {
		t.Fatalf("expected java/io/IOException, got %v", frame.PendingException.Class)
	}
}
