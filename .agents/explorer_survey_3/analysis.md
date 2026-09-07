# Technical Analysis Report: JAR Loading, JVM Main Invocation, and Input Forwarding

**Author:** explorer_survey_3  
**Date:** 2026-08-09  
**Target System:** J2ME Go Engine (`/home/neuro/Documents/Tools/java-app/`)

---

## Executive Summary

This report presents a detailed analysis of embedding a Java Virtual Machine (JVM) inside a Go executable using CGO/JNI, loading external J2ME game archives (`KnightOnline_402.jar`), capturing and forwarding Ebitengine input events to Java Canvas, and maintaining thread synchronization between Ebitengine's 60 FPS main thread loop and Java's background execution threads.

Key Findings:
1. **JVM Classpath Resolution**: The current classpath parser in `pkg/jvm/jvm.go` fails when handling multi-path composite classpaths containing `: ` or single `.jar` files because `os.Stat(cand)` is invoked on the raw string and requires `info.IsDir() == true`. Supporting composite classpaths (`j2me-stubs/bin:KnightOnline_402.jar`) requires path list splitting via `filepath.SplitList` and absolute path expansion for both directories and JAR archives.
2. **Input Forwarding Pipeline**: Ebitengine's `Update()` loop currently uses `ebiten.IsKeyPressed` (held key detection), which is insufficient for J2ME games expecting discrete `keyPressed` and `keyReleased` state transitions. Upgrading to `inpututil.IsKeyJustPressed` and `inpututil.IsKeyJustReleased`, along with mouse/cursor pointer event interception (`pointerPressed`, `pointerReleased`, `pointerDragged`), allows full touch and keypad input capture.
3. **Thread Synchronization & Deadlock Prevention**: Calling JNI methods directly on Ebitengine's main thread inside `Update()` introduces deadlock and window freeze risks if Java code blocks or waits on thread synchronization primitives. Implementing an asynchronous, mutex-protected `InputQueue` and maintaining double-buffered graphics via `RenderQueue` decouples the Ebitengine window thread from JVM execution.

---

## 1. JVM Initialization & Classpath Configuration

### 1.1 CGO JVM Initialization Mechanism
- **Location**: `pkg/jvm/jvm.go` (lines 1-135)
- **Implementation**: The JVM is initialized via CGO invoking `JNI_CreateJavaVM` inside `create_java_vm` (C preamble, lines 11-28).
- **Initialization Flags**:
  - `-Djava.class.path=<classpath>`: Defines system classpath for JVM class loading.
  - `-Xrs`: Disables JVM signal handling for OS signals (SIGSEGV, SIGILL, SIGINT), preventing HotSpot JVM from interfering with Go runtime signal handlers.
- **Singleton Lifecycle**: `InitJVM(classpath string)` locks an OS thread via `runtime.LockOSThread()`, acquires `jvmMutex`, and returns `globalJVM` if already initialized (lines 65-72).

```go
// pkg/jvm/jvm.go:63-72
func InitJVM(classpath string) (*JVM, error) {
	runtime.LockOSThread()

	jvmMutex.Lock()
	defer jvmMutex.Unlock()

	if globalJVM != nil {
		return globalJVM, nil
	}
	...
}
```

### 1.2 Classpath Resolution Limitations & Multi-Entry Enhancement
- **Current Logic Analysis** (`pkg/jvm/jvm.go`, lines 78-92):
  ```go
  candidates := []string{cpPath, "./j2me-stubs/bin", "../j2me-stubs/bin", "../../j2me-stubs/bin"}
  finalCP := cpPath
  for _, cand := range candidates {
      if info, err := os.Stat(cand); err == nil && info.IsDir() {
          if abs, err := filepath.Abs(cand); err == nil {
              finalCP = abs
              break
          }
      }
  }
  ```
- **Defects Identified**:
  1. **Composite Classpath Failure**: If `classpath` contains multiple paths separated by `:` (e.g. `./j2me-stubs/bin:/home/neuro/Downloads/KnightOnline_402.jar`), `os.Stat(cand)` is executed on the concatenated string `"./j2me-stubs/bin:..."` and fails with `os.ErrNotExist`.
  2. **JAR File Exclusion**: Line 86 checks `info.IsDir()`. For a `.jar` archive file, `info.IsDir()` is `false`, causing `os.Stat` verification to fail for JAR files even when the file exists.

