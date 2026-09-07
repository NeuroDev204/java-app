package vm

import (
	"java-app/pkg/classfile"
	"sync"
)

type Object struct {
	Class  *classfile.ClassFile
	Fields map[string]Slot
	Extra  interface{}
	mu     sync.RWMutex
}

func NewObject(cf *classfile.ClassFile) *Object {
	return &Object{
		Class:  cf,
		Fields: make(map[string]Slot),
	}
}

func (o *Object) SetField(name string, val Slot) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.Fields == nil {
		o.Fields = make(map[string]Slot)
	}
	o.Fields[name] = val
}

func (o *Object) GetField(name string) Slot {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.Fields == nil {
		return Slot{}
	}
	return o.Fields[name]
}

// HasField distinguishes an explicitly initialized zero value from a field
// that is absent; appearance synchronization needs that distinction for
// optional character layers.
func (o *Object) HasField(name string) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.Fields == nil {
		return false
	}
	_, present := o.Fields[name]
	return present
}

func (o *Object) GetFieldsCopy() []Slot {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.Fields == nil {
		return nil
	}
	res := make([]Slot, 0, len(o.Fields))
	for _, v := range o.Fields {
		res = append(res, v)
	}
	return res
}
