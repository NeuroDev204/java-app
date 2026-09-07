# Knight Online J2ME Hashtable & RMS Cache Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Resolve missing render graphics and the permanently stuck "Đang lấy dữ liệu" modal dialog in Knight Online J2ME by implementing `java.util.Hashtable` and `java.util.Enumeration` VM natives, enforcing MIDP `RecordStore` exception and nil-normalization semantics, preloading the 90 embedded RMS graphics stores into `VMContext`, and empirically verifying gameplay on the Android emulator.

**Architecture:** 
- `pkg/vm/hashtable.go`: Provides thread-safe `HashtableData` (`sync.RWMutex`) and `EnumerationData` (`sync.Mutex`) backing `java.util.Hashtable` and `java.util.Enumeration`.
- `pkg/vm/interpreter.go`: Intercepts `java/util/Hashtable` and `java/util/Enumeration` method calls; enforces `RecordStoreNotFoundException` and `InvalidRecordIDException` on RMS cache misses.
- `pkg/vm/frame.go`: Normalizes nil typed slices (`[]byte(nil)`) in `OperandStack.PushRef` to untyped `nil` so JVM `OpIfnull` behaves correctly.
- `pkg/vm/default_rms.json` & `pkg/vm/rms_embed.go`: Embeds `rms_data.json` via `//go:embed` to preload NPC, tree, item, and effect sprites directly into `ctx.RecordStores` on startup.

**Tech Stack:** Go 1.24, gomobile, Flutter 3.47 / Dart, Android SDK, J2ME bytecode interpreter, JVM MIDP specification.

## Global Constraints
- Strictly adhere to `AGENTS.md` (no guesswork, zero compiler errors, 100% test pass).
- Thread safety: `HashtableData` and `EnumerationData` must be thread-safe across concurrent Go goroutines running Java threads (`h.run()`, `ax.run()`, `al.run()`, UI event loop).
- Retain `IsIdentityXOR` architecture where Go `SocketTap` manages stream encryption/decryption while Java XOR methods return identity bytes.
- Do not introduce external dependencies; use Go standard library (`sync`, `encoding/json`, `embed`).

---

### Task 1: Implement `pkg/vm/hashtable.go` with `HashtableData` and `EnumerationData`

**Files:**
- Create: `pkg/vm/hashtable.go`
- Test: `pkg/vm/hashtable_test.go`

**Interfaces:**
- Produces:
  ```go
  type HashtableData struct { ... }
  func NewHashtableData() *HashtableData
  func (h *HashtableData) Put(key, val interface{}) interface{}
  func (h *HashtableData) Get(key interface{}) interface{}
  func (h *HashtableData) ContainsKey(key interface{}) bool
  func (h *HashtableData) Contains(val interface{}) bool
  func (h *HashtableData) Remove(key interface{}) interface{}
  func (h *HashtableData) Clear()
  func (h *HashtableData) Size() int
  func (h *HashtableData) IsEmpty() bool
  func (h *HashtableData) Keys() []interface{}
  func (h *HashtableData) Elements() []interface{}

  type EnumerationData struct { ... }
  func NewEnumerationData(items []interface{}) *EnumerationData
  func (e *EnumerationData) HasMoreElements() bool
  func (e *EnumerationData) NextElement() interface{}
  ```

- [x] **Step 1: Write failing tests in `pkg/vm/hashtable_test.go`**

```go
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
```

- [x] **Step 2: Run test to verify failure**
Run: `go test -v ./pkg/vm -run "TestHashtable|TestEnumeration"`
Expected: Compilation failure due to missing types.

- [x] **Step 3: Implement `pkg/vm/hashtable.go`**

```go
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
```

- [x] **Step 4: Run tests to verify pass**
Run: `go test -v ./pkg/vm -run "TestHashtable|TestEnumeration"`
Expected: PASS.

- [x] **Step 5: Commit**
```bash
git add pkg/vm/hashtable.go pkg/vm/hashtable_test.go
git commit -m "feat(vm): implement thread-safe HashtableData and EnumerationData"
```

---

### Task 2: Dispatch `java/util/Hashtable` and `java/util/Enumeration` in `interpreter.go`

