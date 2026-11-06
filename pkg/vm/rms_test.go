package vm

import (
	"bytes"
	"strings"
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
	if frame.PendingException.Class == nil || frame.PendingException.Class.GetClassName() != "javax/microedition/rms/RecordStoreNotFoundException" {
		t.Fatalf("expected RecordStoreNotFoundException, got %v", frame.PendingException)
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
	if frame.PendingException.Class == nil || frame.PendingException.Class.GetClassName() != "javax/microedition/rms/InvalidRecordIDException" {
		t.Fatalf("expected InvalidRecordIDException, got %v", frame.PendingException)
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

func TestRMSGetRecordDefensiveCopy(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	originalData := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	storeObj := &Object{Extra: &RecordStoreData{
		Records: [][]byte{
			{0xDE, 0xAD, 0xBE, 0xEF},
		},
	}}

	// First getRecord(1) call
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef(storeObj)
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "getRecord", "(I)[B", frame, OpInvokevirtual)

	resRef := frame.OperandStack.PopRef()
	bytes1, ok := resRef.([]byte)
	if !ok || !bytes.Equal(bytes1, originalData) {
		t.Fatalf("expected %v, got %v", originalData, resRef)
	}

	// Mutate the returned slice (simulating Java XOR loop)
	bytes1[0] ^= 0xFF
	bytes1[1] ^= 0xFF
	bytes1[2] ^= 0xFF
	bytes1[3] ^= 0xFF

	// Second getRecord(1) call
	frame.OperandStack.PushRef(storeObj)
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "getRecord", "(I)[B", frame, OpInvokevirtual)

	resRef2 := frame.OperandStack.PopRef()
	bytes2, ok := resRef2.([]byte)
	if !ok || !bytes.Equal(bytes2, originalData) {
		t.Fatalf("defensive copy failed: expected original data %v, got mutated %v", originalData, bytes2)
	}
}

func TestRMSListRecordStores(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	vmCtx.RecordStores = make(map[string]*RecordStoreData)
	frame := &Frame{OperandStack: NewOperandStack(8)}

	// 1. Empty RecordStores should return nil
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "listRecordStores", "()[Ljava/lang/String;", frame, OpInvokestatic)
	res := frame.OperandStack.PopRef()
	if res != nil {
		t.Fatalf("expected nil when no stores exist, got %v", res)
	}

	// 2. Populate stores
	vmCtx.RecordStores["store_b"] = &RecordStoreData{Records: [][]byte{{1}}}
	vmCtx.RecordStores["store_a"] = &RecordStoreData{Records: [][]byte{{2}}}
	vmCtx.RecordStores["store_c"] = &RecordStoreData{Records: [][]byte{{3}}}

	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "listRecordStores", "()[Ljava/lang/String;", frame, OpInvokestatic)
	res = frame.OperandStack.PopRef()
	arr, ok := res.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", res)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 stores, got %d", len(arr))
	}

	storeSet := make(map[string]bool)
	for _, item := range arr {
		str := extractString(item)
		storeSet[str] = true
	}
	for _, expected := range []string{"store_a", "store_b", "store_c"} {
		if !storeSet[expected] {
			t.Errorf("expected store '%s' in listRecordStores, but was missing", expected)
		}
	}
}

func TestRMSDeleteRecordStore(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	vmCtx.RecordStores["test_store"] = &RecordStoreData{Records: [][]byte{{1, 2, 3}}}

	frame := &Frame{OperandStack: NewOperandStack(8)}

	// 1. Delete existing store
	frame.OperandStack.PushRef("test_store")
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "deleteRecordStore", "(Ljava/lang/String;)V", frame, OpInvokestatic)

	if frame.PendingException != nil {
		t.Fatalf("unexpected exception deleting existing store: %v", frame.PendingException)
	}
	if _, exists := vmCtx.RecordStores["test_store"]; exists {
		t.Fatalf("store 'test_store' should have been deleted from vmCtx.RecordStores")
	}

	// 2. Delete non-existent store should throw RecordStoreNotFoundException
	frame.OperandStack.PushRef("test_store")
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "deleteRecordStore", "(Ljava/lang/String;)V", frame, OpInvokestatic)

	if frame.PendingException == nil {
		t.Fatal("expected RecordStoreNotFoundException when deleting non-existent store, got nil")
	}
	if frame.PendingException.Class == nil || frame.PendingException.Class.GetClassName() != "javax/microedition/rms/RecordStoreNotFoundException" {
		t.Fatalf("expected RecordStoreNotFoundException, got %v", frame.PendingException)
	}
}

func TestDefaultRMSImagesAllXORedPNG(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	if vmCtx.RecordStores == nil {
		t.Fatal("expected RecordStores to be non-nil")
	}

	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	imageStoreCount := 0

	for name, store := range vmCtx.RecordStores {
		if !strings.HasPrefix(name, "image") {
			continue
		}
		imageStoreCount++
		if len(store.Records) == 0 {
			t.Errorf("store %s has 0 records", name)
			continue
		}
		for recIdx, recData := range store.Records {
			if len(recData) < len(pngMagic) {
				t.Errorf("store %s rec %d has length %d < %d", name, recIdx+1, len(recData), len(pngMagic))
				continue
			}
			// Decrypt record by XORing with 0xFF
			decrypted := make([]byte, len(recData))
			for i, b := range recData {
				decrypted[i] = b ^ 0xFF
			}
			if !bytes.Equal(decrypted[:len(pngMagic)], pngMagic) {
				t.Errorf("store %s rec %d failed PNG magic check: decrypted header %X, expected %X",
					name, recIdx+1, decrypted[:len(pngMagic)], pngMagic)
			}
		}
	}

	if imageStoreCount != 0 {
		t.Fatalf("expected no stale dynamic image stores in the runtime preload, got %d", imageStoreCount)
	}
}
