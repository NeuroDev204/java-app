package vm

import (
	"encoding/binary"
	"math"
	"testing"
)

// 1. Switch jump tables (tableswitch / lookupswitch) with negative/zero/boundary indices.
func TestChallenger_SwitchJumpTables(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	t.Run("Tableswitch_NegativeAndZeroBounds", func(t *testing.T) {
		// startPC = 1
		// low = -3, high = 1
		// Table data: PC 1..35
		// Targets placed at PC >= 40 (no overlap with table data!)
		code := make([]byte, 60)
		code[0] = OpNop
		code[1] = OpTableswitch
		// pad = 2 (PC 2, 3)
		vLow := int32(-3)
		binary.BigEndian.PutUint32(code[4:8], 50)           // defaultOffset = 50 -> target PC 51
		binary.BigEndian.PutUint32(code[8:12], uint32(vLow)) // low = -3
		binary.BigEndian.PutUint32(code[12:16], 1)          // high = 1
		binary.BigEndian.PutUint32(code[16:20], 39)         // -3 -> target PC 40
		binary.BigEndian.PutUint32(code[20:24], 41)         // -2 -> target PC 42
		binary.BigEndian.PutUint32(code[24:28], 43)         // -1 -> target PC 44
		binary.BigEndian.PutUint32(code[28:32], 45)         //  0 -> target PC 46
		binary.BigEndian.PutUint32(code[32:36], 47)         //  1 -> target PC 48

		code[40] = OpReturn
		code[42] = OpReturn
		code[44] = OpReturn
		code[46] = OpReturn
		code[48] = OpReturn
		code[51] = OpReturn

		testIndex := func(idx int32, expectedTargetPC int) {
			frame := &Frame{
				OperandStack: NewOperandStack(5),
				PC:           1,
			}
			frame.OperandStack.PushInt(idx)
			_, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
			if err != nil {
				t.Fatalf("ExecuteBytecode returned error for idx %d: %v", idx, err)
			}
			// After executing OpReturn at expectedTargetPC, final PC is expectedTargetPC + 1
			if frame.PC != expectedTargetPC+1 {
				t.Errorf("tableswitch idx=%d expected final PC=%d, got %d", idx, expectedTargetPC+1, frame.PC)
			}
		}

		testIndex(-3, 40)
		testIndex(-2, 42)
		testIndex(-1, 44)
		testIndex(0, 46)
		testIndex(1, 48)
		testIndex(-4, 51)  // Below low -> default
		testIndex(2, 51)   // Above high -> default
		testIndex(100, 51)
		testIndex(-100, 51)
	})

	t.Run("Tableswitch_ZeroBound", func(t *testing.T) {
		// tableswitch with low = 0, high = 0
		// Table data: PC 1..19
		// Targets placed at PC >= 25
		code := make([]byte, 40)
		code[0] = OpNop
		code[1] = OpTableswitch
		binary.BigEndian.PutUint32(code[4:8], 30)  // defaultOffset = 30 -> target PC 31
		binary.BigEndian.PutUint32(code[8:12], 0)  // low = 0
		binary.BigEndian.PutUint32(code[12:16], 0) // high = 0
		binary.BigEndian.PutUint32(code[16:20], 24) // 0 -> target PC 25
		code[25] = OpReturn
		code[31] = OpReturn

		testIndex := func(idx int32, expectedTargetPC int) {
			frame := &Frame{
				OperandStack: NewOperandStack(5),
				PC:           1,
			}
			frame.OperandStack.PushInt(idx)
			_, _ = vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
			if frame.PC != expectedTargetPC+1 {
				t.Errorf("tableswitch low=0 high=0 idx=%d expected final PC=%d, got %d", idx, expectedTargetPC+1, frame.PC)
			}
		}

		testIndex(0, 25)
		testIndex(-1, 31)
		testIndex(1, 31)
	})

	t.Run("Lookupswitch_NegativeAndBoundaryKeys", func(t *testing.T) {
		// lookupswitch with keys: INT_MIN, -100, 0, 100, INT_MAX
		// Table data: PC 1..51
		// Targets placed at PC >= 54
		code := make([]byte, 80)
		code[0] = OpNop
		code[1] = OpLookupswitch
		// pad = 2
		binary.BigEndian.PutUint32(code[4:8], 70) // defaultOffset = 70 -> target PC 71
		binary.BigEndian.PutUint32(code[8:12], 5) // npairs = 5

		var vMinKey int32 = math.MinInt32
		var vNeg100Key int32 = -100
		var vMaxKey int32 = math.MaxInt32

		// Pair 0: INT_MIN -> offset 53 (target PC 54)
		binary.BigEndian.PutUint32(code[12:16], uint32(vMinKey))
		binary.BigEndian.PutUint32(code[16:20], 53)

		// Pair 1: -100 -> offset 55 (target PC 56)
		binary.BigEndian.PutUint32(code[20:24], uint32(vNeg100Key))
		binary.BigEndian.PutUint32(code[24:28], 55)

		// Pair 2: 0 -> offset 57 (target PC 58)
		binary.BigEndian.PutUint32(code[28:32], 0)
		binary.BigEndian.PutUint32(code[32:36], 57)

		// Pair 3: 100 -> offset 59 (target PC 60)
		binary.BigEndian.PutUint32(code[36:40], 100)
		binary.BigEndian.PutUint32(code[40:44], 59)

		// Pair 4: INT_MAX -> offset 61 (target PC 62)
		binary.BigEndian.PutUint32(code[44:48], uint32(vMaxKey))
		binary.BigEndian.PutUint32(code[48:52], 61)

		code[54] = OpReturn
		code[56] = OpReturn
		code[58] = OpReturn
		code[60] = OpReturn
		code[62] = OpReturn
		code[71] = OpReturn

		testKey := func(key int32, expectedTargetPC int) {
			frame := &Frame{
				OperandStack: NewOperandStack(5),
				PC:           1,
			}
			frame.OperandStack.PushInt(key)
			_, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
			if err != nil {
				t.Fatalf("ExecuteBytecode error for key %d: %v", key, err)
			}
			if frame.PC != expectedTargetPC+1 {
				t.Errorf("lookupswitch key=%d expected final PC=%d, got %d", key, expectedTargetPC+1, frame.PC)
			}
		}

		testKey(math.MinInt32, 54)
		testKey(-100, 56)
		testKey(0, 58)
		testKey(100, 60)
		testKey(math.MaxInt32, 62)
		testKey(-50, 71) // Unmatched -> default
		testKey(1, 71)   // Unmatched -> default
	})
}

