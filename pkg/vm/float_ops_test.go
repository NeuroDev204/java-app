package vm

import (
	"math"
	"testing"
)

func fbits(v float32) int32 { return int32(math.Float32bits(v)) }

func TestFloatOps(t *testing.T) {
	cases := []struct {
		name string
		code []byte
		init []int32
		want float32
	}{
		{"fconst_2", []byte{OpFconst2}, nil, 2},
		{"i2f", []byte{OpI2f}, []int32{-3}, -3},
		{"fadd", []byte{OpFadd}, []int32{fbits(1.5), fbits(2)}, 3.5},
		{"fsub", []byte{OpFsub}, []int32{fbits(1.5), fbits(2)}, -0.5},
		{"fmul", []byte{OpFmul}, []int32{fbits(1.5), fbits(2)}, 3},
		{"fdiv", []byte{OpFdiv}, []int32{fbits(3), fbits(2)}, 1.5},
		{"frem", []byte{OpFrem}, []int32{fbits(7), fbits(2)}, 1},
		{"fneg", []byte{OpFneg}, []int32{fbits(4)}, -4},
		{"fstore_1/fload_1", []byte{OpFstore1, OpFload1}, []int32{fbits(9)}, 9},
	}
	for _, c := range cases {
		f := &Frame{OperandStack: NewOperandStack(8), LocalVars: NewLocalVars(4)}
		for _, v := range c.init {
			f.OperandStack.PushInt(v)
		}
		if _, err := NewVMContext(nil, nil).ExecuteBytecode(nil, codeWithFrame{code: append(c.code, OpReturn), frame: f}); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if f.OperandStack.size != 1 || popFloat(f) != c.want {
			t.Fatalf("%s: want %v", c.name, c.want)
		}
	}

	nan := fbits(float32(math.NaN()))
	for op, want := range map[byte]int32{OpFcmpl: -1, OpFcmpg: 1} {
		f := runOps(t, []byte{op}, fbits(1), nan)
		if got := f.OperandStack.PopInt(); got != want {
			t.Fatalf("op %#x with NaN = %d, want %d", op, got, want)
		}
	}
	if f := runOps(t, []byte{OpFcmpl}, fbits(2), fbits(1)); f.OperandStack.PopInt() != 1 {
		t.Fatal("fcmpl 2 vs 1 must be 1")
	}
	for in, want := range map[float32]int32{2.9: 2, -2.9: -2, 1e20: math.MaxInt32, float32(math.NaN()): 0} {
		if got := f2i(in); got != want {
			t.Fatalf("f2i(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestMathAbs(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	f := &Frame{OperandStack: NewOperandStack(4)}
	f.OperandStack.PushInt(-7)
	ctx.handleNativeCall("java/lang/Math", "abs", "(I)I", f, OpInvokestatic)
	if f.OperandStack.PopInt() != 7 {
		t.Fatal("Math.abs(-7) must be 7")
	}
	pushFloat(f, -2.5)
	ctx.handleNativeCall("java/lang/Math", "abs", "(F)F", f, OpInvokestatic)
	if popFloat(f) != 2.5 {
		t.Fatal("Math.abs(-2.5f) must be 2.5")
	}
}

func TestWideIinc(t *testing.T) {
	// wide iinc local[300] += 1000; wide iload local[300]
	code := []byte{OpWide, OpIinc, 0x01, 0x2C, 0x03, 0xE8, OpWide, OpIload, 0x01, 0x2C, OpReturn}
	f := &Frame{OperandStack: NewOperandStack(4), LocalVars: NewLocalVars(4)}
	if _, err := NewVMContext(nil, nil).ExecuteBytecode(nil, codeWithFrame{code: code, frame: f}); err != nil {
		t.Fatal(err)
	}
	if f.OperandStack.size != 1 || f.OperandStack.PopInt() != 1000 {
		t.Fatal("wide iinc/iload must use the 16-bit index and constant")
	}
}

// j.<init> builds "name +N" via dup_x1/swap around a StringBuffer; swap must keep refs.
func TestSwapKeepsRefs(t *testing.T) {
	f := &Frame{OperandStack: NewOperandStack(4)}
	f.OperandStack.PushRef("name")
	f.OperandStack.PushInt(7)
	if _, err := NewVMContext(nil, nil).ExecuteBytecode(nil, codeWithFrame{code: []byte{OpSwap, OpReturn}, frame: f}); err != nil {
		t.Fatal(err)
	}
	if extractString(f.OperandStack.PopRef()) != "name" || f.OperandStack.PopInt() != 7 {
		t.Fatal("swap must move the ref slot intact")
	}
}
