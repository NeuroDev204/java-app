# J2ME Emulator Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a functional J2ME (CLDC 1.1 / MIDP 2.0) Java emulator in Go using Ebitengine for rendering and keyboard/touch input, running on Linux and portable to iOS.

**Architecture:** A modular Go architecture where `pkg/jar` extracts JAR class files and metadata, `pkg/classfile` parses Java `.class` binary structures, `pkg/vm` executes JVM bytecode opcodes and handles method frames/native bridging, `pkg/midp` implements J2ME `javax.microedition.*` APIs, and `pkg/gfx` renders the UI via Ebitengine.

**Tech Stack:** Go 1.24, Ebitengine v2 (`github.com/hajimehoshi/ebiten/v2`), standard `testing` package.

## Global Constraints
- Target Language: Go 1.24+
- Directory: `/home/neuro/Documents/Tools/java-app`
- All tests must pass via `go test ./...` with 0 errors before completion.

---

### Task 1: Project Module Setup & Directory Skeleton

**Files:**
- Create: `java-app/go.mod`
- Create: `java-app/cmd/j2me/main.go`
- Test: `java-app/cmd/j2me/main_test.go`

**Interfaces:**
- Consumes: None
- Produces: Base Go module `java-app` and executable entrypoint `cmd/j2me`

- [ ] **Step 1: Write failing main test**

```go
// java-app/cmd/j2me/main_test.go
package main

import "testing"

func TestAppTitle(t *testing.T) {
	expected := "J2ME Go Emulator"
	if AppTitle != expected {
		t.Errorf("expected %s, got %s", expected, AppTitle)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./cmd/j2me`
Expected: FAIL (AppTitle undefined)

- [ ] **Step 3: Write minimal implementation & go.mod**

```go
// java-app/cmd/j2me/main.go
package main

import "fmt"

const AppTitle = "J2ME Go Emulator"

func main() {
	fmt.Println(AppTitle)
}
```

Initialize `go.mod`:
```bash
cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go mod init java-app
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./cmd/j2me`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add java-app/go.mod java-app/cmd/j2me
git commit -m "feat: initialize java-app Go module and main package"
```

---

### Task 2: JAR & MANIFEST.MF Reader (`pkg/jar`)

**Files:**
- Create: `java-app/pkg/jar/reader.go`
- Test: `java-app/pkg/jar/reader_test.go`

**Interfaces:**
- Consumes: ZIP archive bytes (`.jar` file)
- Produces: `JarArchive` struct with `Manifest map[string]string`, `MainMIDletClass string`, `GetClassBytes(name string) ([]byte, error)`

- [ ] **Step 1: Write failing unit test for JAR Reader**

```go
// java-app/pkg/jar/reader_test.go
package jar

import "testing"

func TestParseManifest(t *testing.T) {
	manifestContent := "MIDlet-Name: TestGame\nMIDlet-1: TestGame,,com.vendor.TestMIDlet\nMIDlet-Vendor: Vendor\n"
	manifestMap := parseManifest([]byte(manifestContent))

	if manifestMap["MIDlet-Name"] != "TestGame" {
		t.Errorf("expected MIDlet-Name to be TestGame, got %s", manifestMap["MIDlet-Name"])
	}
	if manifestMap["MIDlet-1"] != "TestGame,,com.vendor.TestMIDlet" {
		t.Errorf("unexpected MIDlet-1: %s", manifestMap["MIDlet-1"])
	}

	mainClass := extractMainClass(manifestMap["MIDlet-1"])
	if mainClass != "com/vendor/TestMIDlet" {
		t.Errorf("expected main class com/vendor/TestMIDlet, got %s", mainClass)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/jar`
Expected: FAIL (parseManifest undefined)

- [ ] **Step 3: Write minimal implementation**

```go
// java-app/pkg/jar/reader.go
package jar

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
)

type JarArchive struct {
	Manifest         map[string]string
	MainMIDletClass  string
	Classes          map[string][]byte
}

func parseManifest(data []byte) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			result[key] = val
		}
	}
	return result
}

func extractMainClass(midlet1Val string) string {
	parts := strings.Split(midlet1Val, ",")
	if len(parts) >= 3 {
		cls := strings.TrimSpace(parts[2])
		return strings.ReplaceAll(cls, ".", "/")
	}
	return ""
}

