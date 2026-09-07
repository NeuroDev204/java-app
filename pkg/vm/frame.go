package vm

type OperandStack struct {
	slots []Slot
	size  uint
}

func NewOperandStack(maxStack uint) *OperandStack {
	if maxStack == 0 {
		maxStack = 16
	}
	return &OperandStack{slots: make([]Slot, maxStack)}
}

func (s *OperandStack) PushSlot(slot Slot) {
	if s.size >= uint(len(s.slots)) {
		s.slots = append(s.slots, Slot{})
	}
	s.slots[s.size] = slot
	s.size++
}

func (s *OperandStack) PushInt(val int32) {
	if s.size >= uint(len(s.slots)) {
		s.slots = append(s.slots, Slot{})
	}
	s.slots[s.size] = Slot{Num: val}
	s.size++
}

func (s *OperandStack) PopInt() int32 {
	if s.size == 0 {
		return 0
	}
	s.size--
	return s.slots[s.size].Num
}

func (s *OperandStack) PushRef(ref interface{}) {
	if s.size >= uint(len(s.slots)) {
		s.slots = append(s.slots, Slot{})
	}
	if b, ok := ref.([]byte); ok && b == nil {
		ref = nil
	}
	s.slots[s.size] = Slot{Ref: ref}
	s.size++
}

func (s *OperandStack) PopRef() interface{} {
	if s.size == 0 {
		return nil
	}
	s.size--
	return s.slots[s.size].Ref
}

func (s *OperandStack) PopSlot() Slot {
	if s.size == 0 {
		return Slot{}
	}
	s.size--
	return s.slots[s.size]
}

func (s *OperandStack) Clear() {
	s.size = 0
}

type LocalVars []Slot

func NewLocalVars(maxLocals uint) LocalVars {
	if maxLocals == 0 {
		maxLocals = 16
	}
	return make(LocalVars, maxLocals)
}

func (l *LocalVars) EnsureSize(index uint) {
	for uint(len(*l)) <= index {
		*l = append(*l, Slot{})
	}
}

func (l *LocalVars) SetInt(index uint, val int32) {
	l.EnsureSize(index)
	(*l)[index] = Slot{Num: val}
}

func (l *LocalVars) GetInt(index uint) int32 {
	if index >= uint(len(*l)) {
		return 0
	}
	return (*l)[index].Num
}

func (l *LocalVars) SetRef(index uint, ref interface{}) {
	l.EnsureSize(index)
	(*l)[index] = Slot{Ref: ref}
}

func (l *LocalVars) GetRef(index uint) interface{} {
	if index >= uint(len(*l)) {
		return nil
	}
	return (*l)[index].Ref
}

type Frame struct {
	LocalVars        LocalVars
	OperandStack     *OperandStack
	PC               int
	PendingException *Object
}