- **Proposed Multi-Entry Classpath Resolution**:
  To allow JVM System ClassLoader to load both stub classes (`javax.microedition.lcdui.*`) and game classes (`com.silverknight.*` inside `KnightOnline_402.jar`), `InitJVM` should parse composite classpaths as follows:

  ```go
  func buildCompositeClasspath(cp string) string {
      if cp == "" {
          cp = "./j2me-stubs/bin"
      }
      if strings.HasPrefix(cp, "-Djava.class.path=") {
          cp = cp[17:]
      }
      parts := filepath.SplitList(cp)
      var resolvedParts []string
      for _, part := range parts {
          absPath, err := filepath.Abs(part)
          if err == nil {
              if _, err := os.Stat(absPath); err == nil {
                  resolvedParts = append(resolvedParts, absPath)
                  continue
              }
          }
          resolvedParts = append(resolvedParts, part)
      }
      return strings.Join(resolvedParts, string(os.PathListSeparator))
  }
  ```

- **System Classpath vs Custom ClassLoader Architecture**:
  By passing `-Djava.class.path=/abs/path/j2me-stubs/bin:/abs/path/KnightOnline_402.jar` to `JNI_CreateJavaVM`, JVM's System ClassLoader (`ClassLoader.getSystemClassLoader()`) natively locates and loads classes from both directories and JAR archives without requiring complex bytecode manipulation or custom Java classloaders.

### 1.3 Java `main` Invocation Architecture
- **Location**: `pkg/jvm/callbacks.go` (lines 124-138), `pkg/jvm/bridge.c` (lines 53-113)
- **Current Functionality**: `safe_invoke_main(vm, env, className)` performs JNI lookup:
  1. `FindClass(env, className)`
  2. `GetStaticMethodID(env, clazz, "main", "([Ljava/lang/String;)V")`
  3. Creates empty `String[]` array (`NewObjectArray(env, 0, stringClass, NULL)`).
  4. `CallStaticVoidMethod(env, clazz, mainMethod, args)`
- **Game Launcher Integration Strategy**:
  1. Create Java game launcher class `com.emulator.Main` inside `j2me-stubs/src/com/emulator/Main.java`:
     ```java
     package com.emulator;

     public class Main {
         public static void main(String[] args) {
             String jarPath = (args.length > 0) ? args[0] : "KnightOnline_402.jar";
             System.out.println("[JAVA LAUNCHER] Booting J2ME Game: " + jarPath);
             // Launch MIDlet class using game initialization sequence
         }
     }
     ```
  2. Enhance `safe_invoke_main` in `bridge.c` / `callbacks.go` to accept an array of String arguments (`args []string`) so Go can pass the path to `KnightOnline_402.jar` directly to `Main.main`.

---

## 2. Ebitengine Input Capture & JNI Canvas Forwarding

### 2.1 Input Event Capture in Go (Ebitengine)
- **Location**: `pkg/gfx/renderer.go` (lines 26-59), `pkg/gfx/keypad.go` (lines 5-28)
- **Current State**: `renderer.go` checks keys via `ebiten.IsKeyPressed(k)` in a loop during `Update()`.
- **Defects Identified**:
  - `ebiten.IsKeyPressed(k)` fires repeatedly on every 60 FPS tick while a key is held down.
  - J2ME Canvas expects state transition callbacks (`keyPressed` when pressed down, `keyReleased` when released).
  - Pointer/mouse input (`pointerPressed`, `pointerReleased`, `pointerDragged`) is not currently captured.

- **Enhanced Input Event Polling Model**:
  Using `github.com/hajimehoshi/ebiten/v2/inpututil`:

  ```go
  // Keyboard events
  for _, k := range monitoredKeys {
      if inpututil.IsKeyJustPressed(k) {
          jKey := MapEbitenKeyToJ2ME(k)
          enqueueInputEvent(EventKeyPressed, jKey, 0, 0)
      }
      if inpututil.IsKeyJustReleased(k) {
          jKey := MapEbitenKeyToJ2ME(k)
          enqueueInputEvent(EventKeyReleased, jKey, 0, 0)
      }
  }

  // Pointer / Mouse events
  x, y := ebiten.CursorPosition()
  if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
      enqueueInputEvent(EventPointerPressed, 0, x, y)
  }
  if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
      enqueueInputEvent(EventPointerReleased, 0, x, y)
  }
  if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
      if x != g.lastMouseX || y != g.lastMouseY {
          enqueueInputEvent(EventPointerDragged, 0, x, y)
          g.lastMouseX, g.lastMouseY = x, y
      }
  }
  ```