func ReadJar(jarBytes []byte) (*JarArchive, error) {
	reader, err := zip.NewReader(bytes.NewReader(jarBytes), int64(len(jarBytes)))
	if err != nil {
		return nil, err
	}

	archive := &JarArchive{
		Manifest: make(map[string]string),
		Classes:  make(map[string][]byte),
	}

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		if file.Name == "META-INF/MANIFEST.MF" {
			archive.Manifest = parseManifest(data)
			if val, ok := archive.Manifest["MIDlet-1"]; ok {
				archive.MainMIDletClass = extractMainClass(val)
			}
		} else if strings.HasSuffix(file.Name, ".class") {
			className := strings.TrimSuffix(file.Name, ".class")
			archive.Classes[className] = data
		}
	}

	return archive, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/jar`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add java-app/pkg/jar
git commit -m "feat: add JAR and MANIFEST.MF parser in pkg/jar"
```

---

### Task 3: Java ClassFile Parser (`pkg/classfile`)

**Files:**
- Create: `java-app/pkg/classfile/classfile.go`
- Create: `java-app/pkg/classfile/constant_pool.go`
- Test: `java-app/pkg/classfile/classfile_test.go`

**Interfaces:**
- Consumes: `.class` file byte slice
- Produces: `ClassFile` struct containing Magic, Version, ConstantPool, AccessFlags, ThisClass, SuperClass, Methods (with `Code` attribute).

- [ ] **Step 1: Write failing unit test for Classfile Parser**

```go
// java-app/pkg/classfile/classfile_test.go
package classfile

import "testing"

func TestMagicHeaderValidation(t *testing.T) {
	// Minimal invalid class header
	invalidData := []byte{0x12, 0x34, 0x56, 0x78}
	_, err := Parse(invalidData)
	if err == nil {
		t.Error("expected error for invalid magic bytes, got nil")
	}

	// Mock valid header
	validHeader := []byte{
		0xCA, 0xFE, 0xBA, 0xBE, // Magic
		0x00, 0x00, 0x00, 0x32, // Minor=0, Major=50 (Java 6 / CLDC 1.1)
		0x00, 0x01, // Constant pool count = 1
	}
	cf, err := Parse(validHeader)
	if err != nil {
		t.Fatalf("unexpected error parsing valid header: %v", err)
	}
	if cf.Magic != 0xCAFEBABE {
		t.Errorf("expected magic 0xCAFEBABE, got 0x%X", cf.Magic)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/classfile`
Expected: FAIL (Parse undefined)

- [ ] **Step 3: Write minimal implementation**

```go
// java-app/pkg/classfile/constant_pool.go
package classfile

import "encoding/binary"

const (
	TagUtf8               = 1
	TagInteger            = 3
	TagFloat              = 4
	TagLong               = 5
	TagDouble             = 6
	TagClass              = 7
	TagString             = 8
	TagFieldref           = 9
	TagMethodref          = 10
	TagInterfaceMethodref = 11
	TagNameAndType        = 12
)

type ConstantInfo interface{}

type ConstantClassInfo struct {
	NameIndex uint16
}

type ConstantUtf8Info struct {
	Str string
}

type ConstantPool []ConstantInfo
```

```go
// java-app/pkg/classfile/classfile.go
package classfile

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MethodInfo struct {
	AccessFlags uint16
	NameIndex   uint16
	DescIndex   uint16
	Code        []byte
	MaxStack    uint16
	MaxLocals   uint16
}

type ClassFile struct {
	Magic        uint32
	MinorVersion uint16
	MajorVersion uint16
	ConstantPool ConstantPool
	AccessFlags  uint16
	ThisClass    uint16
	SuperClass   uint16
	Interfaces   []uint16
	Methods      []MethodInfo
}

func Parse(data []byte) (*ClassFile, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("classfile data too short")
	}

	cf := &ClassFile{}
	cf.Magic = binary.BigEndian.Uint32(data[0:4])
	if cf.Magic != 0xCAFEBABE {
		return nil, fmt.Errorf("invalid classfile magic: 0x%X", cf.Magic)
	}

	cf.MinorVersion = binary.BigEndian.Uint16(data[4:6])
	cf.MajorVersion = binary.BigEndian.Uint16(data[6:8])
	cpCount := binary.BigEndian.Uint16(data[8:10])

	cf.ConstantPool = make(ConstantPool, cpCount)
	// Additional parser code for ConstantPool & Methods...
	return cf, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/classfile`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add java-app/pkg/classfile
