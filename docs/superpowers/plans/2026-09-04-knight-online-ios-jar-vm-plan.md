# Knight Online iOS JAR-on-Go-VM Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run `KnightOnline_402.jar` on iOS through the pure-Go interpreter, with a Flutter shell (240×320 canvas, direct touch, AFK silent audio, auto-upgrade HUD) that never opens a second game TCP session.

**Architecture:** Flutter talks to `pkg/gamebridge` over MethodChannel. The VM executes the JAR. `Connector.open("socket://…")` is a Go native that owns one `SocketTap` (`net.Conn` + HSO XOR). Auto-upgrade injects `protocol.Message` values on that tap's write mutex. Knight Online Java XOR methods are identity stubs so Go holds the only cipher pointer. `pkg/autoupgrade` is CGO/JNI and must **not** be imported by `gamebridge` (gomobile iOS cannot link it).

**Tech Stack:** Go 1.24, `pkg/vm` + `pkg/midp`, `pkg/network/protocol`, Flutter 3.x, gomobile bind (no CGO), `audioplayers`, `wakelock_plus`, GitHub Actions `macos-latest`.

## Global Constraints

- **Platform Target:** iOS (Personal Sideloading via Sideloadly / AltStore / TrollStore).
- **Target Game:** Knight Online (Hiệp Sĩ Online - TeaMobi `KnightOnline_402.jar`). Server URL comes from the JAR (`hs1.teamobi.com:19129`); do not dial a second host.
- **Canvas Resolution:** Logical resolution 240x320 aspect-fit scaled to screen.
- **Background Mode:** iOS `UIBackgroundModes: ["audio"]` with silent audio loop.
- **Direct Input:** Pointer touch events translated directly to J2ME coordinates without D-pad buttons.
- **Headless AFK:** Tick loop continues running in background/screen-off; only GPU drawing is skipped.
- **One socket:** Auto-upgrade injects on the JAR's `SocketTap`. Never `Session.Connect` from the bot.
- **XOR ownership:** Go `protocol.Crypto` is the only cipher on the tap. JAR XOR methods are identity.
- **Dead socket:** EOF to the JAR; do not swap a new `net.Conn` under existing streams.
- **`in_game`:** true after inbound opcodes `-100`, `24`, or `26`.
- **CGO:** `gamebridge` and all packages it imports must be CGO-free.
- **Coding Standard:** Clean architecture, strict type safety, zero linter warnings, unit tests for all domain logic.

## File structure

| File | Responsibility |
|------|----------------|
| `pkg/network/protocol/frame.go` | Shared plaintext frame encode/decode (`WriteMessage`, `ReadPlainMessage`). |
| `pkg/midp/sockettap.go` | One `net.Conn`, handshake, XOR, JAR read/write, inject mutex, EOF on close. |
| `pkg/upgradebot/packet.go` | `BuildUpgradeAction` — payload of `q.b(byte,short,byte)`. |
| `pkg/upgradebot/rules.go` | Clover/target/stop-condition (CGO-free copy of the rule engine). |
| `pkg/upgradebot/controller.go` | Start/Stop loop; injects on `SocketTap`; never talks to Flutter. |
| `pkg/vm/interpreter.go` | Native `Connector.open` + socket stream read/write; XOR identity table. |
| `pkg/vm/xorstub.go` | Class/method names treated as identity XOR. |
| `pkg/gamebridge/bridge.go` | gomobile API: LoadJAR, framebuffer, touch, upgrade, `SetShouldPaint`. |
| `mobile/lib/core/bridge/game_bridge.dart` | Dart MethodChannel wrapper matching Bridge. |
| `mobile/lib/features/autoupgrade/controller.dart` | HUD state; Start/Stop call `GameBridge`, do not open Dart sockets. |
| `mobile/lib/main.dart` | Auto-boot bundled JAR; no picker. |
| `mobile/ios/Runner/AppDelegate.swift` | MethodChannel + `AVAudioSession` playback/mixWithOthers. |
| `mobile/android/.../MainActivity.kt` | MethodChannel → gomobile (replace stubs). |
| `.github/workflows/build-ios.yml` | `gomobile bind` then unsigned IPA. |

Do **not** import `java-app/pkg/autoupgrade` or `java-app/pkg/jvm` from any of the files above.

---

### Task 1: HSO frame helpers + SocketTap

**Files:**
- Create: `pkg/network/protocol/frame.go`
- Create: `pkg/network/protocol/frame_test.go`
- Create: `pkg/midp/sockettap.go`
- Create: `pkg/midp/sockettap_test.go`
- Modify: `pkg/network/protocol/session.go` (`SendMessage` body calls `WriteMessage`)

**Interfaces:**
- Consumes: `protocol.Crypto`, `protocol.Message`, `net.Conn`
- Produces:
  - `func WriteMessage(w io.Writer, crypto *Crypto, msg *Message) error`
  - `func ReadPlainMessage(r io.Reader) (*Message, error)`
  - `type SocketTap struct`
  - `func NewSocketTap(conn net.Conn) *SocketTap`
  - `func (t *SocketTap) ReadPlain(p []byte) (int, error)`
  - `func (t *SocketTap) WritePlain(p []byte) (int, error)`
  - `func (t *SocketTap) Inject(msg *protocol.Message) error`
  - `func (t *SocketTap) Close() error`
  - `func (t *SocketTap) SetOnPlainMessage(fn func(*protocol.Message))`
  - `func (t *SocketTap) Crypto() *protocol.Crypto`

- [ ] **Step 1: Write the failing tests**

Create `pkg/network/protocol/frame_test.go`:

```go
package protocol

import (
	"bytes"
	"testing"
)

func TestWriteMessagePlainRoundtrip(t *testing.T) {
	msg := NewMessage(10)
	msg.WriteByte(42)
	msg.WriteShort(1234)

	var buf bytes.Buffer
	if err := WriteMessage(&buf, nil, msg); err != nil {
		t.Fatal(err)
	}
	got, err := ReadPlainMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got.Command != 10 {
		t.Fatalf("command=%d", got.Command)
	}
	b, _ := got.ReadByte()
	if b != 42 {
		t.Fatalf("byte=%d", b)
	}
	s, _ := got.ReadShort()
	if s != 1234 {
		t.Fatalf("short=%d", s)
	}
}

func TestWriteMessageLargeOpcodeUses4ByteLength(t *testing.T) {
	msg := NewMessage(-51)
	msg.WriteBytes(make([]byte, 3))
	var buf bytes.Buffer
	if err := WriteMessage(&buf, nil, msg); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	if len(raw) != 1+4+3 {
		t.Fatalf("want 8 byte frame, got %d", len(raw))
	}
}

func TestWriteMessageNilRejected(t *testing.T) {
	if err := WriteMessage(bytes.NewBuffer(nil), nil, nil); err == nil {
		t.Fatal("expected error")
	}
}
```

Create `pkg/midp/sockettap_test.go`:

