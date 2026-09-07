# SUPERSEDED — do not implement

**Superseded on 2026-09-04 by:** [`2026-09-04-knight-online-ios-jar-vm-design.md`](./2026-09-04-knight-online-ios-jar-vm-design.md)

This document described a Dart rewrite of the HSO protocol (Flutter as the game client). The product is now: **run `KnightOnline_402.jar` on iOS via the Go interpreter**, Flutter shell only, auto-upgrade on the JAR's socket.

The original text is kept below for history.

---

# Knight Online J2ME — iOS Flutter Client & Auto-Upgrade Tool System Design Spec

**Date:** 2026-09-04  
**Author:** AI Senior Systems Engineer & Pair Programmer  
**Target Platform:** iOS (Personal Sideloading via AltStore / Sideloadly / TrollStore)  
**Development Platform:** Cross-platform (Linux / Windows development, Android Studio / Flutter Web testing, GitHub Actions macOS build)  
**Status:** SUPERSEDED  

---

## 1. Executive Summary & Goals

### 1.1. Context & Motivation
The existing repository hosts a Go-based J2ME emulator running the Vietnamese online mobile game **Knight Online** (Hiệp Sĩ Online - TeaMobi, `KnightOnline_402.jar`) using Ebitengine, HotSpot JVM via CGO/JNI, and custom Go packages for network protocol analysis (`pkg/network`) and automated equipment enhancement (`pkg/autoupgrade`).

The user desires to port this experience to a personal **iOS application** built with **Flutter**, designed specifically for personal AFK gaming ("treo máy") and equipment auto-upgrades, without publishing to the official Apple App Store.

### 1.2. Primary Goals
1. **Full Game Experience on iOS:** Render the full original LCDUI Canvas graphics at smooth frame rates, supporting direct on-screen touch/pointer interaction (`pointerPressed`, `pointerDragged`, `pointerReleased`) without intrusive on-screen D-pads.
2. **24/7 AFK Background Persistence ("Treo máy tắt màn hình"):** The game logic, network socket, and auto-upgrade bot must continue running uninterrupted when the user locks the iPhone screen or switches to other apps. Screen rendering must be skipped when the screen is dark to conserve battery and keep the device cool.
3. **Integrated Auto-Upgrade Tool:** A sleek, non-intrusive floating HUD drawer allowing real-time configuration of auto-upgrade targets, lucky clover protection rules, speed delays, and live transaction logs.
4. **Zero-Mac Build Pipeline:** Automated GitHub Actions CI workflow to compile and package unsigned `.ipa` bundles, enabling painless personal sideloading via Sideloadly, AltStore, or TrollStore.
5. **Local Dev & Testing via Android Studio / Web:** Seamless testing on Linux using Android Studio (Android Emulator / device) or Flutter Web, before committing code for iOS IPA generation.

### 1.3. Non-Goals
- Publishing to Apple App Store or complying with strict App Store commercial guidelines (e.g. In-App Purchases, non-silent audio review checks).
- Supporting generic multi-JAR emulators for arbitrary J2ME games (the solution is tailored specifically for `KnightOnline_402.jar` and the HSO protocol).
- Adding complex on-screen keyboard overlays that obscure game graphics.

---

## 2. System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            FLUTTER APPLICATION                              │
│                                                                             │
│  ┌───────────────────────────────┐     ┌─────────────────────────────────┐  │
│  │       UI & Input Layer        │     │       Auto-Upgrade Drawer       │  │
│  │ - Direct Screen Touch Scaling │     │ - Floating draggable trigger    │  │
│  │ - Pointer event forwarding    │     │ - Target level & clover rules   │  │
│  │ - Virtual text input dialog   │     │ - Real-time upgrade statistics  │  │
│  └───────────────┬───────────────┘     └────────────────┬────────────────┘  │
│                  │                                      │                   │
│                  ▼                                      ▼                   │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    Core Engine & Logic Manager                        │  │
│  │                                                                       │  │
│  │   ┌───────────────────────────┐     ┌─────────────────────────────┐   │  │
│  │   │     Logic Tick Loop       │     │       Render Loop (GPU)     │   │  │
│  │   │ - 20-60 Hz steady tick    │     │ - LCDUI Canvas painter      │   │  │
│  │   │ - Packet handling         │     │ - If foreground: Draw frame │   │  │
│  │   │ - Bot auto-actions        │     │ - If background: SKIP PAINT │   │  │
│  │   │ - NEVER PAUSES            │     │   (zero GPU, low battery)   │   │  │
│  │   └─────────────┬─────────────┘     └─────────────────────────────┘   │  │
│  │                 │                                                     │  │
│  └─────────────────┼─────────────────────────────────────────────────────┘  │
│                    │                                                        │
│                    ▼                                                        │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                     iOS Background Services                           │  │
│  │ - UIBackgroundModes: audio (Silent Audio Loop via AVAudioPlayer)      │  │
│  │ - Wakelock Plus (Prevent sleep while observing in foreground)         │  │
│  │ - TCP Keep-Alive / Heartbeat & Auto-Reconnect                         │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Subsystem Detailed Specifications