// 2. Bitwise arithmetic (iand, ior, ixor, ishl, ishr, iushr, irem) with edge values (0, negative shift amounts, INT_MIN/MAX).
func TestChallenger_BitwiseArithmetic(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	execOp := func(op byte, v1, v2 int32) int32 {
		code := []byte{op, OpIreturn}
		frame := &Frame{
			OperandStack: NewOperandStack(5),
			PC:           0,
		}
		frame.OperandStack.PushInt(v1)
		frame.OperandStack.PushInt(v2)
		res, err := vmCtx.ExecuteBytecode(nil, codeWithFrame{code: code, frame: frame})
		if err != nil {
			t.Fatalf("op 0x%X(%d, %d) returned error: %v", op, v1, v2, err)
		}
		return res.Num
	}

	t.Run("iand_ior_ixor_EdgeValues", func(t *testing.T) {
		min := int32(math.MinInt32)
		max := int32(math.MaxInt32)

		if got := execOp(OpIand, min, max); got != 0 {
			t.Errorf("iand MinInt32 & MaxInt32 expected 0, got %d", got)
		}
		if got := execOp(OpIand, min, -1); got != min {
			t.Errorf("iand MinInt32 & -1 expected %d, got %d", min, got)
		}
		if got := execOp(OpIor, min, max); got != -1 {
			t.Errorf("ior MinInt32 | MaxInt32 expected -1, got %d", got)
		}
		if got := execOp(OpIor, 0, min); got != min {
			t.Errorf("ior 0 | MinInt32 expected %d, got %d", min, got)
		}
		if got := execOp(OpIxor, min, max); got != -1 {
			t.Errorf("ixor MinInt32 ^ MaxInt32 expected -1, got %d", got)
		}
		if got := execOp(OpIxor, min, min); got != 0 {
			t.Errorf("ixor MinInt32 ^ MinInt32 expected 0, got %d", got)
		}
	})

	t.Run("Shifts_NegativeAndEdgeDistances", func(t *testing.T) {
		min := int32(math.MinInt32)
		max := int32(math.MaxInt32)

		// Test shift distance = 0
		if got := execOp(OpIshl, 12345, 0); got != 12345 {
			t.Errorf("ishl 12345 << 0 expected 12345, got %d", got)
		}
		if got := execOp(OpIshr, -12345, 0); got != -12345 {
			t.Errorf("ishr -12345 >> 0 expected -12345, got %d", got)
		}
		if got := execOp(OpIushr, -12345, 0); got != -12345 {
			t.Errorf("iushr -12345 >>> 0 expected -12345, got %d", got)
		}

		// Test shift distance = 31
		if got := execOp(OpIshl, 1, 31); got != min {
			t.Errorf("ishl 1 << 31 expected %d (MinInt32), got %d", min, got)
		}
		if got := execOp(OpIshr, min, 31); got != -1 {
			t.Errorf("ishr MinInt32 >> 31 expected -1, got %d", got)
		}
		if got := execOp(OpIushr, min, 31); got != 1 {
			t.Errorf("iushr MinInt32 >>> 31 expected 1, got %d", got)
		}

		// Test shift distance = 32 (should use 32 & 0x1f = 0)
		if got := execOp(OpIshl, 100, 32); got != 100 {
			t.Errorf("ishl 100 << 32 expected 100, got %d", got)
		}

		// Test negative shift distance = -1 (uint32(-1) & 0x1f = 31)
		if got := execOp(OpIshl, 1, -1); got != min {
			t.Errorf("ishl 1 << -1 expected %d, got %d", min, got)
		}
		if got := execOp(OpIshr, min, -1); got != -1 {
			t.Errorf("ishr MinInt32 >> -1 expected -1, got %d", got)
		}
		if got := execOp(OpIushr, min, -1); got != 1 {
			t.Errorf("iushr MinInt32 >>> -1 expected 1, got %d", got)
		}

		// Test MaxInt32 shifts
		if got := execOp(OpIshl, max, 1); got != -2 {
			t.Errorf("ishl MaxInt32 << 1 expected -2, got %d", got)
		}
	})

	t.Run("irem_EdgeValues", func(t *testing.T) {
		min := int32(math.MinInt32)
		max := int32(math.MaxInt32)

		if got := execOp(OpIrem, 17, 5); got != 2 {
			t.Errorf("irem 17 %% 5 expected 2, got %d", got)
		}
		if got := execOp(OpIrem, -17, 5); got != -2 {
			t.Errorf("irem -17 %% 5 expected -2, got %d", got)
		}
		if got := execOp(OpIrem, 17, -5); got != 2 {
			t.Errorf("irem 17 %% -5 expected 2, got %d", got)
		}
		if got := execOp(OpIrem, -17, -5); got != -2 {
			t.Errorf("irem -17 %% -5 expected -2, got %d", got)
		}
		if got := execOp(OpIrem, 0, 10); got != 0 {
			t.Errorf("irem 0 %% 10 expected 0, got %d", got)
		}
		if got := execOp(OpIrem, min, max); got != -1 {
			t.Errorf("irem MinInt32 %% MaxInt32 expected -1, got %d", got)
		}
		// Divide by zero safety check
		if got := execOp(OpIrem, 10, 0); got != 0 {
			t.Errorf("irem 10 %% 0 expected 0 (safe fallback), got %d", got)
		}
	})
}

