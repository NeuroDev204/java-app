# Handoff Report: Go Architecture, CGO Bridge & VRAM Image Management Investigation

**Agent:** `explorer_survey_2`  
**Working Directory:** `/home/neuro/Documents/Tools/java-app/.agents/explorer_survey_2/`  
**Target:** `/home/neuro/Documents/Tools/java-app/`  
**Date:** 2026-08-09  

---

## 1. Observation

1. **Root Main & Thread Lock**:
   - `main.go:41`: `runtime.LockOSThread()` is invoked prior to calling `jvm.InitJVM(...)` at `main.go:44` and `ebiten.RunGame(game)` at `main.go:59`.
2. **JVM Initialization & JNI Registration**:
   - `pkg/jvm/jvm.go:65`: Locks OS thread inside `InitJVM` and calls `C.create_java_vm(&vm, &env, cOpt)` at line 102.
   - `pkg/jvm/jvm.go:112`: Calls `globalJVM.RegisterNatives()` which calls `C.register_graphics_natives` in `pkg/jvm/bridge.c:13`.
   - `pkg/jvm/bridge.c:41-44`: Binds `Graphics.fillRect` and `Graphics.setColor` using `JNINativeMethod methods[]`.
3. **CGO Callback Exports & Render Queue**:
   - `pkg/jvm/callbacks.go:87-108`: Functions `go_fillRect` and `go_setColor` are annotated with `//export`. They push commands into a thread-safe `RenderQueue` (`sync.Mutex` guarding `[]RenderCommand`).
4. **Current Ebitengine Rendering**:
   - `pkg/gfx/renderer.go:90-115`: `EmulatorGame.Draw` copies CPU software pixel buffer (`g.Graphics.Buffer.Pix`) to GPU memory using `g.offscreen.WritePixels(...)`.
5. **Test Suite Status**:
   - Running `go test ./...` in `/home/neuro/Documents/Tools/java-app` exits with code 0 (`ok java-app`, `ok java-app/pkg/jvm`, `ok java-app/pkg/gfx`, etc.).

---

## 2. Logic Chain

1. **From Observation 1 & 2**: JNI calls and Ebitengine graphics rendering both depend on OS thread stability. Calling `runtime.LockOSThread()` early in `main.go` and within `InitJVM()` ensures that JNI function pointers (`JNIEnv*`) and graphics renderer context remain bound to the same OS thread without goroutine migration issues.
2. **From Observation 2 & 3**: The existing CGO bridge in `pkg/jvm/bridge.c` and `callbacks.go` proves that JNI calls from Java stubs safely execute Go logic. Using `RegisterNatives` allows clean signature binding (`"(IIII)V"`) without complex C name mangling.
3. **From Observation 4**: The existing renderer copies CPU `image.RGBA` pixel buffers to Ebitengine per frame. To achieve high performance in Phase 2, images created via `Image.createImage` should be decoded directly in Go and stored in a VRAM registry (`map[int32]*ebiten.Image`) returning integer IDs (`int nativeId`) back to Java.
4. **From Observation 2, 3 & 4**: Standard J2ME offscreen rendering (`Image.getGraphics()`) can be supported by passing target handle IDs (0 for main screen canvas, >0 for offscreen images) to Go drawing callbacks, executing direct drawing operations on the underlying `*ebiten.Image`.

---

## 3. Caveats

- **JAR Loading**: `com.emulator.Main` launcher class and custom JAR classloading mechanism were analyzed conceptually based on `ORIGINAL_REQUEST.md` and phase 2 plans, but Java JAR byte loading code was not directly executed against `KnightOnline_402.jar` as this is a read-only investigation.
- **Audio & RecordStore (RMS)**: Focused on graphics, VRAM, and JNI callback architecture. Audio (`javax.microedition.media`) and RMS file persistence stubs were not part of this specific survey step.

---

## 4. Conclusion

The Go CGO/JNI architecture in `/home/neuro/Documents/Tools/java-app/` is structurally sound and ready for full J2ME stub migration. 

Recommended implementation steps:
1. Implement `pkg/jvm/vram.go` defining `VRAMRegistry` (`map[int32]*ebiten.Image`).
2. Write native proxy Java stubs for `Image.java` and `Canvas.java` in `j2me-stubs/src/javax/microedition/lcdui/`.
3. Expand `pkg/jvm/bridge.c` and `pkg/jvm/callbacks.go` to register `createImage`, `getGraphics`, `serviceRepaints`, and draw image calls via `RegisterNatives`.
4. Update `main.go` to invoke `com.emulator.Main.startApp("/path/to/KnightOnline_402.jar")` and process Ebitengine frame repaints.

Full details are documented in `.agents/explorer_survey_2/analysis.md`.

---

## 5. Verification Method

1. **Inspect Analysis File**:
   - Check file existence and contents at `/home/neuro/Documents/Tools/java-app/.agents/explorer_survey_2/analysis.md`.
2. **Verify Go Build & Test Suite**:
   - Run `go test ./...` inside `/home/neuro/Documents/Tools/java-app/` — must return 0 errors.
   - Run `go build -o j2mego` — must link cleanly against `libjvm.so`.
3. **Verify Java Stub Compilation**:
   - Run `javac -d j2me-stubs/bin $(find j2me-stubs/src -name "*.java")` — must compile cleanly without AWT/Swing dependencies.
