# J2ME Go Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite the J2ME Java Desktop Emulator into a Go application using Ebitengine for rendering and an embedded JVM (via CGO/JNI) for executing the game logic.

**Architecture:** A standalone Go executable running Ebitengine. The Go application initializes a JVM via JNI. Java J2ME stub classes (`Canvas`, `Graphics`, `Image`) are stripped of Swing/AWT and redefined with `native` methods. Go implements these native methods to draw directly onto the Ebiten screen.

**Tech Stack:** Go (1.21+), Ebitengine (`github.com/hajimehoshi/ebiten/v2`), JNI (`github.com/tebeka/jni`), Java (JDK 21).

## Global Constraints

- Working directory for Go code: `/home/neuro/Documents/Tools/java-app`
- Working directory for Java code: `/home/neuro/Documents/Tools/java-app/j2me-stubs`
- No Swing or AWT imports allowed in the Java stubs.
- Go MUST manage `ebiten.Image` objects and return simple integer IDs to Java to avoid memory overhead.

---

### Task 1: Project Scaffolding & Ebitengine Setup

**Files:**
- Create: `/home/neuro/Documents/Tools/java-app/go.mod`
- Create: `/home/neuro/Documents/Tools/java-app/main.go`

**Interfaces:**
- Produces: A basic runnable Go application opening an Ebitengine window.

- [ ] **Step 1: Initialize Go Modules**

```bash
cd /home/neuro/Documents/Tools/java-app
go mod init java-app
go get github.com/hajimehoshi/ebiten/v2
```

- [ ] **Step 2: Create basic Ebitengine Window in main.go**

```go
package main

import (
	"log"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "J2ME Go Engine Initialized")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 320
}

func main() {
	ebiten.SetWindowSize(640, 640)
	ebiten.SetWindowTitle("J2ME Go Engine")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Test Go Build**

Run: `go build -o j2mego`
Expected: Exits 0, creates executable `j2mego`.

- [ ] **Step 4: Commit**

```bash
cd /home/neuro/Documents/Tools/java-app
git add go.mod go.sum main.go
git commit -m "feat: scaffold Ebitengine project"
```

---

### Task 2: Embed JVM via CGO & JNI

**Files:**
- Create: `/home/neuro/Documents/Tools/java-app/jvm.go`
- Modify: `/home/neuro/Documents/Tools/java-app/main.go:1-32`

**Interfaces:**
- Consumes: Go Ebitengine loop.
- Produces: An active `*jni.Env` ready to execute Java classes.

- [ ] **Step 1: Install JNI package**

```bash
cd /home/neuro/Documents/Tools/java-app
go get github.com/tebeka/jni
```

- [ ] **Step 2: Initialize JVM in jvm.go**

```go
package main

/*
#cgo linux CFLAGS: -I/usr/lib/jvm/default-java/include -I/usr/lib/jvm/default-java/include/linux
#cgo linux LDFLAGS: -L/usr/lib/jvm/default-java/lib/server -ljvm
*/
import "C"

import (
	"log"
	"github.com/tebeka/jni"
)

var env *jni.Env

func initJVM() {
	opts := []string{
		"-Djava.class.path=./j2me-stubs/bin",
	}
	var err error
	_, env, err = jni.CreateJVM(opts...)
	if err != nil {
		log.Fatalf("Failed to create JVM: %v", err)
	}
}
```

- [ ] **Step 3: Hook JVM into main.go**

```go
package main

import (
	"log"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct{}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "JVM Embedded and Running")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 320
}

