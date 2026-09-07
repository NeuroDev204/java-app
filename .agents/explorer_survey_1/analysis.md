# Comprehensive J2ME Codebase & KnightOnline_402.jar Survey Analysis

## 1. Executive Summary

This report presents a thorough survey and architectural mapping of the Java codebase in `j2me-stubs/` and the target game archive `KnightOnline_402.jar`. The objective is to enable headless J2ME execution embedded in a Go/Ebitengine host process via CGO and JNI, eliminating all Java AWT/Swing dependencies.

Key findings:
- `j2me-stubs/src/javax/microedition/lcdui/Graphics.java` currently exists and has **zero AWT/Swing dependencies**. However, missing methods (`drawImage`, `drawRegion`, `setClip`, `translate`, etc.) must be added.
- `j2me-stubs/src/javax/microedition/lcdui/Image.java`, `Canvas.java`, `Displayable.java`, `Display.java`, `javax/microedition/midlet/MIDlet.java`, and `com/emulator/Main.java` are **currently missing** and must be implemented as thin native proxies.
- `KnightOnline_402.jar` was located at `/home/neuro/Downloads/KnightOnline_402.jar`.
- Target MIDlet entry point: `com.silverknight.TemMidlet` (defined in `META-INF/MANIFEST.MF`).
- Primary rendering canvas class: `com.silverknight.a` (extends `javax.microedition.lcdui.Canvas` and implements `java.lang.Runnable`).

---

## 2. Assessment of Existing Java Codebase (`j2me-stubs/`)

### 2.1 File Inventory
- `j2me-stubs/src/javax/microedition/lcdui/Graphics.java` (Active)
- `j2me-stubs/src/ConcurrentJavaTest.java`, `ExceptionTest.java`, `MultiFillRectTest.java`, `Test.java`, `TestConcurrent.java`, `TestMultiple.java`, `TestNoMain.java`, `TestUnregisteredNative.java` (Test stubs)

### 2.2 AWT/Swing Audit
- **Findings**: `Graphics.java` contains **0 imports of `java.awt.*` or `javax.swing.*`**. All graphics methods are declared with `public native`.
- **Missing Required LCDUI / MIDP Stubs**:
  1. `Image.java` (`javax.microedition.lcdui.Image`)
  2. `Canvas.java` (`javax.microedition.lcdui.Canvas`)
  3. `Displayable.java` (`javax.microedition.lcdui.Displayable`)
  4. `Display.java` (`javax.microedition.lcdui.Display`)
  5. `MIDlet.java` (`javax.microedition.midlet.MIDlet`)
  6. `Main.java` (`com.emulator.Main`)

---

## 3. Inspection of `KnightOnline_402.jar`

### 3.1 Metadata & Entry Point
- **Location**: `/home/neuro/Downloads/KnightOnline_402.jar`
- **Manifest (`META-INF/MANIFEST.MF`)**:
  ```manifest
  Manifest-Version: 1.0
  MIDlet-1: KnightOnline,/icon.png,com.silverknight.TemMidlet
  MIDlet-Vendor: Midlet Suite Vendor
  MicroEdition-Configuration: CLDC-1.0
  MIDlet-Name: KnightOnline Midlet Suite
  MIDlet-Version: 1.8.2
  MicroEdition-Profile: MIDP-2.0
  ```
- **MIDlet Main Class**: `com.silverknight.TemMidlet`
  - Hierarchy: `com.silverknight.TemMidlet extends javax.microedition.midlet.MIDlet`
  - Instantiates `com.silverknight.a` canvas during startup.
- **Canvas Class**: `com.silverknight.a`
  - Hierarchy: `public final class com.silverknight.a extends javax.microedition.lcdui.Canvas implements java.lang.Runnable`
  - Key Overridden Methods:
    - `protected final void paint(javax.microedition.lcdui.Graphics g)`
    - `public final void run()`
    - `protected final void keyPressed(int keyCode)`
    - `protected final void keyReleased(int keyCode)`
    - `protected final void pointerPressed(int x, int y)`
    - `protected final void pointerReleased(int x, int y)`
    - `protected final void pointerDragged(int x, int y)`

### 3.2 Game Asset Structure
- ~200 PNG textures located under `x1/` (e.g., `x1/logo.png`, `x1/interface/18.png`, `x1/c/hair/`, `x1/weapon/`).
- Native image decoding via Go will intercept calls to `Image.createImage(String path)` and load these resource streams into Go VRAM (`*ebiten.Image`).

---

## 4. JNI & Native Signature Mapping

### 4.1 `Image.java` (`javax.microedition.lcdui.Image`)

