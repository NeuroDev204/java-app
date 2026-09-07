# Handoff Report — Requirement R2: JVM Bytecode Interpreter Engine & MIDP Runtime APIs

## 1. Observation

### 1.1 Overview & Key File Locations
Direct inspection of the codebase in `/home/neuro/Documents/Tools/java-app` reveals the exact file paths and line ranges defining the JVM execution engine and MIDP runtime:

| Subsystem / Component | Absolute File Path | Relevant Line Ranges |
| --- | --- | --- |
| **JVM Interpreter Loop** | `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` | `lines 235–796` (`ExecuteBytecode`) |
| **Bytecode Opcode Definitions** | `/home/neuro/Documents/Tools/java-app/pkg/vm/opcodes.go` | `lines 1–149` (Consts & `OpcodeLengths`) |
| **JVM Stack Frame & LocalVars** | `/home/neuro/Documents/Tools/java-app/pkg/vm/frame.go` | `lines 1–107` (`OperandStack`, `LocalVars`) |
| **JVM Slot Memory Primitive** | `/home/neuro/Documents/Tools/java-app/pkg/vm/slot.go` | `lines 1–7` (`Slot` struct) |
| **JVM Object Representation** | `/home/neuro/Documents/Tools/java-app/pkg/vm/object.go` | `lines 1–52` (`Object` struct & mutex) |
| **Native Method Dispatcher** | `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` | `lines 722–786`, `882–1787` (`handleNativeCall`) |
| **String & StringBuffer APIs** | `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` | `lines 931–1089`, `1789–1816` (`java/lang/String`) |
| **InputStream & DataStream** | `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` | `lines 68–121`, `1090–1243` (`DataStreamReader`) |
| **RecordStore Persistent Storage** | `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` | `lines 1244–1345` (`javax/microedition/rms`) |
| **Image Creation & Decoding** | `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go`<br>`/home/neuro/Documents/Tools/java-app/pkg/midp/graphics.go` | `lines 1565–1750`<br>`lines 203–206` (`DecodeImage`) |
| **ClassFile & Attribute Parser** | `/home/neuro/Documents/Tools/java-app/pkg/classfile/classfile.go` | `lines 1–205` (`Parse`, `skipAttributes`) |

---

### 1.2 Verbatim Code Evidence

#### A. Interpreter Default Unhandled Opcode Handler (`interpreter.go:787-793`)
```go
default:
    startPC := frame.PC - 1
    length := OpcodeLengths[opcode]
    if length > 1 {
        frame.PC = startPC + length
    }
```

#### B. `String.equals` Missing in `handleNativeCall` (`interpreter.go:931-1038`)
`java/lang/String` handles `valueOf`, `indexOf`, `getBytes`, `toCharArray`, `charAt`, `length`, `substring`. Method `equals(Ljava/lang/Object;)Z` is **missing**, falling into the generic native fallback (`lines 1777-1787`):
```go
paramCount := countMethodParamSlots(mDesc)
for i := 0; i < paramCount; i++ {
    frame.OperandStack.PopRef()
}
if opcode != OpInvokestatic {
    frame.OperandStack.PopRef()
}
if hasReturnValue(mDesc) {
    frame.OperandStack.PushInt(0)
}
```
*Result:* All `str1.equals(str2)` calls return `0` (false).

#### C. `String.charAt` Byte Index Bug (`interpreter.go:1012`)
```go
ch := strVal[idx]
log.Printf("[STRING-CHARAT] '%s'.charAt(%d) -> '%c' (%d)", strVal, idx, ch, ch)
frame.OperandStack.PushInt(int32(ch))
```
*Result:* `strVal[idx]` accesses UTF-8 bytes in Go instead of Unicode runes / UTF-16 code points.

#### D. `DataStreamReader.ReadBytes` Destination Out-of-Bounds Skip (`interpreter.go:116-118`)
```go
if off+n <= len(dst) {
    copy(dst[off:off+n], r.Data[r.Pos:r.Pos+n])
}
r.Pos += n
return int32(n)
```
*Result:* If `off+n > len(dst)`, `copy` is skipped, but `r.Pos += n` advances the stream and returns `n` bytes read while `dst` remains unpopulated.

#### E. Duplicate `read()I` Branch (`interpreter.go:1117-1129` & `1159-1171`)
Lines 1159–1171 repeat the exact same `else if mName == "read" && mDesc == "()I"` condition block as lines 1117–1129.

#### F. `RecordStore.setRecord` Dummy Implementation (`interpreter.go:1335-1345`)
```go
else if mName == "closeRecordStore" || mName == "setRecord" {
    paramCount := countMethodParamSlots(mDesc)
    for i := 0; i < paramCount; i++ {
        frame.OperandStack.PopRef()
    }
    frame.OperandStack.PopRef()
    if hasReturnValue(mDesc) {
        frame.OperandStack.PushInt(1)
    }
    return
}
```
*Result:* `setRecord` pops parameters and pushes `1`, silently discarding saved data updates.

---

## 2. Logic Chain

1. **Interpreter Opcode Coverage & Fallback Flaw**:
   - *Observation:* `interpreter.go` handles ~50 opcodes out of 256 Java bytecodes. Unhandled opcodes hit `default:` (`lines 787-793`), skipping `OpcodeLengths[opcode]` bytes without modifying the stack or local variables.
   - *Reasoning:* For opcodes like `instanceof` (`0xC1`), skipping 3 bytes leaves an object reference on top of the operand stack instead of replacing it with boolean integer `0` or `1`. Next instructions expecting an integer pop the object reference and crash or misbehave.
   - *Switch & Jump Tables:* `tableswitch` (`0xAA`) and `lookupswitch` (`0xAB`) are padding-aligned variable-length branch instructions used extensively in J2ME game state machines. `opcodes.go:144` sets their length to `1`. When hit, `default:` skips 1 byte, causing `frame.PC` to point into raw jump table offsets, which are interpreted as invalid bytecodes.

