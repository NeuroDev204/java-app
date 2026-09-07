package vm

import (
	"encoding/binary"
	"fmt"
	"testing"

	"java-app/pkg/classfile"
)

func TestTableswitch(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// Build tableswitch instruction code:
	// PC 0: nop (0x00) -> startPC = 1
	// PC 1: tableswitch (0xAA)
	// (1 + 1 + pad) % 4 == 0 -> pad = 2 (PC 2, 3)
	// PC 4..7: defaultOffset = 34 -> target PC 35
	// PC 8..11: low = 1
	// PC 12..15: high = 3
	// PC 16..19: offset for 1 = 28 -> target PC 29
	// PC 20..23: offset for 2 = 30 -> target PC 31
	// PC 24..27: offset for 3 = 32 -> target PC 33

	code := make([]byte, 40)
	code[0] = OpNop
	code[1] = OpTableswitch

	binary.BigEndian.PutUint32(code[4:8], 34)  // defaultOffset -> target PC 35
	binary.BigEndian.PutUint32(code[8:12], 1)  // low
	binary.BigEndian.PutUint32(code[12:16], 3) // high
	binary.BigEndian.PutUint32(code[16:20], 28)
	binary.BigEndian.PutUint32(code[20:24], 30)
	binary.BigEndian.PutUint32(code[24:28], 32)

	code[29] = OpReturn
	code[31] = OpReturn // Target PC for index 2
	code[33] = OpReturn
	code[35] = OpReturn // Target PC for default offset

	// Case A: Index 2 -> jump to startPC + 30 = 31, hits OpReturn, PC becomes 32
	frame1 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           1,
	}
	frame1.OperandStack.PushInt(2)
	_, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame1})
	_ = err
	if frame1.PC != 32 {
		t.Errorf("expected PC 32 after OpReturn at target 31, got %d", frame1.PC)
	}

	// Case B: Index 99 (out of bounds) -> jump to startPC + 34 = 35, hits OpReturn, PC becomes 36
	frame2 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           1,
	}
	frame2.OperandStack.PushInt(99)
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame2})
	if frame2.PC != 36 {
		t.Errorf("expected PC 36 after OpReturn at target 35, got %d", frame2.PC)
	}
}

func TestLookupswitch(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// Build lookupswitch instruction:
	// PC 0: nop -> startPC = 1
	// PC 1: lookupswitch (0xAB)
	// pad = 2 (PC 2, 3)
	// PC 4..7: defaultOffset = 40 -> target PC 41
	// PC 8..11: npairs = 2
	// PC 12..15: match 10, PC 16..19: offset 20 -> target PC 21
	// PC 20..23: match 20, PC 24..27: offset 30 -> target PC 31

	code := make([]byte, 50)
	code[0] = OpNop
	code[1] = OpLookupswitch
	binary.BigEndian.PutUint32(code[4:8], 40)   // defaultOffset
	binary.BigEndian.PutUint32(code[8:12], 2)   // npairs
	binary.BigEndian.PutUint32(code[12:16], 10) // match 10
	binary.BigEndian.PutUint32(code[16:20], 20)
	binary.BigEndian.PutUint32(code[20:24], 20) // match 20
	binary.BigEndian.PutUint32(code[24:28], 30)

	code[31] = OpReturn // Target PC for match 20
	code[41] = OpReturn // Target PC for default offset

	// Key = 20 -> jump to startPC + 30 = 31, hits OpReturn, PC becomes 32
	frame1 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           1,
	}
	frame1.OperandStack.PushInt(20)
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame1})
	if frame1.PC != 32 {
		t.Errorf("expected PC 32 for key 20, got %d", frame1.PC)
	}

	// Key = 77 (no match) -> jump to startPC + 40 = 41, hits OpReturn, PC becomes 42
	frame2 := &Frame{
		OperandStack: NewOperandStack(5),
		PC:           1,
	}
	frame2.OperandStack.PushInt(77)
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame2})
	if frame2.PC != 42 {
		t.Errorf("expected PC 42 for default jump, got %d", frame2.PC)
	}
}

