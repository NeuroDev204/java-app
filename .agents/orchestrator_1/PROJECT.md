# Project: J2ME Java Game Emulator Rendering Pipeline & JVM Engine (`java-app`)

## Architecture
The J2ME Emulator is structured into three main Go layers:
1. **JVM Interpreter Engine (`pkg/vm/`)**: `interpreter.go`, `opcodes.go`, `frame.go`, `object.go`, `slot.go`. Executes Java CLDC 1.1/MIDP 2.0 bytecode instructions, manages operand stack/local variables, and dispatches native methods (`java/lang/*`, `javax/microedition/*`).
2. **J2ME MIDP UI & Graphics Engine (`pkg/midp/`)**: `graphics.go`. Manages offscreen 240x320 RGBA buffer, clipping rectangle state, coordinate translation, anchor alignment, sprite transformations, and text rendering.
3. **Ebitengine Renderer & Main App (`pkg/gfx/`, `cmd/j2me/`)**: `renderer.go`, `keypad.go`, `main.go`. Manages window event loop, key mappings, frame buffer flushing, and PNG dump export (`emulator_buffer.png`).

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Clipping & Erase Prevention | Fix `setClip`/`clipRect` parameter passing in `interpreter.go`, fix `getClippedDst` bounds check when `ClipW<=0`, and eliminate black overwrite | M1 | Survey (R1) |
| 2 | Bitmap Font & Text Rendering | Implement glyph mask text rendering in `DrawString` and native `Font` stubs (`getFont`, `getDefaultFont`, `stringWidth`, `getHeight`) | M1 | Survey (R1) |
| 3 | Anchor & Transform Sprites | Implement MIDP 2.0 anchor alignment (`LEFT`, `RIGHT`, `HCENTER`, `TOP`, `BOTTOM`, `VCENTER`) and transform flips/rotations in `DrawImage`/`DrawRegion` | M1 | Survey (R1) |
| 4 | Buffer Export & Flushing | Continuous offscreen buffer flushing to `emulator_buffer.png` and `/tmp/emulator_buffer.png` without 5-frame cap | M1 | Survey (R1) |
| 5 | Jump Tables & Switch Opcodes | Implement variable-length padding-aligned `tableswitch` (0xAA) and `lookupswitch` (0xAB) opcodes | M2 | Survey (R2) |
| 6 | Bitwise & Arithmetic Opcodes | Implement `iand`, `ior`, `ixor`, `ishl`, `ishr`, `iushr`, `irem` opcodes | M2 | Survey (R2) |
| 7 | Control Flow & Object Opcodes | Implement comparison branches (`iflt`, `ifge`, `ifgt`, `ifle`, `if_icmpgt`, `if_icmple`, `if_acmpeq`, `if_acmpne`), `instanceof`, `checkcast`, `monitorenter`, `monitorexit`, and 5-byte `invokeinterface` | M2 | Survey (R2) |
| 8 | String & InputStream APIs | Implement `String.equals` match logic, `String.charAt` UTF-16/rune indexing, `indexOf` descriptors, `DataStreamReader.ReadBytes` buffer copy fix, and `readUTF()` | M2 | Survey (R2) |
| 9 | RecordStore Persistent Storage | Implement functional `RecordStore.setRecord` updates and `enumerateRecords` iterator stub | M2 | Survey (R2) |
| 10 | Unit & E2E Verification Suite | Expand unit test suite across `pkg/vm`, `pkg/midp`, `pkg/gfx`, `pkg/jar` and add headless E2E buffer validation test in `cmd/j2me/e2e_test.go` | M3 | Survey (Test) |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | J2ME Canvas & MIDP UI Rendering | Features 1, 2, 3, 4 (Clipping, `setClip`, font text rendering, `DrawImage`/`DrawRegion` anchor/transforms, buffer flushing) | none | DONE |
| M2 | JVM Bytecode Engine & MIDP APIs | Features 5, 6, 7, 8, 9 (`tableswitch`, `lookupswitch`, bitwise opcodes, branch/object opcodes, String.equals/charAt, InputStream, RecordStore) | M1 | DONE |
| M3 | Unit & E2E Verification & Hardening | Feature 10 (Unit test suite expansion, headless E2E `emulator_buffer.png` validation, 100% test pass under `LIBRARY_PATH=/tmp go test ./...`, adversarial check) | M1, M2 | DONE |

## Interface Contracts
### `pkg/midp` ↔ `pkg/vm`
- `ctx.Graphics.SetClip(x, y, w, h int)`: Called by `interpreter.go` native dispatch for `setClip` and `clipRect` with translation offset `(TransX, TransY)`.
- `ctx.Graphics.GetClipX() / GetClipY() / GetClipWidth() / GetClipHeight()`: Called by `interpreter.go` to return dynamic clip state.
- `DrawString(str string, x, y, anchor int)`: Renders text using Go basicfont/glyph mask onto `g.Buffer`.
- `DrawRegion(src *Image, x, y, w, h, transform, destX, destY, anchor int)`: Applies transformation matrix and anchor offsets to draw sub-rectangle.

### `pkg/vm` Native Method Dispatch
- `java/lang/String.equals(Ljava/lang/Object;)Z`: Returns `1` if string values match, `0` otherwise.
- `java/lang/String.charAt(I)C`: Pushes `int32([]rune(strVal)[idx])`.
- `javax/microedition/rms/RecordStore.setRecord(I[BII)V`: Updates `store.Records[recID-1]` slice.

## Code Layout
- Codebase root: `/home/neuro/Documents/Tools/java-app`
- Implementation packages: `cmd/j2me/`, `pkg/vm/`, `pkg/midp/`, `pkg/gfx/`, `pkg/jar/`, `pkg/classfile/`
- Test files: `cmd/j2me/*_test.go`, `pkg/vm/*_test.go`, `pkg/midp/*_test.go`, `pkg/gfx/*_test.go`, `pkg/jar/*_test.go`, `pkg/classfile/*_test.go`