2. **Bitwise Arithmetic & Graphic Text Rendering**:
   - *Observation:* KnightOnline uses bitmap font rendering and bitmask packing. Opcodes `iand` (`0x7E`), `ior` (`0x80`), `ixor` (`0x82`), `ishl` (`0x78`), `ishr` (`0x7A`), `iushr` (`0x7C`), `irem` (`0x70`) are defined in `opcodes.go` but absent from `ExecuteBytecode`.
   - *Reasoning:* Bitwise operations hit `default:` and skip 1 byte without performing bit shifts or masking, leading to corrupt coordinate calculations and broken font rendering.

3. **String Comparison & Standard Library Failures**:
   - *Observation:* In J2ME games, `String.equals()` is used for menu item selection, state comparison, and resource key lookup.
   - *Reasoning:* `java/lang/String.equals` is unhandled in `interpreter.go:931-1038`. It falls into generic fallback (`lines 1777-1787`), returning `0` (false). Consequently, string comparisons always fail.
   - *Indexing:* `String.charAt(idx)` uses Go byte indexing `strVal[idx]`. Non-ASCII text or multi-byte glyphs break because byte indexing splits UTF-8 bytes instead of returning rune code points.

4. **Data Stream & Persistent Storage (RMS) Incompleteness**:
   - *Observation:* `DataStreamReader.ReadBytes` (`interpreter.go:108-121`) has an unsafe `if off+n <= len(dst)` guard. `DataInputStream.readUTF()` is missing.
   - *Reasoning:* Binary resource streams reading UTF-8 strings or large buffers are silently ignored or return zeroed buffers.
   - *RMS:* `setRecord` is a stub. `enumerateRecords` is completely missing. RMS state is kept in-memory (`VMContext.RecordStores map[string]*RecordStoreData`), so data disappears on exit.

---

## 3. Caveats

- **No Source Code Execution or Test Commands**: Per guidelines for Explorer 2, no build/test commands were run, and no source files outside `.agents/explorer_2` were modified.
- **Dynamic Opcode Frequency**: The exact list of missing opcodes is derived from static inspection of `interpreter.go` and `opcodes.go` relative to standard J2ME CLDC 1.1 / MIDP 2.0 bytecode usage in KnightOnline MIDlet.

---

## 4. Conclusion

Requirement R2 requires a robust JVM bytecode execution engine and complete MIDP runtime APIs. The existing Go implementation provides a solid framework but suffers from key gaps:
1. Missing support for `tableswitch`, `lookupswitch`, bitwise opcodes (`iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr`), comparison branches (`iflt`, `ifge`, `ifgt`, `ifle`, `if_icmpgt`, `if_icmple`, `if_acmpeq`, `if_acmpne`), `instanceof`, `checkcast`, `monitorenter`, `monitorexit`, and `invokeinterface`.
2. Broken `java/lang/String.equals` (returns false), `charAt` (byte indexing bug), and `indexOf` descriptor parameter resolution.
3. Stream buffer reading boundary bugs in `DataStreamReader.ReadBytes` and missing `readUTF()`.
4. Dummy `RecordStore.setRecord` implementation and missing `enumerateRecords`.

---

## 5. Recommended Fix Strategy

1. **Interpreter Loop (`pkg/vm/interpreter.go` & `pkg/vm/opcodes.go`)**:
   - Add explicit handlers for `tableswitch` (`0xAA`) and `lookupswitch` (`0xAB`) calculating 4-byte padding alignment relative to `startPC`.
   - Implement bitwise opcodes: `iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr`, `irem`.
   - Implement conditional branches: `iflt`, `ifge`, `ifgt`, `ifle`, `if_icmpgt`, `if_icmple`, `if_acmpeq`, `if_acmpne`.
   - Implement `instanceof` (pop ref, push `0`/`1`), `checkcast` (verify/leave ref), `monitorenter`/`monitorexit` (pop ref), and `invokeinterface` (5-byte opcode handling).

2. **String APIs (`pkg/vm/interpreter.go`)**:
   - Add `java/lang/String.equals` returning `1` if string contents match.
   - Update `charAt` to index `[]rune(strVal)[idx]`.
   - Disambiguate `indexOf(Ljava/lang/String;I)I` vs `indexOf(II)I`.

3. **InputStream & RMS (`pkg/vm/interpreter.go`)**:
   - Fix `DataStreamReader.ReadBytes` to copy `min(n, len(dst)-off)` bytes without skipping when `len(dst)` is smaller than requested `length`.
   - Add `readUTF()` helper using 2-byte big-endian length prefix.
   - Implement `setRecord` to write into `store.Records[recID-1]`.
   - Implement `enumerateRecords` returning a record iterator stub object.

---

## 6. Verification Method

To independently verify these findings:
1. Inspect `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` at lines 252–796 to check opcode switch cases.
2. Inspect `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` at lines 931–1038 to verify `java/lang/String` method dispatches.
3. Inspect `/home/neuro/Documents/Tools/java-app/pkg/vm/interpreter.go` at lines 1335–1345 to verify `RecordStore.setRecord` stub.
4. After implementation, run `LIBRARY_PATH=/tmp go test ./...` to verify all tests pass cleanly.