```go
package midp

import (
	"io"
	"net"
	"testing"
	"time"

	"java-app/pkg/network/protocol"
)

func TestSocketTapHandshakeThenXORInjectSharesSendPointer(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)

	key := []byte{0x11, 0x22, 0x33}
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Handshake: 1 byte length + key (unencrypted).
		if _, err := server.Write(append([]byte{byte(len(key))}, key...)); err != nil {
			t.Errorf("handshake write: %v", err)
		}
		// Read one injected encrypted frame from the client side of the pipe.
		buf := make([]byte, 64)
		_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := server.Read(buf)
		if err != nil {
			t.Errorf("server read: %v", err)
			return
		}
		peer := protocol.NewCrypto(key)
		dec := make([]byte, n)
		for i := 0; i < n; i++ {
			dec[i] = peer.DecryptByte(buf[i])
		}
		if dec[0] != 7 {
			t.Errorf("injected command = %d, want 7", int8(dec[0]))
		}
	}()

	hs := make([]byte, 1+len(key))
	n, err := tap.ReadPlain(hs)
	if err != nil || n != len(hs) {
		t.Fatalf("ReadPlain handshake n=%d err=%v", n, err)
	}
	if tap.Crypto() == nil {
		t.Fatal("expected crypto after handshake bytes")
	}

	msg := protocol.NewMessage(7)
	msg.WriteUTF("Smiley")
	if err := tap.Inject(msg); err != nil {
		t.Fatal(err)
	}
	<-done
}

func TestSocketTapWritePlainAndInjectSerialized(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	tap := NewSocketTap(client)
	key := []byte{0xAA, 0xBB}
	go func() {
		_, _ = server.Write(append([]byte{byte(len(key))}, key...))
	}()
	hs := make([]byte, 1+len(key))
	_, _ = tap.ReadPlain(hs)

	got := make(chan []byte, 2)
	go func() {
		buf := make([]byte, 256)
		for i := 0; i < 2; i++ {
			_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := server.Read(buf)
			if err != nil {
				return
			}
			cp := append([]byte(nil), buf[:n]...)
			got <- cp
		}
	}()

	if _, err := tap.WritePlain([]byte{0x01, 0x00, 0x01, 0xFF}); err != nil {
		t.Fatal(err)
	}
	inj := protocol.NewMessage(2)
	inj.WriteByte(0)
	if err := tap.Inject(inj); err != nil {
		t.Fatal(err)
	}

	a := <-got
	b := <-got
	if len(a) == 0 || len(b) == 0 {
		t.Fatal("empty write")
	}
}

func TestSocketTapCloseReturnsEOF(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	tap := NewSocketTap(client)
	if err := tap.Close(); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8)
	_, err := tap.ReadPlain(buf)
	if err != io.EOF {
		t.Fatalf("ReadPlain after Close: %v", err)
	}
	_, err = tap.WritePlain([]byte{1})
	if err == nil {
		t.Fatal("WritePlain after Close should fail")
	}
	if err := tap.Inject(protocol.NewMessage(1)); err == nil {
		t.Fatal("Inject after Close should fail")
	}
}

func TestSocketTapDoesNotSwapConn(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	tap := NewSocketTap(client)
	if err := tap.Close(); err != nil {
		t.Fatal(err)
	}
	replacement, _ := net.Pipe()
	defer replacement.Close()
	// There is no SetConn API. Replacing the connection requires a new SocketTap.
	other := NewSocketTap(replacement)
	if other == tap {
		t.Fatal("new tap must be a distinct object")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/network/protocol ./pkg/midp -count=1 -run 'TestWriteMessage|TestSocketTap'`

Expected: FAIL with `WriteMessage` / `NewSocketTap` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/network/protocol/frame.go`:

```go
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

func isLargePayloadOpcode(cmd int8) bool {
	return cmd == -51 || cmd == -52 || cmd == -54 || cmd == 126
}

// WriteMessage writes one HSO frame to w. If crypto is non-nil, command,
// length, and payload are XOR-stream encrypted with crypto's send pointer.
func WriteMessage(w io.Writer, crypto *Crypto, msg *Message) error {
	if msg == nil {
		return fmt.Errorf("cannot send nil message")
	}
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	data := msg.GetData()
	size := len(data)
	cmd := msg.Command

	var header []byte
	var payload []byte
	if crypto != nil {
		encCmd := crypto.EncryptByte(byte(cmd))
		if isLargePayloadOpcode(cmd) {
			header = []byte{
				encCmd,
				crypto.EncryptByte(byte(size >> 24)),
				crypto.EncryptByte(byte(size >> 16)),
				crypto.EncryptByte(byte(size >> 8)),
				crypto.EncryptByte(byte(size & 0xFF)),
			}
		} else {
			header = []byte{
				encCmd,
				crypto.EncryptByte(byte(size >> 8)),
				crypto.EncryptByte(byte(size & 0xFF)),
			}
		}
		payload = crypto.Encrypt(data)
	} else {
		if isLargePayloadOpcode(cmd) {
			header = []byte{byte(cmd), byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size & 0xFF)}
		} else {
			header = []byte{byte(cmd), byte(size >> 8), byte(size & 0xFF)}
		}
		payload = data
	}
	packet := append(header, payload...)
	_, err := w.Write(packet)
	return err
}

// ReadPlainMessage reads one unencrypted HSO frame from r.
func ReadPlainMessage(r io.Reader) (*Message, error) {
	var cmdBuf [1]byte
	if _, err := io.ReadFull(r, cmdBuf[:]); err != nil {
		return nil, err
	}
	cmd := int8(cmdBuf[0])
	var payloadLen int
	if isLargePayloadOpcode(cmd) {
		var lenBuf [4]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return nil, err
		}
		payloadLen = int(binary.BigEndian.Uint32(lenBuf[:]))
	} else {
		var lenBuf [2]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return nil, err
		}
		payloadLen = int(binary.BigEndian.Uint16(lenBuf[:]))
	}
	if payloadLen < 0 {
		return nil, fmt.Errorf("negative payload length %d", payloadLen)
	}
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}
	return NewMessageWithData(cmd, payload), nil
}
```

Remove the old unexported `isLargePayloadOpcode` from `session.go` (now in `frame.go`). Change `SendMessage` encrypted/plain write bodies to:

```go
return WriteMessage(conn, crypto, msg)
```

Create `pkg/midp/sockettap.go`:

```go
package midp

import (
	"fmt"
	"io"
	"net"
	"sync"

	"java-app/pkg/network/protocol"
)

// SocketTap owns the JAR's single TCP connection.
type SocketTap struct {
	mu        sync.Mutex
	conn      net.Conn
	crypto    *protocol.Crypto
	closed    bool
	readBuf   []byte
	hsDone    bool
	hsNeed    int
	hsPending []byte
	onPlain   func(*protocol.Message)
	plainAcc  []byte
}

func NewSocketTap(conn net.Conn) *SocketTap {
	return &SocketTap{conn: conn, hsNeed: 1}
}

func (t *SocketTap) Crypto() *protocol.Crypto {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.crypto
}

func (t *SocketTap) SetOnPlainMessage(fn func(*protocol.Message)) {
	t.mu.Lock()
	t.onPlain = fn
	t.mu.Unlock()
}

func (t *SocketTap) ReadPlain(p []byte) (int, error) {
	t.mu.Lock()
	if t.closed || t.conn == nil {
		t.mu.Unlock()
		return 0, io.EOF
	}
	conn := t.conn
	t.mu.Unlock()

	n, err := conn.Read(p)
	if n > 0 {
		t.mu.Lock()
		if !t.hsDone {
			t.ingestHandshakeLocked(p[:n])
		} else if t.crypto != nil {
			plain := t.crypto.Decrypt(p[:n])
			copy(p[:n], plain)
			t.feedPlain(plain)
		}
		t.mu.Unlock()
	}
	return n, err
}

func (t *SocketTap) ingestHandshakeLocked(raw []byte) {
	t.hsPending = append(t.hsPending, raw...)
	if len(t.hsPending) < 1 {
		return
	}
	keyLen := int(t.hsPending[0])
	if keyLen <= 0 || len(t.hsPending) < 1+keyLen {
		return
	}
	key := append([]byte(nil), t.hsPending[1:1+keyLen]...)
	t.crypto = protocol.NewCrypto(key)
	t.hsDone = true
	rest := t.hsPending[1+keyLen:]
	t.hsPending = nil
	if len(rest) > 0 {
		plain := t.crypto.Decrypt(rest)
		t.feedPlain(plain)
	}
}

func (t *SocketTap) feedPlain(plain []byte) {
	t.plainAcc = append(t.plainAcc, plain...)
	for {
		msg, rest, ok := tryReadPlain(t.plainAcc)
		if !ok {
			return
		}
		t.plainAcc = rest
		if t.onPlain != nil {
			t.onPlain(msg)
		}
	}
}

func tryReadPlain(buf []byte) (*protocol.Message, []byte, bool) {
	if len(buf) < 3 {
		return nil, buf, false
	}
	r := bytesReader(buf)
	msg, err := protocol.ReadPlainMessage(r)
	if err != nil {
		return nil, buf, false
	}
	consumed := len(buf) - r.Len()
	return msg, buf[consumed:], true
}

type sliceReader struct {
	b []byte
}

func bytesReader(b []byte) *sliceReader { return &sliceReader{b: b} }
func (s *sliceReader) Read(p []byte) (int, error) {
	if len(s.b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.b)
	s.b = s.b[n:]
	return n, nil
}
func (s *sliceReader) Len() int { return len(s.b) }

func (t *SocketTap) WritePlain(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.conn == nil {
		return 0, fmt.Errorf("socket closed")
	}
	out := p
	if t.crypto != nil {
		out = t.crypto.Encrypt(p)
	}
	return t.conn.Write(out)
}