func TestBitwiseAndShiftOpcodes(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	runOp := func(op byte, v1, v2 int32) int32 {
		code := []byte{op, OpIreturn}
		frame := &Frame{
			OperandStack: NewOperandStack(5),
			PC:           0,
		}
		frame.OperandStack.PushInt(v1)
		frame.OperandStack.PushInt(v2)
		res, _ := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
		return res.Num
	}

	if got := runOp(OpIand, 0x0F, 0x33); got != (0x0F & 0x33) {
		t.Errorf("iand expected %d, got %d", 0x0F&0x33, got)
	}
	if got := runOp(OpIor, 0x0F, 0x30); got != (0x0F | 0x30) {
		t.Errorf("ior expected %d, got %d", 0x0F|0x30, got)
	}
	if got := runOp(OpIxor, 0xFF, 0x0F); got != (0xFF ^ 0x0F) {
		t.Errorf("ixor expected %d, got %d", 0xFF^0x0F, got)
	}
	if got := runOp(OpIshl, 1, 4); got != 16 {
		t.Errorf("ishl expected 16, got %d", got)
	}
	if got := runOp(OpIshr, -16, 2); got != -4 {
		t.Errorf("ishr expected -4, got %d", got)
	}
	if got := runOp(OpIushr, -1, 16); got != 0xFFFF {
		t.Errorf("iushr expected 65535, got %d", got)
	}
	if got := runOp(OpIrem, 17, 5); got != 2 {
		t.Errorf("irem expected 2, got %d", got)
	}
}

func TestConditionalBranchOpcodes(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// test single-operand branch: op (1 byte) + offset int16 (2 bytes)
	testBranch1 := func(op byte, val int32, expectBranch bool) {
		code := []byte{op, 0x00, 0x10} // offset = 16
		frame := &Frame{
			OperandStack: NewOperandStack(5),
			PC:           0,
		}
		frame.OperandStack.PushInt(val)
		_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
		expectedPC := 3
		if expectBranch {
			expectedPC = 16
		}
		if frame.PC != expectedPC {
			t.Errorf("op 0x%X val %d expected PC %d, got %d", op, val, expectedPC, frame.PC)
		}
	}

	testBranch1(OpIlt, -5, true)
	testBranch1(OpIlt, 5, false)
	testBranch1(OpIge, 0, true)
	testBranch1(OpIge, -1, false)
	testBranch1(OpIgt, 1, true)
	testBranch1(OpIgt, 0, false)
	testBranch1(OpIle, 0, true)
	testBranch1(OpIle, 1, false)
	testBranch1(OpIeq, 0, true)
	testBranch1(OpIeq, 1, false)
	testBranch1(OpIne, 1, true)
	testBranch1(OpIne, 0, false)

	// test two-operand icmp branch
	testBranch2 := func(op byte, v1, v2 int32, expectBranch bool) {
		code := []byte{op, 0x00, 0x10} // offset = 16
		frame := &Frame{
			OperandStack: NewOperandStack(5),
			PC:           0,
		}
		frame.OperandStack.PushInt(v1)
		frame.OperandStack.PushInt(v2)
		_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
		expectedPC := 3
		if expectBranch {
			expectedPC = 16
		}
		if frame.PC != expectedPC {
			t.Errorf("op 0x%X v1=%d v2=%d expected PC %d, got %d", op, v1, v2, expectedPC, frame.PC)
		}
	}

	testBranch2(OpIfIcmpgt, 10, 5, true)
	testBranch2(OpIfIcmpgt, 5, 5, false)
	testBranch2(OpIfIcmple, 5, 5, true)
	testBranch2(OpIfIcmple, 6, 5, false)
	testBranch2(OpIfIcmpeq, 5, 5, true)
	testBranch2(OpIfIcmpeq, 5, 6, false)
	testBranch2(OpIfIcmpne, 5, 6, true)
	testBranch2(OpIfIcmpne, 5, 5, false)
	testBranch2(OpIfIcmplt, 4, 5, true)
	testBranch2(OpIfIcmplt, 5, 5, false)
	testBranch2(OpIfIcmpge, 5, 5, true)
	testBranch2(OpIfIcmpge, 4, 5, false)

	// test goto branch
	codeGoto := []byte{OpGoto, 0x00, 0x10}
	frameGoto := &Frame{OperandStack: NewOperandStack(5), PC: 0}
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: codeGoto, frame: frameGoto})
	if frameGoto.PC != 16 {
		t.Errorf("OpGoto expected PC 16, got %d", frameGoto.PC)
	}

	// test acmp branch
	refA := &Object{}
	refB := &Object{}
	testAcmp := func(op byte, r1, r2 interface{}, expectBranch bool) {
		code := []byte{op, 0x00, 0x10}
		frame := &Frame{
			OperandStack: NewOperandStack(5),
			PC:           0,
		}
		frame.OperandStack.PushRef(r1)
		frame.OperandStack.PushRef(r2)
		_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
		expectedPC := 3
		if expectBranch {
			expectedPC = 16
		}
		if frame.PC != expectedPC {
			t.Errorf("acmp op 0x%X expected PC %d, got %d", op, expectedPC, frame.PC)
		}
	}

	testAcmp(OpIfAcmpeq, refA, refA, true)
	testAcmp(OpIfAcmpeq, refA, refB, false)
	testAcmp(OpIfAcmpne, refA, refB, true)
	testAcmp(OpIfAcmpne, refA, refA, false)
}