**Files:**
- Modify: `pkg/vm/hashtable.go` (add native dispatch method `handleHashtableNative`, `handleEnumerationNative`)
- Modify: `pkg/vm/interpreter.go` (route in `handleNativeCall`)
- Test: `pkg/vm/hashtable_test.go` (test native invocations via Frame stack)

**Interfaces:**
- Consumes: `*Frame`, `*VMContext`, `HashtableData`, `EnumerationData`
- Produces: Complete native dispatch for `java/util/Hashtable` and `java/util/Enumeration`

- [x] **Step 1: Write integration test in `pkg/vm/hashtable_test.go`**

```go
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
```

- [x] **Step 2: Run test to verify failure**
Run: `go test -v ./pkg/vm -run TestHashtableNativeDispatch`
Expected: FAIL.

- [x] **Step 3: Implement dispatch in `pkg/vm/hashtable.go` and wire in `pkg/vm/interpreter.go`**

In `pkg/vm/hashtable.go`:
```go
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
```

In `pkg/vm/interpreter.go` inside `handleNativeCall`:
```go
	if clsName == "java/util/Hashtable" && ctx.handleHashtableNative(mName, mDesc, frame, opcode) {
		return
	}
	if clsName == "java/util/Enumeration" && ctx.handleEnumerationNative(mName, mDesc, frame, opcode) {
		return
	}
```

- [x] **Step 4: Run tests to verify pass**
Run: `go test -v ./pkg/vm -run TestHashtableNativeDispatch`
Expected: PASS.

- [x] **Step 5: Commit**
```bash
git add pkg/vm/hashtable.go pkg/vm/interpreter.go pkg/vm/hashtable_test.go
git commit -m "feat(vm): route java.util.Hashtable and java.util.Enumeration natives"
```

---

### Task 3: Enforce `RecordStore` (RMS) Standard Exceptions & Nil-Normalization

**Files:**
- Modify: `pkg/vm/frame.go` (normalize nil slices in `PushRef`)
- Modify: `pkg/vm/interpreter.go` (throw `RecordStoreNotFoundException` & `InvalidRecordIDException`)
- Test: `pkg/vm/rms_test.go`

**Interfaces:**
- Produces: Correct JVM exception throwing on cache miss and accurate nil representation for bytecode `ifnull`.

- [x] **Step 1: Write failing test in `pkg/vm/rms_test.go`**

```go
package vm

import (
	"testing"
)

func TestRecordStoreNotFoundExceptionWhenNoCreate(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(8)}

	// openRecordStore("nonexistent_store", false)
	frame.OperandStack.PushRef("nonexistent_store")
	frame.OperandStack.PushInt(0) // createIfNecessary = false
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "openRecordStore", "(Ljava/lang/String;Z)Ljavax/microedition/rms/RecordStore;", frame, OpInvokestatic)

	if frame.PendingException == nil {
		t.Fatal("expected PendingException for non-existent store with create=false, got nil")
	}
	if frame.PendingException.ClassName != "javax/microedition/rms/RecordStoreNotFoundException" {
		t.Fatalf("expected RecordStoreNotFoundException, got %s", frame.PendingException.ClassName)
	}
}

func TestRecordStoreInvalidRecordIDException(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	storeObj := &Object{Extra: &RecordStoreData{Records: make([][]byte, 0)}}

	// getRecord(1) on empty store
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(storeObj)
	frame.OperandStack.PushInt(1) // recordId 1
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "getRecord", "(I)[B", frame, OpInvokevirtual)

	if frame.PendingException == nil {
		t.Fatal("expected PendingException for invalid record ID 1 on empty store, got nil")
	}
	if frame.PendingException.ClassName != "javax/microedition/rms/InvalidRecordIDException" {
		t.Fatalf("expected InvalidRecordIDException, got %s", frame.PendingException.ClassName)
	}
}

func TestOperandStackPushRefNormalizesNilSlice(t *testing.T) {
	stack := NewOperandStack(4)
	var nilBytes []byte = nil
	stack.PushRef(nilBytes)

	popped := stack.PopRef()
	if popped != nil {
		t.Fatalf("expected untyped nil interface, got %T(%v)", popped, popped)
	}
}
```

- [x] **Step 2: Run test to verify failure**
Run: `go test -v ./pkg/vm -run "TestRecordStoreNotFound|TestRecordStoreInvalid|TestOperandStackPushRefNormalizesNil"`
Expected: FAIL.

