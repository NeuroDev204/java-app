# Pure-Go Desktop Windows Executable Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone native Windows executable (`j2me-emulator.exe`) running the J2ME Knight Online client via pure-Go (`pkg/gamebridge` and Ebitengine) without requiring CGO or any external Java/JVM installation.

**Architecture:** A desktop application powered by Ebitengine (`ebiten.Game`) that directly wraps the pure-Go `pkg/gamebridge.Bridge`. The UI maintains an initial JAR selection menu (with native file picker dialog or command-line argument), dispatches keyboard (`gfx.MapEbitenKeyToJ2ME`) and mouse/touch events to the bridge, supports in-game text box input (`GetTextInputInfoJSON` / `SubmitTextInput`), and renders 60 FPS QVGA graphics from `bridge.GetFrameBuffer()` via `screen.WritePixels`.

**Tech Stack:** Go 1.24+, Ebitengine v2 (`github.com/hajimehoshi/ebiten/v2`), `github.com/sqweek/dialog`, pure-Go JVM engine (`pkg/vm`, `pkg/gamebridge`, `pkg/midp`, `pkg/jar`, `pkg/classfile`).

## Global Constraints

- **Zero CGO on Windows:** Must compile with `CGO_ENABLED=0` without requiring GCC/MinGW or external C headers.
- **Zero JVM Dependency:** Must not load `jvm.dll` or require JDK/JRE installed on the target machine.
- **Resolution & Aspect Ratio:** Fixed logical canvas of 240x320 scaled to 480x640 default window size with free resizing.
- **Non-destructive:** Keep existing Linux CGO HotSpot paths intact via Go build tags (`//go:build cgo` vs `//go:build !cgo`).

## Review Focus

1. **Window resizing / scaling:** Ebitengine's `Layout(w, h)` returning `(240, 320)` guarantees automatic aspect-ratio scaling without stretching or distortion.
2. **Cursor / Pointer mapping:** Mouse coordinates must scale properly from window space to 240x320 canvas space.
3. **Key mapping:** Softkeys (F1 = Left softkey, F2 = Right softkey), Arrow keys, Enter/Fire, and Alphanumeric keys mapped via `gfx.MapEbitenKeyToJ2ME`.
4. **Text Input Dialog:** When game opens a J2ME `TextBox` (e.g. login username/password), render an overlay and accept keyboard text typing, submitting with Enter or cancelling with Escape.
5. **Persistence:** RMS store saved to disk (`./rms.json` via `gamebridge.SetDataDir(".")`) so game progress and downloaded assets persist across restarts.

---

### Task 1: Verify & Finalize pure-Go compatibility in `pkg/gfx` and shared packages

**Files:**
- Modify: `pkg/gfx/renderer.go` (remove any leftover CGO directives)
- Test: `pkg/gfx/renderer_test.go`

**Interfaces:**
- Consumes: `pkg/midp.Graphics`, `github.com/hajimehoshi/ebiten/v2`
- Produces: pure-Go `gfx.EmulatorGame` and `gfx.MapEbitenKeyToJ2ME`

- [x] **Step 1: Verify `pkg/gfx` compiles cleanly with `CGO_ENABLED=0`**
- [x] **Step 2: Run all tests in `pkg/gfx` and verify PASS**

---

### Task 2: Implement Desktop Pure-Go Ebiten Runner

**Files:**
- Create: `cmd/desktop/main.go`
- Create / Modify: `main_purego.go` (build tag `!cgo`) and `main_cgo.go` (build tag `cgo`)
- Test: `cmd/desktop/main_test.go`

**Interfaces:**
- Consumes: `pkg/gamebridge.Bridge`, `pkg/gfx`, `github.com/hajimehoshi/ebiten/v2`, `github.com/sqweek/dialog`
- Produces: Standalone desktop executable entrypoint

- [x] **Step 1: Write desktop launcher tests in `cmd/desktop/main_test.go`**
  - Verify initialization of `DesktopGame` struct.
  - Verify default layout returns (240, 320).
  - Verify key and touch translation logic.

- [x] **Step 2: Implement `cmd/desktop/main.go`**
  - Implement `DesktopGame` implementing `ebiten.Game`.
  - Handle Menu state (Select JAR via `dialog.File()` or run default/passed JAR).
  - Call `gamebridge.SetDataDir(".")` and `bridge.LoadJAR(jarBytes)`.
  - In `Update()`:
    - Handle text input if `GetTextInputInfoJSON()` reports active.
    - Check ebiten keys and forward to `bridge.SendKeyDown` / `bridge.SendKeyUp`.
    - Check cursor and mouse buttons and forward to `bridge.SendTouchDown` / `bridge.SendTouchDrag` / `bridge.SendTouchUp`.
  - In `Draw()`:
    - If in menu, draw menu UI.
    - If in game, write framebuffer (`screen.WritePixels(bridge.GetFrameBuffer())`).
    - If text box active, draw input prompt overlay.
    - Render debug FPS and upgrade status.

- [x] **Step 3: Run desktop tests and verify PASS**

---

### Task 3: Build Windows Executable `j2me-emulator.exe` & Verify

**Files:**
- Output: `j2me-emulator.exe`

- [x] **Step 1: Build `j2me-emulator.exe` with `go build -o j2me-emulator.exe ./cmd/desktop`**
- [x] **Step 2: Verify binary file type (PE32+ Windows x64 executable)**
- [x] **Step 3: Test launching headless / smoke test verification**
