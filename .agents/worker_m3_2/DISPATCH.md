# DISPATCH — Worker M3 (Remediation Round 2)

## Mission
Fix the test failure bugs, cross-package shared resource race conditions, and buffer flush truncation issues so that `LIBRARY_PATH=/tmp go test ./... -v -count=1` passes 100% cleanly across all packages.

## Context & Feedback
- Reviewer 2 Handoff Report: /home/neuro/Documents/Tools/java-app/.agents/reviewer_m3_2/handoff.md
- Scope document: /home/neuro/Documents/Tools/java-app/.agents/sub_orch_m3/SCOPE.md
- Master Project Plan: /home/neuro/Documents/Tools/java-app/.agents/orchestrator_1/PROJECT.md

## Issues to Remediate
1. **Cross-Package Resource Race Condition**:
   - `cmd/j2me/e2e_test.go` and `pkg/gfx/renderer_stress_test.go` collide when `go test ./...` runs packages in parallel because both write to global fixed file paths (`/tmp/emulator_buffer.png` and `emulator_buffer.png`).
   - Fix: Ensure tests use isolated or unique buffer file paths or configurable buffer dump paths (e.g. environment variable override, function parameter, or package test isolation/synchronization) so `cmd/j2me/e2e_test.go` and `pkg/gfx` tests do not overwrite each other's test outputs during parallel `go test ./...`.

2. **Buffer Flush Truncation Bug (`pkg/gfx/renderer.go`)**:
   - `flushBufferToDisk()` in `pkg/gfx/renderer.go` creates 33-byte partial PNG files under continuous flushing (`TestContinuousBufferFlushing`).
   - Fix: Ensure `png.Encode()` output is completely flushed/synced with `f.Sync()`, check return errors from `f.Sync()` and `f.Close()`, and ensure `os.Rename()` only replaces target files after complete write and flush.

3. **Verification Command**:
   - Execute `LIBRARY_PATH=/tmp go test ./... -v -count=1` and ensure ALL tests in ALL packages pass with 0 errors and exit code 0.
   - Include authentic command execution output log in `/home/neuro/Documents/Tools/java-app/.agents/worker_m3_2/handoff.md`.

## Mandatory Integrity Warning
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or fabricate log evidence. A teamwork_preview_auditor will independently verify your work.
