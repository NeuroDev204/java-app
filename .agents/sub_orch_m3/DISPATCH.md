# DISPATCH — 2026-08-08T22:56:03Z

<USER_REQUEST>
You are Sub-Orchestrator for Milestone 3 (Unit & E2E Verification & Hardening).
Working directory: /home/neuro/Documents/Tools/java-app/.agents/sub_orch_m3
Scope document: /home/neuro/Documents/Tools/java-app/.agents/sub_orch_m3/SCOPE.md
Master Project Plan: /home/neuro/Documents/Tools/java-app/.agents/orchestrator_1/PROJECT.md
Original Request: /home/neuro/Documents/Tools/java-app/.agents/ORIGINAL_REQUEST.md
Explorer 3 Report: /home/neuro/Documents/Tools/java-app/.agents/explorer_3/handoff.md

Your Mission:
Execute Milestone 3 according to the Project Orchestrator procedure:

1. **Initialize State**:
   - Create `BRIEFING.md` and `progress.md` in your working directory.
   - Set up your liveness heartbeat.

2. **Iteration Loop (TestWriter/Worker -> Reviewers -> Challengers -> Auditor -> Gate)**:
   - **Test Writer & Worker Dispatch**: Spawn `teamwork_preview_test_writer` / `teamwork_preview_worker` to:
     a. Expand unit tests across `pkg/vm` (bitwise opcodes, switch tables, string operations, input stream reading, RecordStore), `pkg/midp` (clipping bounds, font rendering, sprite anchor/transforms, image decoding), `pkg/gfx` (buffer flushing to `emulator_buffer.png`), and `pkg/jar`.
     b. Create headless E2E verification test harness in `cmd/j2me/e2e_test.go` that:
        - Renders a full J2ME Canvas frame (background fill, menu buttons, server selection text strings, gameplay sprites).
        - Flushes offscreen buffer to `emulator_buffer.png` and `/tmp/emulator_buffer.png`.
        - Decodes `emulator_buffer.png` using Go `image/png`.
        - Asserts non-zero pixel count > 50,000, button region non-black pixels, bitmap font text rendering, and zero full-screen black overwrite.
     - MANDATORY INTEGRITY WARNING: Include in prompt: "DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work."
     - Require execution of `LIBRARY_PATH=/tmp go test ./... -v -count=1` and inclusion of output in handoff.md.

   - **Reviewer Dispatch**: Spawn 2 independent `teamwork_preview_reviewer` agents to inspect test code quality, E2E assertion coverage, and zero side-effect verification.

   - **Challenger Dispatch**: Spawn 2 `teamwork_preview_challenger` agents to stress test E2E pixel assertion boundaries, PNG file generation, and unit test coverage.

   - **Forensic Auditor Dispatch**: Spawn 1 `teamwork_preview_auditor` agent to perform integrity verification (verify authentic test suite expansion and genuine PNG buffer generation).

   - **Gate Check**: Record verdicts in `GATE_STATUS.md`. Gate passes ONLY if build and all unit & E2E tests pass cleanly under `LIBRARY_PATH=/tmp go test ./...`, both Reviewers APPROVE, both Challengers confirm correctness, and Auditor verdict is CLEAN.

3. **Handoff & Reporting**:
   - Write `handoff.md` in your working directory.
   - Send completion message to parent orchestrator.

Do NOT write source code directly — delegate implementation to TestWriter/Worker, review to Reviewers, testing to Challengers, and audit to Auditor.
</USER_REQUEST>
