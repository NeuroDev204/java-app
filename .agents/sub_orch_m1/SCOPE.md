# Scope: Milestone 1 — J2ME Canvas & MIDP UI Rendering Pipeline

## Scope Overview
Complete the J2ME Canvas & MIDP UI Rendering pipeline for KnightOnline MIDlet suite in Go (`java-app`) so that main menu graphics, server selection, dialog popups, font text strings, and game world sprites render cleanly on offscreen canvas without black clipping.

## Target Package Directories
- `/home/neuro/Documents/Tools/java-app/pkg/midp/`
- `/home/neuro/Documents/Tools/java-app/pkg/vm/`
- `/home/neuro/Documents/Tools/java-app/pkg/gfx/`

## Detailed Implementation Requirements
1. **Clipping & Erase Prevention (`pkg/midp/graphics.go` & `pkg/vm/interpreter.go`)**:
   - `pkg/midp/graphics.go`: Update `getClippedDst(dst image.Rectangle)` to return `image.ZR` (empty rect) whenever `g.ClipW <= 0 || g.ClipH <= 0`.
   - `pkg/vm/interpreter.go`: In `handleNativeCall` for `setClip` and `clipRect`, extract `(x, y, w, h)`, apply `g.TransX` / `g.TransY`, and call `ctx.Graphics.SetClip(x, y, w, h)` / `ClipRect(x, y, w, h)` instead of discarding parameters.
   - Update `getClipX`, `getClipY`, `getClipWidth`, `getClipHeight` in `interpreter.go` to return dynamic clip values from `ctx.Graphics` relative to current translation.
   - Remove/refactor the fragile `[GFX-PREVENT-ERASE]` workaround in `graphics.go` once clipping state is properly managed.

2. **Bitmap Font & Text Rendering (`pkg/midp/graphics.go` & `pkg/vm/interpreter.go`)**:
   - `pkg/midp/graphics.go`: Replace solid box uniform block drawing in `DrawString` with actual character glyph rendering (using standard 8x13 font or basicfont glyph masks onto `g.Buffer`).
   - `pkg/vm/interpreter.go`: Implement native stubs for `javax/microedition/lcdui/Font` methods (`getFont`, `getDefaultFont`, `stringWidth`, `charWidth`, `getHeight`, `getBaselinePosition`).

3. **Anchor & Transform Sprites (`pkg/midp/graphics.go` & `pkg/vm/interpreter.go`)**:
   - `pkg/midp/graphics.go`: Implement J2ME anchor offset calculations (`LEFT`, `RIGHT`, `HCENTER`, `TOP`, `BOTTOM`, `VCENTER`) for `DrawImage` and `DrawRegion`.
   - Implement pixel transformation matrix / axis swaps for `transform` flags in `DrawRegion` (`TRANS_NONE`, `TRANS_ROT90`, `TRANS_ROT180`, `TRANS_ROT270`, `TRANS_MIRROR`, `TRANS_MIRROR_ROT90`, etc.).

4. **Offscreen Buffer Flushing (`pkg/gfx/renderer.go`)**:
   - Update `renderer.go` frame dump logic so `/tmp/emulator_buffer.png` and `./emulator_buffer.png` are flushed on paint updates without a 5-frame cap (`drawLogCnt < 5`).

## Verification Commands
- `LIBRARY_PATH=/tmp go test ./pkg/midp ./pkg/gfx ./pkg/vm -v`

## Milestone Status
Status: **DONE**
Gate Verdict: **PASS** (Iteration 2)
Verified by: Reviewer 1, Reviewer 2, Challenger 1, Challenger 2, Forensic Auditor

