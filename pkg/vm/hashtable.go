package vm

import (
	"fmt"
	"sync"
)

// normalizeHashKey extracts comparable keys from Go strings and Java String Objects.
func normalizeHashKey(ref interface{}) interface{} {
	if ref == nil {
		return nil
	}
	if slot, ok := ref.(Slot); ok {
		if slot.Ref != nil {
			return normalizeHashKey(slot.Ref)
		}
		return slot.Num
	}
	if s, ok := ref.(string); ok {
		return s
	}
	if obj, ok := ref.(*Object); ok && obj != nil {
		if s, ok := obj.Extra.(string); ok {
			return s
		}
		if s := extractString(obj); s != "" {
			return s
		}
		if obj.Class != nil && obj.Class.GetClassName() == "java/lang/String" {
			return ""
		}
		return obj
	}
	return ref
}

// HashtableData provides a thread-safe backing map for java.util.Hashtable.
type HashtableData struct {
	mu      sync.RWMutex
	entries map[interface{}]interface{}
	keyList []interface{}
}

// NewHashtableData creates a new HashtableData.
func NewHashtableData() *HashtableData {
	return &HashtableData{
		entries: make(map[interface{}]interface{}),
		keyList: make([]interface{}, 0),
	}
}

func (h *HashtableData) Put(rawKey, val interface{}) interface{} {
	if rawKey == nil || val == nil {
		return nil
	}
	key := normalizeHashKey(rawKey)

	h.mu.Lock()
	defer h.mu.Unlock()

	old, exists := h.entries[key]
	if !exists {
		h.keyList = append(h.keyList, key)
	}
	h.entries[key] = val
	return old
}

func (h *HashtableData) Get(rawKey interface{}) interface{} {
	if rawKey == nil {
		return nil
	}
	key := normalizeHashKey(rawKey)

	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.entries[key]
}

func (h *HashtableData) ContainsKey(rawKey interface{}) bool {
	if rawKey == nil {
		return false
	}
	key := normalizeHashKey(rawKey)

	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.entries[key]
	return exists
}

func (h *HashtableData) Contains(val interface{}) bool {
	if val == nil {
		return false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, v := range h.entries {
		if v == val {
			return true
		}
	}
	return false
}

func (h *HashtableData) Remove(rawKey interface{}) interface{} {
	if rawKey == nil {
		return nil
	}
	key := normalizeHashKey(rawKey)

	h.mu.Lock()
	defer h.mu.Unlock()

	old, exists := h.entries[key]
	if !exists {
		return nil
	}
	delete(h.entries, key)
	for i, k := range h.keyList {
		if k == key {
			h.keyList = append(h.keyList[:i], h.keyList[i+1:]...)
			break
		}
	}
	return old
}

func (h *HashtableData) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = make(map[interface{}]interface{})
	h.keyList = h.keyList[:0]
}

func (h *HashtableData) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.entries)
}

func (h *HashtableData) IsEmpty() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.entries) == 0
}

func (h *HashtableData) Keys() []interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	copied := make([]interface{}, len(h.keyList))
	copy(copied, h.keyList)
	return copied
}

func (h *HashtableData) Elements() []interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	elems := make([]interface{}, 0, len(h.entries))
	for _, k := range h.keyList {
		if v, ok := h.entries[k]; ok {
			elems = append(elems, v)
		}
	}
	return elems
}

func (h *HashtableData) String() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return fmt.Sprintf("Hashtable(size=%d)", len(h.entries))
}

// EnumerationData provides an iterator for java.util.Enumeration.
type EnumerationData struct {
	mu     sync.Mutex
	items  []interface{}
	cursor int
}

// NewEnumerationData creates a new EnumerationData.
func NewEnumerationData(items []interface{}) *EnumerationData {
	return &EnumerationData{
		items:  items,
		cursor: 0,
	}
}

func (e *EnumerationData) HasMoreElements() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cursor < len(e.items)
}

