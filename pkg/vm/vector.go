package vm

import (
	"strings"
	"sync"
)

// VectorData provides thread-safe backing storage for java.util.Vector.
type VectorData struct {
	mu    sync.RWMutex
	items []interface{}
}

// NewVectorData constructs an initialized VectorData.
func NewVectorData() *VectorData {
	return &VectorData{
		items: make([]interface{}, 0),
	}
}

// Add appends an item thread-safely.
func (v *VectorData) Add(elem interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.items = append(v.items, elem)
}

// Get returns the item at idx, or nil if out of bounds.
func (v *VectorData) Get(idx int) interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if idx < 0 || idx >= len(v.items) {
		return nil
	}
	return v.items[idx]
}

// Set sets the element at idx thread-safely.
func (v *VectorData) Set(idx int, elem interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if idx >= 0 && idx < len(v.items) {
		v.items[idx] = elem
	}
}

// Insert inserts an element at idx thread-safely.
func (v *VectorData) Insert(idx int, elem interface{}) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if idx < 0 {
		idx = 0
	}
	if idx >= len(v.items) {
		v.items = append(v.items, elem)
		return
	}
	v.items = append(v.items[:idx], append([]interface{}{elem}, v.items[idx:]...)...)
}

// RemoveAt removes an item at idx thread-safely.
func (v *VectorData) RemoveAt(idx int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if idx < 0 || idx >= len(v.items) {
		return
	}
	v.items = append(v.items[:idx], v.items[idx+1:]...)
}

// RemoveElement removes the first occurrence of elem thread-safely, returning true if found.
func (v *VectorData) RemoveElement(elem interface{}) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	for i, item := range v.items {
		if item == elem {
			v.items = append(v.items[:i], v.items[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes all elements thread-safely.
func (v *VectorData) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.items = v.items[:0]
}

// Size returns the count of elements.
func (v *VectorData) Size() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.items)
}

// IsEmpty returns true if there are no elements.
func (v *VectorData) IsEmpty() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.items) == 0
}

func vectorFromRef(ref interface{}) *VectorData {
	obj, ok := ref.(*Object)
	if !ok || obj == nil {
		return nil
	}
	vd, _ := obj.Extra.(*VectorData)
	return vd
}

// handleVectorNative handles native calls on java/util/Vector.
func (ctx *VMContext) handleVectorNative(mName, mDesc string, frame *Frame, opcode byte) bool {
	switch mName {
	case "<init>":
		paramCount := countMethodParamSlots(mDesc)
		for i := 0; i < paramCount; i++ {
			frame.OperandStack.PopSlot()
		}
		objRef := frame.OperandStack.PopRef()
		if obj, ok := objRef.(*Object); ok && obj != nil {
			obj.Extra = NewVectorData()
		}
		return true
	case "addElement":
		itemRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			v.Add(itemRef)
		}
		return true
	case "elementAt":
		idx := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			frame.OperandStack.PushRef(v.Get(int(idx)))
		} else {
			frame.OperandStack.PushRef(nil)
		}
		return true
	case "setElementAt":
		idx := frame.OperandStack.PopInt()
		itemRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			v.Set(int(idx), itemRef)
		}
		return true
	case "insertElementAt":
		idx := frame.OperandStack.PopInt()
		itemRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			v.Insert(int(idx), itemRef)
		}
		return true
	case "removeElementAt":
		idx := frame.OperandStack.PopInt()
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			v.RemoveAt(int(idx))
		}
		return true
	case "removeElement":
		itemRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil && v.RemoveElement(itemRef) {
			frame.OperandStack.PushInt(1)
		} else {
			frame.OperandStack.PushInt(0)
		}
		return true
	case "removeAllElements":
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			v.Clear()
		}
		return true
	case "size":
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			frame.OperandStack.PushInt(int32(v.Size()))
		} else {
			frame.OperandStack.PushInt(0)
		}
		return true
	case "isEmpty":
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil && v.IsEmpty() {
			frame.OperandStack.PushInt(1)
		} else {
			frame.OperandStack.PushInt(0)
		}
		return true
	case "firstElement":
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			frame.OperandStack.PushRef(v.Get(0))
		} else {
			frame.OperandStack.PushRef(nil)
		}
		return true
	case "lastElement":
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil && v.Size() > 0 {
			frame.OperandStack.PushRef(v.Get(v.Size() - 1))
		} else {
			frame.OperandStack.PushRef(nil)
		}
		return true
	case "contains":
		itemRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		found := false
		if v := vectorFromRef(objRef); v != nil {
			v.mu.RLock()
			for _, it := range v.items {
				if it == itemRef {
					found = true
					break
				}
			}
			v.mu.RUnlock()
		}
		if found {
			frame.OperandStack.PushInt(1)
		} else {
			frame.OperandStack.PushInt(0)
		}
		return true
	case "indexOf":
		startIdx := int32(0)
		if strings.HasPrefix(mDesc, "(Ljava/lang/Object;I)") {
			startIdx = frame.OperandStack.PopInt()
		}
		itemRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		idx := int32(-1)
		if v := vectorFromRef(objRef); v != nil {
			v.mu.RLock()
			for i := int(startIdx); i < len(v.items); i++ {
				if i >= 0 && v.items[i] == itemRef {
					idx = int32(i)
					break
				}
			}
			v.mu.RUnlock()
		}
		frame.OperandStack.PushInt(idx)
		return true
	case "lastIndexOf":
		startIdx := int32(-1)
		if strings.HasPrefix(mDesc, "(Ljava/lang/Object;I)") {
			startIdx = frame.OperandStack.PopInt()
		}
		itemRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		idx := int32(-1)
		if v := vectorFromRef(objRef); v != nil {
			v.mu.RLock()
			start := len(v.items) - 1
			if startIdx >= 0 && int(startIdx) < len(v.items) {
				start = int(startIdx)
			}
			for i := start; i >= 0; i-- {
				if v.items[i] == itemRef {
					idx = int32(i)
					break
				}
			}
			v.mu.RUnlock()
		}
		frame.OperandStack.PushInt(idx)
		return true
	case "copyInto":
		arrRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		if v := vectorFromRef(objRef); v != nil {
			v.mu.RLock()
			if slots, ok := arrRef.([]Slot); ok {
				for i := 0; i < len(v.items) && i < len(slots); i++ {
					slots[i] = Slot{Ref: v.items[i]}
				}
			}
			v.mu.RUnlock()
		}
		return true
	default:
		return false
	}
}