**Java Class Definition:**
```java
package javax.microedition.lcdui;
import java.io.IOException;

public class Image {
    private int nativeId;

    public Image(int nativeId) {
        this.nativeId = nativeId;
    }

    public int getId() {
        return nativeId;
    }

    public static native Image createImage(String path) throws IOException;
    public static native Image createImage(int width, int height);
    public static native Image createImage(byte[] data, int offset, int length);
    public static native Image createImage(Image source, int x, int y, int width, int height, int transform);

    public native int getWidth();
    public native int getHeight();
    public native Graphics getGraphics();
}
```

**JNI Signature Mapping Table:**

| Method Signature (Java) | JNI Native Symbol / C Signature | Descriptor |
| --- | --- | --- |
| `createImage(String)` | `Java_javax_microedition_lcdui_Image_createImage__Ljava_lang_String_2(JNIEnv*, jclass, jstring)` | `(Ljava/lang/String;)Ljavax/microedition/lcdui/Image;` |
| `createImage(int, int)` | `Java_javax_microedition_lcdui_Image_createImage__II(JNIEnv*, jclass, jint, jint)` | `(II)Ljavax/microedition/lcdui/Image;` |
| `createImage(byte[], int, int)` | `Java_javax_microedition_lcdui_Image_createImage___3BII(JNIEnv*, jclass, jbyteArray, jint, jint)` | `([BII)Ljavax/microedition/lcdui/Image;` |
| `createImage(Image, int, int, int, int, int)` | `Java_javax_microedition_lcdui_Image_createImage__Ljavax_microedition_lcdui_Image_2IIIII(JNIEnv*, jclass, jobject, jint, jint, jint, jint, jint)` | `(Ljavax/microedition/lcdui/Image;IIIII)Ljavax/microedition/lcdui/Image;` |
| `getWidth()` | `Java_javax_microedition_lcdui_Image_getWidth(JNIEnv*, jobject)` | `()I` |
| `getHeight()` | `Java_javax_microedition_lcdui_Image_getHeight(JNIEnv*, jobject)` | `()I` |
| `getGraphics()` | `Java_javax_microedition_lcdui_Image_getGraphics(JNIEnv*, jobject)` | `()Ljavax/microedition/lcdui/Graphics;` |

---

### 4.2 `Graphics.java` (`javax.microedition.lcdui.Graphics`)

**Java Class Extension Definition:**
```java
package javax.microedition.lcdui;

public class Graphics {
    public native void fillRect(int x, int y, int width, int height);
    public native void drawRect(int x, int y, int width, int height);
    public native void drawLine(int x1, int y1, int x2, int y2);
    public native void setColor(int rgb);
    public native void drawString(String str, int x, int y, int anchor);
    public native void drawImage(Image img, int x, int y, int anchor);
    public native void drawRegion(Image src, int x_src, int y_src, int width, int height, int transform, int x_dest, int y_dest, int anchor);
    public native void setClip(int x, int y, int width, int height);
    public native int getClipX();
    public native int getClipY();
    public native int getClipWidth();
    public native int getClipHeight();
    public native void translate(int x, int y);
}
```

**JNI Signature Mapping Table:**

| Method Signature (Java) | JNI Native Symbol / C Signature | Descriptor |
| --- | --- | --- |
| `fillRect(int, int, int, int)` | `Java_javax_microedition_lcdui_Graphics_fillRect(JNIEnv*, jobject, jint, jint, jint, jint)` | `(IIII)V` |
| `drawRect(int, int, int, int)` | `Java_javax_microedition_lcdui_Graphics_drawRect(JNIEnv*, jobject, jint, jint, jint, jint)` | `(IIII)V` |
| `drawLine(int, int, int, int)` | `Java_javax_microedition_lcdui_Graphics_drawLine(JNIEnv*, jobject, jint, jint, jint, jint)` | `(IIII)V` |
| `setColor(int)` | `Java_javax_microedition_lcdui_Graphics_setColor(JNIEnv*, jobject, jint)` | `(I)V` |
| `drawString(String, int, int, int)` | `Java_javax_microedition_lcdui_Graphics_drawString(JNIEnv*, jobject, jstring, jint, jint, jint)` | `(Ljava/lang/String;III)V` |
| `drawImage(Image, int, int, int)` | `Java_javax_microedition_lcdui_Graphics_drawImage(JNIEnv*, jobject, jobject, jint, jint, jint)` | `(Ljavax/microedition/lcdui/Image;III)V` |
| `drawRegion(Image, int, int, int, int, int, int, int, int)` | `Java_javax_microedition_lcdui_Graphics_drawRegion(JNIEnv*, jobject, jobject, jint, jint, jint, jint, jint, jint, jint, jint)` | `(Ljavax/microedition/lcdui/Image;IIIIIIII)V` |
| `setClip(int, int, int, int)` | `Java_javax_microedition_lcdui_Graphics_setClip(JNIEnv*, jobject, jint, jint, jint, jint)` | `(IIII)V` |
| `getClipX()` | `Java_javax_microedition_lcdui_Graphics_getClipX(JNIEnv*, jobject)` | `()I` |
| `getClipY()` | `Java_javax_microedition_lcdui_Graphics_getClipY(JNIEnv*, jobject)` | `()I` |
| `getClipWidth()` | `Java_javax_microedition_lcdui_Graphics_getClipWidth(JNIEnv*, jobject)` | `()I` |
| `getClipHeight()` | `Java_javax_microedition_lcdui_Graphics_getClipHeight(JNIEnv*, jobject)` | `()I` |
| `translate(int, int)` | `Java_javax_microedition_lcdui_Graphics_translate(JNIEnv*, jobject, jint, jint)` | `(II)V` |

