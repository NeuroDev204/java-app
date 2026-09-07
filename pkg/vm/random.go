package vm

import "time"

const (
	javaRandomMultiplier uint64 = 0x5DEECE66D
	javaRandomMask       uint64 = (1 << 48) - 1
)

// javaRandomState is protected by its owning Object's mutex. Keep Java's
// algorithm so seeded game behavior is identical across emulator platforms.
type javaRandomState struct {
	seed uint64
}

func (r *javaRandomState) next(bits uint) int32 {
	r.seed = (r.seed*javaRandomMultiplier + 0xB) & javaRandomMask
	return int32(r.seed >> (48 - bits))
}

func (ctx *VMContext) handleRandomCall(name, desc string, frame *Frame) bool {
	switch name + desc {
	case "<init>()V", "<init>(J)V", "setSeed(J)V", "nextInt()I", "nextInt(I)I", "nextLong()J":
	default:
		return false
	}
	params := make([]Slot, countMethodParamSlots(desc))
	for i := len(params) - 1; i >= 0; i-- {
		params[i] = frame.OperandStack.PopSlot()
	}
	obj, ok := frame.OperandStack.PopRef().(*Object)
	if !ok || obj == nil {
		frame.PendingException = ctx.NewException("java/lang/NullPointerException", "Random receiver is null")
		return true
	}
	obj.mu.Lock()
	defer obj.mu.Unlock()
	random, ok := obj.Extra.(*javaRandomState)
	if !ok {
		random = &javaRandomState{seed: (uint64(time.Now().UnixMilli()) ^ javaRandomMultiplier) & javaRandomMask}
		obj.Extra = random
	}
	switch name {
	case "<init>", "setSeed":
		seed := uint64(time.Now().UnixMilli())
		if len(params) == 2 {
			seed = uint64(uint32(params[0].Num))<<32 | uint64(uint32(params[1].Num))
		}
		random.seed = (seed ^ javaRandomMultiplier) & javaRandomMask
	case "nextInt":
		if len(params) == 0 {
			frame.OperandStack.PushInt(random.next(32))
			return true
		}
		bound := params[0].Num
		if bound <= 0 {
			frame.PendingException = ctx.NewException("java/lang/IllegalArgumentException", "Random bound must be positive")
			return true
		}
		if bound&(bound-1) == 0 {
			frame.OperandStack.PushInt(int32((int64(bound) * int64(random.next(31))) >> 31))
			return true
		}
		for {
			bits := random.next(31)
			value := bits % bound
			// Java's signed 32-bit overflow rejects the biased tail of the range.
			if bits-value+(bound-1) >= 0 {
				frame.OperandStack.PushInt(value)
				return true
			}
		}
	case "nextLong":
		value := int64(random.next(32))<<32 + int64(random.next(32))
		frame.OperandStack.PushInt(int32(value >> 32))
		frame.OperandStack.PushInt(int32(value))
	}
	return true
}