- [x] **Step 3: Update `pkg/vm/frame.go` and `pkg/vm/interpreter.go`**

In `pkg/vm/frame.go`:
```go
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
```

In `pkg/vm/interpreter.go`:
```go
		if mName == "openRecordStore" {
			createIfNecessary := frame.OperandStack.PopInt()
			nameRef := frame.OperandStack.PopRef()
			name := "default"
			if s, ok := nameRef.(string); ok {
				name = s
			} else if obj, ok := nameRef.(*Object); ok {
				name = extractString(obj)
			}
			store, exists := ctx.RecordStores[name]
			if !exists {
				if createIfNecessary == 0 {
					frame.PendingException = ctx.NewException("javax/microedition/rms/RecordStoreNotFoundException", "RecordStore not found: "+name)
					return
				}
				store = &RecordStoreData{Records: make([][]byte, 0)}
				ctx.RecordStores[name] = store
			}
			log.Printf("[RMS] Opened RecordStore '%s' (records=%d)", name, len(store.Records))
			frame.OperandStack.PushRef(&Object{Extra: store})
			return
```
And in `getRecord`:
```go
				recID := frame.OperandStack.PopInt()
				rsObj := frame.OperandStack.PopRef()
				if obj, ok := rsObj.(*Object); ok && obj != nil {
					if store, ok := obj.Extra.(*RecordStoreData); ok && store != nil {
						idx := int(recID) - 1
						if idx >= 0 && idx < len(store.Records) {
							result := store.Records[idx]
							log.Printf("[RMS] getRecord #%d -> %d bytes", recID, len(result))
							if len(result) == 0 {
								frame.OperandStack.PushRef(nil)
							} else {
								frame.OperandStack.PushRef(result)
							}
							return
						}
					}
				}
				frame.PendingException = ctx.NewException("javax/microedition/rms/InvalidRecordIDException", "Invalid record id")
				return
```

- [x] **Step 4: Run tests to verify pass**
Run: `go test -v ./pkg/vm -run "TestRecordStoreNotFound|TestRecordStoreInvalid|TestOperandStackPushRefNormalizesNil"`
Expected: PASS.

- [x] **Step 5: Run existing VM test suite to ensure 0 regressions**
Run: `go test -v ./pkg/vm`
Expected: ALL PASS.

- [x] **Step 6: Commit**
```bash
git add pkg/vm/frame.go pkg/vm/interpreter.go pkg/vm/rms_test.go
git commit -m "fix(vm): enforce RMS exceptions on cache misses and normalize nil slice refs"
```

---

### Task 4: Embed `default_rms.json` and Preload Assets on `NewVMContext`

**Files:**
- Create: `pkg/vm/default_rms.json` (copy from `java-app/rms_data.json`)
- Create: `pkg/vm/rms_embed.go`
- Test: `pkg/vm/rms_embed_test.go`
- Modify: `pkg/vm/interpreter.go` (call `ctx.loadDefaultRMS()`)

**Interfaces:**
- Produces: Automatic preloading of all 90 standard game texture and sprite stores on startup.

- [x] **Step 1: Copy `rms_data.json` to `pkg/vm/default_rms.json`**
Run: `cp rms_data.json pkg/vm/default_rms.json`

- [x] **Step 2: Write test in `pkg/vm/rms_embed_test.go`**

```go
package vm

import (
	"testing"
)

func TestEmbeddedRMSPreloaded(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	if vmCtx.RecordStores == nil {
		t.Fatal("expected RecordStores to be non-nil")
	}

	requiredStores := []string{"image3005", "image3023", "image3029", "image3044"}
	for _, name := range requiredStores {
		store, exists := vmCtx.RecordStores[name]
		if !exists {
			t.Fatalf("expected preloaded store %s to exist", name)
		}
		if len(store.Records) == 0 {
			t.Fatalf("expected store %s to contain records, got 0", name)
		}
		if len(store.Records[0]) == 0 {
			t.Fatalf("expected store %s record 0 to contain bytes, got 0", name)
		}
	}
}
```

- [x] **Step 3: Implement `pkg/vm/rms_embed.go`**

