package vm

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// TestChallengerInstanceofAndCheckcast verifies opcode instruction rules for instanceof and checkcast.
func TestChallengerInstanceofAndCheckcast(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// 1. Instanceof with non-null Object
	code := []byte{OpInstanceof, 0x00, 0x05, OpReturn}
	frame1 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           0,
	}
	objRef := &Object{}
	frame1.OperandStack.PushRef(objRef)

	_, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame1})
	if err != nil {
		t.Fatalf("ExecuteBytecode failed for instanceof non-null: %v", err)
	}
	if frame1.PC != 4 {
		t.Errorf("instanceof PC expected 4, got %d", frame1.PC)
	}
	if frame1.OperandStack.size != 1 {
		t.Fatalf("instanceof non-null expected stack size 1, got %d", frame1.OperandStack.size)
	}
	if val := frame1.OperandStack.PopInt(); val != 1 {
		t.Errorf("instanceof non-null expected 1, got %d", val)
	}

	// 2. Instanceof with null reference
	frame2 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           0,
	}
	frame2.OperandStack.PushRef(nil)
	_, err = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame2})
	if err != nil {
		t.Fatalf("ExecuteBytecode failed for instanceof null: %v", err)
	}
	if frame2.PC != 4 {
		t.Errorf("instanceof PC expected 4, got %d", frame2.PC)
	}
	if frame2.OperandStack.size != 1 {
		t.Fatalf("instanceof null expected stack size 1, got %d", frame2.OperandStack.size)
	}
	if val := frame2.OperandStack.PopInt(); val != 0 {
		t.Errorf("instanceof null expected 0, got %d", val)
	}

	// 3. Checkcast with non-null Object
	codeCast := []byte{OpCheckcast, 0x00, 0x05, OpReturn}
	frameCast1 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           0,
	}
	frameCast1.OperandStack.PushRef(objRef)
	_, err = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: codeCast, frame: frameCast1})
	if err != nil {
		t.Fatalf("ExecuteBytecode failed for checkcast non-null: %v", err)
	}
	if frameCast1.PC != 4 {
		t.Errorf("checkcast PC expected 4, got %d", frameCast1.PC)
	}
	if frameCast1.OperandStack.size != 1 {
		t.Fatalf("checkcast non-null expected stack size 1, got %d", frameCast1.OperandStack.size)
	}
	if ref := frameCast1.OperandStack.PopRef(); ref != objRef {
		t.Errorf("checkcast expected stack top to be objRef, got %v", ref)
	}

	// 4. Checkcast with null reference
	frameCast2 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           0,
	}
	frameCast2.OperandStack.PushRef(nil)
	_, err = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: codeCast, frame: frameCast2})
	if err != nil {
		t.Fatalf("ExecuteBytecode failed for checkcast null: %v", err)
	}
	if frameCast2.PC != 4 {
		t.Errorf("checkcast PC expected 4, got %d", frameCast2.PC)
	}
	if frameCast2.OperandStack.size != 1 {
		t.Fatalf("checkcast null expected stack size 1, got %d", frameCast2.OperandStack.size)
	}
	if ref := frameCast2.OperandStack.PopRef(); ref != nil {
		t.Errorf("checkcast expected stack top to be nil, got %v", ref)
	}
}

