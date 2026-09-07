# Technical Analysis Report: Go Architecture, CGO Bridge & VRAM Image Management

**Author:** `explorer_survey_2`  
**Target System:** J2ME Go Engine (`/home/neuro/Documents/Tools/java-app/`)  
**Date:** 2026-08-09  

---

## 1. Executive Summary

This investigation analyzes the architecture required to migrate the J2ME emulator from a CPU-buffered Java/Go interpreter model to a high-performance Go Engine embedding OpenJDK via CGO/JNI. Rendering and input are delegated to Ebitengine (`github.com/hajimehoshi/ebiten/v2`), while game logic runs inside an embedded JVM.

Key findings include:
1. **Thread Stability**: `runtime.LockOSThread()` is critical in `main.go` and `InitJVM` because JNI thread-local states (`JNIEnv*`) and Ebitengine / OpenGL graphics contexts require OS thread pinning.
2. **VRAM Handle Mapping**: Image memory must be kept entirely in Go VRAM as `*ebiten.Image` textures. Go allocates auto-incrementing `int32` handle IDs returned to Java proxy objects (`javax.microedition.lcdui.Image`).
3. **JNI Registration Strategy**: Using explicit JNI `RegisterNatives` dynamic registration in C (`pkg/jvm/bridge.c`) is far cleaner and more maintainable than relying on mangled JNI symbol exports (`Java_javax_microedition_lcdui_...`).

---

## 2. Go Architecture & CGO/JNI Bridge Inspection

### 2.1 Entry Point (`main.go`) & Thread Locking
- `main.go` initializes the system by calling `runtime.LockOSThread()`, ensuring the primary goroutine stays pinned to OS thread 0.
- `jvm.InitJVM("./j2me-stubs/bin")` uses `C.create_java_vm` to spawn a single HotSpot JVM instance with classpath set to the compiled Java stubs directory.
- `jvm.RegisterNatives()` invokes `register_graphics_natives` in `pkg/jvm/bridge.c` to bind Java native methods to C/Go callbacks.
- `vm.InvokeMain("Test")` triggers the launcher entry point.
- Finally, Ebitengine is configured (`SetWindowSize(240, 320)`, `SetTPS(60)`) and launched via `ebiten.RunGame(game)`.

### 2.2 Existing `pkg/jvm` CGO Bridge Design
The current bridge consists of three primary source files:
- **`pkg/jvm/jvm.go`**: Handles JVM creation (`JNI_CreateJavaVM`), lifecycle management, and thread safety via `sync.Mutex`.
- **`pkg/jvm/callbacks.go`**: Exports CGO functions (`go_fillRect`, `go_setColor`) annotated with `//export`. Defines a thread-safe `RenderQueue` (`sync.Mutex` guarding `[]RenderCommand`) to collect commands executed from Java threads.
- **`pkg/jvm/bridge.c`**: Implements C wrappers `Java_javax_microedition_lcdui_Graphics_fillRect` and `Java_javax_microedition_lcdui_Graphics_setColor`, which invoke the exported Go functions via `_cgo_export.h`. Registers native methods on `javax/microedition/lcdui/Graphics` using `RegisterNatives`.

### 2.3 Existing Package Layout & Legacy Comparison
- `pkg/gfx`: Contains `EmulatorGame` implementing `ebiten.Game`. Currently converts CPU `image.RGBA` pixel buffers into Ebitengine textures per frame via `offscreen.WritePixels()`.
- `pkg/midp`: Pure Go implementation of MIDP graphics primitives, font rendering (`font8x13.go`), and image decoding (`DecodeImage` supporting `.smd` XOR decryption).
- `pkg/vm`, `pkg/classfile`, `pkg/jar`: Legacy pure-Go JVM bytecode interpreter. In the CGO/JNI engine, bytecode execution is handled natively by HotSpot JVM, while MIDP graphics calls bridge directly to Go.

---

## 3. Ebitengine Initialization & VRAM Handle Mapping

### 3.1 VRAM Architecture Rationale
In standard J2ME, Java `Image` instances store raw byte arrays or AWT `BufferedImage` objects on the JVM heap. For the Go Engine:
- Java heap memory usage is minimized by storing zero pixel data in Java.
- Images are decoded immediately upon creation in Go and stored in GPU VRAM as `*ebiten.Image`.
- Java holds only a 32-bit primitive integer ID (`int nativeId`).

### 3.2 VRAM Handle Registry Design in Go
We propose a thread-safe VRAM registry module (`pkg/jvm/vram.go`):

```go
package jvm

import (
	"sync"
	"sync/atomic"
	"github.com/hajimehoshi/ebiten/v2"
)

type VRAMRegistry struct {
	mu     sync.RWMutex
	nextID int32
	images map[int32]*ebiten.Image
}

var GlobalVRAM = &VRAMRegistry{
	nextID: 1,
	images: make(map[int32]*ebiten.Image),
}

func (v *VRAMRegistry) Register(img *ebiten.Image) int32 {
	if img == nil {
		return 0
	}
	id := atomic.AddInt32(&v.nextID, 1)
	v.mu.Lock()
	v.images[id] = img
	v.mu.Unlock()
	return id
}

func (v *VRAMRegistry) Get(id int32) (*ebiten.Image, bool) {
	v.mu.RLock()
	img, ok := v.images[id]
	v.mu.RUnlock()
	return img, ok
}

func (v *VRAMRegistry) Delete(id int32) {
	v.mu.Lock()
	delete(v.images, id)
	v.mu.Unlock()
}
```

