package vm

import "testing"

func runOps(t *testing.T, code []byte, init ...int32) *Frame {
	t.Helper()
	ctx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(16)}
	for _, v := range init {
		frame.OperandStack.PushInt(v)
	}
	if _, err := ctx.ExecuteBytecode(nil, codeWithFrame{code: append(code, OpReturn), frame: frame}); err != nil {
		t.Fatalf("ExecuteBytecode: %v", err)
	}
	return frame
}

// Bag equipment is built with new j(..., lconst_0, ...); a missing lconst_0
// shifted every constructor argument and the items never rendered.
func TestLongOpsKeepTwoSlots(t *testing.T) {
	f := runOps(t, []byte{OpLconst0, OpLconst1, OpLadd})
	if f.OperandStack.size != 2 || popLong(f) != 1 {
		t.Fatal("lconst_0 + lconst_1 must be the 2-slot long 1")
	}
	f = runOps(t, []byte{OpI2l}, -7)
	if f.OperandStack.size != 2 || popLong(f) != -7 {
		t.Fatal("i2l must sign-extend into 2 slots")
	}
	f = runOps(t, []byte{OpL2i}, 0, 42)
	if f.OperandStack.size != 1 || f.OperandStack.PopInt() != 42 {
		t.Fatal("l2i must leave the low word in 1 slot")
	}
	for op, want := range map[byte]int64{OpLmul: -210, OpLdiv: -8, OpLrem: -2} {
		f = runOps(t, []byte{op}, -1, -42, 0, 5) // -42 op 5
		if f.OperandStack.size != 2 || popLong(f) != want {
			t.Fatalf("op %#x: want %d", op, want)
		}
	}
	f = runOps(t, []byte{OpDupX1}, 1, 2)
	if f.OperandStack.PopInt() != 2 || f.OperandStack.PopInt() != 1 || f.OperandStack.PopInt() != 2 {
		t.Fatal("dup_x1 must turn [1 2] into [2 1 2]")
	}

	ctx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(4)}
	pushLong(frame, -5)
	ret, err := ctx.ExecuteBytecode(nil, codeWithFrame{code: []byte{OpLreturn}, frame: frame})
	if lo, ok := ret.Ref.(wideLo); err != nil || !ok || int64(ret.Num)<<32|int64(uint32(lo)) != -5 {
		t.Fatalf("lreturn must carry both words, got %+v err=%v", ret, err)
	}
}