func (t *SocketTap) Inject(msg *protocol.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.conn == nil {
		return fmt.Errorf("socket closed")
	}
	return protocol.WriteMessage(t.conn, t.crypto, msg)
}

func (t *SocketTap) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
	if t.conn == nil {
		return nil
	}
	err := t.conn.Close()
	t.conn = nil
	return err
}
```

`ingestHandshakeLocked` only looks at raw bytes until `1+keyLen` is complete, then `t.crypto = protocol.NewCrypto(key)` and `t.hsDone = true`. Handshake bytes stay unencrypted for the JAR. After `hsDone`, `ReadPlain` decrypts into `p` (identity XOR in the JAR sees plaintext) and `feedPlain` copies the same plaintext for the inbound parser. Never decrypt the handshake. Never add `SetConn`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/network/protocol ./pkg/midp -count=1 -run 'TestWriteMessage|TestSocketTap'`

Expected: PASS. Also run `go test ./pkg/network/protocol -count=1` so existing session tests still pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/network/protocol/frame.go pkg/network/protocol/frame_test.go pkg/network/protocol/session.go pkg/midp/sockettap.go pkg/midp/sockettap_test.go
git commit -m "feat(midp): add SocketTap with shared HSO frame encode and XOR inject"
```

---

### Task 2: Inbound parser — `in_game`, inventory, stop-condition

**Files:**
- Create: `pkg/upgradebot/parse.go`
- Create: `pkg/upgradebot/parse_test.go`
- Create: `pkg/upgradebot/rules.go`
- Create: `pkg/upgradebot/rules_test.go`

**Interfaces:**
- Consumes: `protocol.Message`
- Produces:
  - `func IsInGameOpcode(cmd int8) bool` — true for `-100`, `24`, `26`
  - `func IsStopCondition(msg string) bool`
  - `func IsIgnorableResult(msg string) bool`
  - `type Snapshot struct { InGame bool; Gold int64; Items []*Item; LastServerMsg string }`
  - `func (s *Snapshot) Apply(msg *protocol.Message)`
  - `type Item struct { ID int16; Name string; Type int8; Level int8; Quantity int; IsClover3, IsClover4 bool }`
  - `type Config struct { TargetLevel, Clover3Min, Clover3Max, Clover4Min, Clover4Max int8; StopIfOutOfClover bool; DelayMs int; ItemID int16; ItemType int8 }`
  - `type Decision struct { CanProceed, UseClover3, UseClover4 bool; Reason string }`
  - `func Evaluate(cfg Config, level int8, snap Snapshot) Decision`

- [ ] **Step 1: Write the failing tests**

Create `pkg/upgradebot/parse_test.go`:

```go
package upgradebot

import (
	"testing"

	"java-app/pkg/network/protocol"
)

func TestIsInGameOpcode(t *testing.T) {
	if !IsInGameOpcode(-100) || !IsInGameOpcode(24) || !IsInGameOpcode(26) {
		t.Fatal("expected -100, 24, 26 to mark in_game")
	}
	if IsInGameOpcode(1) {
		t.Fatal("login opcode is not in_game")
	}
}

func TestSnapshotApplySetsInGame(t *testing.T) {
	var s Snapshot
	s.Apply(protocol.NewMessage(-100))
	if !s.InGame {
		t.Fatal("expected InGame")
	}
}

func TestStopConditionVietnamese(t *testing.T) {
	stops := []string{
		"Không đủ xu", "khong du vang", "không đủ ngọc",
		"thiếu đá", "Hành trang đầy", "hanh trang khong du",
	}
	for _, m := range stops {
		if !IsStopCondition(m) {
			t.Fatalf("expected stop: %q", m)
		}
	}
	if IsStopCondition("Cố gắng hết sức") || IsStopCondition("Thành công") || IsStopCondition("Thất bại") {
		t.Fatal("success/fail dialogue must not stop")
	}
}
```

Create `pkg/upgradebot/rules_test.go`:

```go
package upgradebot

import "testing"

func TestEvaluateCloverRanges(t *testing.T) {
	cfg := Config{TargetLevel: 10, Clover3Min: 4, Clover3Max: 7, Clover4Min: 8, Clover4Max: 15, StopIfOutOfClover: true}
	snap := Snapshot{Items: []*Item{
		{ID: 10, Name: "Co 3 la", Quantity: 2, IsClover3: true},
		{ID: 11, Name: "Co 4 la", Quantity: 1, IsClover4: true},
	}}
	d := Evaluate(cfg, 6, snap)
	if !d.CanProceed || !d.UseClover3 || d.UseClover4 {
		t.Fatalf("%+v", d)
	}
	d = Evaluate(cfg, 8, snap)
	if !d.CanProceed || !d.UseClover4 {
		t.Fatalf("%+v", d)
	}
	d = Evaluate(cfg, 10, snap)
	if d.CanProceed {
		t.Fatal("target reached")
	}
}

