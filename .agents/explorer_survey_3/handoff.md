# Handoff Report: JAR Loading, JVM Main Invocation, and Input Forwarding Analysis

**Agent:** explorer_survey_3  
**Date:** 2026-08-09  
**Working Directory:** `/home/neuro/Documents/Tools/java-app/.agents/explorer_survey_3`  
**Target Project:** `/home/neuro/Documents/Tools/java-app`

---

## 1. Observation

1. **JVM Initialization & Classpath Handling**:
   - `pkg/jvm/jvm.go` lines 63-115 (`InitJVM`): Classpath parsing checks `candidates := []string{cpPath, "./j2me-stubs/bin", ...}` and evaluates `os.Stat(cand)` with `info.IsDir()`.
   - When a multi-path string separated by `:` (e.g., `./j2me-stubs/bin:/home/neuro/Downloads/KnightOnline_402.jar`) is passed, `os.Stat` fails because `:` is a path separator.
   - For `.jar` files, `info.IsDir()` returns `false`, causing single JAR path checks to fail.

2. **JVM Java Main Invocation**:
   - `pkg/jvm/callbacks.go` lines 124-138 (`InvokeMain`) and `pkg/jvm/bridge.c` lines 53-113 (`safe_invoke_main`).
   - `safe_invoke_main` resolves static `main([Ljava/lang/String;)V` and passes an empty string array `args` (`NewObjectArray(env, 0, stringClass, NULL)`).

3. **Input Handling**:
   - `pkg/gfx/renderer.go` lines 26-59 (`Update()`): Uses `ebiten.IsKeyPressed(k)` to check key state per frame.
   - `pkg/gfx/keypad.go` lines 5-28 (`MapEbitenKeyToJ2ME`): Maps `KeyUp` (-1), `KeyDown` (-2), `KeyLeft` (-3), `KeyRight` (-4), `KeyEnter` (-5), and digits 0-9.
   - Discrete key release transitions and mouse/touch pointer events (`pointerPressed`, `pointerReleased`, `pointerDragged`) are not currently polled or forwarded to JNI Canvas.

4. **Thread Concurrency & Render Queue**:
   - `pkg/jvm/callbacks.go` lines 36-85 (`RenderQueue`): Implements thread-safe FIFO queue (`sync.Mutex`) for native `go_fillRect` and `go_setColor` JNI callbacks.
   - Unit tests in `pkg/jvm/callbacks_test.go` and `pkg/jvm/challenger_m4_test.go` confirm `RenderQueue` handles multi-threaded JNI callbacks from Java without data races (`go test ./...` passes in 0.129s).

---

## 2. Logic Chain

1. **Observation**: `InitJVM` in `pkg/jvm/jvm.go` checks `os.Stat(cand)` and `info.IsDir()` on raw input string `cpPath`.
   - **Reasoning**: Composite classpaths formatted like `pathA:pathB` or JAR files are rejected by `os.Stat` and `info.IsDir()`.
   - **Deduction**: To support loading both `j2me-stubs/bin` and `KnightOnline_402.jar`, `InitJVM` must use `filepath.SplitList` to split by `os.PathListSeparator`, resolve absolute paths for both directories and JAR archives, and reconstruct the `-Djava.class.path=` JVM option.

2. **Observation**: `safe_invoke_main` in `bridge.c` builds an empty `String[]` array when invoking Java `main`.
   - **Reasoning**: A game launcher (e.g. `com.emulator.Main`) requires passing the JAR file path or configuration flags as command-line arguments.
   - **Deduction**: Extending `safe_invoke_main` or adding `InvokeMainWithArgs` in `callbacks.go` and `bridge.c` allows Go to pass `KnightOnline_402.jar` dynamically to Java `Main.main(String[] args)`.

3. **Observation**: Ebitengine `renderer.go` polls `ebiten.IsKeyPressed` in `Update()`, while J2ME `Canvas` expects discrete `keyPressed` and `keyReleased` events.
   - **Reasoning**: Polling `IsKeyPressed` on every frame sends repeated press events every tick rather than single state transition events.
   - **Deduction**: Utilizing `inpututil.IsKeyJustPressed`, `inpututil.IsKeyJustReleased`, and mouse cursor position helpers enables clean event generation for `keyPressed`, `keyReleased`, `pointerPressed`, `pointerReleased`, and `pointerDragged`.

4. **Observation**: Ebitengine owns thread 0 at 60 FPS, while Java code runs on background JVM threads.
   - **Reasoning**: Executing synchronous JNI calls directly from Ebitengine's `Update()` thread blocks window event processing if Java thread holds synchronization locks.
   - **Deduction**: An asynchronous, thread-safe `InputQueue` must buffer events from Ebitengine `Update()`, while a dedicated JVM-attached worker goroutine dispatches them to Java Canvas. Graphics callbacks should continue using `RenderQueue` or mutex-guarded offscreen buffers to prevent UI stalls.

---

## 3. Caveats

- **Assumptions**: Assumes JDK 21 OpenJDK is installed at `/usr/lib/jvm/java-21-openjdk-amd64` (as defined in `pkg/jvm/jvm.go` `#cgo CFLAGS` / `LDFLAGS`).
- **Unexplored Areas**: Network socket stubs (`javax.microedition.io`) and RMS storage (`RecordStore`) were not evaluated as part of this input/loading survey.
- **Alternative Interpretations**: Loading `KnightOnline_402.jar` via Java custom `URLClassLoader` instead of JVM system classpath was considered. System classpath via composite `-Djava.class.path=` was selected because it avoids reflection overhead and native JNI complexity.

---

## 4. Conclusion

1. **JVM Classpath**: `pkg/jvm/jvm.go` requires composite path resolution using `filepath.SplitList` to load both `j2me-stubs/bin` and `KnightOnline_402.jar` via system classpath.
2. **Input Forwarding**: Input handling should upgrade to `inpututil.IsKeyJustPressed`/`Released` and pointer events, mapped via `pkg/gfx/keypad.go` key codes, and registered against `g_current_canvas` in `bridge.c`.
3. **Thread Safety**: An asynchronous `InputQueue` combined with `RenderQueue` isolates the 60 FPS Ebitengine main thread from JVM thread locks, preventing window freezes and deadlocks.

---

## 5. Verification Method

1. **Unit Test Execution**:
   Run the full test suite in `/home/neuro/Documents/Tools/java-app`:
   ```bash
   go test -v ./pkg/jvm/...
   ```
   *Expected Output*: All tests pass, including `TestInitJVM`, `TestRenderQueueConcurrent`, `TestInvokeMainEndToEnd`, and `TestChallenger_ConcurrentJavaThreads`.

2. **File Inspection**:
   Inspect analysis report:
   `/home/neuro/Documents/Tools/java-app/.agents/explorer_survey_3/analysis.md`

3. **Invalidation Conditions**:
   - `go test ./...` fails.
   - `InitJVM` panics when passed composite classpath strings.
   - Ebitengine main thread blocks when JNI methods are invoked.
