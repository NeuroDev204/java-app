# Project: J2ME Go Engine Migration

## Architecture
- **Java Stub Layer (`j2me-stubs/`)**: Pure Java J2ME proxies (`javax.microedition.lcdui.Image`, `Canvas`, `Graphics`, `Display`, `Displayable`, `Screen`, `TextBox`, `MIDlet`, `com.emulator.Main`). Zero AWT/Swing dependencies. Graphic operations delegate to `native` methods.
- **Go Engine & CGO Bridge (`pkg/`)**:
  - `pkg/jvm`: OpenJDK 21 JVM initialization via CGO/JNI with composite classpath (`j2me-stubs/bin` + `/home/neuro/Downloads/KnightOnline_402.jar`).
  - `pkg/vram`: Thread-safe VRAM image registry (`map[int32]image.Image`) managing integer texture handles and offscreen canvas targets.
  - `pkg/bridge` / `pkg/jvm/bridge.c`: Dynamic JNI `RegisterNatives` bridge mapping all Java native proxy declarations to Go handlers (27/27 native methods registered in `Graphics.java`, `Image.java`, `Canvas.java`, `Display.java`, `RecordStore.java`).
  - `pkg/input`: Discrete Ebitengine input mapper polling `inpututil.IsKeyJustPressed`/`IsKeyJustReleased` and pointer events, queueing events thread-safely to forward to Java `Canvas`.
  - `main.go`: Ebitengine application entry point implementing 60 FPS update loop and JVM execution trigger via `com.emulator.Main`.

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Native Image & Canvas Stubs | Rewrite Image.java, Canvas.java, Main.java as thin native proxies without AWT/Swing | M1 | R1 |
| 2 | Pure Native Graphics & Display Stubs | Implement Graphics.java, Display.java, Displayable.java, MIDlet.java | M1 | R1 |
| 3 | VRAM Registry & Texture Management | Go map[int32]image.Image handle table for VRAM texture IDs | M2 | R2 |
| 4 | CGO JNI Dynamic RegisterNatives Bridge | Native bridge intercepting Image/Canvas/Graphics calls from Java | M2 | R2 |
| 5 | Screen Rendering & Ebitengine Draw Loop | Render VRAM textures & offscreen buffers to Ebiten screen | M2 | R2 |
| 6 | Composite Classpath & JVM Launcher | Update jvm.go to resolve composite classpath and launch Main.java | M3 | R3 |
| 7 | KnightOnline_402.jar Game Loading | Main.java loads KnightOnline_402.jar and launches TemMidlet | M3 | R3 |
| 8 | Keyboard & Mouse Input Forwarding | Ebiten inpututil polling -> InputQueue -> JNI canvas key/pointer events | M3 | R3 |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| 1 | Core J2ME Stubs Migration | Java stubs: Image.java, Canvas.java, Graphics.java, Display.java, MIDlet.java, javac compilation | None | DONE |
| 2 | Go Native & VRAM Bridge | Go VRAM registry, CGO JNI bridge, RegisterNatives, Ebitengine offscreen draw loop | M1 | DONE |
| 3 | JAR Launcher & E2E Integration | Main.java, KnightOnline_402.jar launch via JNI, Ebiten input forwarding, E2E verification | M1, M2 | DONE |

## Interface Contracts
### Java Proxy ↔ Go CGO Bridge
- `Image.nCreateImage(String path) -> int id`
- `Image.nCreateImageBytes(byte[] data, int off, int len) -> int id`
- `Image.nCreateOffscreen(int w, int h) -> int id`
- `Canvas.nRepaint(int x, int y, int w, int h) -> void`
- `Canvas.nServiceRepaints() -> void`
- `Canvas.nDispatchKeyEvent(int type, int key) -> void`
- `Canvas.nDispatchPointerEvent(int type, int x, int y) -> void`

## Code Layout
- `j2me-stubs/src/javax/microedition/lcdui/`: J2ME LCDUI stubs (`Image.java`, `Canvas.java`, `Graphics.java`, `Display.java`, `Displayable.java`, `Screen.java`, `TextBox.java`)
- `j2me-stubs/src/javax/microedition/midlet/`: MIDlet stub (`MIDlet.java`)
- `j2me-stubs/src/com/emulator/Main.java`: Game launcher class
- `pkg/jvm/`: JVM CGO initialization & JNI bindings (`jvm.go`, `bridge.c`, `callbacks.go`)
- `pkg/vram/`: VRAM handle manager
- `pkg/input/` / `pkg/gfx/keypad.go`: Key & mouse input translation queue
- `main.go`: Executable entry point
