# Knight Online iOS — Run `KnightOnline_402.jar` via Go VM

**Date:** 2026-09-04  
**Status:** Draft pending user review  
**Supersedes:** `docs/superpowers/specs/2026-09-04-knight-online-ios-flutter-design.md` (Dart HSO client rewrite)  
**Target platform:** iOS personal sideload (Sideloadly / AltStore / TrollStore)  
**Dev platform:** Linux/Windows + Android emulator; GitHub Actions `macos-latest` for IPA  

---

## 1. Executive summary

The iOS app **executes** TeaMobi Knight Online (`KnightOnline_402.jar`) on a **pure-Go CLDC/MIDP interpreter** (`pkg/vm`). Flutter is the shell: 240×320 canvas, direct touch, auto-upgrade HUD, and iOS background keep-alive.

This is **not** a Dart rewrite of the HSO protocol, **not** HotSpot/JNI (`j2mego` cannot run on iOS — no JIT), and **not** a generic multi-JAR emulator. One game, optimized until it is stable.

Auto-upgrade rides the **same TCP socket** the JAR opens. A second login session is forbidden (the server kicks the character).

---

## 2. Goals and non-goals

### 2.1 Goals

1. **Boot the real JAR on iOS.** Flutter loads `mobile/assets/KnightOnline_402.jar` and calls `gamebridge.Bridge.LoadJAR`. The MIDlet entry is `META-INF/MANIFEST.MF` `MIDlet-1`, fallback `com/silverknight/TemMidlet`.
2. **Playable LCDUI.** 240×320 RGBA frames, aspect-fit, `pointerPressed` / `pointerDragged` / `pointerReleased` mapped through `ViewportScaler`. No on-screen D-pad.
3. **Treo máy (AFK).** `UIBackgroundModes: audio` + silent looping asset. Tick and socket keep running. GPU paint skipped while `AppLifecycleState` is not `resumed`.
4. **Auto-upgrade HUD.** Floating button → bottom sheet: target item, target level, 3-leaf / 4-leaf clover ranges, stop-if-out-of-clover, delay, live stats and log. Logic runs in Go on the game socket.
5. **Sideload IPA.** GitHub Actions unsigned `flutter build ipa --no-codesign`.

### 2.2 Non-goals (v1)

- Apple App Store.
- Arbitrary `.jar` picker / generic J2ME compatibility.
- Dart `HsoSession` as the game client (delete from the play path).
- HotSpot JVM, CGO JNI, or `AutoUpgrade.java` reflection on iOS.
- On-screen keypad overlay.

---

## 3. Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ Flutter (iOS / Android)                                      │
│  Canvas 240×320 · touch · HUD · silent audio · skip-paint    │
│  Auto-boots assets/KnightOnline_402.jar                      │
└────────────────────────────┬─────────────────────────────────┘
                             │ MethodChannel
                             │ com.example.knight_online/game_bridge
                             ▼
