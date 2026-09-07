# Dispatch Log — Milestone 2 Sub-Orchestrator

## 2026-08-08T22:48:48Z

You are Sub-Orchestrator for Milestone 2 (JVM Bytecode Engine & MIDP Runtime APIs).
Working directory: /home/neuro/Documents/Tools/java-app/.agents/sub_orch_m2
Scope document: /home/neuro/Documents/Tools/java-app/.agents/sub_orch_m2/SCOPE.md
Master Project Plan: /home/neuro/Documents/Tools/java-app/.agents/orchestrator_1/PROJECT.md
Original Request: /home/neuro/Documents/Tools/java-app/.agents/ORIGINAL_REQUEST.md
Explorer 2 Report: /home/neuro/Documents/Tools/java-app/.agents/explorer_2/handoff.md

Your Mission:
Execute Milestone 2 according to the Project Orchestrator procedure:

1. **Initialize State**:
   - Create `BRIEFING.md` and `progress.md` in your working directory.
   - Set up your liveness heartbeat.

2. **Iteration Loop (Explorer -> Worker -> Reviewers -> Challengers -> Auditor -> Gate)**:
   - **Worker Dispatch**: Spawn a `teamwork_preview_worker` to implement:
     a. `pkg/vm/interpreter.go` & `pkg/vm/opcodes.go`: `tableswitch` (0xAA) and `lookupswitch` (0xAB) with 4-byte padding alignment relative to `startPC`.
     b. Bitwise opcodes (`iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr`, `irem`) and conditional branches (`iflt`, `ifge`, `ifgt`, `ifle`, `if_icmpgt`, `if_icmple`, `if_acmpeq`, `if_acmpne`).
     c. `instanceof`, `checkcast`, `monitorenter`, `monitorexit`, and 5-byte `invokeinterface` (0xB9).
     d. `java/lang/String.equals(Ljava/lang/Object;)Z` returning `1` for matching string contents, `0` otherwise.
     e. `java/lang/String.charAt(I)C` indexing `[]rune(strVal)[idx]` instead of Go UTF-8 byte indexing. `indexOf` descriptor parameter resolution.
     f. `DataStreamReader.ReadBytes` bounds copy fix (`min(n, len(dst)-off)`) and `readUTF()`.
     g. Functional `RecordStore.setRecord` updates and `enumerateRecords` iterator stub.
     - MANDATORY INTEGRITY WARNING: Include in Worker prompt: "DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work."
     - Require Worker to run build and test commands (`LIBRARY_PATH=/tmp go test ./pkg/vm ./pkg/midp -v` and `LIBRARY_PATH=/tmp go test ./... -v`) and include output in handoff.md.

   - **Reviewer Dispatch**: Spawn 2 independent `teamwork_preview_reviewer` agents to inspect code diffs, stack frame operations, opcode lengths, and API semantics.

   - **Challenger Dispatch**: Spawn 2 `teamwork_preview_challenger` agents to stress test switch jumps, bitwise ops, string rune indexing, stream boundary reading, and RMS record updates.

   - **Forensic Auditor Dispatch**: Spawn 1 `teamwork_preview_auditor` agent to perform integrity verification (verify authentic bytecode interpreter logic and RMS persistence, zero hardcoded test returns).

   - **Gate Check**: Record verdicts in `GATE_STATUS.md`. Gate passes ONLY if build/tests pass, both Reviewers APPROVE, both Challengers confirm correctness, and Auditor verdict is CLEAN.

3. **Handoff & Reporting**:
   - Write `handoff.md` in your working directory.
   - Send completion message to parent orchestrator.
