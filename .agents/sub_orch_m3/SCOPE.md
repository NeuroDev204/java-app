# Scope: Milestone 3 — Unit & E2E Verification & Hardening

## Status: DONE

## Scope Overview
Expand the unit test suite across all Go packages (`pkg/vm`, `pkg/midp`, `pkg/gfx`, `pkg/jar`, `pkg/classfile`, `cmd/j2me`), build a headless E2E verification test harness in `cmd/j2me/e2e_test.go` that decodes and validates `emulator_buffer.png` PNG pixel rendering without requiring an interactive X11 GUI display, and perform adversarial coverage hardening.

## Target Package Directories
- `/home/neuro/Documents/Tools/java-app/cmd/j2me/`
- `/home/neuro/Documents/Tools/java-app/pkg/vm/`
- `/home/neuro/Documents/Tools/java-app/pkg/midp/`
- `/home/neuro/Documents/Tools/java-app/pkg/gfx/`
- `/home/neuro/Documents/Tools/java-app/pkg/jar/`

## Detailed Implementation Requirements
1. **Unit Test Expansion across Packages**:
   - `pkg/vm`: Add tests for `tableswitch`, `lookupswitch`, bitwise opcodes (`iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr`, `irem`), conditional branch instructions, `instanceof`, `checkcast`, `monitorenter`/`monitorexit`, `invokeinterface`, `String.equals`, `String.charAt` (rune indexing), `String.indexOf`, `DataStreamReader.ReadBytes`, `readUTF()`, and `RecordStore.setRecord`.
   - `pkg/midp`: Add tests for clipping bounds handling (`getClippedDst`), `DrawString` bitmap font rendering, `DrawImage` anchor alignments, `DrawRegion` sprite transformations, and `DecodeImage`.
   - `pkg/gfx`: Add tests for continuous offscreen buffer flushing and `emulator_buffer.png` file output.
   - `pkg/jar`: Add tests for JAR manifest parsing and class file reader.

2. **Headless E2E Test Harness (`cmd/j2me/e2e_test.go`)**:
   - Instantiate `midp.NewGraphics(240, 320)` and `vm.NewVMContext`.
   - Simulate a full J2ME Canvas `paint()` frame cycle executing background fill, menu buttons, server selection text strings, font text, and gameplay sprites.
   - Flush the offscreen canvas to `./emulator_buffer.png` and `/tmp/emulator_buffer.png`.
   - Decode `./emulator_buffer.png` or `/tmp/emulator_buffer.png` using Go's `image/png`.
   - Verify acceptance criteria:
     - Non-zero pixel count > 50,000 (ensuring screen is not black/blank).
     - Menu button regions contain expected non-black color pixels.
     - Bitmap font text string bounding boxes contain rendered font pixels.
     - No full-screen black overwrite occurs during `paint()` cycles.

3. **Phase 2 Hardening**:
   - Perform adversarial white-box coverage analysis on implementation packages to verify zero untested edge cases, zero fake stubs, and zero regressions.

## Verification Commands
- `LIBRARY_PATH=/tmp go test ./... -v -count=1`
