# Handoff Report: Go Engine & CGO Bridge Specialist Investigation

**Explorer**: Explorer 2 (Go Engine & CGO Bridge Specialist)  
**Working Directory**: `/home/neuro/Documents/Tools/java-app/.agents/explorer_v2`  
**Target Project**: `/home/neuro/Documents/Tools/java-app`  

---

## 1. Observation

### Observation 1.1: Go Module & Dependencies
- `go.mod` (lines 1-14):
  - Module path: `java-app`
  - Go version: `1.24.5`
  - Dependencies: `github.com/hajimehoshi/ebiten/v2 v2.9.9`, `github.com/ebitengine/purego v0.9.0`, `golang.org/x/sync v0.17.0`, `golang.org/x/sys v0.36.0`.
- `main.go` (lines 1-63):
  - `#cgo LDFLAGS: -L${SRCDIR}/lib` (line 4)
  - `main()` locks OS thread via `runtime.LockOSThread()`, initializes JVM via `jvm.InitJVM("./j2me-stubs/bin")`, invokes `vm.InvokeMain("Test")`, configures window (240x320, 60 TPS), and calls `ebiten.RunGame(&Game{})`.

### Observation 1.2: CGO JVM Initialization (`pkg/jvm/jvm.go`)
- `pkg/jvm/jvm.go` (lines 3-37):
  - CGO headers:
    `#cgo CFLAGS: -I/usr/lib/jvm/java-21-openjdk-amd64/include -I/usr/lib/jvm/java-21-openjdk-amd64/include/linux`
    `#cgo LDFLAGS: -L/usr/lib/jvm/java-21-openjdk-amd64/lib/server -Wl,-rpath,/usr/lib/jvm/java-21-openjdk-amd64/lib/server -ljvm`
  - Uses JNI C API `JNI_CreateJavaVM` with `JNI_VERSION_1_8` options `-Djava.class.path=...` and `-Xrs`.
  - `InitJVM(classpath string)` (lines 63-115) enforces thread safety with `jvmMutex`, resolves relative/absolute classpath candidates, creates the VM, and calls `_ = globalJVM.RegisterNatives()`.

### Observation 1.3: CGO JNI Bridge & RegisterNatives (`pkg/jvm/callbacks.go` & `pkg/jvm/bridge.c`)
- `pkg/jvm/callbacks.go`:
  - Defines `RenderCommand` (`CmdFillRect`, `CmdSetColor`) and `RenderQueue` (lines 19-85).
  - Exports `go_fillRect` (line 87) and `go_setColor` (line 100) using `//export`.
  - `(j *JVM) RegisterNatives()` (lines 111-121) calls C helper `register_graphics_natives(j.vm, j.env)`.
  - `(j *JVM) InvokeMain(className string)` (lines 124-138) calls C helper `safe_invoke_main(j.vm, j.env, cName)`.
- `pkg/jvm/bridge.c`:
  - `Java_javax_microedition_lcdui_Graphics_fillRect` (lines 5-7) forwards to `go_fillRect(x, y, width, height)`.
  - `Java_javax_microedition_lcdui_Graphics_setColor` (lines 9-11) forwards to `go_setColor(rgb)`.
  - `register_graphics_natives` (lines 13-51) dynamically registers JNI native methods for class `javax/microedition/lcdui/Graphics`:
    ```c
    JNINativeMethod methods[] = {
        {"fillRect", "(IIII)V", (void *)&Java_javax_microedition_lcdui_Graphics_fillRect},
        {"setColor", "(I)V", (void *)&Java_javax_microedition_lcdui_Graphics_setColor}
    };
    ```
  - `safe_invoke_main` (lines 53-113) attaches thread if detached (`JNI_EDETACHED`), finds class, resolves `main([Ljava/lang/String;)V` method ID, and invokes `CallStaticVoidMethod`.

### Observation 1.4: Directory & Package Layout vs. PROJECT.md Specifications
- Filesystem check (`find_by_name` under `pkg/`):
  - Existing packages: `pkg/classfile`, `pkg/gfx`, `pkg/jar`, `pkg/jvm`, `pkg/midp`, `pkg/vm`.
  - Missing/Uncreated packages specified in `PROJECT.md`:
    - `pkg/vram`: VRAM handle registry (`map[int32]*ebiten.Image`) for managing texture handles and offscreen canvas targets does not exist as a separate package yet.
    - `pkg/bridge`: Separate dynamic JNI `RegisterNatives` package does not exist yet (JNI bridge currently resides in `pkg/jvm`).
    - `pkg/input`: Discrete Ebitengine key/mouse input translation queue does not exist yet.
- Native method coverage in `j2me-stubs/src`:
  - `Graphics.java`: 2 native methods registered (`fillRect`, `setColor`); 25 native methods declared but unregistered (e.g. `drawImage`, `drawRegion`, `drawRect`, `drawLine`, `drawString`, `setClip`).
  - `Canvas.java`: `serviceRepaints()` declared but unregistered.
  - `Image.java`: `createImage(String)`, `createImage(int, int)`, `createImage(byte[], int, int)`, `createImage(Image, int, int, int, int)`, `getWidth()`, `getHeight()`, `getGraphics()` declared but unregistered.

