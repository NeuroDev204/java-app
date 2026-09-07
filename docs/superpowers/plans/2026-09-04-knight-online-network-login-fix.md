# Knight Online J2ME Mobile Network & Login Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the network login freeze in the Knight Online J2ME Flutter/Go emulator by implementing missing `ByteArrayOutputStream` and `java.util.Vector` VM natives, correcting the TeaMobi `0xD8` handshake protocol in `SocketTap`, and verifying successful character screen login on the Android emulator.

**Architecture:** 
- The J2ME MIDP app running inside Go VM communicates via `Connector.open("socket://...")` backed by `midp.SocketTap`.
- Packet payloads are constructed in Java using `java.io.ByteArrayOutputStream` wrapped in `DataOutputStream`.
- Outgoing packets are queued in `ax.a` (`java.util.Vector`) and sent by background thread `ax.run()`.
- Go VM intercepts `ByteArrayOutputStream` and `Vector` to provide memory-backed, thread-safe buffering and queueing.
- `SocketTap.ingestHandshakeLocked` detects the TeaMobi server's raw `0xD8` handshake frame (`[0xD8, len_hi, len_lo, key_len, key]`), initializes `protocol.Crypto(key)` and enables live TCP session communication.

**Tech Stack:** Go 1.24, gomobile, Flutter/Dart, Android SDK, J2ME bytecode interpreter, Java standard library native stubs.

## Global Constraints
- Strictly adhere to `AGENTS.md` (no guesswork, zero compiler errors, 100% test pass).
- Thread safety: `Vector` and stream buffer manipulations must be thread-safe across concurrent Go goroutines running Java threads (`h.run()`, `ax.run()`, `al.run()`, UI event loop).
- Retain `IsIdentityXOR` architecture where Go `SocketTap` manages stream encryption/decryption while Java XOR methods return identity bytes.

---

### Task 1: Implement `java/io/ByteArrayOutputStream` Native

**Files:**
- Modify: `pkg/vm/connector.go`
- Modify: `pkg/vm/interpreter.go`
- Test: `pkg/vm/streams_test.go`

**Interfaces:**
- Produces: Native support for `java/io/ByteArrayOutputStream`:
  - `<init>()V`, `<init>(I)V`
  - `write(I)V`, `write([B)V`, `write([BII)V`
  - `toByteArray()[B`
  - `reset()V`
  - `size()I`
  - `close()V`
- Consumes: `bytes.Buffer`, `*vm.Object`

- [ ] **Step 1: Inspect existing failing test**
  Run `go test -v ./pkg/vm -run TestByteArrayOutputStreamViaDataOutputStream` and confirm failure with `got []`.

- [ ] **Step 2: Add `handleByteArrayOutputStreamNative` in `pkg/vm/connector.go`**
  Implement native handler mapping `Object.Extra` to a `*bytes.Buffer`:
  ```go
  func (ctx *VMContext) handleByteArrayOutputStreamNative(mName, mDesc string, frame *Frame, opcode byte) bool
  ```
  Ensure `streamBufFromRef` extracts `*bytes.Buffer` from either `Object.Extra` or unwraps nested streams.
  Implement `toByteArray` returning `[]byte` (or `[]Slot` as required by JVM stack).

- [ ] **Step 3: Route `java/io/ByteArrayOutputStream` in `pkg/vm/interpreter.go`**
  In `handleNativeCall`, check `clsName == "java/io/ByteArrayOutputStream"` and invoke `handleByteArrayOutputStreamNative`.
  Also ensure `new DataOutputStream(baos)` properly shares the underlying `*bytes.Buffer` in `handleOutputStreamNative`.

- [ ] **Step 4: Verify test passes**
  Run `go test -v ./pkg/vm -run TestByteArrayOutputStreamViaDataOutputStream` and ensure PASS.

---

### Task 2: Implement Thread-Safe `java/util/Vector` Native

**Files:**
- Create: `pkg/vm/vector.go`
- Modify: `pkg/vm/interpreter.go`
- Test: `pkg/vm/streams_test.go`

**Interfaces:**
- Produces: `VectorData` struct with `sync.RWMutex` backing `java/util/Vector` methods:
  - `<init>()V`, `<init>(I)V`, `<init>(II)V`
  - `addElement(Ljava/lang/Object;)V`
  - `elementAt(I)Ljava/lang/Object;`
  - `removeElementAt(I)V`
  - `removeAllElements()V`
  - `size()I`
  - `isEmpty()Z`
- Consumes: `*vm.Object`, `vm.Slot`

- [ ] **Step 1: Inspect failing test**
  Run `go test -v ./pkg/vm -run TestVectorQueue` and confirm failure with `Vector.size after addElement`.

