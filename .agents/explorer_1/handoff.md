# Handoff Report — Requirement R1: J2ME Canvas & MIDP UI Rendering Exploration

**Explorer**: Explorer 1  
**Target Component**: Requirement R1 (J2ME Canvas & MIDP UI Rendering Pipeline)  
**Working Directory**: `/home/neuro/Documents/Tools/java-app/.agents/explorer_1`  
**Codebase Directory**: `/home/neuro/Documents/Tools/java-app`  

---

## 1. Observation

### Focus Area 1: Offscreen Buffer Rendering & Image Flushing
- **File**: `pkg/midp/graphics.go` (Lines 22–36)
  - `NewGraphics(w, h int)` initializes `g.Buffer` as an `*image.RGBA` of size `w x h` (default 240x320) with background color `RGBA{R: 15, G: 25, B: 45, A: 255}`.
- **File**: `pkg/gfx/renderer.go` (Lines 56–86)
  - `EmulatorGame.Draw(screen *ebiten.Image)` writes `g.Graphics.Buffer.Pix` directly into Ebitengine's offscreen surface (`g.offscreen.WritePixels(...)`) and renders it to `screen`.
  - Lines 76–81 save the frame buffer to disk:
    ```go
    f, err := os.Create("/tmp/emulator_buffer.png")
    if err == nil {
        png.Encode(f, g.Graphics.Buffer)
        f.Close()
        log.Printf("[RENDER-DUMP] Wrote /tmp/emulator_buffer.png")
    }
    ```
  - **Issue**: The dump logic is guarded by `if drawLogCnt < 5` (Line 67). After 5 frames, no further updates to `/tmp/emulator_buffer.png` are saved. Furthermore, the acceptance criteria requires `emulator_buffer.png` to contain the full game menu graphics, UI buttons, and text.

---

### Focus Area 2: Root Cause of Full-Screen Black Overwrite & Canvas Clipping
- **File**: `pkg/midp/graphics.go` (Lines 47–70)
  ```go
  func (g *Graphics) SetClip(x, y, w, h int) {
      g.ClipX = x + g.TransX
      g.ClipY = y + g.TransY
      g.ClipW = w
      g.ClipH = h
  }

  func (g *Graphics) getClippedDst(dst image.Rectangle) image.Rectangle {
      if g.ClipW <= 0 || g.ClipH <= 0 {
          return dst // <--- BUG 1: Returns unclipped destination rectangle when clip is empty!
      }
      clip := image.Rect(g.ClipX, g.ClipY, g.ClipX+g.ClipW, g.ClipY+g.ClipH)
      return dst.Intersect(clip)
  }
  ```
  - **BUG 1**: `getClippedDst` checks `if g.ClipW <= 0 || g.ClipH <= 0` and returns `dst` (the full target rectangle). In standard J2ME graphics, an empty or non-positive clip means **nothing** should be rendered (return `image.ZR` / empty rectangle).
- **File**: `pkg/vm/interpreter.go` (Lines 1425–1447)
  ```go
  } else if mName == "setClip" || mName == "clipRect" {
      h := frame.OperandStack.PopInt()
      w := frame.OperandStack.PopInt()
      y := frame.OperandStack.PopInt()
      x := frame.OperandStack.PopInt()
      frame.OperandStack.PopRef()
      _ = x
      _ = y
      _ = w
      _ = h
      return
  }
  ```
  - **BUG 2**: `handleNativeCall` intercepts `setClip` and `clipRect` calls from J2ME bytecode and discards the parameters (`_ = x`, etc.) without calling `SetClip` or `ClipRect` on `ctx.Graphics`. As a result, `g.ClipX`, `ClipY`, `ClipW`, `ClipH` on `ctx.Graphics` remain at default or stale state.
  - **BUG 3**: `getClipX` / `getClipY` / `getClipWidth` / `getClipHeight` in `handleNativeCall` (Lines 1436–1447) return hardcoded constants `0, 0, 240, 320` instead of querying `ctx.Graphics`.
- **File**: `pkg/midp/graphics.go` (Lines 80–83)
  ```go
  r, gVal, b, _ := g.CurrentColor.RGBA()
  if w >= g.Width && h >= g.Height && r == 0 && gVal == 0 && b == 0 {
      log.Printf("[GFX-PREVENT-ERASE] Ignored full-screen black FillRect(%d,%d,%d,%d)", x, y, w, h)
      return
  }
  ```
  - **HACK**: The existing codebase contains a fragile hardcoded workaround `[GFX-PREVENT-ERASE]` that skips full-screen black `FillRect` calls. However, when J2ME games call `FillRect` with clipping active or slightly smaller dimensions (e.g. 240x319), or when `TransY` is non-zero, this hack is bypassed and the full-screen buffer is overwritten with opaque black.

---