func TestObjectAndInterfaceOpcodes(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// Test instanceof
	code := []byte{OpInstanceof, 0x00, 0x01}
	frame1 := &Frame{OperandStack: NewOperandStack(5), PC: 0}
	frame1.OperandStack.PushRef(&Object{})
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame1})
	if res := frame1.OperandStack.PopInt(); res != 1 {
		t.Errorf("instanceof non-null expected 1, got %d", res)
	}

	frame2 := &Frame{OperandStack: NewOperandStack(5), PC: 0}
	frame2.OperandStack.PushRef(nil)
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame2})
	if res := frame2.OperandStack.PopInt(); res != 0 {
		t.Errorf("instanceof null expected 0, got %d", res)
	}

	// Test checkcast
	codeCast := []byte{OpCheckcast, 0x00, 0x01}
	frameCast := &Frame{OperandStack: NewOperandStack(5), PC: 0}
	obj := &Object{}
	frameCast.OperandStack.PushRef(obj)
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: codeCast, frame: frameCast})
	if ref := frameCast.OperandStack.PopRef(); ref != obj {
		t.Errorf("checkcast expected original obj, got %v", ref)
	}

	// Test monitorenter / monitorexit
	codeMonitor := []byte{OpMonitorenter, OpMonitorexit}
	frameMon := &Frame{OperandStack: NewOperandStack(5), PC: 0}
	frameMon.OperandStack.PushRef(obj)
	frameMon.OperandStack.PushRef(obj)
	_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: codeMonitor, frame: frameMon})
	if frameMon.OperandStack.size != 0 {
		t.Errorf("monitorenter/exit expected empty stack, size=%d", frameMon.OperandStack.size)
	}
}

