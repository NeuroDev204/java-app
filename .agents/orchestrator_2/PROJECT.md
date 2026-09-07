# Project: J2ME Go Engine (java-app)

## Architecture
A hybrid Go-Java engine where Go (using Ebitengine v2) handles the 60FPS application window, input, and 2D rendering, while an embedded OpenJDK 21 HotSpot JVM (initialized via CGO and standard JNI headers `<jni.h>`) executes Java J2ME bytecode. 

J2ME UI and graphics operations in `javax.microedition.lcdui.Graphics` are stubbed as pure `native` Java methods without AWT/Swing dependencies. CGO native method interception (`RegisterNatives` or dynamic JNI symbol mapping `Java_javax_microedition_lcdui_Graphics_fillRect`) routes graphics calls from Java to Go. A thread-safe `RenderQueue` (`sync.Mutex`) decouples JVM threads from Ebitengine's 60FPS render loop.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Go Ebitengine Scaffold | Go module, 60FPS Ebitengine window update/draw loop (`j2mego` binary build), X11 symlink fix | M1 | R1 |
| 2 | J2ME Java Stubs | `j2me-stubs/src/javax/microedition/lcdui/Graphics.java` with `native fillRect(...)`, zero AWT/Swing, compiled via `javac` | M2 | R3 |
| 3 | JVM Embedding (CGO/JNI) | JVM initialization (`JNI_CreateJavaVM`) via CGO linked against `libjvm.so` at `/usr/lib/jvm/java-21-openjdk-amd64/lib/server` | M3 | R2 |
| 4 | JNI Callbacks (Java -> Go) | CGO bridge intercepting Java `Graphics.fillRect`, delegating to Go `RenderQueue`, and test harness `Test.java` | M4 | R4 |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Go Ebitengine Scaffold | Setup Ebitengine game loop, `lib/libXxf86vm.so` workaround, `j2mego` executable build harness | None | DONE |
| M2 | J2ME Java Stubs | Create `j2me-stubs/src/javax/microedition/lcdui/Graphics.java` and build `j2me-stubs/bin` | None | DONE |
| M3 | JVM Embedding | Create CGO JVM embedding wrapper (`pkg/jvm`), link against `libjvm.so` with embedded `-Wl,-rpath` | M1 | DONE |
| M4 | JNI Callbacks & E2E Integration | Wire JNI `fillRect` callback to Go `RenderQueue`, create `Test.java`, verify runtime output & E2E tests | M2, M3 | DONE |

## Interface Contracts
### Java J2ME Graphics ↔ Go Engine
- Java class: `package javax.microedition.lcdui; public class Graphics { public native void fillRect(int x, int y, int width, int height); }`
- JNI Symbol: `Java_javax_microedition_lcdui_Graphics_fillRect(JNIEnv *env, jobject thisObj, jint x, jint y, jint w, jint h)` or `RegisterNatives` mapping.
- Go Interception:
  ```go
  //export go_fillRect
  func go_fillRect(env *C.JNIEnv, obj C.jobject, x C.jint, y C.jint, w C.jint, h C.jint)
  ```
- Command Queue: `RenderQueue.Push(FillRectCmd{X: int(x), Y: int(y), Width: int(w), Height: int(h)})`

## Code Layout
- `go.mod`, `go.sum` — Go module configuration
- `lib/libXxf86vm.so` — Symlink for Ubuntu CGO X11 linker build
- `main.go` — Main entry point: JVM startup, JNI registration, Ebitengine game loop
- `pkg/jvm/` — CGO JVM initialization & JNI helper functions
- `j2me-stubs/src/javax/microedition/lcdui/Graphics.java` — J2ME stub class
- `j2me-stubs/src/Test.java` — Test harness class invoking `Graphics.fillRect`
- `j2me-stubs/bin/` — Compiled bytecode directory
