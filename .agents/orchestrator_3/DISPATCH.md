# Dispatch Assignment — Project Orchestrator (orchestrator_3)

## 2026-08-09T12:39:44Z

Migrate existing J2ME emulator logic to the new Go Engine architecture.
Requirements:
- R1: Core J2ME Stubs Migration (rewrite Image.java and Canvas.java in j2me-stubs/src/javax/microedition/lcdui/ as thin native proxies without AWT/Swing).
- R2: Go Native Implementations (implement CGO bridge intercepting native calls from Image.java and Canvas.java, storing *ebiten.Image in VRAM, returning integer IDs, drawing to Ebiten screen).
- R3: JAR Loading and Game Execution (write launcher class Main.java in Java stubs to load KnightOnline_402.jar; Go main invokes Main.java via JNI after initializing Ebiten & JVM).

Acceptance Criteria:
- javac compiles Java stubs without syntax errors or AWT/Swing dependencies.
- go build -o j2mego succeeds and links against libjvm.so.
- Running ./j2mego loads KnightOnline_402.jar through JVM.
- Ebitengine window renders game graphics without crashing.
- Input events (keys/mouse) forward to Java game logic, preventing freezing.