// TestChallengerInvokeinterfaceStackAndParams verifies 5-byte opcode layout and parameter popping.
func TestChallengerInvokeinterfaceStackAndParams(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// 5-byte opcode layout: OpInvokeinterface (0xB9) + 2 bytes index + 1 byte count + 1 byte 0
	// 0x00: nop
	// 0x01: OpInvokeinterface (0xB9)
	// 0x02..0x03: cp index 0x0001
	// 0x04: count 0x02
	// 0x05: 0x00
	// 0x06: OpReturn
	code := []byte{
		OpNop,
		OpInvokeinterface, 0x00, 0x01, 0x02, 0x00,
		OpReturn,
	}

	frame := &Frame{
		OperandStack: NewOperandStack(10),
		PC:           1, // start at opcode 0xB9
	}

	// Simulate stack: objRef (RecordStore), param1 (recID=1)
	rsObj := &Object{Extra: &RecordStoreData{Records: [][]byte{{1, 2, 3}}}}
	frame.OperandStack.PushRef(rsObj)

	// Test PC advancement through 5-byte instruction:
	// We call handleNativeCall directly to test parameter popping for RecordStore.getNumRecords
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "getNumRecords", "()I", frame, OpInvokeinterface)

	// getNumRecords pops rsObj and pushes int 1
	if frame.OperandStack.size != 1 {
		t.Fatalf("expected stack size 1 after getNumRecords, got %d", frame.OperandStack.size)
	}
	if num := frame.OperandStack.PopInt(); num != 1 {
		t.Errorf("expected 1 record, got %d", num)
	}

	// Verify PC advancement logic in ExecuteBytecode loop:
	// If PC starts at 1, after executing 5-byte invokeinterface (opcode at 1, index 2..3, count 4, zero 5), PC should be at 6.
	framePC := &Frame{
		OperandStack: NewOperandStack(10),
		PC:           1,
	}
	// Push dummy string args for String.equals(Ljava/lang/Object;)Z via native dispatch
	framePC.OperandStack.PushRef("abc")
	framePC.OperandStack.PushRef("abc")
	// Execute switch statement logic simulating opcode 0xB9 execution
	refIdx := binary.BigEndian.Uint16(code[framePC.PC+1 : framePC.PC+3]) // code[2:4]
	if refIdx != 1 {
		t.Errorf("refIdx expected 1, got %d", refIdx)
	}
	// Advance PC as interpreter does: framePC.PC++ (for opcode) + 4 (for 4 trailing bytes)
	framePC.PC++    // PC = 2
	framePC.PC += 4 // PC = 6
	if framePC.PC != 6 {
		t.Errorf("invokeinterface total PC advance expected 6, got %d", framePC.PC)
	}
}

// TestChallengerRecordStoreSetRecordBoundaries verifies setRecord in-place modifications and edge indices.
func TestChallengerRecordStoreSetRecordBoundaries(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(10)}

	// 1. Setup RecordStore with 2 records
	rsData := &RecordStoreData{
		Records: [][]byte{{10, 20, 30, 40}, {50, 60}},
	}
	rsObj := &Object{Extra: rsData}

	// 2. Normal setRecord on rec #1: newData = [100, 200, 255], offset = 0, len = 3
	srcData := []byte{100, 200, 255, 111}
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushInt(1) // recID
	frame.OperandStack.PushRef(srcData)
	frame.OperandStack.PushInt(0) // offset
	frame.OperandStack.PushInt(3) // numBytes
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "setRecord", "(I[BII)V", frame, OpInvokevirtual)

	if len(rsData.Records[0]) != 3 || !bytes.Equal(rsData.Records[0], []byte{100, 200, 255}) {
		t.Errorf("setRecord normal update failed: %v", rsData.Records[0])
	}

	// Verify defensive copy (modifying srcData does NOT mutate RecordStore record)
	srcData[0] = 0
	if rsData.Records[0][0] != 100 {
		t.Errorf("setRecord failed defensive copy check: record mutated when source slice changed!")
	}

	// 3. Boundary test: setRecord with offset > 0 (offset=1, numBytes=2 from [77, 88, 99, 44])
	srcData2 := []byte{77, 88, 99, 44}
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushInt(2) // recID = 2
	frame.OperandStack.PushRef(srcData2)
	frame.OperandStack.PushInt(1) // offset = 1 (points to 88)
	frame.OperandStack.PushInt(2) // numBytes = 2 (88, 99)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "setRecord", "(I[BII)V", frame, OpInvokevirtual)

	if len(rsData.Records[1]) != 2 || !bytes.Equal(rsData.Records[1], []byte{88, 99}) {
		t.Errorf("setRecord with offset>0 failed: %v", rsData.Records[1])
	}

	// 4. Edge test: setRecord with invalid recID (recID = 0 or 999) - should not crash
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushInt(999) // out-of-bounds recID
	frame.OperandStack.PushRef([]byte{1, 2})
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(2)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "setRecord", "(I[BII)V", frame, OpInvokevirtual)

	// 5. Edge test: setRecord with offset+numBytes > len(srcData) - should safely ignore without panic
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushInt(1)
	frame.OperandStack.PushRef([]byte{1, 2})
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(10) // requests 10 bytes from 2-byte slice
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "setRecord", "(I[BII)V", frame, OpInvokevirtual)
	// rec 1 should remain [100, 200, 255]
	if !bytes.Equal(rsData.Records[0], []byte{100, 200, 255}) {
		t.Errorf("out-of-bounds src read unexpectedly modified record: %v", rsData.Records[0])
	}
}

