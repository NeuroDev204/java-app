package vm

import (
	"sync"
	"testing"
)

func TestHashtableBasicOperations(t *testing.T) {
	h := NewHashtableData()
	if !h.IsEmpty() || h.Size() != 0 {
		t.Fatalf("expected empty hashtable, got size %d", h.Size())
	}

	// Put and Get with string keys
	old := h.Put("key1", "val1")
	if old != nil {
		t.Fatalf("expected nil old value, got %v", old)
	}
	if h.Size() != 1 || h.IsEmpty() {
		t.Fatalf("expected size 1, got %d", h.Size())
	}
	if val := h.Get("key1"); val != "val1" {
		t.Fatalf("expected val1, got %v", val)
	}
	if !h.ContainsKey("key1") {
		t.Fatal("expected ContainsKey(key1) = true")
	}
	if h.ContainsKey("nonexistent") {
		t.Fatal("expected ContainsKey(nonexistent) = false")
	}

	// Overwrite
	old = h.Put("key1", "val2")
	if old != "val1" {
		t.Fatalf("expected old value val1, got %v", old)
	}
	if val := h.Get("key1"); val != "val2" {
		t.Fatalf("expected val2, got %v", val)
	}

	// Remove
	removed := h.Remove("key1")
	if removed != "val2" {
		t.Fatalf("expected removed val2, got %v", removed)
	}
	if h.Size() != 0 || h.ContainsKey("key1") {
		t.Fatal("expected key1 to be removed")
	}
}

func TestHashtableKeyNormalization(t *testing.T) {
	h := NewHashtableData()
	keyObj := &Object{Extra: "3029"}
	h.Put(keyObj, "sprite3029")

	// Query with raw string
	if val := h.Get("3029"); val != "sprite3029" {
		t.Fatalf("expected sprite3029 querying with raw string, got %v", val)
	}

	// Query with another Object holding string
	keyObj2 := &Object{Extra: "3029"}
	if val := h.Get(keyObj2); val != "sprite3029" {
		t.Fatalf("expected sprite3029 querying with another Object, got %v", val)
	}
}

func TestEnumeration(t *testing.T) {
	items := []interface{}{"a", "b", "c"}
	enum := NewEnumerationData(items)

	var collected []interface{}
	for enum.HasMoreElements() {
		collected = append(collected, enum.NextElement())
	}

	if len(collected) != 3 || collected[0] != "a" || collected[1] != "b" || collected[2] != "c" {
		t.Fatalf("unexpected collected enumeration items: %v", collected)
	}
	if enum.HasMoreElements() {
		t.Fatal("expected HasMoreElements to be false at end")
	}
	if val := enum.NextElement(); val != nil {
		t.Fatalf("expected nil after end, got %v", val)
	}
}

func TestHashtableConcurrentAccess(t *testing.T) {
	h := NewHashtableData()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h.Put(id, j)
				h.Get(id)
				h.ContainsKey(id)
			}
		}(i)
	}
	wg.Wait()
	if h.Size() != 50 {
		t.Fatalf("expected size 50, got %d", h.Size())
	}
}

func TestHashtableNativeDispatch(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// 1. <init>
	frame := &Frame{OperandStack: NewOperandStack(8)}
	htObj := &Object{}
	frame.OperandStack.PushRef(htObj)
	vmCtx.handleNativeCall("java/util/Hashtable", "<init>", "()V", frame, OpInvokespecial)
	if _, ok := htObj.Extra.(*HashtableData); !ok {
		t.Fatalf("expected htObj.Extra to be *HashtableData, got %T", htObj.Extra)
	}

	// 2. put("100", "Sprite100")
	frame.OperandStack.PushRef(htObj)
	frame.OperandStack.PushRef("100")
	frame.OperandStack.PushRef("Sprite100")
	vmCtx.handleNativeCall("java/util/Hashtable", "put", "(Ljava/lang/Object;Ljava/lang/Object;)Ljava/lang/Object;", frame, OpInvokevirtual)
	oldVal := frame.OperandStack.PopRef()
	if oldVal != nil {
		t.Fatalf("expected nil oldVal on first put, got %v", oldVal)
	}

	// 3. containsKey("100")
	frame.OperandStack.PushRef(htObj)
	frame.OperandStack.PushRef("100")
	vmCtx.handleNativeCall("java/util/Hashtable", "containsKey", "(Ljava/lang/Object;)Z", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != 1 {
		t.Fatal("expected containsKey('100') = 1")
	}

	// 4. get("100")
	frame.OperandStack.PushRef(htObj)
	frame.OperandStack.PushRef("100")
	vmCtx.handleNativeCall("java/util/Hashtable", "get", "(Ljava/lang/Object;)Ljava/lang/Object;", frame, OpInvokevirtual)
	got := frame.OperandStack.PopRef()
	if got != "Sprite100" {
		t.Fatalf("expected 'Sprite100', got %v", got)
	}

	// 5. keys() -> Enumeration
	frame.OperandStack.PushRef(htObj)
	vmCtx.handleNativeCall("java/util/Hashtable", "keys", "()Ljava/util/Enumeration;", frame, OpInvokevirtual)
	enumRef := frame.OperandStack.PopRef()
	enumObj, ok := enumRef.(*Object)
	if !ok || enumObj == nil {
		t.Fatalf("expected Object with EnumerationData, got %T", enumRef)
	}

	// 6. hasMoreElements()
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("java/util/Enumeration", "hasMoreElements", "()Z", frame, OpInvokeinterface)
	if frame.OperandStack.PopInt() != 1 {
		t.Fatal("expected hasMoreElements() = 1")
	}

	// 7. nextElement()
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("java/util/Enumeration", "nextElement", "()Ljava/lang/Object;", frame, OpInvokeinterface)
	k := frame.OperandStack.PopRef()
	if k != "100" {
		t.Fatalf("expected key '100', got %v", k)
	}
}