### 2.2 Keycode Mapping Table (`pkg/gfx/keypad.go`)

| Ebitengine Key | J2ME Key Constant | J2ME Code Value | Description |
|---|---|---|---|
| `KeyUp` | `Canvas.UP` | `-1` | Directional Up |
| `KeyDown` | `Canvas.DOWN` | `-2` | Directional Down |
| `KeyLeft` | `Canvas.LEFT` | `-3` | Directional Left |
| `KeyRight` | `Canvas.RIGHT` | `-4` | Directional Right |
| `KeyEnter` / `KeySpace` | `Canvas.FIRE` | `-5` | Action / Selection |
| `KeyF1` / `KeyQ` | `Canvas.GAME_A` / Soft1 | `-6` | Left Softkey |
| `KeyF2` / `KeyE` | `Canvas.GAME_B` / Soft2 | `-7` | Right Softkey |
| `KeyDigit0` / `KeyNumpad0` | `'0'` | `48` | Numeric 0 |
| `KeyDigit1`..`KeyDigit9` | `'1'`..`'9'` | `49`..`57` | Numeric 1 to 9 |
| `KeyAsterisk` | `'*'` | `42` | Star key |
| `KeySlash` / `KeyPound` | `'#'` | `35` | Pound key |

### 2.3 JNI Canvas Management & Event Forwarding
- **Canvas Tracking in Java & Go**:
  When `Display.setCurrent(Displayable d)` is called in Java, Java invokes native `setNativeCanvas(Canvas canvas)` via JNI.
  In CGO (`bridge.c`):
  ```c
  static jobject g_current_canvas = NULL;

  JNIEXPORT void JNICALL Java_javax_microedition_lcdui_Display_setNativeCanvas(JNIEnv *env, jobject obj, jobject canvas) {
      if (g_current_canvas != NULL) {
          (*env)->DeleteGlobalRef(env, g_current_canvas);
          g_current_canvas = NULL;
      }
      if (canvas != NULL) {
          g_current_canvas = (*env)->NewGlobalRef(env, canvas);
      }
  }
  ```

- **JNI Dispatch Method Signatures**:
  - `keyPressed(I)V` — `methodID = (*env)->GetMethodID(env, canvasClass, "keyPressed", "(I)V")`
  - `keyReleased(I)V` — `methodID = (*env)->GetMethodID(env, canvasClass, "keyReleased", "(I)V")`
  - `pointerPressed(II)V` — `methodID = (*env)->GetMethodID(env, canvasClass, "pointerPressed", "(II)V")`
  - `pointerReleased(II)V` — `methodID = (*env)->GetMethodID(env, canvasClass, "pointerReleased", "(II)V")`
  - `pointerDragged(II)V` — `methodID = (*env)->GetMethodID(env, canvasClass, "pointerDragged", "(II)V")`

---

## 3. Multi-Threaded Synchronization & Concurrency Architecture

### 3.1 Threading Model Constraints Comparison

| Subsystem | Threading Model | Execution Requirements & Constraints |
|---|---|---|
| **Ebitengine Engine** | Single Main Thread (Thread 0) | `Update()` and `Draw()` MUST execute on OS thread 0 at 60 FPS. Any blocking JNI call halts OS window event processing. |
| **HotSpot JVM** | Multi-threaded Java Threads | Java MIDlet game loops run in separate threads (`java.lang.Thread`). Native callbacks run on calling Java thread. |
| **CGO / JNI Bridge** | Thread-Local `JNIEnv*` | `JNIEnv` pointer is unique per OS thread. Goroutines calling JNI MUST execute `runtime.LockOSThread()` and attach via `AttachCurrentThread`. |

### 3.2 Deadlock & Freezing Risk Scenarios

1. **Risk Scenario A: Synchronous JNI Dispatch in `Update()`**:
   - If `game.Update()` directly calls `(*env)->CallVoidMethod(canvas, keyPressedID, keyCode)` on Ebitengine's main thread:
   - If Java `keyPressed` code blocks on a synchronized lock held by Java game thread or calls `serviceRepaints()` waiting for render completion, Ebitengine main thread stalls.
   - **Result**: Application window stops responding, frame rate drops to 0, operating system marks window as hung.