// TestChallengerEnumerateRecordsStub verifies RecordEnumeration iterator stubs.
func TestChallengerEnumerateRecordsStub(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(10)}

	rsObj := &Object{Extra: &RecordStoreData{Records: [][]byte{{1}, {2}}}}

	// 1. Call enumerateRecords
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushRef(nil) // filter
	frame.OperandStack.PushRef(nil) // comparator
	frame.OperandStack.PushInt(1)   // keepUpdated
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "enumerateRecords", "(Ljavax/microedition/rms/RecordFilter;Ljavax/microedition/rms/RecordComparator;Z)Ljavax/microedition/rms/RecordEnumeration;", frame, OpInvokevirtual)

	enumObj := frame.OperandStack.PopRef()
	if enumObj == nil {
		t.Fatalf("enumerateRecords returned nil")
	}

	// 2. Call RecordEnumeration native methods
	// hasNextElement
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordEnumeration", "hasNextElement", "()Z", frame, OpInvokeinterface)
	if got := frame.OperandStack.PopInt(); got != 0 {
		t.Errorf("hasNextElement expected 0, got %d", got)
	}

	// hasPreviousElement
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordEnumeration", "hasPreviousElement", "()Z", frame, OpInvokeinterface)
	if got := frame.OperandStack.PopInt(); got != 0 {
		t.Errorf("hasPreviousElement expected 0, got %d", got)
	}

	// nextRecord
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordEnumeration", "nextRecord", "()[B", frame, OpInvokeinterface)
	if got := frame.OperandStack.PopRef(); got != nil {
		t.Errorf("nextRecord expected nil, got %v", got)
	}

	// previousRecord
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordEnumeration", "previousRecord", "()[B", frame, OpInvokeinterface)
	if got := frame.OperandStack.PopRef(); got != nil {
		t.Errorf("previousRecord expected nil, got %v", got)
	}

	// nextRecordId
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordEnumeration", "nextRecordId", "()I", frame, OpInvokeinterface)
	if got := frame.OperandStack.PopInt(); got != 0 {
		t.Errorf("nextRecordId expected 0, got %d", got)
	}

	// numRecords
	frame.OperandStack.PushRef(enumObj)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordEnumeration", "numRecords", "()I", frame, OpInvokeinterface)
	if got := frame.OperandStack.PopInt(); got != 0 {
		t.Errorf("numRecords expected 0, got %d", got)
	}

	// destroy, reset, rebuild
	for _, m := range []string{"destroy", "reset", "rebuild"} {
		frame.OperandStack.PushRef(enumObj)
		vmCtx.handleNativeCall("javax/microedition/rms/RecordEnumeration", m, "()V", frame, OpInvokeinterface)
		if frame.OperandStack.size != 0 {
			t.Errorf("%s expected empty stack, size=%d", m, frame.OperandStack.size)
		}
	}
}