### 3.3 Image Creation Workflows
1. **`Image.createImage(String path)`**:
   - Java passes string path via JNI.
   - Go opens resource/file, decodes payload using `midp.DecodeImage(bytes)` (handling SMD/PNG/JPEG formats).
   - Go converts `image.Image` to `*ebiten.Image` via `ebiten.NewImageFromImage(img)`.
   - Go registers texture in `GlobalVRAM` and returns the allocated `int32` ID.

2. **`Image.createImage(int width, int height)`**:
   - Go creates blank offscreen image: `img := ebiten.NewImage(width, height)`.
   - Registers in `GlobalVRAM` and returns `int32` ID.

3. **`Image.createImage(byte[] data, int offset, int length)`**:
   - Go retrieves raw bytes from JVM via JNI `GetByteArrayElements`.
   - Decodes slice `data[offset : offset+length]`, converts to `*ebiten.Image`, registers, returns `int32` ID.

### 3.4 Offscreen Graphics & Target Handling (`Image.getGraphics()`)
J2ME allows offscreen rendering (`Image.getGraphics()`). To support this:
- Java `Graphics` object holds `private final int targetImageId` (0 = screen canvas, >0 = offscreen image ID).
- When a drawing operation occurs (`fillRect`, `drawImage`, etc.), Go resolves `targetImageId`:
  - `targetImageId == 0`: Renders to the main screen buffer/canvas.
  - `targetImageId > 0`: Looks up target `*ebiten.Image` from `GlobalVRAM` and draws directly onto it using Ebitengine options.

---

## 4. CGO JNI Native Callback Registration & Signatures

### 4.1 Registration Strategy: `RegisterNatives` vs Exported C Names
- **Exported C Names (`Java_javax_microedition_lcdui_...`)**:
  - Overloaded JNI names require double underscores and mangled descriptor strings (e.g. `Java_javax_microedition_lcdui_Image_createImage__Ljava_lang_String_2`, `Java_javax_microedition_lcdui_Image_createImage___3BII`).
  - Fragile, difficult to read, prone to linking failures.
- **Dynamic Registration (`RegisterNatives`)**:
  - Explicit table mapping method names and descriptors to C functions.
  - Handles overloaded methods cleanly without symbol mangling.
  - **Recommended Approach**: Extend `pkg/jvm/bridge.c` to register methods for `Image`, `Canvas`, and `Graphics`.

### 4.2 JNI Method Descriptor Table

| Class | Method | JNI Descriptor | C Implementation |
|---|---|---|---|
| `Image` | `createImage(String)` | `(Ljava/lang/String;)Ljavax/microedition/lcdui/Image;` | `native_Image_createImage_String` |
| `Image` | `createImage(int, int)` | `(II)Ljavax/microedition/lcdui/Image;` | `native_Image_createImage_Dimensions` |
| `Image` | `createImage(byte[], int, int)` | `([BII)Ljavax/microedition/lcdui/Image;` | `native_Image_createImage_Bytes` |
| `Image` | `getWidth()` | `()I` | `native_Image_getWidth` |
| `Image` | `getHeight()` | `()I` | `native_Image_getHeight` |
| `Image` | `getGraphics()` | `()Ljavax/microedition/lcdui/Graphics;` | `native_Image_getGraphics` |
| `Canvas` | `serviceRepaints()` | `()V` | `native_Canvas_serviceRepaints` |
| `Graphics` | `fillRect(int, int, int, int)` | `(IIII)V` | `Java_javax_microedition_lcdui_Graphics_fillRect` |
| `Graphics` | `setColor(int)` | `(I)V` | `Java_javax_microedition_lcdui_Graphics_setColor` |
| `Graphics` | `drawImage(Image, int, int, int)` | `(Ljavax/microedition/lcdui/Image;III)V` | `Java_javax_microedition_lcdui_Graphics_drawImage` |

### 4.3 `Canvas.repaint()` & `Canvas.serviceRepaints()` Sync Protocol
- Java's `Canvas.repaint()` invokes `serviceRepaints()`.
- `native_Canvas_serviceRepaints` in CGO triggers a synchronous call back into Java's `Canvas.paint(Graphics g)` method or signals Ebitengine's `Update()` loop.
- In Ebitengine's 60 FPS update cycle, Go checks for queued repaints and invokes `paint(Graphics g)` on the active Java `Canvas` object.

---

## 5. Architectural Recommendations for Next Steps

1. **Java Proxy Classes**:
   Update `j2me-stubs/src/javax/microedition/lcdui/Image.java` and `Canvas.java` as pure native proxies with zero AWT dependencies.
2. **Go VRAM Implementation**:
   Implement `pkg/jvm/vram.go` for managing integer-to-`*ebiten.Image` handle mappings.
3. **Bridge Expansion**:
   Expand `pkg/jvm/bridge.c` and `pkg/jvm/callbacks.go` to register all MIDP native callbacks for `Image`, `Canvas`, and `Graphics`.
4. **Game Launcher**:
   Provide `com/emulator/Main.java` launcher to invoke `KnightOnline_402.jar` via custom `ClassLoader`.