2. **Risk Scenario B: Concurrent Graphics Buffer Modification**:
   - While Java background thread is executing native `fillRect` or `drawImage` callbacks (writing to `Graphics.Buffer.Pix`), Ebitengine `Draw()` thread reads `Pix` to call `offscreen.WritePixels(Pix)`.
   - **Result**: Data race, tearing artifacts, memory corruption, or slice index panics.

3. **Risk Scenario C: Detached JNI Thread Access**:
   - Calling JNI methods from arbitrary Go goroutines without calling `AttachCurrentThread` returns `JNI_EDETACHED` or causes instant JVM segfault (`SIGSEGV`).

### 3.3 Recommended Decoupled Synchronization Architecture

```
+-------------------------------------------------------------------+
|                        Ebitengine Main Thread                     |
|                                                                   |
|  Update()                                  Draw()                 |
|  - Capture Input Events                    - Drain RenderQueue /  |
|  - Push to InputQueue (Mutex)                Lock Buffer Mutex    |
|  - Return IMMEDIATELY                      - Render to Screen     |
+---------------------------------+---------------------------------+
                                  |
            InputQueue (Mutex)    | RenderQueue / Buffer Lock
                                  |
+---------------------------------+---------------------------------+
|                        JVM Execution Thread                       |
|                                                                   |
|  Event Dispatch Loop / Native Callback Handler                    |
|  - LockOSThread() & AttachCurrentThread()                         |
|  - Pop InputQueue & Dispatch via JNI (keyPressed/pointerPressed)  |
|  - Handle Graphics native calls (fillRect) into RenderQueue       |
+-------------------------------------------------------------------+
```

#### Key Architecture Components:

1. **Asynchronous `InputQueue`**:
   - Thread-safe FIFO queue (`sync.Mutex` or Go channel).
   - Ebitengine `Update()` enqueues `InputEvent` structs non-blockingly.
   - A dedicated worker goroutine locked to OS thread (`runtime.LockOSThread()`) drains `InputQueue` and executes JNI calls onto `g_current_canvas`.

2. **Thread-Safe Graphics Buffer Synchronization**:
   - Native graphics methods (`go_fillRect`, `go_drawImage`) either:
     - Push `RenderCommand` to `RenderQueue` (as tested in `pkg/jvm/callbacks.go`), which Ebitengine drains in `Draw()`.
     - Modify a double-buffered `image.RGBA` guarded by a `sync.RWMutex`, which `Draw()` briefly RLock-copies.

3. **JNI Environment Lifetime Helper (`WithEnv`)**:
   - Standard helper pattern in Go for safe JNI execution from any goroutine:
     ```go
     func (j *JVM) WithEnv(fn func(env *C.JNIEnv) error) error {
         runtime.LockOSThread()
         defer runtime.UnlockOSThread()

         if j == nil || j.vm == nil {
             return fmt.Errorf("JVM uninitialized")
         }

         var env *C.JNIEnv
         res := C.GetEnv(j.vm, (unsafe.Pointer)(&env), C.JNI_VERSION_1_8)
         if res == C.JNI_EDETACHED {
             if C.AttachCurrentThread(j.vm, (unsafe.Pointer)(&env), nil) != C.JNI_OK {
                 return fmt.Errorf("failed to attach OS thread to JVM")
             }
             defer C.DetachCurrentThread(j.vm)
         }
         return fn(env)
     }
     ```

---

## 4. Summary of Verification & Recommendations

1. **Enhance Classpath Resolver**: Update `InitJVM` in `pkg/jvm/jvm.go` using `filepath.SplitList` and absolute path verification to seamlessly accept combined classpaths (`j2me-stubs/bin:KnightOnline_402.jar`).
2. **Implement Input Event Queue**: Replace `ebiten.IsKeyPressed` polling with `inpututil.IsKeyJustPressed`/`Released` and pointer events, routing them through an async `InputQueue` to prevent Ebiten main thread stalls.
3. **Double Buffer Graphics**: Retain and expand `RenderQueue` (`pkg/jvm/callbacks.go`) or mutex-guarded offscreen buffer to decouple JNI render callbacks from Ebiten frame draw ticks.