func TestStringEqualsAndCharAt(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(10)}

	// Test String.equals true
	frame.OperandStack.PushRef("Hello")
	frame.OperandStack.PushRef("Hello")
	vmCtx.handleNativeCall("java/lang/String", "equals", "(Ljava/lang/Object;)Z", frame, OpInvokevirtual)
	if got := frame.OperandStack.PopInt(); got != 1 {
		t.Errorf("String.equals expected 1 for matching strings, got %d", got)
	}

	// Test String.equals false
	frame.OperandStack.PushRef("Hello")
	frame.OperandStack.PushRef("World")
	vmCtx.handleNativeCall("java/lang/String", "equals", "(Ljava/lang/Object;)Z", frame, OpInvokevirtual)
	if got := frame.OperandStack.PopInt(); got != 0 {
		t.Errorf("String.equals expected 0 for non-matching strings, got %d", got)
	}

	// Test String.charAt with UTF-8 runes: "J2ME_🎮_Game" -> charAt(5) is '🎮' (rune 127918)
	unicodeStr := "J2ME_🎮_Game"
	frame.OperandStack.PushRef(unicodeStr)
	frame.OperandStack.PushInt(5) // Index 5 in rune slice
	vmCtx.handleNativeCall("java/lang/String", "charAt", "(I)C", frame, OpInvokevirtual)
	if ch := frame.OperandStack.PopInt(); ch != 127918 {
		t.Errorf("charAt(5) expected rune 127918, got %d", ch)
	}
}

func TestStringIndexOfDescriptors(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(10)}

	// 1. indexOf(Ljava/lang/String;I)I
	frame.OperandStack.PushRef("hello world hello")
	frame.OperandStack.PushRef("hello")
	frame.OperandStack.PushInt(5) // fromIndex
	vmCtx.handleNativeCall("java/lang/String", "indexOf", "(Ljava/lang/String;I)I", frame, OpInvokevirtual)
	if idx := frame.OperandStack.PopInt(); idx != 12 {
		t.Errorf("indexOf(String,int) expected 12, got %d", idx)
	}

	// 2. indexOf(II)I
	frame.OperandStack.PushRef("banana")
	frame.OperandStack.PushInt(int32('a'))
	frame.OperandStack.PushInt(2) // fromIndex
	vmCtx.handleNativeCall("java/lang/String", "indexOf", "(II)I", frame, OpInvokevirtual)
	if idx := frame.OperandStack.PopInt(); idx != 3 {
		t.Errorf("indexOf(int,int) expected 3, got %d", idx)
	}

	// 3. indexOf(Ljava/lang/String;)I
	frame.OperandStack.PushRef("knight online")
	frame.OperandStack.PushRef("online")
	vmCtx.handleNativeCall("java/lang/String", "indexOf", "(Ljava/lang/String;)I", frame, OpInvokevirtual)
	if idx := frame.OperandStack.PopInt(); idx != 7 {
		t.Errorf("indexOf(String) expected 7, got %d", idx)
	}
}

func TestDataStreamReaderReadBytesAndUTF(t *testing.T) {
	data := []byte{
		0x00, 0x05, 'H', 'e', 'l', 'l', 'o', // ReadUTF prefix
		0x01, 0x02, 0x03, 0x04, 0x05, // Extra bytes
	}
	reader := &DataStreamReader{Data: data, Pos: 0}

	// Test ReadUTF
	utfStr := reader.ReadUTF()
	if utfStr != "Hello" {
		t.Errorf("ReadUTF expected 'Hello', got '%s'", utfStr)
	}

	// Test ReadBytes into small buffer (len=3, requested length=5)
	dst := make([]byte, 3)
	n := reader.ReadBytes(dst, 0, 5)
	if n != 3 {
		t.Errorf("ReadBytes into small dst expected 3 bytes read, got %d", n)
	}
	if dst[0] != 0x01 || dst[1] != 0x02 || dst[2] != 0x03 {
		t.Errorf("ReadBytes copied data incorrect: %v", dst)
	}
}