┌──────────────────────────────────────────────────────────────┐
│ gomobile xcframework  (pkg/gamebridge)                       │
│  LoadJAR · GetFrameBuffer · Touch · Start/StopUpgrade · Stop │
│                                                              │
│  pkg/jar + pkg/classfile + pkg/vm + pkg/midp                 │
│  Connector.open("socket://…") → SocketTap (one net.Conn)     │
│       write queue ──XOR──► hs1.teamobi.com:19129             │
│       read  copy ──XOR──► HSO parser → Inventory / results   │
│       inject ◄── AutoUpgrade controller (Go)                 │
└──────────────────────────────────────────────────────────────┘
```

| Layer | Does | Does not |
|-------|------|----------|
| Flutter | Paint frames, map touch, HUD, iOS audio/wakelock | Game TCP, HSO crypto, bytecode |
| `gamebridge` | gomobile API, tick loop, upgrade commands | UI widgets |
| `pkg/vm` + MIDP | Execute the JAR, LCDUI, RMS, native Connector | Flutter |
| `SocketTap` | Own the only `net.Conn`, XOR, sniff, inject | Second TCP session |

HotSpot `pkg/jvm` and `j2me-stubs` `java.net.Socket` stay on desktop only.

---

## 4. Components and file responsibilities

### 4.1 Flutter (`mobile/`)

| File | Responsibility |
|------|----------------|
| `lib/main.dart` | Boot bundled JAR on launch. No JAR picker on the play path. |
| `lib/core/bridge/game_bridge.dart` | Typed MethodChannel to gomobile. |
| `lib/presentation/game_canvas.dart` | Draw RGBA buffer; forward pointer events. |
| `lib/core/engine/viewport_scaler.dart` | Aspect-fit 240×320 → canvas coords, clamp. |
| `lib/core/engine/lifecycle_manager.dart` | `shouldPaint == (state == resumed)`. |
| `lib/core/services/background_keepalive.dart` | Silent `audioplayers` loop + `wakelock_plus`. |
| `lib/features/autoupgrade/*` | HUD state only. Start/Stop/config/stats via `GameBridge`. |
| `lib/features/jar_runner/*` | Remove from the play path. `main.dart` must not open a JAR picker. |
| `lib/core/network/crypto.dart`, `message.dart`, `session.dart` | Remove from play path. Do not open a Dart socket to the game server. |

`pubspec.yaml` keeps `audioplayers`, `wakelock_plus`, `assets/audio/silence.mp3`, `assets/KnightOnline_402.jar`. `file_picker` is not required for v1.

### 4.2 iOS glue

| File | Responsibility |
|------|----------------|
| `mobile/ios/Runner/AppDelegate.swift` | MethodChannel ↔ gomobile `Bridge`. Configure `AVAudioSession` category `.playback` with `.mixWithOthers`. |
| `mobile/ios/Runner/Info.plist` | `UIBackgroundModes: audio`; `NSAllowsArbitraryLoads: true`; hide status bar. |
| gomobile output | `KnightBridge.xcframework` from `pkg/gamebridge` (`gomobile bind -target=ios`). No CGO, no `libjvm`. |

### 4.3 Go engine

| File | Responsibility |
|------|----------------|
| `pkg/gamebridge/bridge.go` | Singleton VM + public gomobile API (see §5). |
| `pkg/jar`, `pkg/classfile`, `pkg/vm` | Existing interpreter. Extend native dispatcher for Connector and Knight Online XOR identity stubs. |
| `pkg/midp/graphics.go` | 240×320 RGBA front/back buffer. |
| `pkg/midp/connector.go` | Native `javax.microedition.io.Connector.open` for `socket://host:port`. |
| `pkg/midp/sockettap.go` | `net.Conn` wrapper: single writer mutex, handshake, XOR, copy-tap inbound, inject queue. |
| `pkg/network/protocol` | Existing `Crypto`, `Message`, `Session` framing (reuse; do not open a second `Session.Connect`). |
| `pkg/network/game` | Inventory model filled from sniffed packets. |
| `pkg/autoupgrade` | JNI-free controller used by `gamebridge` (new intercept path; existing `controller.go` CGO/JNI stays desktop-only). |

---

## 5. `gamebridge` API (gomobile-safe)

All methods are on the singleton `Bridge`. Types are primitives / `[]byte` / `string` so `gomobile bind` works.

```
LoadJAR(jarBytes []byte) string     // "" success, else error text
GetFrameBuffer() []byte             // 240×320×4 RGBA, copy
GetWidth() int                      // 240
GetHeight() int                     // 320
SendTouchDown(x, y int)
SendTouchDrag(x, y int)
SendTouchUp(x, y int)
SetShouldPaint(paint bool)          // Flutter lifecycle → skip GPU copy when false
StartUpgrade(configJSON string) string
StopUpgrade()
GetUpgradeStatus() string           // JSON: running, stats, last logs, inventory snapshot
IsStarted() bool
Stop()
```

`StartUpgrade` JSON:

```json
{
  "item_id": 1,
  "item_type": 0,
  "target_level": 10,
  "clover3_min": 4,
  "clover3_max": 7,
  "clover4_min": 8,
  "clover4_max": 15,
  "stop_if_out_of_clover": true,
  "delay_ms": 500
}
```

`GetUpgradeStatus` JSON:

```json
{
  "running": true,
  "connected": true,
  "in_game": true,
  "attempts": 0,
  "success": 0,
  "fail": 0,
  "item_level": 6,
  "logs": [
    "[12:01:04] Bat dau Auto-Upgrade +10",
    "[12:01:05] Thanh cong: cap +7"
  ]
}
```

---

## 6. Data flows

### 6.1 Boot

1. `main()` → immersive portrait → `KnightApp`.
2. `GameHomeScreen` starts `GameLoop` (50 ms) and `BackgroundKeepAliveService` when the VM is running or auto-upgrade is running.
3. Load `assets/KnightOnline_402.jar` via `rootBundle`.
4. `GameBridge.loadJAR(bytes)` → Go `ReadJar` → class load → `<clinit>` → `<init>` → `startApp`.
5. `runLoop` goroutine: drain touch queue, invoke MIDlet `run()` if present, `Graphics.Flush`, copy front buffer.

### 6.2 Frame and skip-paint

- Foreground: Flutter polls `getFrameBuffer` ~20 Hz and paints `GameCanvasWidget`.
- `AppLifecycleState.paused` / `inactive` / `hidden`: `SetShouldPaint(false)`. Tick, Connector, and auto-upgrade **keep running**. `GetFrameBuffer` may return the last frame; Flutter must not rasterize it.
- Resume: `SetShouldPaint(true)` and paint immediately.

### 6.3 Touch

`ViewportScaler` (uniform scale, letterbox offsets, clamp to `[0,239]×[0,319]`):

- down → `pointerPressed(x,y)`
- move → `pointerDragged(x,y)`
- up → `pointerReleased(x,y)`

Dispatch on the VM tick goroutine (existing `touchQueue`), never from the Dart isolate into the interpreter unsynchronized.

### 6.4 Network — one socket

The interpreter has no `java.net.Socket`. `Connector.open("socket://host:port")` is a Go native that constructs a `SocketTap` around `net.DialTimeout` (10 s, TCP_NODELAY, keepalive).

```
JAR OutputStream.write  →  write queue (mutex)  →  Crypto.Encrypt  →  net.Conn
net.Conn                →  Crypto.Decrypt       →  JAR InputStream.read
                        ↘  copy of plaintext    →  HSO parser (inventory / upgrade result)
AutoUpgrade inject      →  same write queue
```

Server: `hs1.teamobi.com:19129` (the JAR chooses the URL; the tap does not hard-dial a second host).

**Handshake** (existing `Session.PerformHandshake`): first inbound record is `1 byte key length + key bytes`. Tap installs `protocol.NewCrypto(key)` and then XOR-frames the rest.

**Framing** (existing `protocol.Session`): 1-byte command + 2-byte big-endian length (4-byte length for opcodes `-51,-52,-54,126`) + payload. Encrypt/decrypt command and payload with the stream cipher, independent `r` (read) and `s` (send) pointers.

### 6.5 XOR ownership (Knight Online specific)

The JAR encrypts in Java **before** `OutputStream.write`. Extra ciphertext on the wire advances the **server** pointer but not the JAR pointer → next game write desyncs → disconnect.

**Rule:** Go owns the only `protocol.Crypto` on this socket.

Implementation:

1. After handshake, all socket I/O in the tap is XOR'd in Go.
2. In `pkg/vm` native dispatch, the Knight Online stream-cipher methods (TeaMobi `key[i] ^ b` with independent read/send indexes) are registered as **identity**: they return the input byte unchanged. Identify those methods by matching the XOR pattern in `KnightOnline_402.jar` at implementation time; they are not used by any other JAR in this product.
3. The write queue serializes JAR writes and auto-upgrade injects under one mutex so `s` stays consistent.

Do not call `Session.Connect` from auto-upgrade. Reuse the tap's `net.Conn`.

### 6.6 Auto-upgrade loop

Port of `j2me-stubs/src/com/emulator/AutoUpgrade.java` **behavior**, without JNI:

Sequence per attempt (delay `delay_ms`, minimum 120 ms between steps):

1. Action `0`, `(itemId, itemType)` — select equipment.
2. If clover required for current level: action `0` on the clover item.
3. Action `2`, `(0, 0)` — gold/xu.
4. Action `5`, `(itemId, itemType)` — blacksmith upgrade.
5. Wait for inbound result (level change or server message). Update stats. Stop on target level, out-of-clover (if configured), or stop-condition messages (not enough gold/gems/stones, full bag).

Clover rule (existing):

- current level in `[clover3_min, clover3_max]` → 3-leaf
- current level in `[clover4_min, clover4_max]` → 4-leaf
- else none

Inject those actions as `protocol.Message` values on the tap write queue, matching the on-wire bytes `q.b(byte,short,byte)` would emit **after** identity-XOR (i.e. plaintext then Go XOR). Inventory and results come from the inbound copy-tap into `pkg/network/game.Inventory`, not from Java reflection on `bw.V`.

HUD never talks to the game server. Flutter only calls `StartUpgrade` / `StopUpgrade` / `GetUpgradeStatus`.

`in_game` becomes true when the inbound parser has seen opcode `-100`, `24`, or `26` (inventory / world sync used by `pkg/autoupgrade/tcp_bridge.go`). HUD Start stays disabled until then.

On disconnect, the controller **stops** (`running=false`, no inject). It does not auto-Start after the JAR logs in again; the user presses Start.

### 6.7 Background keep-alive

- Info.plist `UIBackgroundModes = audio`.
- Loop `assets/audio/silence.mp3` at volume 0 with `ReleaseMode.loop`.
- Wakelock on while foreground play or HUD running.
- Socket keepalive + the JAR's own heartbeat. If the tap's `net.Conn` dies: close it, return EOF / `-1` on the JAR's `InputStream.read`, and fail subsequent writes. **Do not** swap a new `net.Conn` under the existing streams (protocol state would be wrong).
- Knight Online then opens a new `Connector.open("socket://…")` if it reconnects. That creates a **new** `SocketTap` and a **new** handshake / `Crypto`. Never reuse an old XOR pointer.
- HUD: `connected=false` until the new tap is up; `in_game=false` until opcodes `-100`/`24`/`26`. Auto-upgrade remains stopped (§6.6).

---

## 7. Error handling

| Situation | Behavior |
|-----------|----------|
| `LoadJAR` fails | Return error string; Flutter shows it on the canvas; HUD stays closed; log in Go |
| Missing opcode / unregistered native | Log + safe stub (default return); do not panic. If the missing API blocks Knight Online, surface the name on the HUD log |
| TCP drop | EOF to the JAR; `connected=false`; upgrade stopped (`running=false`) |
| Handshake / XOR desync | Close socket immediately; reconnect with fresh `Crypto`; do not keep writing |
| Out of clover / gold / full bag | Stop upgrade; Vietnamese log line (same stop-condition strings as `AutoUpgrade.java`) |
| Concurrent writes | Single mutex on the tap write queue |
| iOS still suspends | On resume, if socket dead → reconnect; paint last framebuffer immediately |
| `StartUpgrade` before in-game | `StartUpgrade` returns error string; Flutter keeps the button disabled |

No empty `catch`. Every failure is logged. HUD log ring is 100 lines, newest first.

---

## 8. Security, storage, permissions

- `NSAppTransportSecurity` / `NSAllowsArbitraryLoads`: true (raw TCP to `hs1.teamobi.com`).
- RMS and settings: app documents directory (existing MIDP RecordStore native).
- HUD config: `SharedPreferences`.
- No App Store crypto/export review. Personal sideload only.

---

## 9. Testing and verification

Run on Linux. IPA is a packaging check, not a unit-test gate.

1. **`pkg/vm` + MIDP** — existing opcode/Canvas/RMS tests stay green. Add Connector native tests with a fake `net.Conn`.
2. **`SocketTap`** — mock conn: copy-tap sees the same plaintext the JAR would read; inject is serialized; two writers cannot interleave bytes.
3. **`protocol.Crypto`** — encrypt/decrypt roundtrip; pointer wrap; handshake `len+key`; after inject of N messages, a subsequent JAR write still decrypts on a paired peer (shared `s`).
4. **Auto-upgrade rules** — clover 3/4 ranges, stop-if-out-of-clover, delay; no live server.
5. **Flutter widget tests** — scaler letterbox/clamp; HUD start/stop; `shouldPaint` false when paused.
6. **Android emulator** — manual: boot JAR, login, touch, HUD, background.

CI: existing `.github/workflows/build-ios.yml` — `flutter build ipa --no-codesign`, zip `Payload/Runner.app` → artifact. The workflow must link the gomobile xcframework before `flutter build`.

---

## 10. Definition of done

- [ ] App launch boots `KnightOnline_402.jar` (no file picker) and reaches login / in-game on Android emulator; IPA sideloads on iOS.
- [ ] Direct touch maps to 240×320; no D-pad.
- [ ] Lock screen: socket stays up, character does not disconnect; GPU paint skipped.
- [ ] HUD selects an item, applies clover rules, Start/Stop, stats/log match the Go controller.
- [ ] Network loss: reconnect with new handshake; bot does not inject while disconnected.
- [ ] GitHub Actions produces an installable `.ipa`.
- [ ] Linux `go test` for tap/crypto/upgrade + `flutter test` pass. Interpreter does not panic on unknown opcodes.

---

## 11. Relationship to existing docs

| Doc | Role after this spec |
|-----|----------------------|
| `2026-09-04-knight-online-ios-flutter-design.md` | Superseded. Dart HSO client is not the iOS product. |
| `2026-09-04-knight-online-ios-flutter-plan.md` | Do not execute. Replace after this spec is approved. |
| `2026-08-08-j2me-emulator-design.md` | Interpreter / `pkg/vm` — still the engine. |
| `2026-08-09-j2me-go-engine-design.md` | Desktop HotSpot + Ebiten — not iOS. |