---

### 4.3 `Canvas.java` (`javax.microedition.lcdui.Canvas`)

**Java Class Definition:**
```java
package javax.microedition.lcdui;

public abstract class Canvas extends Displayable {
    public static final int UP = -1;
    public static final int DOWN = -2;
    public static final int LEFT = -3;
    public static final int RIGHT = -4;
    public static final int FIRE = -5;
    public static final int KEY_NUM0 = 48;
    public static final int KEY_NUM1 = 49;
    public static final int KEY_NUM2 = 50;
    public static final int KEY_NUM3 = 51;
    public static final int KEY_NUM4 = 52;
    public static final int KEY_NUM5 = 53;
    public static final int KEY_NUM6 = 54;
    public static final int KEY_NUM7 = 55;
    public static final int KEY_NUM8 = 56;
    public static final int KEY_NUM9 = 57;
    public static final int KEY_STAR = 42;
    public static final int KEY_POUND = 35;

    public Canvas() {}
    public void setFullScreenMode(boolean mode) {}
    public boolean hasPointerEvents() { return true; }
    public int getWidth() { return 240; }
    public int getHeight() { return 320; }
    public void repaint() { serviceRepaints(); }
    public native void serviceRepaints();

    protected abstract void paint(Graphics g);
    protected void keyPressed(int keyCode) {}
    protected void keyReleased(int keyCode) {}
    protected void pointerPressed(int x, int y) {}
    protected void pointerReleased(int x, int y) {}
    protected void pointerDragged(int x, int y) {}
}
```

**JNI Signature Mapping Table:**

| Method Signature (Java) | JNI Native Symbol / C Signature | Descriptor |
| --- | --- | --- |
| `serviceRepaints()` | `Java_javax_microedition_lcdui_Canvas_serviceRepaints(JNIEnv*, jobject)` | `()V` |

---

### 4.4 `Main.java` Launcher (`com.emulator.Main`)

**Java Launcher Definition:**
```java
package com.emulator;

import java.io.File;
import java.net.URL;
import java.net.URLClassLoader;
import javax.microedition.midlet.MIDlet;

public class Main {
    public static void startApp(String jarPath) {
        try {
            File file = new File(jarPath);
            URLClassLoader classLoader = new URLClassLoader(
                new URL[]{ file.toURI().toURL() },
                Main.class.getClassLoader()
            );
            Class<?> midletClass = classLoader.loadClass("com.silverknight.TemMidlet");
            MIDlet midlet = (MIDlet) midletClass.getDeclaredConstructor().newInstance();
            java.lang.reflect.Method startAppMethod = midletClass.getDeclaredMethod("startApp");
            startAppMethod.setAccessible(true);
            startAppMethod.invoke(midlet);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    public static void main(String[] args) {
        String path = (args != null && args.length > 0) ? args[0] : "/home/neuro/Downloads/KnightOnline_402.jar";
        startApp(path);
    }
}
```

---

## 5. Verification Strategy & Next Steps

1. **Java Stubs Compilation**:
   Run `javac -d j2me-stubs/bin $(find j2me-stubs/src -name "*.java")` to ensure clean bytecode generation without any AWT/Swing dependencies.
2. **Go CGO JNI Native Registration**:
   Register C functions corresponding to the `JNINativeMethod` tables in `pkg/jvm/bridge.c` or Go `RegisterNatives()`.
3. **Execution Test**:
   Invoke `com.emulator.Main.startApp("/home/neuro/Downloads/KnightOnline_402.jar")` via JNI inside Go `cmd/j2me/main.go` and verify that graphics render to the Ebitengine window.