### 3.1. Game Loop: Logic vs. Rendering Separation (AFK Mode)
To satisfy the requirement that the game must continue running while the screen is turned off:

1. **Decoupled Architecture:**
   - **Game Tick Loop (Headless Core):** Runs on a continuous timer (e.g. 50ms / 20 FPS game tick matching original MIDP specifications). Handles incoming network packets, sprite animations, world updates, player movement, combat calculations, and bot decisions.
   - **Render Loop:** Triggered on vsync / Flutter `Ticker`. Checks `WidgetsBindingObserver.didChangeAppLifecycleState`:
     - **Foreground (`AppLifecycleState.resumed`):** Renders the back-buffer image to the active Flutter canvas.
     - **Background (`AppLifecycleState.paused` / `inactive` / screen locked):** Skips the paint pipeline entirely (`skipPaint = true`).
2. **State Resynchronization:** When returning to foreground, the render loop instantly paints the current world state without frame drops, desynchronization, or socket lag.

### 3.2. Background Execution & Network Persistence (iOS)
iOS aggressively terminates or suspends apps in the background unless designated background modes are active:

1. **Silent Audio Loop Technique:**
   - Configure `UIBackgroundModes: ["audio"]` in `ios/Runner/Info.plist`.
   - On application startup, initialize `audioplayers` or native `AVAudioSession` configured with `AVAudioSessionCategoryPlayback` and `.mixWithOthers`.
   - Play an infinite loop of an empty, zero-volume 1-second WAV/MP3 track.
   - **Impact:** iOS maintains full CPU execution and unrestricted BSD socket networking for the app process, preventing iOS from freezing the socket when the screen locks.
2. **Socket Keep-Alive & Auto-Reconnect:**
   - Port the TCP packet protocol from `pkg/network/protocol` (symmetric dynamic XOR cipher and frame structure).
   - Periodically transmit empty / ping packets to prevent server-side inactivity timeout.
   - If Wi-Fi/Cellular handover drops the TCP stream, trigger exponential backoff reconnect and session re-authentication.
3. **Foreground Screen Wakelock:**
   - Integrate `wakelock_plus`. Allow users to toggle "Keep Screen On" while actively playing or monitoring upgrades.

### 3.3. Input Handling: Direct Canvas Touch Mapping
No bulky D-pads or numpads are required:

1. **Touch Coordinate Translation:**
   - Let $(W_{view}, H_{view})$ be the Flutter view dimensions and $(W_{game}, H_{game}) = (240, 320)$ (or dynamic J2ME resolution).
   - Calculate uniform aspect ratio scale factor $S = \min(W_{view} / W_{game}, H_{view} / H_{game})$ and offsets $(O_x, O_y)$.
   - For any touch event $(X_{touch}, Y_{touch})$:
     $$X_{game} = \lfloor (X_{touch} - O_x) / S \rfloor, \quad Y_{game} = \lfloor (Y_{touch} - O_y) / S \rfloor$$
2. **Event Dispatching:**
   - `Listener.onPointerDown` $\rightarrow$ `pointerPressed(x, y)`
   - `Listener.onPointerMove` $\rightarrow$ `pointerDragged(x, y)`
   - `Listener.onPointerUp` $\rightarrow$ `pointerReleased(x, y)`