// 3. Multi-byte UTF-8 Unicode rune indexing in String.charAt.
func TestChallenger_StringCharAt_Unicode(t *testing.T) {
	vmCtx := NewVMContext(nil, nil)

	testCharAt := func(str string, idx int32, expectedRune int32) {
		frame := &Frame{OperandStack: NewOperandStack(5)}
		frame.OperandStack.PushRef(str)
		frame.OperandStack.PushInt(idx)
		vmCtx.handleNativeCall("java/lang/String", "charAt", "(I)C", frame, OpInvokevirtual)
		got := frame.OperandStack.PopInt()
		if got != expectedRune {
			t.Errorf("charAt('%s', %d) expected rune %d ('%c'), got %d ('%c')",
				str, idx, expectedRune, rune(expectedRune), got, rune(got))
		}
	}

	// 1. Multi-byte UTF-8 (Emoji / 4-byte): "J2ME_🎮_Game"
	// Runes: [J(74), 2(50), M(77), E(69), _(95), 🎮(127918), _(95), G(71), a(97), m(109), e(101)]
	testCharAt("J2ME_🎮_Game", 0, 74)
	testCharAt("J2ME_🎮_Game", 5, 127918) // 🎮
	testCharAt("J2ME_🎮_Game", 6, 95)     // _
	testCharAt("J2ME_🎮_Game", 7, 71)     // G

	// 2. 3-byte UTF-8 CJK Characters: "Knight骑士"
	// Runes: [K(75), n(110), i(105), g(103), h(104), t(116), 骑(39569), 士(22763)]
	testCharAt("Knight骑士", 6, 39569) // 骑
	testCharAt("Knight骑士", 7, 22763) // 士

	// 3. 2-byte UTF-8 Accented Characters: "Café"
	// Runes: [C(67), a(97), f(102), é(233)]
	testCharAt("Café", 3, 233) // é

	// 4. Out of bounds and edge indices
	testCharAt("Test", -1, 0)
	testCharAt("Test", 4, 0)
	testCharAt("Test", 100, 0)
	testCharAt("", 0, 0)
}

