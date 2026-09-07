# Design Specification: J2ME Emulator in Go (Linux & iOS target)

## 1. Overview
The goal of this project is to build a lightweight, cross-platform Java ME (J2ME CLDC 1.1 / MIDP 2.0) emulator written in Go using Ebitengine as the rendering & input backend. It targets Linux desktop for local development and execution, with built-in cross-compilation support for iOS (iPhone) via `gomobile` / GitHub Actions CI.

## 2. Architecture & Directory Structure
The application lives in `/home/neuro/Documents/Tools/java-app` and follows standard Go project organization:

```
java-app/
├── cmd/
│   └── j2me/             # Main entry point
│       └── main.go
├── pkg/
│   ├── jar/              # ZIP/JAR reader & MANIFEST.MF parser
│   ├── classfile/        # Java .class binary format parser (Constant pool, methods, attributes)
│   ├── vm/               # JVM bytecode interpreter & execution engine (Stack frames, local variables, opcodes)
│   ├── midp/             # J2ME MIDP 2.0 / CLDC 1.1 runtime APIs (javax/microedition/* stubs)
│   └── gfx/              # Ebitengine renderer, Nokia 240x320 phone canvas scaling, input mapping
├── docs/                 # Documentation & specs
└── go.mod                # Go module file
```

## 3. Subsystem Specifications

### 3.1 JAR & Manifest Parser (`pkg/jar`)
- Opens `.jar` files (ZIP format).
- Parses `META-INF/MANIFEST.MF` to extract:
  - `MIDlet-Name`, `MIDlet-Version`, `MIDlet-Vendor`
  - `MIDlet-1`: Specifies main MIDlet entry class name (e.g. `com.vendor.game.GameMIDlet`).
- Extracts `.class` files into an in-memory class reader registry.

### 3.2 Java ClassFile Parser (`pkg/classfile`)
- Parses standard Java `.class` binary format:
  - Magic number (`0xCAFEBABE`), Minor/Major versions.
  - Constant Pool: UTF8, Integer, Float, Long, Double, Class, String, Fieldref, Methodref, InterfaceMethodref, NameAndType.
  - Access Flags, ThisClass, SuperClass, Interfaces.
  - Fields & Methods definitions.
  - Attributes: `Code` attribute (max_stack, max_locals, code byte array, exception table, line number table).

### 3.3 JVM Bytecode Interpreter (`pkg/vm`)
- **Execution Stack & Local Variables**: Operates on 32-bit/64-bit stack values (`Slot` type for Primitive / Object references).
- **Stack Frame**: Contains `LocalVariables []Slot`, `OperandStack []Slot`, `ConstantPool`, `ProgramCounter (PC)`, `Method`.
- **Supported Core Opcodes**:
  - Constants & Push: `nop`, `aconst_null`, `iconst_m1`..`iconst_5`, `lconst_0`..`lconst_1`, `bipush`, `sipush`, `ldc`, `ldc_w`, `ldc2_w`.
  - Load/Store: `iload`, `lload`, `fload`, `dload`, `aload` (+ `_0`..`_3`), `istore`, `lstore`, `fstore`, `dstore`, `astore` (+ `_0`..`_3`), `bastore`, `castore`, `sastore`, `iastore`, `aastore`, `baload`, `caload`, `saload`, `iaload`, `aaload`.
  - Arithmetic & Logic: `iadd`, `isub`, `imul`, `idiv`, `irem`, `ineg`, `ishl`, `ishr`, `iushr`, `iand`, `ior`, `ixor`, `iinc`.
  - Conversions: `i2b`, `i2c`, `i2s`, `i2l`, `i2f`, `i2d`, `l2i`, `f2i`, `d2i`.
  - Branching: `ifeq`, `ifne`, `iflt`, `ifge`, `ifgt`, `ifle`, `if_icmpeq`, `if_icmpne`, `if_icmplt`, `if_icmpge`, `if_icmpgt`, `if_icmple`, `if_acmpeq`, `if_acmpne`, `goto`, `tableswitch`, `lookupswitch`.
  - Method Invocation & Return: `invokevirtual`, `invokestatic`, `invokespecial`, `ireturn`, `lreturn`, `freturn`, `dreturn`, `areturn`, `return`.
  - Object/Class Operations: `new`, `newarray`, `anewarray`, `arraylength`, `getfield`, `putfield`, `getstatic`, `putstatic`, `checkcast`, `instanceof`.
  - Stack Manipulation: `pop`, `pop2`, `dup`, `dup_x1`, `dup_x2`, `dup2`, `dup2_x1`, `dup2_x2`, `swap`.
- **Native Method Dispatcher**: Bridges JVM method calls to Go Go-native implementations (e.g. `java/lang/Object.<init>`, `java/lang/System.currentTimeMillis`, `javax/microedition/*`).

### 3.4 J2ME MIDP 2.0 / CLDC 1.1 API Emulation (`pkg/midp`)
- `javax.microedition.midlet.MIDlet`: Life-cycle management (`startApp`, `pauseApp`, `destroyApp`).
- `javax.microedition.lcdui.Display`: Manages current active `Displayable` (`Canvas`, `Form`, etc.).
- `javax.microedition.lcdui.Canvas`:
  - 60 FPS repainting loop.
  - Event dispatch: `keyPressed(int key)`, `keyReleased(int key)`, `pointerPressed(int x, int y)` (for touchscreen/iOS).
  - J2ME Key codes: `KEY_NUM0`..`9`, `KEY_STAR`, `KEY_POUND`, `UP`, `DOWN`, `LEFT`, `RIGHT`, `FIRE`.
- `javax.microedition.lcdui.Graphics`:
  - Off-screen drawing API: `setColor(int r, int g, int b)`, `drawLine`, `drawRect`, `fillRect`, `drawString`, `drawImage`, `setClip`, `clipRect`.
- `javax.microedition.lcdui.Image`: `createImage(w, h)`, `createImage(byte[], offset, len)`, `getGraphics()`.

### 3.5 Graphics & UI Engine (`pkg/gfx`)
- Built on **Ebitengine** (`github.com/hajimehoshi/ebiten/v2`).
- Screen Layout:
  - Outer Phone Case / Virtual Keypad border.
  - Central J2ME Game Viewport (default 240x320, pixel integer scaled).
- Input Handling:
  - **Linux Desktop**: Keyboard keys (Arrow keys / Numpad 0-9 / Enter / Esc / Space) mapped to J2ME key codes.
  - **iOS Mobile**: Virtual key pad buttons rendered on touch screen.

## 4. Error Handling & Fallbacks
- Logging missing opcodes or unhandled native methods gracefully without crashing the VM.
- Safe Stack bounds checking and null pointer exception simulation.

## 5. Build & Deployment Pipeline
- **Linux**: Standard `go build -o j2me ./cmd/j2me`.
- **iOS**: Cross-compilation via `gomobile build -target=ios` or GitHub Actions workflow using macOS runners to package `.ipa`.
