# J2ME Go Engine Design Spec

## Overview
A complete rewrite of the J2ME Java Desktop Emulator into a lightweight, high-performance Go application. The system uses CGO and JNI to embed a Java Virtual Machine (JVM) inside a Go executable, delegating all rendering and input handling to the Go layer (via Ebitengine) while preserving the game's internal logic inside the Java ecosystem.

## Architecture

The architecture consists of two deeply integrated layers:

1.  **The Go Layer (Host / Frontend)**
    *   **Engine:** Ebitengine (`github.com/hajimehoshi/ebiten/v2`).
    *   **Responsibilities:**
        *   Creating the application window and running the 60FPS update/draw loop.
        *   Initializing the embedded JVM via CGO / `github.com/tebeka/jni`.
        *   Implementing native JNI functions called by the Java layer.
        *   Capturing keyboard/mouse input and passing it to Java.
        *   Managing graphics memory (storing loaded images directly in Go memory as `*ebiten.Image`).
        *   Handling File I/O for `RecordStore` (RMS).

2.  **The Java Layer (Guest / Game Logic)**
    *   **Environment:** Minimal J2ME stub classes (`Canvas`, `Graphics`, `Image`, `RecordStore`).
    *   **Responsibilities:**
        *   Loading the original game JAR (`KnightOnline_402.jar`) using a CustomClassLoader.
        *   Executing the game's business logic, networking, and threading.
        *   Translating J2ME API calls into `native` methods.
    *   **Constraint:** Absolutely zero dependencies on Java AWT or Swing.

## Data Flow & Components

### 1. Graphics and Rendering
*   `Graphics.java` will declare methods like `public native void fillRect(int x, int y, int w, int h);`.
*   The Go engine registers these native methods. When called, Go directly modifies the Ebitengine screen buffer.
*   **Double Buffering:** Ebitengine inherently handles double buffering, meaning the game's `serviceRepaints()` will just trigger the Go engine to render the current state in the next frame.

### 2. Image Management (VRAM Optimization)
*   `Image.java` will no longer store `byte[]` or `BufferedImage` in Java heap.
*   Instead, when `Image.createImage()` is called, Java passes the byte array to Go.
*   Go decodes the image, stores it as `*ebiten.Image` in a Go map, and returns a unique `int imageId` to Java.
*   Java calls `Graphics.drawImage(Image img, ...)` by passing `img.getId()`. Go retrieves the `*ebiten.Image` and draws it.

### 3. Input Handling
*   Ebitengine's `Update()` function checks for key presses and mouse clicks.
*   When an event occurs, Go uses JNI to call `Canvas.keyPressed(int keyCode)` in the Java layer.
*   Coordinates will be dynamically scaled by Go before being passed to Java.

### 4. RecordStore (RMS) I/O
*   `RecordStore` methods will be entirely native.
*   Go handles creating directories, reading `.dat` files, and writing data, bypassing Java's file I/O overhead.

## Error Handling & Fallbacks
*   **JNI Panics:** Proper CGO exception checking must be implemented after every JNI call to prevent silent JVM crashes.
*   **Missing Images:** If Go fails to decode an image, it must return a specific error code to Java so Java can throw an `IOException`, triggering the game's automatic network download logic.

## Testing Strategy
*   **Compilation:** Ensure the Go wrapper builds successfully with CGO enabled and links to `libjvm.so`.
*   **Integration:** Run `KnightOnline_402.jar`, verify that the main menu renders correctly.
*   **Performance:** Check CPU usage compared to the old Swing emulator; it should be significantly lower and visually smoother (60FPS).