func main() {
	initJVM() // Initialize JVM before starting graphics
	
	ebiten.SetWindowSize(640, 640)
	ebiten.SetWindowTitle("J2ME Go Engine")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Verify CGO Build**

Run: `CGO_ENABLED=1 go build -o j2mego`
Expected: Exits 0. (Requires `libjvm.so` in `LD_LIBRARY_PATH` during execution).

- [ ] **Step 5: Commit**

```bash
git add jvm.go main.go go.mod go.sum
git commit -m "feat: embed JVM using JNI"
```

---

### Task 3: Create Native J2ME Java Stubs

**Files:**
- Create: `/home/neuro/Documents/Tools/java-app/j2me-stubs/src/javax/microedition/lcdui/Graphics.java`

**Interfaces:**
- Produces: A Java class with `native` methods.

- [ ] **Step 1: Write Graphics.java with native declarations**

```bash
mkdir -p /home/neuro/Documents/Tools/java-app/j2me-stubs/src/javax/microedition/lcdui
```

Create `/home/neuro/Documents/Tools/java-app/j2me-stubs/src/javax/microedition/lcdui/Graphics.java`:
```java
package javax.microedition.lcdui;

public class Graphics {
    public native void setColor(int rgb);
    public native void fillRect(int x, int y, int width, int height);
    
    // Add dummy constructor for initialization
    public Graphics() {}
}
```

- [ ] **Step 2: Compile Java Stub**

```bash
cd /home/neuro/Documents/Tools/java-app/j2me-stubs
mkdir -p bin
javac -d bin src/javax/microedition/lcdui/Graphics.java
```
Expected: Exits 0.

- [ ] **Step 3: Commit**

```bash
cd /home/neuro/Documents/Tools/java-app
git add j2me-stubs/
git commit -m "feat: add native Graphics Java stub"
```

---

### Task 4: Implement JNI Callbacks in Go

**Files:**
- Create: `/home/neuro/Documents/Tools/java-app/graphics_native.go`
- Modify: `/home/neuro/Documents/Tools/java-app/jvm.go`

**Interfaces:**
- Consumes: Java `native` method calls.
- Produces: Ebitengine drawing instructions.

- [ ] **Step 1: Write CGO bridge for Graphics_fillRect**

Create `/home/neuro/Documents/Tools/java-app/graphics_native.go`:
```go
package main

/*
#include <jni.h>

// Forward declaration of the Go function
extern void Java_javax_microedition_lcdui_Graphics_fillRect(JNIEnv *env, jobject thisObj, jint x, jint y, jint w, jint h);
*/
import "C"
import (
	"fmt"
)

//export Java_javax_microedition_lcdui_Graphics_fillRect
func Java_javax_microedition_lcdui_Graphics_fillRect(env *C.JNIEnv, thisObj C.jobject, x C.jint, y C.jint, w C.jint, h C.jint) {
	fmt.Printf("Java called fillRect(%d, %d, %d, %d)\n", int(x), int(y), int(w), int(h))
	// In the real implementation, this will queue a draw command to Ebiten
}
```

- [ ] **Step 2: Test Call from Java**

Create `/home/neuro/Documents/Tools/java-app/j2me-stubs/src/Test.java`:
```java
import javax.microedition.lcdui.Graphics;

public class Test {
    public static void main(String[] args) {
        Graphics g = new Graphics();
        g.fillRect(10, 20, 100, 100);
    }
}
```

Compile and run:
```bash
cd /home/neuro/Documents/Tools/java-app/j2me-stubs
javac -d bin src/Test.java src/javax/microedition/lcdui/Graphics.java
cd ..
CGO_ENABLED=1 go build -o j2mego
```

Update `main.go` to invoke `Test.main` before running Ebiten:
```go
func main() {
	initJVM()
	
	// Invoke Test.main
	class, err := env.FindClass("Test")
	if err != nil {
		log.Fatalf("Test class not found: %v", err)
	}
	methodId := env.GetStaticMethodID(class, "main", "([Ljava/lang/String;)V")
	env.CallStaticVoidMethod(class, methodId)
	
	ebiten.SetWindowSize(640, 640)
	ebiten.SetWindowTitle("J2ME Go Engine")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Verify the output**

Run: `LD_LIBRARY_PATH=/usr/lib/jvm/default-java/lib/server ./j2mego`
Expected: Console prints `Java called fillRect(10, 20, 100, 100)`

- [ ] **Step 4: Commit**

```bash
git add graphics_native.go main.go j2me-stubs/src/Test.java
git commit -m "feat: implement JNI callbacks from Java to Go"
```
