# Original User Request

## Follow-up — 2026-08-09T04:24:12Z

Build a high-performance J2ME Go Engine that embeds a JVM via CGO/JNI to run Knight Online. Go (using Ebitengine) will handle all graphics rendering and input, while Java J2ME stubs will be rewritten as native methods to bridge logic to Go.

Working directory: /home/neuro/Documents/Tools/java-app
Integrity mode: benchmark

## Requirements

### R1. Go Ebitengine Scaffold
Initialize a Go module and create a basic Ebitengine application window that runs a 60FPS update loop.

### R2. JVM Embedding (CGO/JNI)
Use `github.com/tebeka/jni` and CGO to embed a Java Virtual Machine (JVM) inside the Go executable. The Go application must initialize the JVM before starting the graphics loop.

### R3. J2ME Java Stubs
Create a minimal `Graphics.java` class in a `j2me-stubs` directory. It must contain only `native` method declarations (e.g. `public native void fillRect(...)`) and have zero dependencies on Java AWT/Swing.

### R4. JNI Callbacks (Java -> Go)
Implement the CGO bridge in Go to intercept the `native` calls from Java. Go should successfully capture a Java call to `Graphics.fillRect` and execute Go code.

## Acceptance Criteria

### Compilation & Building
- [ ] `go build -o j2mego` executes without errors and successfully links against `libjvm.so`.
- [ ] `javac -d bin src/javax/microedition/lcdui/Graphics.java` executes without syntax errors.

### Runtime Verification
- [ ] Running `./j2mego` successfully initializes the JVM without crashing.
- [ ] A test Java class (`Test.java`) called from Go can successfully invoke the native `fillRect` method, and Go can print an interception message to standard output.

## Follow-up — 2026-08-09T05:39:08Z

Migrate the existing J2ME emulator logic to the new Go Engine architecture. The Java J2ME stubs (Canvas, Image, RecordStore) must be rewritten as thin native proxies to replace Swing/AWT. Go must implement these native methods via CGO JNI to manage VRAM (via Ebitengine images) and handle I/O. Finally, Go must instruct the embedded JVM to load the `KnightOnline_402.jar` file and start the game.

Working directory: /home/neuro/Documents/Tools/java-app
Integrity mode: benchmark

## Requirements

### R1. Core J2ME Stubs Migration
Rewrite `Image.java` and `Canvas.java` in `j2me-stubs/src/javax/microedition/lcdui/` to act as thin native proxies. Remove all Java AWT/Swing logic. Any graphic operation (like creating an image) must call a `native` method.

### R2. Go Native Implementations
Implement the Go CGO bridge to intercept native calls from `Image.java` and `Canvas.java`. Go must decode images, store them directly in its own memory space as `*ebiten.Image` objects (VRAM), and return integer IDs back to Java. Go must draw these images to the Ebiten screen when instructed.

### R3. JAR Loading and Game Execution
Write a launcher class (e.g., `Main.java`) in the Java stubs to handle loading `KnightOnline_402.jar`. Go's `main()` function must invoke this launcher via JNI to start the game after initializing the Ebiten engine and the JVM.

## Acceptance Criteria

### Compilation & Building
- [ ] `javac` successfully compiles the newly modified Java stubs (`Image.java`, `Canvas.java`, `Main.java`) without any syntax errors or AWT/Swing dependencies.
- [ ] `go build -o j2mego` executes without errors and successfully links against `libjvm.so`.

### Runtime Verification
- [ ] Running `./j2mego` successfully loads the `KnightOnline_402.jar` file through the JVM.
- [ ] The Ebitengine window appears and successfully renders the game graphics (using the Go-managed `*ebiten.Image` textures) without crashing.
- [ ] Input events (keys/mouse) from Go are correctly forwarded to the Java game logic, preventing freezing or locking up.

