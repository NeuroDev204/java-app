## 2026-08-08T15:49:26Z
You are the Worker assigned to implement Milestone 2 (JVM Bytecode Engine & MIDP Runtime APIs) for the J2ME Java Game Emulator in `/home/neuro/Documents/Tools/java-app`.

Your working directory is: `/home/neuro/Documents/Tools/java-app/.agents/worker_m2`

Read these files before starting:
- `/home/neuro/Documents/Tools/java-app/.agents/ORIGINAL_REQUEST.md`
- `/home/neuro/Documents/Tools/java-app/.agents/orchestrator_1/PROJECT.md`
- `/home/neuro/Documents/Tools/java-app/.agents/sub_orch_m2/SCOPE.md`
- `/home/neuro/Documents/Tools/java-app/.agents/explorer_2/handoff.md`

Your Tasks:
1. Implement in `pkg/vm/interpreter.go` & `pkg/vm/opcodes.go`:
   a. `tableswitch` (0xAA) opcode with 4-byte padding alignment relative to `startPC`:
      - Calculate byte alignment padding after opcode (0 to 3 bytes so `(startPC + 1 + pad) % 4 == 0`).
      - Read 4-byte default offset, 4-byte low, 4-byte high.
      - Pop int index from stack. If `index < low || index > high`, jump to `startPC + defaultOffset`. Else jump to `startPC + offsets[index - low]`.
   b. `lookupswitch` (0xAB) opcode with 4-byte padding alignment relative to `startPC`:
      - Calculate byte alignment padding after opcode.
      - Read 4-byte default offset, 4-byte npairs.
      - Read npairs of (match, offset) 4-byte pairs.
      - Pop int key from stack. Match key against pairs. If matched, jump to `startPC + pair.offset`. If no match, jump to `startPC + defaultOffset`.
   c. Bitwise opcodes (`iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr`, `irem`):
      - Implement complete stack arithmetic popping 2 ints, performing operation (with 0x1f bitmask shift for shifts, modulo for `irem`), pushing int result.
   d. Conditional branch instructions:
      - `iflt`, `ifge`, `ifgt`, `ifle`: pop 1 int, compare to 0, branch to `startPC + branchOffset` if condition holds.
      - `if_icmpgt`, `if_icmple`, `if_acmpeq`, `if_acmpne`: pop 2 values (ints or refs), compare, branch if condition holds.
   e. Object & Interface operations:
      - `instanceof` (pop objref, push `1` if valid non-null instance, `0` otherwise).
      - `checkcast` (verify object ref type, leave ref on stack or raise ClassCastException if incompatible).
      - `monitorenter` and `monitorexit` (pop objref, synchronization lock handling or no-op).
      - `invokeinterface` (0xB9): 5-byte opcode (opcode + 2-byte index + count + zero). Pop count/params, dispatch interface method.

2. Implement/Fix in `pkg/vm/interpreter.go`:
   a. `java/lang/String.equals(Ljava/lang/Object;)Z`:
      - In `handleNativeCall` / `java/lang/String` dispatch: return `1` (true) if target string content matches parameter string content, `0` (false) otherwise.
   b. `java/lang/String.charAt(I)C`:
      - Index `[]rune(strVal)[idx]` instead of Go UTF-8 byte indexing `strVal[idx]`.
   c. `indexOf` descriptor parameter resolution:
      - Disambiguate `indexOf(Ljava/lang/String;I)I` vs `indexOf(II)I` vs `indexOf(Ljava/lang/String;)I`.
   d. `DataStreamReader.ReadBytes`:
      - Fix bounds copy logic so it copies `min(n, len(dst)-off)` bytes into `dst` without skipping copy when `len(dst)` is smaller than requested `n`.
   e. Add `readUTF()` helper using 2-byte big-endian length prefix.
   f. RecordStore persistent storage:
      - Functional `RecordStore.setRecord(I[BII)V` updating `store.Records[recID-1]`.
      - Implement `enumerateRecords` iterator stub returning a valid RecordEnumeration object.

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work.

Required Verification:
- Run build and test commands:
  `LIBRARY_PATH=/tmp go test ./pkg/vm ./pkg/midp -v`
  `LIBRARY_PATH=/tmp go test ./... -v`
- Document commands and full pass outputs in your `handoff.md` file in `/home/neuro/Documents/Tools/java-app/.agents/worker_m2/handoff.md`.
- Send completion message to parent when done.