func TestDataStreamReaderReadLong(t *testing.T) {
	reader := &DataStreamReader{Data: []byte{0x01, 0x02, 0x03, 0x04, 0xA5, 0xB6, 0xC7, 0xD8}}

	if got := reader.ReadLong(); got != int64(0x01020304A5B6C7D8) {
		t.Fatalf("ReadLong() = %#x, want %#x", got, int64(0x01020304A5B6C7D8))
	}
	if reader.Pos != 8 {
		t.Fatalf("ReadLong() advanced position to %d, want 8", reader.Pos)
	}
}

func TestDataInputStreamReadLongAndBoolean(t *testing.T) {
	ctx := NewVMContext(nil, nil)
	stream := &Object{Extra: &DataStreamReader{Data: []byte{
		0x01, 0x02, 0x03, 0x04, 0xA5, 0xB6, 0xC7, 0xD8,
		0x00,
		0xFF,
	}}}
	frame := &Frame{OperandStack: NewOperandStack(8)}

	frame.OperandStack.PushRef(stream)
	ctx.handleNativeCall("java/io/DataInputStream", "readLong", "()J", frame, OpInvokevirtual)
	if frame.OperandStack.PopInt() != -1514747944 || frame.OperandStack.PopInt() != 0x01020304 {
		t.Fatal("readLong did not push JVM high/low words in order")
	}

	frame.OperandStack.PushRef(stream)
	ctx.handleNativeCall("java/io/DataInputStream", "readBoolean", "()Z", frame, OpInvokevirtual)
	if got := frame.OperandStack.PopInt(); got != 0 {
		t.Fatalf("readBoolean(false) = %d, want 0", got)
	}

	frame.OperandStack.PushRef(stream)
	ctx.handleNativeCall("java/io/DataInputStream", "readBoolean", "()Z", frame, OpInvokevirtual)
	if got := frame.OperandStack.PopInt(); got != 1 {
		t.Fatalf("readBoolean(true) = %d, want 1", got)
	}
}

func TestRecordStoreSetAndEnumerate(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(10)}

	// 1. openRecordStore
	frame.OperandStack.PushRef("gamestate")
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "openRecordStore", "(Ljava/lang/String;Z)Ljavax/microedition/rms/RecordStore;", frame, OpInvokestatic)
	rsObj := frame.OperandStack.PopRef()
	if rsObj == nil {
		t.Fatalf("openRecordStore returned nil")
	}

	// 2. addRecord
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushRef([]byte{10, 20, 30, 40})
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(4)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "addRecord", "([BII)I", frame, OpInvokevirtual)
	recID := frame.OperandStack.PopInt()
	if recID != 1 {
		t.Fatalf("addRecord expected id 1, got %d", recID)
	}

	// 3. setRecord (update rec 1 to [99, 88])
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushInt(1) // recID
	frame.OperandStack.PushRef([]byte{99, 88})
	frame.OperandStack.PushInt(0)
	frame.OperandStack.PushInt(2)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "setRecord", "(I[BII)V", frame, OpInvokevirtual)

	// 4. getRecord
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "getRecord", "(I)[B", frame, OpInvokevirtual)
	updatedData := frame.OperandStack.PopRef().([]byte)
	if len(updatedData) != 2 || updatedData[0] != 99 || updatedData[1] != 88 {
		t.Errorf("setRecord update failed, got %v", updatedData)
	}

	// 5. enumerateRecords
	frame.OperandStack.PushRef(rsObj)
	frame.OperandStack.PushRef(nil)
	frame.OperandStack.PushRef(nil)
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("javax/microedition/rms/RecordStore", "enumerateRecords", "(Ljavax/microedition/rms/RecordFilter;Ljavax/microedition/rms/RecordComparator;Z)Ljavax/microedition/rms/RecordEnumeration;", frame, OpInvokevirtual)
	enumObj := frame.OperandStack.PopRef()
	if enumObj == nil {
		t.Errorf("enumerateRecords returned nil")
	}
}