### Focus Area 3: Main Menu, Font Text Strings, Sprites & Dialog Popups
- **File**: `pkg/midp/graphics.go` (Lines 181–201) — `DrawString`
  ```go
  func (g *Graphics) DrawString(str string, x, y, anchor int) {
      ...
      rect := g.getClippedDst(image.Rect(x, y, x+w, y+h))
      if !rect.Empty() {
          textColor := g.CurrentColor
          ...
          draw.Draw(g.Buffer, rect, &image.Uniform{C: textColor}, image.Point{}, draw.Over)
      }
  }
  ```
  - **BUG 4**: `DrawString` does NOT render actual text glyphs! Instead, it draws a solid filled rectangular block (`&image.Uniform{C: textColor}`) over the text area. This causes all text strings (menu titles, server names, button labels, dialog prompts) to render as solid yellow/colored blocks.
  - **BUG 5**: `javax.microedition.lcdui.Font` class is missing completely. J2ME Font methods (`Font.getFont`, `Font.getDefaultFont`, `font.stringWidth`, `font.getHeight`) are missing in `pkg/vm/interpreter.go`.
- **File**: `pkg/midp/graphics.go` (Lines 144–179) — `DrawImage` & `DrawRegion`
  - **BUG 6**: `DrawImage` ignores the J2ME `anchor` parameter (discards anchor in `interpreter.go` Line 1510).
  - **BUG 7**: `DrawRegion` ignores both `anchor` and `transform` parameters (Line 162: `_ = anchor; _ = transform`). Sprite flips (`TRANS_MIRROR`, `TRANS_ROT90`, etc.) and anchor alignments (`HCENTER`, `VCENTER`, `BOTTOM`, `RIGHT`) are completely ignored, resulting in misaligned sprites, upside-down menu borders, and misplaced dialog popups.
- **File**: `pkg/vm/interpreter.go` (Lines 842–880 & 1655–1679) — Resource Image Fallbacks
  - When image resource loading fails or decoding returns an error, `createImage` calls `makeFallbackImage(24, 24)` (Lines 826–840), returning a 24x24 yellow/orange placeholder square. Hardcoded paths in `getResourceBytes` fail to map relative JAR paths (e.g. `/img/logo.png` vs `img/logo.png`).

---

### Focus Area 4: Existing Graphics Package Structure & Missing Routines Summary
- **Existing Package Structure**:
  - `pkg/gfx/`:
    - `renderer.go`: Ebitengine `EmulatorGame` loop (`Update`, `Draw`, `Layout`), image dump to `/tmp/emulator_buffer.png`.
    - `keypad.go`: Mapping Ebiten keys to J2ME key codes (`-1` UP, `-2` DOWN, `-3` LEFT, `-4` RIGHT, `-5` FIRE, `'0'`..`'9'`).
  - `pkg/midp/`:
    - `graphics.go`: `Graphics` struct, canvas buffer drawing primitives (`SetColor`, `Translate`, `SetClip`, `ClipRect`, `FillRect`, `DrawRect`, `DrawLine`, `DrawImage`, `DrawRegion`, `DrawString`, `DecodeImage`).
  - `pkg/vm/`:
    - `interpreter.go`: JVM bytecode execution engine and native method dispatcher for `javax/microedition/lcdui/*` and `java/lang/*`.

- **Complete List of Missing / Incomplete Routines for Requirement R1**:
  1. `pkg/midp/graphics.go` — `getClippedDst`: Fix invalid unclipped fallback when `ClipW <= 0 || ClipH <= 0`.
  2. `pkg/vm/interpreter.go` — `setClip` & `clipRect`: Implement actual state update on `ctx.Graphics` with translation offset.
  3. `pkg/vm/interpreter.go` — `getClipX`, `getClipY`, `getClipWidth`, `getClipHeight`: Return dynamic values from `ctx.Graphics` relative to current translation.
  4. `pkg/midp/graphics.go` — `DrawString`: Replace solid uniform block drawing with character glyph font rendering (e.g. using standard 8x13 / basicfont glyph rendering onto `g.Buffer`).
  5. `pkg/midp/graphics.go` — `DrawImage` & `DrawRegion`: Implement J2ME anchor offset calculations (`TOP`, `BOTTOM`, `VCENTER`, `LEFT`, `RIGHT`, `HCENTER`) and sprite transformation matrix/flips (`TRANS_NONE`, `TRANS_ROT90`, `TRANS_ROT180`, `TRANS_ROT270`, `TRANS_MIRROR`, etc.).
  6. `pkg/vm/interpreter.go` — `Font` stubs: Implement native stubs for `javax/microedition/lcdui/Font` (`getFont`, `getDefaultFont`, `stringWidth`, `charWidth`, `getHeight`, `getBaselinePosition`).
  7. `pkg/vm/interpreter.go` — `drawRGB`: Implement pixel array rendering with ARGB alpha blending.
  8. `pkg/gfx/renderer.go` — Offscreen buffer flushing: Remove 5-frame dump limit, ensuring `emulator_buffer.png` is flushed continuously or upon paint completion to `/tmp/emulator_buffer.png` and `./emulator_buffer.png`.

---

## 2. Logic Chain