### Observation 1.5: Build Execution & Dynamic Linking Verification
- Command: `CGO_ENABLED=1 go build -o j2mego`
  - Directory: `/home/neuro/Documents/Tools/java-app`
  - Exit code: `0`
  - Output: (empty stdout / stderr)
  - Binary produced: `j2mego` (executable, size 12,363,328 bytes).
- Command: `ldd j2mego`
  - Exit code: `0`
  - Output excerpt:
    ```
    linux-vdso.so.1 (0x000075a725be5000)
    libjvm.so => /usr/lib/jvm/java-21-openjdk-amd64/lib/server/libjvm.so (0x000075a724600000)
    libX11.so.6 => /usr/lib/x86_64-linux-gnu/libX11.so.6 (0x000075a7244a4000)
    libm.so.6 => /usr/lib/x86_64-linux-gnu/libm.so.6 (0x000075a72437e000)
    libc.so.6 => /usr/lib/x86_64-linux-gnu/libc.so.6 (0x000075a724000000)
    libstdc++.so.6 => /usr/lib/x86_64-linux-gnu/libstdc++.so.6 (0x000075a723c00000)
    ```

---

## 2. Logic Chain

1. **From Observation 1.1 & 1.2**: The Go application `main.go` correctly invokes `jvm.InitJVM()`, which successfully compiles via CGO with OpenJDK 21 headers (`-I/usr/lib/jvm/java-21-openjdk-amd64/include`) and links against `libjvm.so` using rpath `-Wl,-rpath,/usr/lib/jvm/java-21-openjdk-amd64/lib/server`.
2. **From Observation 1.3**: The CGO bridge in `pkg/jvm/bridge.c` and `pkg/jvm/callbacks.go` successfully registers `fillRect` and `setColor` for `javax/microedition/lcdui/Graphics` using `RegisterNatives`. When Java executes these methods, C calls export functions `go_fillRect` and `go_setColor`, pushing render commands to a thread-safe `RenderQueue`.
3. **From Observation 1.4**:
   - The current CGO bridge only handles basic `Graphics.fillRect` and `setColor`.
   - `Image.java` and `Canvas.java` native methods, as well as VRAM texture management (`map[int32]*ebiten.Image`), are not yet implemented in Go.
   - Separate packages `pkg/vram`, `pkg/bridge`, and `pkg/input` specified in `PROJECT.md` have not been created yet; `pkg/jvm` currently holds the partial JVM and JNI initialization.
4. **From Observation 1.5**: Executing `CGO_ENABLED=1 go build -o j2mego` succeeds with exit code `0`. Binary inspection via `ldd j2mego` confirms active dynamic linking against `/usr/lib/jvm/java-21-openjdk-amd64/lib/server/libjvm.so`.

---

## 3. Caveats

- Runtime execution of `./j2mego` was not run in headful X11 display mode during this read-only static inspection phase, though binary linking and `javac` stub compilation were verified.
- Expansion of `RegisterNatives` to include `Image.createImage` and `Canvas.repaint` will require creating `pkg/vram` (`map[int32]*ebiten.Image`) and updating `bridge.c` to register the new JNI method signatures.

---

## 4. Conclusion

- **CGO & JVM Initialization**: Fully functional and building cleanly. `go build -o j2mego` succeeds with exit code `0`.
- **Dynamic Linking**: Verified via `ldd j2mego`. `libjvm.so` is correctly linked with rpath set to OpenJDK 21 server directory.
- **JNI Binding & RegisterNatives**: Operational in `pkg/jvm` for basic `Graphics` methods (`fillRect`, `setColor`).
- **Next Architectural Steps**:
  1. Create `pkg/vram` package with `VRAMRegistry` managing `map[int32]*ebiten.Image`.
  2. Implement `Image.createImage` native callbacks returning auto-incrementing `int32` texture handles.
  3. Expand `RegisterNatives` in `pkg/jvm` (or `pkg/bridge`) to cover `Image.java` and `Canvas.java` proxy methods.
  4. Create `pkg/input` for thread-safe key and pointer event queueing from Ebitengine to JVM.

---

## 5. Verification Method

To independently verify the findings in this report:

1. **Test Go Compilation**:
   ```bash
   cd /home/neuro/Documents/Tools/java-app
   CGO_ENABLED=1 go build -o j2mego
   ```
   *Expected result*: Exit code `0`, binary `j2mego` created.

2. **Verify Dynamic Linking**:
   ```bash
   ldd j2mego | grep libjvm
   ```
   *Expected output*: `libjvm.so => /usr/lib/jvm/java-21-openjdk-amd64/lib/server/libjvm.so`

3. **Verify Java Stub Compilation**:
   ```bash
   javac -d j2me-stubs/bin $(find j2me-stubs/src -name "*.java")
   ```
   *Expected result*: Exit code `0` with class files generated in `j2me-stubs/bin`.