func TestEvaluateStopsWhenOutOfClover(t *testing.T) {
	cfg := Config{TargetLevel: 10, Clover3Min: 4, Clover3Max: 7, StopIfOutOfClover: true}
	d := Evaluate(cfg, 5, Snapshot{})
	if d.CanProceed {
		t.Fatal("should stop")
	}
	cfg.StopIfOutOfClover = false
	d = Evaluate(cfg, 5, Snapshot{})
	if !d.CanProceed || d.UseClover3 {
		t.Fatalf("%+v", d)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/upgradebot -count=1`

Expected: FAIL package does not exist / undefined symbols.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/upgradebot/parse.go`:

```go
package upgradebot

import (
	"strings"

	"java-app/pkg/network/protocol"
)

func IsInGameOpcode(cmd int8) bool {
	return cmd == -100 || cmd == 24 || cmd == 26
}

func IsStopCondition(msg string) bool {
	if msg == "" {
		return false
	}
	lower := strings.ToLower(msg)
	if IsIgnorableResult(msg) {
		return false
	}
	needles := []string{
		"không đủ xu", "khong du xu",
		"không đủ vàng", "khong du vang",
		"không đủ tiền", "khong du tien",
		"không đủ ngọc", "khong du ngoc",
		"không đủ đá", "khong du da",
		"thiếu đá", "thieu da",
		"hành trang đầy", "hanh trang day",
		"hành trang không đủ", "hanh trang khong du",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

func IsIgnorableResult(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "cố gắng hết sức") ||
		strings.Contains(lower, "co gang het suc") ||
		strings.Contains(lower, "thành công") ||
		strings.Contains(lower, "thanh cong") ||
		strings.Contains(lower, "thất bại") ||
		strings.Contains(lower, "that bai")
}

type Item struct {
	ID         int16  `json:"id"`
	Name       string `json:"name"`
	Type       int8   `json:"type"`
	Level      int8   `json:"level"`
	Quantity   int    `json:"quantity"`
	IsClover3  bool   `json:"is_clover_3"`
	IsClover4  bool   `json:"is_clover_4"`
}

type Snapshot struct {
	InGame        bool
	Gold          int64
	Items         []*Item
	LastServerMsg string
}

func (s *Snapshot) Apply(msg *protocol.Message) {
	if msg == nil {
		return
	}
	if IsInGameOpcode(msg.Command) {
		s.InGame = true
	}
}

func (s *Snapshot) Clover3() *Item {
	for _, it := range s.Items {
		if it.IsClover3 && it.Quantity > 0 {
			return it
		}
	}
	return nil
}

func (s *Snapshot) Clover4() *Item {
	for _, it := range s.Items {
		if it.IsClover4 && it.Quantity > 0 {
			return it
		}
	}
	return nil
}

func (s *Snapshot) Find(id int16, typ int8) *Item {
	for _, it := range s.Items {
		if it.ID == id && it.Type == typ {
			return it
		}
	}
	return nil
}
```

Create `pkg/upgradebot/rules.go`:

```go
package upgradebot

type Config struct {
	TargetLevel       int8   `json:"target_level"`
	Clover3Min        int8   `json:"clover3_min"`
	Clover3Max        int8   `json:"clover3_max"`
	Clover4Min        int8   `json:"clover4_min"`
	Clover4Max        int8   `json:"clover4_max"`
	StopIfOutOfClover bool   `json:"stop_if_out_of_clover"`
	DelayMs           int    `json:"delay_ms"`
	ItemID            int16  `json:"item_id"`
	ItemType          int8   `json:"item_type"`
}

type Decision struct {
	CanProceed bool
	UseClover3 bool
	UseClover4 bool
	Reason     string
}

func Evaluate(cfg Config, level int8, snap Snapshot) Decision {
	if level >= cfg.TargetLevel {
		return Decision{Reason: "Target level reached"}
	}
	need3 := level >= cfg.Clover3Min && level <= cfg.Clover3Max
	need4 := level >= cfg.Clover4Min && level <= cfg.Clover4Max
	if need3 && snap.Clover3() == nil {
		if cfg.StopIfOutOfClover {
			return Decision{Reason: "Out of Clover 3"}
		}
		return Decision{CanProceed: true, Reason: "Ready to upgrade"}
	}
	if need4 && snap.Clover4() == nil {
		if cfg.StopIfOutOfClover {
			return Decision{Reason: "Out of Clover 4"}
		}
		return Decision{CanProceed: true, Reason: "Ready to upgrade"}
	}
	return Decision{
		CanProceed: true,
		UseClover3: need3 && snap.Clover3() != nil,
		UseClover4: need4 && snap.Clover4() != nil,
		Reason:     "Ready to upgrade",
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/upgradebot -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/upgradebot/
git commit -m "feat(upgradebot): add in_game opcodes, stop-condition text, and clover rules"
```

---

### Task 3: Upgrade action packets + intercept controller

**Files:**
- Create: `pkg/upgradebot/packet.go`
- Create: `pkg/upgradebot/packet_test.go`
- Create: `pkg/upgradebot/controller.go`
- Create: `pkg/upgradebot/controller_test.go`

**Interfaces:**
- Consumes: `SocketTap.Inject`, `Evaluate`, `Snapshot`, `Config`
- Produces:
  - `const ActionSelect byte = 0`
  - `const ActionCurrency byte = 2`
  - `const ActionUpgrade byte = 5`
  - `var ActionCommand int8 = 34` — packet command `q.b` writes; Task 5 may correct this constant after JAR scan, tests use `ActionCommand` not a magic number
  - `func BuildUpgradeAction(action byte, itemID int16, itemType int8) *protocol.Message`
  - `type Injector interface { Inject(msg *protocol.Message) error }`
  - `type Controller struct`
  - `func NewController(inj Injector) *Controller`
  - `func (c *Controller) Start(cfg Config) string` — `""` ok, else error
  - `func (c *Controller) Stop()`
  - `func (c *Controller) OnMessage(msg *protocol.Message)`
  - `func (c *Controller) StatusJSON() string`
  - `func (c *Controller) SetSnapshot(s Snapshot)` — tests
  - `func (c *Controller) Running() bool`

`Start` returns `"not in game"` when `!snapshot.InGame`. On `OnMessage` stop-condition or dead inject error, `Running()` becomes false. After Stop, Start is required again (no auto-resume).

- [ ] **Step 1: Write the failing tests**

Create `pkg/upgradebot/packet_test.go`:

```go
package upgradebot

import "testing"

func TestBuildUpgradeActionLayout(t *testing.T) {
	msg := BuildUpgradeAction(ActionUpgrade, 101, 0)
	if msg.Command != ActionCommand {
		t.Fatalf("cmd=%d", msg.Command)
	}
	a, _ := msg.ReadByte()
	id, _ := msg.ReadShort()
	typ, _ := msg.ReadByte()
	if a != ActionUpgrade || id != 101 || typ != 0 {
		t.Fatalf("payload %d %d %d", a, id, typ)
	}
}
```

Create `pkg/upgradebot/controller_test.go`:

```go
package upgradebot

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"java-app/pkg/network/protocol"
)

type fakeInj struct {
	mu   sync.Mutex
	msgs []*protocol.Message
}

func (f *fakeInj) Inject(msg *protocol.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := protocol.NewMessageWithData(msg.Command, append([]byte(nil), msg.GetData()...))
	f.msgs = append(f.msgs, cp)
	return nil
}

func TestStartRejectedWhenNotInGame(t *testing.T) {
	c := NewController(&fakeInj{})
	err := c.Start(Config{ItemID: 1, TargetLevel: 10, DelayMs: 1})
	if err != "not in game" {
		t.Fatalf("got %q", err)
	}
}

func TestLoopInjectsSelectCurrencyUpgrade(t *testing.T) {
	inj := &fakeInj{}
	c := NewController(inj)
	c.SetSnapshot(Snapshot{
		InGame: true,
		Items: []*Item{
			{ID: 1, Type: 0, Name: "Kiem", Level: 6, Quantity: 1},
			{ID: 9, Type: 7, Name: "Co 3 la", Quantity: 5, IsClover3: true},
		},
	})
	if err := c.Start(Config{
		ItemID: 1, ItemType: 0, TargetLevel: 7,
		Clover3Min: 4, Clover3Max: 7,
		DelayMs: 5, StopIfOutOfClover: true,
	}); err != "" {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		inj.mu.Lock()
		n := len(inj.msgs)
		inj.mu.Unlock()
		if n >= 4 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.Stop()
	inj.mu.Lock()
	defer inj.mu.Unlock()
	if len(inj.msgs) < 4 {
		t.Fatalf("want select+clover+currency+upgrade, got %d", len(inj.msgs))
	}
	want := []byte{ActionSelect, ActionSelect, ActionCurrency, ActionUpgrade}
	for i := 0; i < 4; i++ {
		b, _ := inj.msgs[i].ReadByte()
		if b != want[i] {
			t.Fatalf("step %d action=%d want %d", i, b, want[i])
		}
	}
}

func TestStopConditionStopsRunning(t *testing.T) {
	c := NewController(&fakeInj{})
	c.SetSnapshot(Snapshot{InGame: true, Items: []*Item{{ID: 1, Level: 6, Quantity: 1}}})
	_ = c.Start(Config{ItemID: 1, TargetLevel: 10, DelayMs: 50})
	msg := protocol.NewMessage(25)
	msg.WriteUTF("Không đủ xu")
	c.OnMessage(msg)
	if c.Running() {
		t.Fatal("should stop")
	}
}

func TestStatusJSONShape(t *testing.T) {
	c := NewController(&fakeInj{})
	c.SetSnapshot(Snapshot{InGame: true})
	raw := c.StatusJSON()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"running", "connected", "in_game", "attempts", "success", "fail", "item_level", "logs"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing %s", k)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/upgradebot -count=1`

Expected: FAIL undefined `BuildUpgradeAction` / `NewController`.

- [ ] **Step 3: Write minimal implementation**

Create `pkg/upgradebot/packet.go`:

```go
package upgradebot

import "java-app/pkg/network/protocol"

const (
	ActionSelect   byte = 0
	ActionCurrency byte = 2
	ActionUpgrade  byte = 5
)

// ActionCommand is the HSO packet command written by KnightOnline_402.jar
// method q.b(byte,short,byte). Task 5 overwrites this if the JAR scan
// captures a different command byte; tests must use this identifier.
var ActionCommand int8 = 34

func BuildUpgradeAction(action byte, itemID int16, itemType int8) *protocol.Message {
	msg := protocol.NewMessage(ActionCommand)
	_ = msg.WriteByte(action)
	msg.WriteShort(itemID)
	_ = msg.WriteByte(byte(itemType))
	return msg
}
```

Create `pkg/upgradebot/controller.go` with:

- `Injector` interface as specified
- `Controller` holding `inj Injector`, `mu`, `cfg`, `snap Snapshot`, `running bool`, `connected bool`, `stats attempts/success/fail`, `logs []string` (cap 100, newest first)
- `Start`: if `!snap.InGame` return `"not in game"`; set running; start goroutine
- Loop (delay `max(cfg.DelayMs, 120)` ms between steps): `Evaluate`; inject select item; if UseClover3/4 inject that clover's id/type with ActionSelect; inject currency `(0,0)`; inject ActionUpgrade; increment attempts
- `OnMessage`: `snap.Apply`; if `LastServerMsg` / UTF payload is stop-condition, `Stop`. If item level in snapshot >= target, success++ and `Stop`. Ignore ignorable result strings.
- `Stop`: running=false, append log `"Đã dừng Auto-Upgrade."`
- `StatusJSON`: marshal `running, connected, in_game, attempts, success, fail, item_level, logs`
- `connected` defaults true until `NotifyDisconnected()` sets false and `Stop`s

UTF on opcode 25: `OnMessage` should try `msg.ReadUTF()` into `snap.LastServerMsg` when command is 25 or 37 (same as desktop TCP bridge errors).

Keep the goroutine from injecting when `!running` or `!snap.InGame`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/upgradebot -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/upgradebot/packet.go pkg/upgradebot/packet_test.go pkg/upgradebot/controller.go pkg/upgradebot/controller_test.go
git commit -m "feat(upgradebot): inject q.b actions on SocketTap without a second TCP session"
```

---

### Task 4: Native Connector + socket streams in the interpreter

**Files:**
- Modify: `pkg/vm/interpreter.go` (`handleNativeCall`)
- Modify: `pkg/vm/object.go` only if Extra docs need a comment (prefer no change)
- Create: `pkg/vm/connector_test.go`

**Interfaces:**
- Consumes: `midp.NewSocketTap`, `net.DialTimeout`
- Produces:
  - Native `javax/microedition/io/Connector.open` stores `*midp.SocketTap` in `Object.Extra`
  - `openInputStream` / `openOutputStream` / `openDataInputStream` / `openDataOutputStream` reuse that Extra
  - `InputStream.read` / `DataInputStream.read*` on Extra `*midp.SocketTap` call `ReadPlain`
  - `OutputStream.write` / `DataOutputStream.write*` call `WritePlain`
  - `close` calls `SocketTap.Close`
  - `VMContext.ActiveTap() *midp.SocketTap` — last opened tap (for gamebridge)

Add on `VMContext`:

```go
ActiveSocket *midp.SocketTap
```

Set it in Connector.open. There is no `SetConn`. A later `Connector.open` replaces `ActiveSocket` with a new tap (JAR reconnect).

- [ ] **Step 1: Write the failing test**

Create `pkg/vm/connector_test.go` that uses `net.Pipe` by injecting a test dialer.

Add in `pkg/midp/sockettap.go`:

```go
var Dial = func(host string, port int) (net.Conn, error) {
    return net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 10*time.Second)
}

func OpenSocket(host string, port int) (*SocketTap, error) {
    conn, err := Dial(host, port)
    if err != nil {
        return nil, err
    }
    if tc, ok := conn.(*net.TCPConn); ok {
        _ = tc.SetNoDelay(true)
        _ = tc.SetKeepAlive(true)
    }
    return NewSocketTap(conn), nil
}
```

Test:

```go
package vm

import (
	"net"
	"testing"
	"java-app/pkg/classfile"
	"java-app/pkg/midp"
)

func TestConnectorOpenStoresSocketTap(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	orig := midp.Dial
	midp.Dial = func(host string, port int) (net.Conn, error) {
		if host != "hs1.teamobi.com" || port != 19129 {
			t.Fatalf("dial %s:%d", host, port)
		}
		return client, nil
	}
	t.Cleanup(func() { midp.Dial = orig })

	ctx := NewVMContext(classfile.NewClassLoader(nil), midp.NewGraphics(240, 320))
	frame := &Frame{OperandStack: NewOperandStack(8)}
	frame.OperandStack.PushRef("socket://hs1.teamobi.com:19129")
	ctx.HandleNativeCall("javax/microedition/io/Connector", "open", "(Ljava/lang/String;)Ljavax/microedition/io/Connection;", frame, OpInvokestatic)
	ref := frame.OperandStack.PopRef()
	obj, ok := ref.(*Object)
	if !ok || obj == nil {
		t.Fatal("expected connection object")
	}
	tap, ok := obj.Extra.(*midp.SocketTap)
	if !ok || tap == nil {
		t.Fatal("expected SocketTap extra")
	}
	if ctx.ActiveSocket != tap {
		t.Fatal("ActiveSocket not set")
	}
}
```

`NewFrame` / `OpInvokestatic` must match existing `pkg/vm` names. If `NewFrame` does not exist, use the same frame constructor as `interpreter_test.go`.

Read `pkg/vm/frame.go` and copy its constructor (`NewFrame(maxLocals, maxStack int)` or similar) — do not invent a different name.

If `handleNativeCall` is not exported, export it as `HandleNativeCall` **or** test via `InvokeMethod` on a tiny synthetic class. Prefer exporting `HandleNativeCall` as a wrapper:

```go
func (ctx *VMContext) HandleNativeCall(clsName, mName, mDesc string, frame *Frame, opcode byte) {
    ctx.handleNativeCall(clsName, mName, mDesc, frame, opcode)
}
```

Use `HandleNativeCall` in the test.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/vm -count=1 -run TestConnectorOpenStoresSocketTap`

Expected: FAIL (tap not stored / Connector branch missing).

- [ ] **Step 3: Write minimal implementation**

In `handleNativeCall`, before the generic native fallback, add:

```go
} else if clsName == "javax/microedition/io/Connector" {
    if mName == "open" {
        // pop extra ints if descriptor has mode/timeouts
        if strings.Contains(mDesc, "I") {
            frame.OperandStack.PopInt()
        }
        if strings.Contains(mDesc, "Z") {
            frame.OperandStack.PopInt()
        }
        nameRef := frame.OperandStack.PopRef()
        name := extractString(nameRef)
        host, port, err := parseSocketURL(name) // "socket://host:port"
        if err != nil {
            frame.OperandStack.PushRef(nil)
            return
        }
        tap, err := midp.OpenSocket(host, port)
        if err != nil {
            log.Printf("[CONNECTOR] open %s: %v", name, err)
            frame.OperandStack.PushRef(nil)
            return
        }
        ctx.ActiveSocket = tap
        frame.OperandStack.PushRef(&Object{Extra: tap})
        return
    }
}
```

For `openInputStream` / `openOutputStream` on classes containing `SocketConnection` / `OutputConnection` / `InputConnection`: pop receiver, push `&Object{Extra: tap}` if receiver Extra is `*midp.SocketTap`.

Extend the existing `java/io/InputStream` / `DataInputStream` branch: if Extra is `*midp.SocketTap`, `read()` one byte via `ReadPlain`. `read([BII)` loop `ReadPlain`.

Add `java/io/OutputStream` / `DataOutputStream` branch: `write(I)` / `write([BII)` → `WritePlain`.

`close` on Extra `*midp.SocketTap` → `Close()` and `ctx.ActiveSocket = nil` if it was that tap.

`parseSocketURL` in `pkg/vm/connector.go` (new small file): trim `socket://`, split host/port.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/vm -count=1 -run 'TestConnector|TestVM'`

Expected: new test PASS; existing VM tests still PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/vm/interpreter.go pkg/vm/connector.go pkg/vm/connector_test.go pkg/midp/sockettap.go
git commit -m "feat(vm): native Connector.open binds SocketTap for Knight Online TCP"
```

---

### Task 5: Knight Online XOR identity stubs

**Files:**
- Create: `pkg/vm/xorstub.go`
- Create: `pkg/vm/xorstub_test.go`
- Modify: `pkg/vm/interpreter.go` (`handleNativeCall` first-line check)

**Interfaces:**
- Consumes: `mobile/assets/KnightOnline_402.jar` (read-only)
- Produces:
  - `func IsIdentityXOR(clsName, mName, mDesc string) bool`
  - `var IdentityXORMethods []string` filled by scan + hardcoded fallbacks
  - `func ScanXORMethods(jarBytes []byte) []string`

Identity semantics: native method returns its last integer argument unchanged (the byte being XOR'd) and does not touch SocketTap crypto.

- [ ] **Step 1: Write the failing tests**

Create `pkg/vm/xorstub_test.go`:

```go
package vm

import (
	"os"
	"testing"
)

func TestScanXORMethodsFindsAtLeastOne(t *testing.T) {
	b, err := os.ReadFile("../../mobile/assets/KnightOnline_402.jar")
	if err != nil {
		t.Skip(err)
	}
	found := ScanXORMethods(b)
	if len(found) == 0 {
		t.Fatal("expected at least one XOR stream-cipher method in KnightOnline_402.jar")
	}
}

func TestIdentityXORReturnsInputByte(t *testing.T) {
	if !IsIdentityXOR("synthetic/Xor", "a", "(B)B") && len(IdentityXORMethods) == 0 {
		// After scan, register. Direct unit: register then check.
	}
	RegisterIdentityXOR("synthetic/Xor.a:(B)B")
	if !IsIdentityXOR("synthetic/Xor", "a", "(B)B") {
		t.Fatal("expected registration")
	}
}
```

`ScanXORMethods` algorithm (deterministic, no TBD):

1. `jar.ReadJar`
2. For each `.class`, `classfile.Parse`
3. For each method with `Code` whose descriptor is `(B)B` or `(I)I` or `(B)I`
4. If bytecode contains `ixor` (`0x82`) and at least one `baload`/`iaload` and `getfield`/`getstatic` — record `ClassName.method:desc`

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/vm -count=1 -run 'TestScanXOR|TestIdentityXOR'`

Expected: FAIL undefined.

- [ ] **Step 3: Write minimal implementation**

`xorstub.go`: mutex-protected set of keys `cls+"."+name+":"+desc`. `ScanXORMethods` as specified. On `gamebridge.LoadJAR`, call `ScanXORMethods` and `RegisterIdentityXOR` for each.

In `handleNativeCall` at the top:

```go
if IsIdentityXOR(clsName, mName, mDesc) {
    // (B)B or (I)I: return the byte argument unchanged.
    if strings.HasPrefix(mDesc, "(B") || strings.HasPrefix(mDesc, "(I") {
        v := frame.OperandStack.PopInt()
        if opcode != OpInvokestatic {
            frame.OperandStack.PopRef()
        }
        if hasReturnValue(mDesc) {
            frame.OperandStack.PushInt(v)
        }
        return
    }
}
```

After scan, if `ActionCommand` needs updating: only change `upgradebot.ActionCommand` when a captured `q.b` frame is documented in `pkg/upgradebot/packet.go` comment with the class name `q` method `b`. Default stays `34` unless scan notes a different command in a comment + assignment in `gamebridge.LoadJAR` is **not** allowed to silently change packet layout. If JAR scan cannot prove the command byte, leave `ActionCommand = 34` and log `[xorstub] methods=%v`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/vm -count=1 -run 'TestScanXOR|TestIdentityXOR'`

Expected: PASS (`Skip` only if the asset JAR is missing).

- [ ] **Step 5: Commit**

```bash
git add pkg/vm/xorstub.go pkg/vm/xorstub_test.go pkg/vm/interpreter.go
git commit -m "feat(vm): identity-stub Knight Online XOR so SocketTap owns the cipher"
```

---

### Task 6: `gamebridge` gomobile API

**Files:**
- Modify: `pkg/gamebridge/bridge.go`
- Modify: `pkg/gamebridge/bridge_test.go`

**Interfaces:**
- Consumes: `upgradebot.Controller`, `VMContext.ActiveSocket`, existing `LoadJAR` / framebuffer / touch
- Produces on `Bridge` (gomobile-safe primitives only):
  - `func (b *Bridge) SendTouchDrag(x, y int)`
  - `func (b *Bridge) SetShouldPaint(paint bool)`
  - `func (b *Bridge) ShouldPaint() bool`
  - `func (b *Bridge) StartUpgrade(configJSON string) string`
  - `func (b *Bridge) StopUpgrade()`
  - `func (b *Bridge) GetUpgradeStatus() string`
  - existing LoadJAR, GetFrameBuffer, GetWidth, GetHeight, SendTouchDown, SendTouchUp, Stop, IsStarted, Tick

`GetFrameBuffer` when `!shouldPaint`: still return the last copy (Flutter must not rasterize). Tick / VM loop always runs.

`LoadJAR` after `startApp`: `ScanXORMethods` + register; construct `upgradebot.NewController(tapInjector{b})` lazily on first `StartUpgrade` using `b.vmCtx.ActiveSocket`.

`tapInjector` implements `upgradebot.Injector` by calling `ActiveSocket.Inject`. If tap is nil, `StartUpgrade` returns `"not connected"`.

Wire `ActiveSocket.SetOnPlainMessage` once when the tap appears (poll in `runLoop`: if `ActiveSocket != nil && not wired`, set callback → `bot.OnMessage` + snapshot in_game).

- [ ] **Step 1: Write the failing tests**

Append to `pkg/gamebridge/bridge_test.go`:

```go
func TestStartUpgradeNotInGame(t *testing.T) {
	b := gamebridge.NewBridge()
	got := b.StartUpgrade(`{"item_id":1,"item_type":0,"target_level":10,"clover3_min":4,"clover3_max":7,"clover4_min":8,"clover4_max":15,"stop_if_out_of_clover":true,"delay_ms":500}`)
	if got == "" {
		t.Fatal("expected error before in_game")
	}
}

func TestSetShouldPaint(t *testing.T) {
	b := gamebridge.NewBridge()
	if !b.ShouldPaint() {
		t.Fatal("default shouldPaint true")
	}
	b.SetShouldPaint(false)
	if b.ShouldPaint() {
		t.Fatal("expected false")
	}
	buf := b.GetFrameBuffer()
	if len(buf) != 240*320*4 {
		t.Fatalf("len=%d", len(buf))
	}
}

func TestSendTouchDragDoesNotPanic(t *testing.T) {
	b := gamebridge.NewBridge()
	b.SendTouchDrag(10, 20)
}

func TestGetUpgradeStatusJSON(t *testing.T) {
	b := gamebridge.NewBridge()
	raw := b.GetUpgradeStatus()
	if raw == "" {
		t.Fatal("expected json")
	}
}
```

Extend `dispatchTouch` to accept drag: add `dragged bool` or a third method name `pointerDragged`. `SendTouchDrag` enqueues `{x,y, dragged:true}`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/gamebridge -count=1`

Expected: FAIL missing methods.

- [ ] **Step 3: Write minimal implementation**

On `Bridge`:

```go
shouldPaint bool // default true in NewBridge
bot *upgradebot.Controller
```

`StartUpgrade` unmarshals JSON into `upgradebot.Config` (field names from spec: `item_id`, `item_type`, `target_level`, `clover3_min`, `clover3_max`, `clover4_min`, `clover4_max`, `stop_if_out_of_clover`, `delay_ms`). Returns `bot.Start(cfg)`.

`StopUpgrade` → `bot.Stop()`.

`GetUpgradeStatus` → `bot.StatusJSON()` or `{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}` if bot nil.

In `dispatchTouch`, if method `pointerDragged` missing on MIDlet, no-op (same as missing `pointerPressed`).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/gamebridge ./pkg/upgradebot ./pkg/midp ./pkg/vm -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/gamebridge/bridge.go pkg/gamebridge/bridge_test.go
git commit -m "feat(gamebridge): expose upgrade, skip-paint, and pointerDragged to gomobile"
```

---

### Task 7: Flutter `GameBridge` + HUD talks to native only

**Files:**
- Modify: `mobile/lib/core/bridge/game_bridge.dart`
- Create: `mobile/test/game_bridge_test.dart`
- Modify: `mobile/lib/features/autoupgrade/controller.dart`
- Modify: `mobile/test/autoupgrade_test.dart`

**Interfaces:**
- Consumes: MethodChannel `com.example.knight_online/game_bridge`
- Produces Dart:
  - `Future<String> startUpgrade(String configJSON)`
  - `Future<void> stopUpgrade()`
  - `Future<String> getUpgradeStatus()`
  - `Future<void> setShouldPaint(bool paint)`
  - `Future<void> sendTouchDrag(int x, int y)`
  - existing loadJAR / getFrameBuffer / sendTouchDown / sendTouchUp / stop / isStarted
  - `AutoUpgradeController.start` calls `GameBridge.startUpgrade` with the spec JSON keys; does not construct `HsoSession`

- [ ] **Step 1: Write the failing tests**

Create `mobile/test/game_bridge_test.dart`:

```dart
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/bridge/game_bridge.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.example.knight_online/game_bridge');
  final log = <MethodCall>[];

  setUp(() {
    log.clear();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
      log.add(call);
      switch (call.method) {
        case 'startUpgrade':
          return 'not in game';
        case 'getUpgradeStatus':
          return '{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}';
        case 'isStarted':
          return false;
        default:
          return null;
      }
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('startUpgrade sends spec JSON', () async {
    final bridge = GameBridge();
    final err = await bridge.startUpgrade(
      '{"item_id":1,"item_type":0,"target_level":10,"clover3_min":4,"clover3_max":7,"clover4_min":8,"clover4_max":15,"stop_if_out_of_clover":true,"delay_ms":500}',
    );
    expect(err, 'not in game');
    expect(log.single.method, 'startUpgrade');
  });

  test('setShouldPaint and sendTouchDrag', () async {
    final bridge = GameBridge();
    await bridge.setShouldPaint(false);
    await bridge.sendTouchDrag(12, 34);
    expect(log[0].method, 'setShouldPaint');
    expect(log[1].method, 'sendTouchDrag');
  });
}
```

Update `mobile/test/autoupgrade_test.dart` `AutoUpgradeController transitions` so `start` with a fake `GameBridge` records a `startUpgrade` call. Add an optional `GameBridge? bridge` constructor argument (existing tests that do not pass a bridge keep local state only for clover unit tests on `AutoUpgradeConfig`).

When `bridge != null`, `start` must `await`/`unawaited` `bridge.startUpgrade(...)` and set `_isRunning` only if the returned string is empty. If non-empty, append that string to `logs` and keep `_isRunning == false`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test test/game_bridge_test.dart`

Expected: FAIL missing methods.

- [ ] **Step 3: Write minimal implementation**

Add methods to `GameBridge` matching the tests. `AutoUpgradeController`:

```dart
AutoUpgradeController({GameBridge? bridge}) : _bridge = bridge;
final GameBridge? _bridge;

Future<void> start({required EquipmentItem targetItem, required AutoUpgradeConfig config}) async {
  _targetItem = targetItem;
  _config = config;
  final json = '{"item_id":${targetItem.id},"item_type":${targetItem.type},"target_level":${config.targetLevel},"clover3_min":${config.clover3Min},"clover3_max":${config.clover3Max},"clover4_min":${config.clover4Min},"clover4_max":${config.clover4Max},"stop_if_out_of_clover":${config.stopIfOutOfClover},"delay_ms":${config.delayMs}}';
  if (_bridge != null) {
    final err = await _bridge!.startUpgrade(json);
    if (err.isNotEmpty) {
      _isRunning = false;
      _addLog(err);
      notifyListeners if ChangeNotifier else log only;
      return;
    }
  }
  _isRunning = true;
  stats.reset();
  _addLog('Bắt đầu Auto-Upgrade cho [${targetItem.name}] lên cấp +${config.targetLevel}');
}

void stop() {
  _isRunning = false;
  _bridge?.stopUpgrade();
  _addLog('Đã dừng Auto-Upgrade.');
}
```

Do **not** import `core/network/session.dart`. Leave those Dart files on disk but unused.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test test/game_bridge_test.dart test/autoupgrade_test.dart`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/core/bridge/game_bridge.dart mobile/lib/features/autoupgrade/controller.dart mobile/test/game_bridge_test.dart mobile/test/autoupgrade_test.dart
git commit -m "feat(mobile): HUD start/stop goes through GameBridge, not Dart HSO sockets"
```

---

### Task 8: Boot bundled JAR, drop picker, skip-paint

**Files:**
- Modify: `mobile/lib/main.dart`
- Modify: `mobile/test/app_scaffold_test.dart`
- Modify: `mobile/lib/presentation/game_canvas.dart` (skip paint when `!shouldPaint`)
- Modify: `mobile/test/game_loop_test.dart` (already covers skip-paint; add bridge call if needed)
- Modify: `mobile/pubspec.yaml` (remove `file_picker` if unused)

**Interfaces:**
- Consumes: `GameBridge.loadJAR`, `AppLifecycleManager.shouldPaint`, `GameBridge.setShouldPaint`
- Produces: Home screen is `GameScreen` (canvas). No `JarPickerScreen` on the play path. Launch loads `assets/KnightOnline_402.jar`.

- [ ] **Step 1: Write the failing test**

Replace picker expectations in `mobile/test/app_scaffold_test.dart`:

```dart
testWidgets('KnightApp initializes game canvas shell without JAR picker', (tester) async {
  await tester.pumpWidget(const KnightApp());
  expect(find.byType(KnightApp), findsOneWidget);
  expect(find.byType(JarPickerScreen), findsNothing);
  expect(find.byType(FloatingHudButton), findsOneWidget);
});
```

Remove the test `'JarPickerScreen shows Start Game and Pick JAR buttons'`.

Add in `mobile/test/game_loop_test.dart` (or a new `lifecycle_bridge_test.dart`):

```dart
test('paused lifecycle requests skip-paint on GameBridge', () async {
  // Use a fake GameBridge MethodChannel as in Task 7.
  // AppLifecycleManager.updateState(AppLifecycleState.paused)
  // must invoke setShouldPaint(false).
});
```

Wire this in `GameHomeScreen`: listen to `_lifecycleManager.addListener((s) { _bridge.setShouldPaint(s == AppLifecycleState.resumed); })`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test test/app_scaffold_test.dart`

Expected: FAIL `JarPickerScreen` still found.

- [ ] **Step 3: Write minimal implementation**

In `_GameHomeScreenState.initState`, after constructing `_jarController`:

```dart
rootBundle.load('assets/KnightOnline_402.jar').then((data) {
  _jarController.loadJARBytes(data.buffer.asUint8List());
});
_lifecycleManager.addListener((state) {
  _jarController; // keep-alive already
  // need GameBridge instance: add GameBridge? to JarRunnerController or hold one on the screen.
});
```

Hold `final GameBridge _bridge = GameBridge();` on the screen (or inject). Pass it into `JarRunnerController(bridge: _bridge)` and `AutoUpgradeController(bridge: _bridge)`.

`build`: `Positioned.fill(child: GameScreen(controller: _jarController))` instead of `JarPickerScreen`.

`GameCanvasWidget` / `GameScreen`: if lifecycle `!shouldPaint`, `CustomPaint` painter returns immediately (empty `paint`).

`JarRunnerController.onTouchMove` → `sendTouchDrag` (add method mirroring down/up).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test`

Expected: ALL PASS. Existing picker tests removed or updated.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/main.dart mobile/lib/presentation mobile/lib/features/jar_runner mobile/test mobile/pubspec.yaml mobile/pubspec.lock
git commit -m "feat(mobile): auto-boot KnightOnline_402.jar and skip-paint when backgrounded"
```

---

### Task 9: iOS and Android MethodChannel → real Bridge

**Files:**
- Modify: `mobile/ios/Runner/AppDelegate.swift`
- Modify: `mobile/android/app/src/main/kotlin/com/example/knight_online/MainActivity.kt`
- Modify: `mobile/ios/Runner/Info.plist` (confirm `UIBackgroundModes` contains `audio`, `NSAllowsArbitraryLoads` true)

**Interfaces:**
- Consumes: gomobile `GamebridgeBridge` (generated name from package `gamebridge`)
- Produces: every Dart method in Task 7 implemented on both platforms. Stubs only allowed when the `.xcframework` / `.aar` is absent **and** `loadJAR` then returns `"native bridge not linked"`.

Linux cannot compile Swift. Tests: keep Kotlin/Swift handlers total and matching names. Add a tiny `mobile/test/channel_methods_test.dart` listing required method strings: `loadJAR`, `getFrameBuffer`, `sendTouchDown`, `sendTouchUp`, `sendTouchDrag`, `stop`, `isStarted`, `setShouldPaint`, `startUpgrade`, `stopUpgrade`, `getUpgradeStatus`.

- [ ] **Step 1: Write the failing test**

```dart
// mobile/test/channel_methods_test.dart
import 'dart:io';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('iOS AppDelegate and Android MainActivity mention every channel method', () {
    const methods = [
      'loadJAR', 'getFrameBuffer', 'sendTouchDown', 'sendTouchUp',
      'sendTouchDrag', 'stop', 'isStarted', 'setShouldPaint',
      'startUpgrade', 'stopUpgrade', 'getUpgradeStatus',
    ];
    final swift = File('ios/Runner/AppDelegate.swift').readAsStringSync();
    final kt = File('android/app/src/main/kotlin/com/example/knight_online/MainActivity.kt').readAsStringSync();
    for (final m in methods) {
      expect(swift.contains('"$m"') || swift.contains("\"$m\""), isTrue, reason: 'swift $m');
      expect(kt.contains('"$m"'), isTrue, reason: 'kotlin $m');
    }
  });
}
```

Run from `mobile/` so relative `ios/` `android/` work.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test test/channel_methods_test.dart`

Expected: FAIL missing method strings.

- [ ] **Step 3: Write minimal implementation**

`AppDelegate.swift` — in `didInitializeImplicitFlutterEngine` (or `application:didFinishLaunching` with `window.rootViewController` messenger):

```swift
import AVFoundation
import gamebridge // gomobile module name; if bind uses KnightBridge, import that

let controller = window?.rootViewController as? FlutterViewController
let channel = FlutterMethodChannel(name: "com.example.knight_online/game_bridge",
                                   binaryMessenger: engineBridge.pluginRegistry.registrar(forPlugin: "GameBridge")!.messenger())
let bridge = GamebridgeBridge() // exact generated class from gomobile bind

try? AVAudioSession.sharedInstance().setCategory(.playback, options: [.mixWithOthers])
try? AVAudioSession.sharedInstance().setActive(true)

channel.setMethodCallHandler { call, result in
  switch call.method {
  case "loadJAR":
    let bytes = call.arguments as? FlutterStandardTypedData
    result(bridge.loadJAR(bytes?.data))
  case "getFrameBuffer":
    result(bridge.getFrameBuffer())
  case "sendTouchDown":
    let args = call.arguments as? [String: Any]
    bridge.sendTouchDown(args?["x"] as? Int ?? 0, y: args?["y"] as? Int ?? 0)
    result(nil)
  case "sendTouchUp":
    let args = call.arguments as? [String: Any]
    bridge.sendTouchUp(args?["x"] as? Int ?? 0, y: args?["y"] as? Int ?? 0)
    result(nil)
  case "sendTouchDrag":
    let args = call.arguments as? [String: Any]
    bridge.sendTouchDrag(args?["x"] as? Int ?? 0, y: args?["y"] as? Int ?? 0)
    result(nil)
  case "setShouldPaint":
    bridge.setShouldPaint(call.arguments as? Bool ?? true)
    result(nil)
  case "startUpgrade":
    result(bridge.startUpgrade(call.arguments as? String))
  case "stopUpgrade":
    bridge.stopUpgrade()
    result(nil)
  case "getUpgradeStatus":
    result(bridge.getUpgradeStatus())
  case "stop":
    bridge.stop()
    result(nil)
  case "isStarted":
    result(bridge.isStarted())
  default:
    result(FlutterMethodNotImplemented)
  }
}
```

If the engine messenger API in this Flutter version differs, get `binaryMessenger` from `engineBridge.applicationRegistrar.messenger()` — same channel name `com.example.knight_online/game_bridge`.

Until xcframework exists, compile-guard:

```swift
#if canImport(gamebridge)
// real calls
#else
result("native bridge not linked")
#endif
```

The string-presence test still requires `"startUpgrade"` etc. in the file — put them in the `switch` even inside `#else` returning `"native bridge not linked"`.

`MainActivity.kt`: extend `when` with `sendTouchDrag`, `setShouldPaint`, `startUpgrade`, `stopUpgrade`, `getUpgradeStatus`. Replace stub `nativeLoadJAR` comment with gomobile `Bridge` when AAR present; otherwise `return "native bridge not linked"` for empty-link builds **except** keep the checkerboard stub only if `BuildConfig.DEBUG && !nativeLinked` — spec DoD on Android emulator needs the real VM. Prefer calling `go.Bridge` from the AAR; if the AAR is missing, `loadJAR` returns `"native bridge not linked"` (no checkerboard success). Remove the checkerboard success path.

Info.plist keys (must exist):

```xml
<key>UIBackgroundModes</key>
<array><string>audio</string></array>
<key>NSAppTransportSecurity</key>
<dict><key>NSAllowsArbitraryLoads</key><true/></dict>
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test test/channel_methods_test.dart`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/ios/Runner/AppDelegate.swift mobile/ios/Runner/Info.plist mobile/android/app/src/main/kotlin/com/example/knight_online/MainActivity.kt mobile/test/channel_methods_test.dart
git commit -m "feat(mobile): wire iOS and Android MethodChannel methods to gamebridge"
```

---

### Task 10: CI gomobile bind + unsigned IPA

**Files:**
- Modify: `/home/neuro/Documents/Tools/.github/workflows/build-ios.yml`
- Create: `mobile/scripts/bind_native.sh`
- Modify: `mobile/ios/Runner.xcodeproj/project.pbxproj` only if a Frameworks path must be added; prefer dropping `KnightBridge.xcframework` into `mobile/ios/Frameworks/` from the script

**Interfaces:**
- Consumes: `pkg/gamebridge` (CGO-free)
- Produces: workflow installs Go + gomobile, binds iOS xcframework, `flutter test`, `flutter build ios --release --no-codesign`, `scripts/package_ipa.sh` artifact `KnightOnline-iOS`

- [ ] **Step 1: Write the failing test**

Create `mobile/scripts/bind_native.sh`:

The test is a shell dry-run check committed as `mobile/test/bind_native_script_test.dart` or simply grep in a test:

```dart
test('CI workflow runs gomobile bind before flutter build', () {
  final yml = File('../../.github/workflows/build-ios.yml').readAsStringSync();
  expect(yml.contains('gomobile bind'), isTrue);
  expect(yml.contains('bind_native.sh'), isTrue);
});
```

Place the test in `mobile/test/ci_workflow_test.dart` with path from `mobile/`: repo root is `../..` because git root is `/home/neuro/Documents/Tools`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test test/ci_workflow_test.dart`

Expected: FAIL `gomobile bind` not in yml.

- [ ] **Step 3: Write minimal implementation**

Create `mobile/scripts/bind_native.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
export PATH="$PATH:$(go env GOPATH)/bin"
gomobile init || true
mkdir -p mobile/ios/Frameworks
gomobile bind -target=ios -o mobile/ios/Frameworks/KnightBridge.xcframework ./pkg/gamebridge
```

Update `.github/workflows/build-ios.yml` after checkout:

```yaml
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24.x'
      - name: Bind gamebridge xcframework
        working-directory: java-app
        run: bash mobile/scripts/bind_native.sh
      - name: Run Go tests
        working-directory: java-app
        run: go test ./pkg/midp ./pkg/upgradebot ./pkg/gamebridge ./pkg/network/protocol ./pkg/vm -count=1
```

Keep existing Flutter test + `flutter build ios --release --no-codesign` + `package_ipa.sh` + upload artifact.

Do not add CGO env (`CGO_ENABLED=1`) and do not install a JDK for the bind step beyond what Flutter already needs.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/neuro/Documents/Tools/java-app/mobile && flutter test test/ci_workflow_test.dart && flutter test`

Run: `cd /home/neuro/Documents/Tools/java-app && go test ./pkg/midp ./pkg/upgradebot ./pkg/gamebridge ./pkg/network/protocol ./pkg/vm -count=1`

Expected: PASS. `gomobile bind` itself runs on the macOS CI runner, not on Linux in this task.

- [ ] **Step 5: Commit**

```bash
git add mobile/scripts/bind_native.sh mobile/test/ci_workflow_test.dart .github/workflows/build-ios.yml
git commit -m "ci: gomobile-bind gamebridge before unsigned iOS IPA package"
```

Note: workflow file lives at repo root `/home/neuro/Documents/Tools/.github/workflows/build-ios.yml`. Stage it from the git root.

---

## Self-review (spec coverage)

| Spec section | Task |
|--------------|------|
| Boot bundled JAR, MIDlet fallback | 8, existing `LoadJAR` |
| 240×320 canvas, pointer down/drag/up, no D-pad | 6, 7, 8 |
| Silent audio AFK + skip-paint | existing keepalive + 6 `SetShouldPaint` + 8 |
| HUD auto-upgrade on JAR socket | 3, 6, 7 |
| Sideload IPA + gomobile | 10 |
| One socket, no Dart HSO client | 1, 3, 7 |
| XOR owned by Go, identity stubs | 1, 5 |
| Dead socket EOF, no conn swap | 1 `Close` / no `SetConn` |
| `in_game` opcodes -100/24/26 | 2 |
| Stop-condition Vietnamese strings | 2, 3 |
| CGO-free gamebridge graph | 3 `pkg/upgradebot` (not `pkg/autoupgrade`) |
| MethodChannel iOS/Android | 9 |
| Tests on Linux | 1–8, 10 script test |

No TBD/TODO left in tasks. Types: `Config` JSON keys match spec and Dart `startUpgrade` payload. `GetUpgradeStatus` keys match spec (`running`, `connected`, `in_game`, `attempts`, `success`, `fail`, `item_level`, `logs`).