1. **Observation**: J2ME MIDlet calls `g.setClip(x, y, w, h)` prior to rendering UI popups or dirty canvas regions.
2. **Observation**: In `interpreter.go` (Line 1425), `setClip` pops parameters and discards them (`_ = x`). `ctx.Graphics.SetClip` is never invoked.
3. **Observation**: `ctx.Graphics` retains its default clip `(0, 0, 240, 320)`. If `setClip(0, 0, 0, 0)` is called to mute drawing, `getClippedDst` in `graphics.go` (Line 65) returns `dst` (unclipped destination) because `ClipW <= 0` returns `dst` instead of an empty rectangle.
4. **Logic Step**: Any call to `g.fillRect(0, 0, 240, 320)` with color black bypasses clipping and paints the entire offscreen `RGBA` buffer black.
5. **Logic Step**: The workaround `[GFX-PREVENT-ERASE]` in `graphics.go` (Line 80) attempts to ignore full-screen black fills, but fails whenever coordinates or dimensions vary slightly or when `TransY` is set.
6. **Observation**: `DrawString` in `graphics.go` (Line 199) uses `draw.Draw(g.Buffer, rect, &image.Uniform{C: textColor}, image.Point{}, draw.Over)` which fills a solid color box instead of rendering font character bitmap masks.
7. **Logic Step**: Consequently, text strings on main menu, server selection list, and dialog popups render as solid yellow/colored rectangular boxes.
8. **Observation**: `DrawRegion` in `graphics.go` (Line 162) explicitly ignores `anchor` and `transform`.
9. **Logic Step**: Sprites rendered using MIDP 2.0 transformations (e.g. character sprites, menu selection arrows, UI borders) draw at incorrect positions without rotation/flips.

---

## 3. Caveats

- **Read-Only Exploration**: This analysis was conducted strictly read-only. No source code was modified, and no build or test execution commands were run.
- **JVM Interpreter Scope**: Requirement R2 (opcodes, String methods, InputStream reading, RecordStore) is analyzed separately by Explorer 2/3.
- **Font Rendering Dependency**: Glyph rendering for `DrawString` can be accomplished using standard Go `golang.org/x/image/font/basicfont` or a embedded 8x13 bitmap font array to maintain zero external Cgo dependencies.

---

## 4. Conclusion

Requirement R1 (J2ME Canvas & MIDP UI Rendering) currently fails due to three main architectural bugs in the rendering pipeline:
1. **Clipping & Black Overwrite Bug**: `setClip`/`clipRect` native stubs ignore arguments, and `getClippedDst` returns unclipped bounds when clip width/height is zero or negative.
2. **Solid Box Text Bug**: `DrawString` draws solid uniform rectangles instead of font character glyphs.
3. **Anchor & Transform Gaps**: `DrawImage` and `DrawRegion` ignore MIDP 2.0 `anchor` and `transform` parameters, misaligning menu buttons and game sprites.

### Recommended Fix Strategy:
1. **Fix Clip Mechanics (`pkg/midp/graphics.go` & `pkg/vm/interpreter.go`)**:
   - Update `getClippedDst`: return `image.ZR` when `ClipW <= 0 || ClipH <= 0`.
   - Update `handleNativeCall` for `setClip` and `clipRect`: invoke `g.SetClip(x, y, w, h)` and `g.ClipRect(x, y, w, h)`.
   - Remove the `[GFX-PREVENT-ERASE]` hack once clipping state is correctly managed.
2. **Implement Glyph Text Rendering (`pkg/midp/graphics.go`)**:
   - Integrate a simple bitmap font renderer (e.g. `basicfont.Face7x13` or ASCII character bitmap table) into `DrawString` to draw character masks onto `g.Buffer`.
3. **Implement MIDP 2.0 Anchor & Transform (`pkg/midp/graphics.go`)**:
   - Compute `(x, y)` offsets for `anchor` flags (`LEFT`, `RIGHT`, `HCENTER`, `TOP`, `BOTTOM`, `VCENTER`).
   - Implement pixel transformation matrix / axis swaps for `transform` flags (`TRANS_MIRROR`, `TRANS_ROT90`, `TRANS_ROT180`, `TRANS_ROT270`, etc.).
4. **Buffer Flushing & Font Native Stubs (`pkg/gfx/renderer.go` & `pkg/vm/interpreter.go`)**:
   - Update `renderer.go` to save `emulator_buffer.png` continuously on paint updates.
   - Add native stubs for `javax/microedition/lcdui/Font` methods.

---

## 5. Verification Method

Once changes are implemented by the builder agent:
1. **Unit Verification**:
   ```bash
   LIBRARY_PATH=/tmp go test ./pkg/midp ./pkg/gfx ./pkg/vm -v
   ```
2. **Visual & Buffer Verification**:
   - Execute the emulator with a test J2ME MIDlet `.jar`:
     ```bash
     go run ./cmd/j2me <path-to-game.jar>
     ```
   - Verify that `/tmp/emulator_buffer.png` and `emulator_buffer.png` contain crisp, readable font text strings, menu buttons, server selection text, and correctly aligned sprites without any full-screen black overwrite flashes.
