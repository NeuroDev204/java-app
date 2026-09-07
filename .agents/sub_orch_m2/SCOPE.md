# Scope: Milestone 2 — JVM Bytecode Engine & MIDP Runtime APIs

## Scope Overview
Complete the JVM Bytecode Interpreter Engine and MIDP Runtime APIs for KnightOnline MIDlet suite in Go (`java-app`) so that all required opcodes, string operations, input streams, image decoding, and persistent storage work authentically.

## Target Package Directories
- `/home/neuro/Documents/Tools/java-app/pkg/vm/`
- `/home/neuro/Documents/Tools/java-app/pkg/midp/`

## Status: DONE

## Detailed Implementation Requirements
1. **Jump Tables & Switch Opcodes (`pkg/vm/interpreter.go` & `pkg/vm/opcodes.go`)**:
   - Implement `tableswitch` (0xAA) with 4-byte padding alignment relative to `startPC`, low/high bounds, and offset jump execution. [DONE]
   - Implement `lookupswitch` (0xAB) with 4-byte padding alignment, pair count, match keys, and default offset jump execution. [DONE]

2. **Bitwise Arithmetic & Branch Instructions (`pkg/vm/interpreter.go` & `pkg/vm/opcodes.go`)**:
   - Implement bitwise opcodes: `iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr`, `irem`. [DONE]
   - Implement conditional branch instructions: `iflt`, `ifge`, `ifgt`, `ifle`, `if_icmpgt`, `if_icmple`, `if_acmpeq`, `if_acmpne`. [DONE]

3. **Object & Interface Operations (`pkg/vm/interpreter.go`)**:
   - Implement `instanceof` (pop objref, push `1` if instance, `0` otherwise). [DONE]
   - Implement `checkcast` (verify object type, leave ref on stack or raise ClassCastException). [DONE]
   - Implement `monitorenter` and `monitorexit` (pop ref, no-op or synchronization lock). [DONE]
   - Implement `invokeinterface` (0xB9) taking 5 bytes (opcode + 2-byte index + count + zero), popping parameters and dispatching method. [DONE]

4. **String & InputStream APIs (`pkg/vm/interpreter.go`)**:
   - Implement `java/lang/String.equals(Ljava/lang/Object;)Z` returning `1` for matching string contents, `0` otherwise. [DONE]
   - Fix `java/lang/String.charAt(I)C` to index `[]rune(strVal)[idx]` instead of Go UTF-8 byte indexing. [DONE]
   - Disambiguate `indexOf(Ljava/lang/String;I)I` vs `indexOf(II)I`. [DONE]
   - Fix `DataStreamReader.ReadBytes` to copy `min(n, len(dst)-off)` bytes without skipping when `len(dst)` is smaller than requested `n`. [DONE]
   - Implement `readUTF()` helper using 2-byte big-endian length prefix. [DONE]

5. **RecordStore Persistent Storage (`pkg/vm/interpreter.go`)**:
   - Implement functional `RecordStore.setRecord(I[BII)V` updating `store.Records[recID-1]`. [DONE]
   - Implement `enumerateRecords` iterator stub returning a valid RecordEnumeration object. [DONE]

## Verification Commands
- `LIBRARY_PATH=/tmp go test ./pkg/vm ./pkg/midp -v` (PASS)
- `LIBRARY_PATH=/tmp go test ./... -v` (PASS)