func (e *EnumerationData) NextElement() interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cursor >= len(e.items) {
		return nil
	}
	item := e.items[e.cursor]
	e.cursor++
	return item
}

func (ctx *VMContext) handleHashtableNative(mName, mDesc string, frame *Frame, opcode byte) bool {
	switch mName {
	case "<init>":
		paramCount := countMethodParamSlots(mDesc)
		for i := 0; i < paramCount; i++ {
			frame.OperandStack.PopSlot()
		}
		objRef := frame.OperandStack.PopRef()
		if obj, ok := objRef.(*Object); ok && obj != nil {
			obj.Extra = NewHashtableData()
		}
		return true

	case "put":
		valRef := frame.OperandStack.PopRef()
		keyRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		var old interface{}
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				old = ht.Put(keyRef, valRef)
			}
		}
		frame.OperandStack.PushRef(old)
		return true

	case "get":
		keyRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		var val interface{}
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				val = ht.Get(keyRef)
			}
		}
		frame.OperandStack.PushRef(val)
		return true

	case "containsKey":
		keyRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		has := int32(0)
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				if ht.ContainsKey(keyRef) {
					has = 1
				}
			}
		}
		frame.OperandStack.PushInt(has)
		return true

	case "contains":
		valRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		has := int32(0)
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				if ht.Contains(valRef) {
					has = 1
				}
			}
		}
		frame.OperandStack.PushInt(has)
		return true

	case "remove":
		keyRef := frame.OperandStack.PopRef()
		objRef := frame.OperandStack.PopRef()
		var old interface{}
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				old = ht.Remove(keyRef)
			}
		}
		frame.OperandStack.PushRef(old)
		return true

	case "clear":
		objRef := frame.OperandStack.PopRef()
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				ht.Clear()
			}
		}
		return true

	case "size":
		objRef := frame.OperandStack.PopRef()
		sz := int32(0)
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				sz = int32(ht.Size())
			}
		}
		frame.OperandStack.PushInt(sz)
		return true

	case "isEmpty":
		objRef := frame.OperandStack.PopRef()
		empty := int32(1)
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				if !ht.IsEmpty() {
					empty = 0
				}
			}
		}
		frame.OperandStack.PushInt(empty)
		return true

	case "keys":
		objRef := frame.OperandStack.PopRef()
		var items []interface{}
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				items = ht.Keys()
			}
		}
		frame.OperandStack.PushRef(&Object{Extra: NewEnumerationData(items)})
		return true

	case "elements":
		objRef := frame.OperandStack.PopRef()
		var items []interface{}
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				items = ht.Elements()
			}
		}
		frame.OperandStack.PushRef(&Object{Extra: NewEnumerationData(items)})
		return true

	case "toString":
		objRef := frame.OperandStack.PopRef()
		str := "{}"
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if ht, ok := obj.Extra.(*HashtableData); ok && ht != nil {
				str = ht.String()
			}
		}
		frame.OperandStack.PushRef(ctx.NewString(str))
		return true

	default:
		return false
	}
}

func (ctx *VMContext) handleEnumerationNative(mName, mDesc string, frame *Frame, opcode byte) bool {
	switch mName {
	case "hasMoreElements":
		objRef := frame.OperandStack.PopRef()
		has := int32(0)
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if e, ok := obj.Extra.(*EnumerationData); ok && e != nil {
				if e.HasMoreElements() {
					has = 1
				}
			}
		}
		frame.OperandStack.PushInt(has)
		return true

	case "nextElement":
		objRef := frame.OperandStack.PopRef()
		var item interface{}
		if obj, ok := objRef.(*Object); ok && obj != nil {
			if e, ok := obj.Extra.(*EnumerationData); ok && e != nil {
				item = e.NextElement()
			}
		}
		frame.OperandStack.PushRef(item)
		return true

	default:
		return false
	}
}