// 4. Boundary reading in DataStreamReader.ReadBytes when n > len(dst) - off.
func TestChallenger_DataStreamReader_ReadBytes_Boundaries(t *testing.T) {
	t.Run("Requested_N_Exceeds_Dst_Remaining_Space", func(t *testing.T) {
		streamData := []byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
		reader := &DataStreamReader{Data: streamData, Pos: 0}

		// Destination buffer of size 5
		dst := make([]byte, 5)
		// Requesting offset=2, length=10 (available space in dst from offset 2 is 3 bytes)
		readN := reader.ReadBytes(dst, 2, 10)

		if readN != 3 {
			t.Errorf("ReadBytes with n > len(dst)-off expected readN=3, got %d", readN)
		}
		if reader.Pos != 3 {
			t.Errorf("ReadBytes expected reader.Pos=3, got %d", reader.Pos)
		}
		if dst[2] != 10 || dst[3] != 20 || dst[4] != 30 {
			t.Errorf("ReadBytes unexpected destination slice content: %v", dst)
		}
	})

	t.Run("Offset_Out_Of_Bounds", func(t *testing.T) {
		streamData := []byte{1, 2, 3, 4, 5}
		reader := &DataStreamReader{Data: streamData, Pos: 0}
		dst := make([]byte, 5)

		// Negative offset
		if readN := reader.ReadBytes(dst, -1, 3); readN != 0 {
			t.Errorf("ReadBytes negative offset expected 0, got %d", readN)
		}
		// Offset equal to len(dst)
		if readN := reader.ReadBytes(dst, 5, 2); readN != 0 {
			t.Errorf("ReadBytes off==len(dst) expected 0, got %d", readN)
		}
		// Offset greater than len(dst)
		if readN := reader.ReadBytes(dst, 10, 2); readN != 0 {
			t.Errorf("ReadBytes off > len(dst) expected 0, got %d", readN)
		}
		// Confirm stream position was not mutated by invalid offset calls
		if reader.Pos != 0 {
			t.Errorf("ReadBytes invalid offset mutated Pos: expected 0, got %d", reader.Pos)
		}
	})

	t.Run("Stream_EOF_And_Exact_Read", func(t *testing.T) {
		streamData := []byte{100, 200}
		reader := &DataStreamReader{Data: streamData, Pos: 2} // Already at EOF
		dst := make([]byte, 5)

		if readN := reader.ReadBytes(dst, 0, 5); readN != -1 {
			t.Errorf("ReadBytes at EOF expected -1, got %d", readN)
		}
	})
}
