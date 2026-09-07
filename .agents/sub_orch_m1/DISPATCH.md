## 2026-08-08T22:41:01Z
You are Sub-Orchestrator for Milestone 1 (J2ME Canvas & MIDP UI Rendering Pipeline).
Working directory: /home/neuro/Documents/Tools/java-app/.agents/sub_orch_m1
Scope document: /home/neuro/Documents/Tools/java-app/.agents/sub_orch_m1/SCOPE.md
Master Project Plan: /home/neuro/Documents/Tools/java-app/.agents/orchestrator_1/PROJECT.md
Original Request: /home/neuro/Documents/Tools/java-app/.agents/ORIGINAL_REQUEST.md
Explorer 1 Report: /home/neuro/Documents/Tools/java-app/.agents/explorer_1/handoff.md

Your Mission:
Execute Milestone 1 according to the Project Orchestrator procedure:

1. **Initialize State**:
   - Create `BRIEFING.md` and `progress.md` in your working directory.
   - Set up your liveness heartbeat.

2. **Iteration Loop (Explorer -> Worker -> Reviewers -> Challengers -> Auditor -> Gate)**:
   - **Worker Dispatch**: Spawn a `teamwork_preview_worker` to implement:
     a. `pkg/midp/graphics.go`: Update `getClippedDst` to return `image.ZR` when `ClipW <= 0 || ClipH <= 0`.
     b. `pkg/vm/interpreter.go`: Implement parameter passing for `setClip` and `clipRect` native calls, invoking `ctx.Graphics.SetClip`/`ClipRect` with `TransX`/`TransY`. Update `getClipX/Y/Width/Height` to query dynamic clip state. Remove/refactor fragile `[GFX-PREVENT-ERASE]` hack.
     c. `pkg/midp/graphics.go` & `pkg/vm/interpreter.go`: Replace solid box uniform drawing in `DrawString` with font glyph rendering (e.g. using standard basicfont / 8x13 font glyph mask). Implement native `Font` stubs (`getFont`, `getDefaultFont`, `stringWidth`, `charWidth`, `getHeight`, `getBaselinePosition`).
     d. `pkg/midp/graphics.go`: Implement J2ME MIDP 2.0 `anchor` alignment (`LEFT`, `RIGHT`, `HCENTER`, `TOP`, `BOTTOM`, `VCENTER`) and sprite `transform` flips/rotations in `DrawImage` and `DrawRegion`.
     e. `pkg/gfx/renderer.go`: Update buffer flushing so `emulator_buffer.png` and `/tmp/emulator_buffer.png` are flushed continuously on paint updates without a 5-frame cap.
     - MANDATORY INTEGRITY WARNING: Include in Worker prompt: "DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work."
     - Require Worker to run build and test commands (`LIBRARY_PATH=/tmp go test ./pkg/midp ./pkg/gfx ./pkg/vm -v`) and include output in handoff.md.

   - **Reviewer Dispatch**: Spawn 2 independent `teamwork_preview_reviewer` agents to inspect the Worker's diff, build output, correctness, and interface compliance.

   - **Challenger Dispatch**: Spawn 2 `teamwork_preview_challenger` agents to stress test clipping, text glyph bounds, sprite anchor alignment, and offscreen buffer rendering.

   - **Forensic Auditor Dispatch**: Spawn 1 `teamwork_preview_auditor` agent to perform integrity verification (verify authentic graphics implementation, zero hardcoded test outputs or dummy image stubs).

   - **Gate Check**: Record verdicts in `GATE_STATUS.md`. Gate passes ONLY if build/tests pass, both Reviewers APPROVE, both Challengers confirm correctness, and Auditor verdict is CLEAN.

3. **Handoff & Reporting**:
   - Write `handoff.md` in your working directory.
   - Send completion message to parent orchestrator.