- [ ] **Step 2: Create `pkg/vm/vector.go`**
  Define `VectorData`:
  ```go
  type VectorData struct {
      mu    sync.RWMutex
      items []interface{}
  }
  ```
  Implement helper functions:
  - `NewVectorData() *VectorData`
  - `Add(elem interface{})`
  - `Get(idx int) interface{}`
  - `RemoveAt(idx int)`
  - `Clear()`
  - `Size() int`
  Implement `(ctx *VMContext) handleVectorNative(mName, mDesc string, frame *Frame, opcode byte) bool`.

- [ ] **Step 3: Route `java/util/Vector` in `pkg/vm/interpreter.go`**
  In `handleNativeCall`, add branch for `clsName == "java/util/Vector"` delegating to `handleVectorNative`.

- [ ] **Step 4: Verify test passes**
  Run `go test -v ./pkg/vm -run TestVectorQueue` and confirm PASS.

---

### Task 3: Fix TeaMobi `0xD8` Handshake Protocol in `SocketTap`

**Files:**
- Modify: `pkg/midp/sockettap.go`
- Modify: `pkg/midp/sockettap_test.go`

**Interfaces:**
- Produces: Correct framing in `SocketTap.ingestHandshakeLocked`:
  - Recognizes `0xD8` header (signed `-40`, unsigned `0xD8` / `216`).
  - Reads `uint16` payload length.
  - Extracts `keyLen` (1 byte) and `key` (`keyLen` bytes).
  - Installs `t.crypto = protocol.NewCrypto(key)`.
  - Sets `t.hsDone = true`.
  - Leaves the raw handshake bytes intact in the read buffer for Java `al.run()` to consume without corruption.
- Consumes: `net.Conn`, `protocol.NewCrypto`

- [ ] **Step 1: Write test for real TeaMobi handshake packet framing**
  In `pkg/midp/sockettap_test.go`, add `TestSocketTapRealTeaMobiHandshake`:
  Server sends `[]byte{0xD8, 0x00, 0x02, 0x01, 0xA8}`.
  Verify `tap.Crypto()` becomes non-nil with key `[]byte{0xA8}`.

- [ ] **Step 2: Update `ingestHandshakeLocked` in `pkg/midp/sockettap.go`**
  Adjust logic to support:
  1. Standard TeaMobi packet header: `0xD8` (or `byte(0xD8)` / `int8(-40)`), followed by 2-byte big-endian length.
  2. Fallback to raw 1-byte length + key for backward compatibility with existing tests.
  Ensure raw handshake bytes are preserved in `p` so Java reader `al.run()` can read `0xD8`, length, key length, and key bytes.

- [ ] **Step 3: Verify all `pkg/midp` tests pass**
  Run `go test -v ./pkg/midp` and ensure 100% PASS.

---

### Task 4: Add `java/lang/System.out` / `PrintStream` Logging

**Files:**
- Modify: `pkg/vm/interpreter.go`

**Interfaces:**
- Produces: Log output on `PrintStream.println` and `print` with prefix `[JAVA-OUT]` for visibility into `l.a` ("------------IP host:...").

- [ ] **Step 1: Handle `java/io/PrintStream` in `handleNativeCall`**
  In `pkg/vm/interpreter.go`, intercept calls to `java/io/PrintStream`:
  - `println(Ljava/lang/String;)V`, `print(Ljava/lang/String;)V`, `println(I)V`, etc.
  - Log via `log.Printf("[JAVA-OUT] %v", msg)`.
  - Pop parameters cleanly.

- [ ] **Step 2: Unit test PrintStream logging**
  Run unit test in `pkg/vm` ensuring no stack imbalance or crash on `System.out.println`.

---

### Task 5: Build Native Bridge & Live Android Verification

**Files:**
- Run: `mobile/scripts/bind_native.sh`
- Deploy to emulator: `emulator-5554`

- [ ] **Step 1: Build `gamebridge.aar`**
  Run `bash mobile/scripts/bind_native.sh`.
  Confirm `mobile/android/app/libs/gamebridge.aar` is updated.

- [ ] **Step 2: Run all tests in repository**
  Run `go test ./pkg/...`. Ensure 0 failures.

- [ ] **Step 3: Rebuild & Launch App on Android Emulator**
  Run `cd mobile && flutter run -d emulator-5554` or build & install APK.
  Launch `com.example.knight_online/.MainActivity`.

- [ ] **Step 4: Tap "Chơi Mới" & Verify Network Login**
  Monitor logcat:
  `adb logcat -s GoLog:V`
  Verify:
  - `[JAVA-OUT] ------------IP host:hs1.teamobi.com--port:19129` appears.
  - `[CONNECTOR] open socket://hs1.teamobi.com:19129` connects.
  - Handshake completes and login packet `0x01` is sent by `ax.run()`.
  - Server responds with character / game state packet.
  - Take screenshot and confirm UI transitions from loading spinner to character selection or creation screen.
