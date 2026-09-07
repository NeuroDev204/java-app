package vm

import "testing"

func TestOperandStackOperations(t *testing.T) {
	stack := NewOperandStack(10)
	stack.PushInt(42)
	stack.PushInt(100)

	val1 := stack.PopInt()
	val2 := stack.PopInt()

	if val1 != 100 || val2 != 42 {
		t.Errorf("expected 100 then 42, got %d then %d", val1, val2)
	}
}

func TestLocalVarsOperations(t *testing.T) {
	locals := NewLocalVars(5)
	locals.SetInt(0, 999)
	if locals.GetInt(0) != 999 {
		t.Errorf("expected 999 at index 0, got %d", locals.GetInt(0))
	}
}