```go
package vm

import (
	_ "embed"
	"encoding/json"
	"log"
)

//go:embed default_rms.json
var defaultRMSData []byte

type jsonRMSRecord struct {
	ID   int    `json:"id"`
	Data []byte `json:"data"` // encoding/json automatically base64-decodes []byte
}

type jsonRMSStore struct {
	Name    string          `json:"name"`
	Records []jsonRMSRecord `json:"records"`
}

func (ctx *VMContext) loadDefaultRMS() {
	if len(defaultRMSData) == 0 {
		return
	}
	var stores []jsonRMSStore
	if err := json.Unmarshal(defaultRMSData, &stores); err != nil {
		log.Printf("[RMS-PRELOAD] failed to unmarshal embedded default_rms.json: %v", err)
		return
	}

	count := 0
	for _, s := range stores {
		rs := &RecordStoreData{Records: make([][]byte, len(s.Records))}
		for _, rec := range s.Records {
			idx := rec.ID - 1
			if idx >= 0 && idx < len(rs.Records) {
				rs.Records[idx] = rec.Data
			} else {
				rs.Records = append(rs.Records, rec.Data)
			}
		}
		ctx.RecordStores[s.Name] = rs
		count++
	}
	log.Printf("[RMS-PRELOAD] Preloaded %d record stores from embedded default_rms.json", count)
}
```

- [x] **Step 4: Invoke `loadDefaultRMS` in `NewVMContext` in `pkg/vm/interpreter.go`**
```go
	ctx.loadDefaultRMS()
```

- [x] **Step 5: Run tests to verify pass**
Run: `go test -v ./pkg/vm -run TestEmbeddedRMSPreloaded`
Expected: PASS with log: `[RMS-PRELOAD] Preloaded 90 record stores from embedded default_rms.json`.

- [x] **Step 6: Commit**
```bash
git add pkg/vm/default_rms.json pkg/vm/rms_embed.go pkg/vm/rms_embed_test.go pkg/vm/interpreter.go
git commit -m "feat(vm): preload 90 embedded RMS graphics stores on VM startup"
```

---

### Task 5: Build AAR, Deploy to Android Emulator & Empirically Verify

**Files:**
- Recompile: `java-app/mobile/android/app/libs/gamebridge-runtime.aar`
- Build & Deploy: `flutter run` / `flutter build apk`
- Log inspection: `adb logcat` / `/tmp/knight-render-trace.log`

- [x] **Step 1: Run full Go test suite**
Run: `cd /home/neuro/Documents/Tools/java-app && go test -v ./pkg/...`
Confirm: 100% PASS across `pkg/vm`, `pkg/midp`, `pkg/gamebridge`.

- [x] **Step 2: Run `go vet`**
Run: `cd /home/neuro/Documents/Tools/java-app && go vet ./pkg/...`
Confirm: 0 errors.

- [x] **Step 3: Recompile gomobile AAR**
Run: `cd /home/neuro/Documents/Tools/java-app/mobile && ./build_aar.sh` (or `gomobile bind -target=android`)
Confirm: `gamebridge-runtime.aar` updated with exit code 0.

- [x] **Step 4: Build and reinstall Android application**
Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter build apk --debug && adb install -r build/app/outputs/flutter-apk/app-debug.apk`

- [x] **Step 5: Launch app and capture logcat trace**
Run: `adb shell am start -n com.example.knight_online/.MainActivity`
Log in to character and enter world.

- [x] **Step 6: Verify empirical criteria**
1. Check `adb logcat -s GoLog`:
   - `[RMS-PRELOAD] Preloaded 90 record stores...` logged.
   - `[RMS] Opened RecordStore 'image3029' (records=1)` hits preloaded records.
   - `[IMAGE-BYTES-FALLBACK] ... dataBytes len=0` drops to 0.
   - `[SOCK-WRITE]` packet flood ceases; connection remains active.
2. Screencap verification:
   - Take screenshot with `adb exec-out screencap -p > /tmp/knight-verified.png`.
   - Confirm NPC and tree sprites render clearly.
   - Tap NPC "Mrs Oda" / "Aman" and verify dialogue / quest menu opens successfully without freezing on "Đang lấy dữ liệu".

- [x] **Step 7: Commit any mobile build updates**
```bash
git add .
git commit -m "fix(mobile): integrate hashtable and rms preloaded assets for full rendering"
```