3. **Text Input:**
   - Whenever the game triggers a `TextBox` or `TextField` dialog (login, character rename, chat, item count), invoke a native Flutter modal text prompt and send the result back to the game thread.

### 3.4. Auto-Upgrade Module (HUD & Engine)
Inherited from `pkg/autoupgrade/controller_tcp.go` and `pkg/autoupgrade/overlay.go`:

1. **Core Capabilities:**
   - Target enhancement tier (e.g. +7, +10, +15).
   - Clover threshold configuration:
     - 3-Leaf Clover range: Tier $L_{min}$ to $L_{max}$ (e.g., +4 to +7).
     - 4-Leaf Clover range: Tier $L'_{min}$ to $L'_{max}$ (e.g., +8 to +15).
     - Safety stop: Automatically halt if lucky clovers are exhausted.
   - Configurable delay (e.g. 500ms–1500ms) to avoid server rate-limiting.
2. **UI Design:**
   - Draggable floating button with translucent icon in the top/right corner.
   - Tapping expands a bottom sheet displaying:
     - Target item selector & current level.
     - Toggle Switch: `[ Start Auto ]` / `[ Stop Auto ]`.
     - Live statistics: Attempts count, Success / Failure ratio, Gems and Gold consumed.
     - Live audit log stream.

---

## 4. Cross-Platform Development & Testing Workflow

Since development occurs on Linux/Windows without a local Mac:

### 4.1. Fast Iteration via Android Studio / Web
- **Android Studio (Emulator or Android Phone):**
  - Run `flutter run` on Android to verify game graphics, direct touch handling, Auto-Upgrade modal UI, network packet decoding, and inventory tracking.
- **Flutter Web:**
  - Run `flutter run -d web-server --web-hostname=0.0.0.0 --web-port=8080`.
  - Open Safari on the local iPhone connected to the same Wi-Fi network (`http://<LINUX_PC_IP>:8080`) for instantaneous mobile touch and layout testing without any compilation or signing delays.

### 4.2. Automated iOS Compilation via GitHub Actions (`build-ios.yml`)
1. Push code to the repository.
2. GitHub Actions runs on `macos-latest`:
   - Installs Flutter SDK and CocoaPods.
   - Injects `KnightOnline_402.jar` into assets.
   - Runs `flutter build ipa --no-codesign`.
   - Packages `build/ios/archive/Runner.xcarchive/Products/Applications/Runner.app` into `Payload/Runner.app` and zips it to `KnightOnline.ipa`.
   - Uploads `KnightOnline.ipa` as an Action Artifact and GitHub Release.
3. **Installation on iPhone:**
   - Download the `.ipa`.
   - Sideload via **Sideloadly** (Free Apple ID, takes 30 seconds via USB cable) or **AltStore** (automatic Wi-Fi resign) or **TrollStore** (permanent, no resign needed).

---

## 5. Security, Permissions & Storage

1. **iOS Permissions (`Info.plist`):**
   - `UIBackgroundModes`: `audio`
   - `NSAppTransportSecurity`: `NSAllowsArbitraryLoads: true` (for raw TCP game sockets to `hs1.teamobi.com:19129`)
   - `UIViewControllerBasedStatusBarAppearance`: `false`
2. **Local Storage:**
   - Game RMS and User Settings stored in iOS `ApplicationDocumentsDirectory`.
   - Credentials and auto-upgrade configurations stored safely in `SharedPreferences`.

---

## 6. Verification & Definition of Done

- [ ] **Direct Touch Input:** Verify character moves and interacts with NPCs upon directly tapping the screen.
- [ ] **24/7 AFK Persistence:** Verify that upon locking the iPhone screen or putting the app in the background, the network socket remains alive and character/bot continues executing without disconnects.
- [ ] **Battery & Thermal Efficiency:** Verify GPU drawing stops when the screen is locked, keeping device temperature normal.
- [ ] **Auto-Upgrade Functionality:** Verify equipment items can be selected and upgraded according to user-defined clover and level rules.
- [ ] **CI/CD Build:** Confirm GitHub Actions produces a clean, installable `.ipa` that successfully launches on iOS via Sideloadly / AltStore.