func TestStringMethods(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(10)}

	// 1. trim
	frame.OperandStack.PushRef("  hello world  \n")
	vmCtx.handleNativeCall("java/lang/String", "trim", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopRef(); res != "hello world" {
		t.Errorf("expected 'hello world', got '%v'", res)
	}

	// 2. toLowerCase & toUpperCase
	frame.OperandStack.PushRef("Hello World")
	vmCtx.handleNativeCall("java/lang/String", "toLowerCase", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopRef(); res != "hello world" {
		t.Errorf("expected 'hello world', got '%v'", res)
	}
	frame.OperandStack.PushRef("Hello World")
	vmCtx.handleNativeCall("java/lang/String", "toUpperCase", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopRef(); res != "HELLO WORLD" {
		t.Errorf("expected 'HELLO WORLD', got '%v'", res)
	}

	// 3. startsWith & endsWith
	frame.OperandStack.PushRef("abc_def")
	frame.OperandStack.PushRef("abc")
	vmCtx.handleNativeCall("java/lang/String", "startsWith", "(Ljava/lang/String;)Z", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopInt(); res != 1 {
		t.Errorf("expected 1, got %d", res)
	}
	frame.OperandStack.PushRef("abc_def")
	frame.OperandStack.PushRef("def")
	vmCtx.handleNativeCall("java/lang/String", "endsWith", "(Ljava/lang/String;)Z", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopInt(); res != 1 {
		t.Errorf("expected 1, got %d", res)
	}

	// 4. valueOf(I)
	frame.OperandStack.PushInt(42)
	vmCtx.handleNativeCall("java/lang/String", "valueOf", "(I)Ljava/lang/String;", frame, OpInvokestatic)
	if res := frame.OperandStack.PopRef(); res != "42" {
		t.Errorf("expected '42', got '%v'", res)
	}

	// 5. valueOf(J)
	frame.OperandStack.PushInt(100) // high
	frame.OperandStack.PushInt(200) // low
	vmCtx.handleNativeCall("java/lang/String", "valueOf", "(J)Ljava/lang/String;", frame, OpInvokestatic)
	v := (int64(100) << 32) | int64(200)
	expectedStr := fmt.Sprintf("%d", v)
	if res := frame.OperandStack.PopRef(); res != expectedStr {
		t.Errorf("expected '%s', got '%v'", expectedStr, res)
	}

	// 6. valueOf(Z)
	frame.OperandStack.PushInt(1)
	vmCtx.handleNativeCall("java/lang/String", "valueOf", "(Z)Ljava/lang/String;", frame, OpInvokestatic)
	if res := frame.OperandStack.PopRef(); res != "true" {
		t.Errorf("expected 'true', got '%v'", res)
	}
}

func TestObjectMethods(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)
	frame := &Frame{OperandStack: NewOperandStack(10)}

	// 1. java/lang/Object.toString() on string
	frame.OperandStack.PushRef("test string")
	vmCtx.handleNativeCall("java/lang/Object", "toString", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopRef(); res != "test string" {
		t.Errorf("expected 'test string', got '%v'", res)
	}

	// 2. java/lang/Object.toString() on Object with Extra string
	obj := &Object{Extra: "extra content"}
	frame.OperandStack.PushRef(obj)
	vmCtx.handleNativeCall("java/lang/Object", "toString", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopRef(); res != "extra content" {
		t.Errorf("expected 'extra content', got '%v'", res)
	}

	// 3. java/lang/String.toString()
	frame.OperandStack.PushRef("hello")
	vmCtx.handleNativeCall("java/lang/String", "toString", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopRef(); res != "hello" {
		t.Errorf("expected 'hello', got '%v'", res)
	}

	// 4. java/lang/Object.equals()
	frame.OperandStack.PushRef("abc")
	frame.OperandStack.PushRef("abc")
	vmCtx.handleNativeCall("java/lang/Object", "equals", "(Ljava/lang/Object;)Z", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopInt(); res != 1 {
		t.Errorf("expected 1, got %d", res)
	}

	frame.OperandStack.PushRef("abc")
	frame.OperandStack.PushRef("xyz")
	vmCtx.handleNativeCall("java/lang/Object", "equals", "(Ljava/lang/Object;)Z", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopInt(); res != 0 {
		t.Errorf("expected 0, got %d", res)
	}

	// 5. Generic fallback toString on unknown class
	frame.OperandStack.PushRef("custom text")
	vmCtx.handleNativeCall("some/random/Class", "toString", "()Ljava/lang/String;", frame, OpInvokevirtual)
	if res := frame.OperandStack.PopRef(); res != "custom text" {
		t.Errorf("expected 'custom text', got '%v'", res)
	}
}

func TestExceptionHandling(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	// Test 1: OpAthrow caught by ExceptionTable
	// PC 0: aconst_null (0x01)
	// PC 1: athrow (0xBF)
	// PC 2: iconst_0 (0x03)
	// PC 3: ireturn (0xAC)
	// PC 4: pop (0x57) -- catch handler
	// PC 5: iconst_5 (0x08)
	// PC 6: ireturn (0xAC)
	code := []byte{
		OpAconstNull,
		OpAthrow,
		OpIconst0,
		OpIreturn,
		OpPop,
		OpIconst5,
		OpIreturn,
	}

	method := &classfile.MethodInfo{
		ExceptionTable: []classfile.ExceptionTableEntry{
			{
				StartPC:   0,
				EndPC:     3,
				HandlerPC: 4,
				CatchType: 0, // catch all
			},
		},
	}

	frame := &Frame{
		OperandStack: NewOperandStack(8),
		PC:           0,
	}

	res, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame, method: method})
	if err != nil {
		t.Fatalf("unexpected error in ExecuteBytecode: %v", err)
	}
	if res.Num != 5 {
		t.Errorf("expected return value 5 from exception handler, got %d", res.Num)
	}

	// Test 2: Native EOFException caught by ExceptionTable with Class java/lang/Exception
	// Build a dummy ClassFile with ConstantPool entry for java/lang/Exception
	cp := make(classfile.ConstantPool, 3)
	cp[1] = classfile.ConstantClassInfo{NameIndex: 2}
	cp[2] = classfile.ConstantUtf8Info{Str: "java/lang/Exception"}
	dummyCF := &classfile.ClassFile{
		ConstantPool: cp,
	}

	method2 := &classfile.MethodInfo{
		ExceptionTable: []classfile.ExceptionTableEntry{
			{
				StartPC:   0,
				EndPC:     2,
				HandlerPC: 3,
				CatchType: 1, // index 1 is java/lang/Exception
			},
		},
	}

	code2 := []byte{
		OpNop,
		OpNop,
		OpIreturn,
		OpPop,
		OpIconst1,
		OpIreturn,
	}

	frame2 := &Frame{
		OperandStack:     NewOperandStack(8),
		PC:               0,
		PendingException: vmCtx.NewException("java/io/EOFException", "End of stream"),
	}

	res2, err2 := vmCtx.ExecuteBytecode(dummyCF, codeWithFrame{code: code2, frame: frame2, method: method2})
	if err2 != nil {
		t.Fatalf("unexpected error in ExecuteBytecode: %v", err2)
	}
	if res2.Num != 1 {
		t.Errorf("expected return value 1 from EOFException handler, got %d", res2.Num)
	}

	// Test 3: Uncaught exception returns JavaException error
	frame3 := &Frame{
		OperandStack:     NewOperandStack(8),
		PC:               0,
		PendingException: vmCtx.NewException("java/lang/RuntimeException", "Uncaught"),
	}
	_, err3 := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: []byte{OpNop}, frame: frame3, method: nil})
	if err3 == nil {
		t.Fatal("expected error for uncaught exception, got nil")
	}
	if _, ok := err3.(*JavaException); !ok {
		t.Errorf("expected *JavaException, got %T: %v", err3, err3)
	}
}