git commit -m "feat: add Java classfile binary structure parser"
```

---

### Task 4: JVM Stack Frame & Operand Data Structures (`pkg/vm`)

**Files:**
- Create: `java-app/pkg/vm/slot.go`
- Create: `java-app/pkg/vm/frame.go`
- Test: `java-app/pkg/vm/frame_test.go`

**Interfaces:**
- Consumes: None
- Produces: `Slot` (int32/ref), `OperandStack` (PushInt, PopInt, PushRef, PopRef), `LocalVars` (GetInt, SetInt, GetRef, SetRef), `Frame` struct.

- [ ] **Step 1: Write failing unit test for Stack Frame**

```go
// java-app/pkg/vm/frame_test.go
package vm

import "testing"

func TestOperandStackOperations(t *testing.T) {
	stack := NewOperandStack(10)
	stack.PushInt(42)
	stack.PushInt(100)

	val1 := stack.PopInt()
	val2 := stack.PopInt()

	if val1 != 100 || val2 != 42 {
		t.Errorf("expected 100 then 42, got %d then %d", val1, val2)
	}
}

func TestLocalVarsOperations(t *testing.T) {
	locals := NewLocalVars(5)
	locals.SetInt(0, 999)
	if locals.GetInt(0) != 999 {
		t.Errorf("expected 999 at index 0, got %d", locals.GetInt(0))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/vm`
Expected: FAIL (NewOperandStack undefined)

- [ ] **Step 3: Write minimal implementation**

```go
// java-app/pkg/vm/slot.go
package vm

type Slot struct {
	Num int32
	Ref interface{}
}
```

```go
// java-app/pkg/vm/frame.go
package vm

type OperandStack struct {
	slots []Slot
	size  uint
}

func NewOperandStack(maxStack uint) *OperandStack {
	return &OperandStack{slots: make([]Slot, maxStack)}
}

func (s *OperandStack) PushInt(val int32) {
	s.slots[s.size] = Slot{Num: val}
	s.size++
}

func (s *OperandStack) PopInt() int32 {
	s.size--
	return s.slots[s.size].Num
}

func (s *OperandStack) PushRef(ref interface{}) {
	s.slots[s.size] = Slot{Ref: ref}
	s.size++
}

func (s *OperandStack) PopRef() interface{} {
	s.size--
	return s.slots[s.size].Ref
}

type LocalVars []Slot

func NewLocalVars(maxLocals uint) LocalVars {
	return make(LocalVars, maxLocals)
}

func (l LocalVars) SetInt(index uint, val int32) {
	l[index] = Slot{Num: val}
}

func (l LocalVars) GetInt(index uint) int32 {
	return l[index].Num
}

func (l LocalVars) SetRef(index uint, ref interface{}) {
	l[index] = Slot{Ref: ref}
}

func (l LocalVars) GetRef(index uint) interface{} {
	return l[index].Ref
}

type Frame struct {
	LocalVars    LocalVars
	OperandStack *OperandStack
	PC           int
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/vm`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add java-app/pkg/vm
git commit -m "feat: add JVM execution stack frame and local variables"
```

---

### Task 5: JVM Bytecode Interpreter (`pkg/vm`)

**Files:**
- Create: `java-app/pkg/vm/interpreter.go`
- Create: `java-app/pkg/vm/opcodes.go`
- Test: `java-app/pkg/vm/interpreter_test.go`

**Interfaces:**
- Consumes: Code byte array `[]byte` and `Frame`
- Produces: `ExecuteBytecode(code []byte, frame *Frame) error`

- [ ] **Step 1: Write failing unit test for Opcode Execution**

```go
// java-app/pkg/vm/interpreter_test.go
package vm

import "testing"

func TestExecuteAdd(t *testing.T) {
	// Bytecode: iconst_5 (0x08), bipush 10 (0x10 0x0A), iadd (0x60), ireturn (0xAC)
	code := []byte{0x08, 0x10, 0x0A, 0x60, 0xAC}
	frame := &Frame{
		LocalVars:    NewLocalVars(2),
		OperandStack: NewOperandStack(5),
		PC:           0,
	}

	result, err := ExecuteBytecode(code, frame)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if result.Num != 15 {
		t.Errorf("expected 15, got %d", result.Num)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/vm`
Expected: FAIL (ExecuteBytecode undefined)

- [ ] **Step 3: Write minimal implementation**

```go
// java-app/pkg/vm/opcodes.go
package vm

const (
	OpNop         = 0x00
	OpIconst5     = 0x08
	OpBipush      = 0x10
	OpIadd        = 0x60
	OpIreturn     = 0xAC
)
```

```go
// java-app/pkg/vm/interpreter.go
package vm

import "fmt"

func ExecuteBytecode(code []byte, frame *Frame) (Slot, error) {
	for frame.PC < len(code) {
		opcode := code[frame.PC]
		frame.PC++

		switch opcode {
		case OpNop:
			// No operation
		case OpIconst5:
			frame.OperandStack.PushInt(5)
		case OpBipush:
			val := int8(code[frame.PC])
			frame.PC++
			frame.OperandStack.PushInt(int32(val))
		case OpIadd:
			v2 := frame.OperandStack.PopInt()
			v1 := frame.OperandStack.PopInt()
			frame.OperandStack.PushInt(v1 + v2)
		case OpIreturn:
			return Slot{Num: frame.OperandStack.PopInt()}, nil
		default:
			return Slot{}, fmt.Errorf("unsupported opcode: 0x%X at PC=%d", opcode, frame.PC-1)
		}
	}
	return Slot{}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/vm`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add java-app/pkg/vm
git commit -m "feat: add bytecode opcode execution interpreter engine"
```

---

### Task 6: J2ME MIDP 2.0 / CLDC APIs (`pkg/midp`)

**Files:**
- Create: `java-app/pkg/midp/display.go`
- Create: `java-app/pkg/midp/canvas.go`
- Create: `java-app/pkg/midp/graphics.go`
- Test: `java-app/pkg/midp/graphics_test.go`

**Interfaces:**
- Consumes: Off-screen draw calls
- Produces: `Graphics` (SetColor, DrawLine, DrawRect, FillRect), `Canvas` off-screen frame buffer.

- [ ] **Step 1: Write failing unit test for Graphics buffer**

```go
// java-app/pkg/midp/graphics_test.go
package midp

import "testing"

func TestGraphicsFillRect(t *testing.T) {
	g := NewGraphics(240, 320)
	g.SetColor(255, 0, 0) // Red
	g.FillRect(0, 0, 10, 10)

	r, gVal, b, a := g.Buffer.At(5, 5).RGBA()
	if r == 0 || gVal != 0 || b != 0 || a == 0 {
		t.Errorf("expected red pixel at (5,5), got r=%d, g=%d, b=%d, a=%d", r, gVal, b, a)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/midp`
Expected: FAIL (NewGraphics undefined)

- [ ] **Step 3: Write minimal implementation**

```go
// java-app/pkg/midp/graphics.go
package midp

import (
	"image"
	"image/color"
	"image/draw"
)

type Graphics struct {
	Width, Height int
	Buffer        *image.RGBA
	CurrentColor  color.Color
}

func NewGraphics(w, h int) *Graphics {
	return &Graphics{
		Width:        w,
		Height:       h,
		Buffer:       image.NewRGBA(image.Rect(0, 0, w, h)),
		CurrentColor: color.Black,
	}
}

func (g *Graphics) SetColor(r, gVal, b int) {
	g.CurrentColor = color.RGBA{R: uint8(r), G: uint8(gVal), B: uint8(b), A: 255}
}

func (g *Graphics) FillRect(x, y, w, h int) {
	rect := image.Rect(x, y, x+w, y+h)
	draw.Draw(g.Buffer, rect, &image.Uniform{C: g.CurrentColor}, image.Point{}, draw.Src)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/midp`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add java-app/pkg/midp
git commit -m "feat: add J2ME Graphics and offscreen buffer rendering"
```

---

### Task 7: Ebitengine UI Renderer & Keypad Mapping (`pkg/gfx`)

**Files:**
- Create: `java-app/pkg/gfx/renderer.go`
- Create: `java-app/pkg/gfx/keypad.go`
- Test: `java-app/pkg/gfx/renderer_test.go`

**Interfaces:**
- Consumes: Ebitengine `ebiten.Game` interface & `midp.Graphics`
- Produces: Window rendering loop (240x320 Nokia UI + virtual keypad), keyboard event handler.

- [ ] **Step 1: Write failing test for Keypad Mapping**

```go
// java-app/pkg/gfx/renderer_test.go
package gfx

import (
	"testing"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestKeycodeMapping(t *testing.T) {
	j2meKey := MapEbitenKeyToJ2ME(ebiten.KeyNumpad1)
	if j2meKey != '1' {
		t.Errorf("expected J2ME key '1', got %d", j2meKey)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/gfx`
Expected: FAIL (MapEbitenKeyToJ2ME undefined)

- [ ] **Step 3: Install Ebitengine dependency & Write implementation**

Install Ebitengine dependency:
```bash
cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go get github.com/hajimehoshi/ebiten/v2
```

```go
// java-app/pkg/gfx/keypad.go
package gfx

import "github.com/hajimehoshi/ebiten/v2"

func MapEbitenKeyToJ2ME(key ebiten.Key) int {
	switch key {
	case ebiten.KeyNumpad0, ebiten.KeyDigit0:
		return '0'
	case ebiten.KeyNumpad1, ebiten.KeyDigit1:
		return '1'
	case ebiten.KeyNumpad2, ebiten.KeyDigit2:
		return '2'
	case ebiten.KeyNumpad3, ebiten.KeyDigit3:
		return '3'
	case ebiten.KeyUp:
		return -1 // UP
	case ebiten.KeyDown:
		return -2 // DOWN
	case ebiten.KeyLeft:
		return -3 // LEFT
	case ebiten.KeyRight:
		return -4 // RIGHT
	case ebiten.KeyEnter:
		return -5 // FIRE
	default:
		return 0
	}
}
```

```go
// java-app/pkg/gfx/renderer.go
package gfx

import (
	"java-app/pkg/midp"
	"github.com/hajimehoshi/ebiten/v2"
)

type EmulatorGame struct {
	Graphics *midp.Graphics
}

func (g *EmulatorGame) Update() error {
	return nil
}

func (g *EmulatorGame) Draw(screen *ebiten.Image) {
	if g.Graphics != nil && g.Graphics.Buffer != nil {
		ebitenImage := ebiten.NewImageFromImage(g.Graphics.Buffer)
		screen.DrawImage(ebitenImage, nil)
	}
}

func (g *EmulatorGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 240, 320
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./pkg/gfx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add java-app/go.mod java-app/go.sum java-app/pkg/gfx
git commit -m "feat: integrate Ebitengine UI renderer and keyboard input mapping"
```

---

### Task 8: Complete End-to-End Main Executable (`cmd/j2me`)

**Files:**
- Modify: `java-app/cmd/j2me/main.go`
- Test: `java-app/cmd/j2me/main_test.go`

**Interfaces:**
- Consumes: `.jar` file path via command line argument `j2me <path-to-game.jar>`
- Produces: Launches Ebitengine window displaying the J2ME game canvas.

- [ ] **Step 1: Update main.go to accept JAR path and start Ebitengine window**

```go
// java-app/cmd/j2me/main.go
package main

import (
	"fmt"
	"log"
	"os"

	"java-app/pkg/gfx"
	"java-app/pkg/jar"
	"java-app/pkg/midp"

	"github.com/hajimehoshi/ebiten/v2"
)

const AppTitle = "J2ME Go Emulator"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: j2me <path-to-game.jar>")
		os.Exit(1)
	}

	jarPath := os.Args[1]
	data, err := os.ReadFile(jarPath)
	if err != nil {
		log.Fatalf("Failed to read jar file: %v", err)
	}

	archive, err := jar.ReadJar(data)
	if err != nil {
		log.Fatalf("Failed to parse jar: %v", err)
	}

	fmt.Printf("Loaded JAR: %s (Main Class: %s)\n", archive.Manifest["MIDlet-Name"], archive.MainMIDletClass)

	graphics := midp.NewGraphics(240, 320)
	graphics.SetColor(0, 0, 128) // Blue background test
	graphics.FillRect(0, 0, 240, 320)

	game := &gfx.EmulatorGame{Graphics: graphics}

	ebiten.SetWindowSize(480, 640)
	ebiten.SetWindowTitle("J2ME Loader - " + archive.Manifest["MIDlet-Name"])

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 2: Run all tests to verify entire project passes**

Run: `cd /home/neuro/Documents/Tools/java-app && /usr/local/go/bin/go test ./...`
Expected: PASS across all packages (`cmd/j2me`, `pkg/jar`, `pkg/classfile`, `pkg/vm`, `pkg/midp`, `pkg/gfx`).

- [ ] **Step 3: Commit**

```bash
git add java-app/cmd/j2me/main.go
git commit -m "feat: complete end-to-end main CLI runner for J2ME JAR files"
```
