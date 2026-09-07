# Knight Online J2ME Aspect Ratio, RMS Icon Corruption, Sprite Transform & NPC Interaction Fix Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the 5 major gameplay and rendering bugs reported in Knight Online J2ME:
1. Deformed vertical aspect ratio & stretched sprites (`fullscreen: true` stretching 240x320 to 1280x2856).
2. HUD notification text overlapping HP/MP bars and punch-hole camera cutout.
3. In-place RMS record mutation (`getRecord` returning master slice reference mutated by Java `TemMidlet.a` XOR 0xFF loop, causing icons to toggle between PNG and corrupted garbage).
4. `image6034` stored as unencrypted PNG while other 86 stores are XORed.
5. Inverted sprite transforms 4 and 7 in `DrawRegion`.
6. NPC interaction failure due to touch coordinate misalignment and lack of `KEY_FIRE` trigger.

**Architecture:**
- `pkg/vm/interpreter.go`: Return defensive slice copy in `getRecord`; implement `listRecordStores` and `deleteRecordStore`.
- `pkg/vm/default_rms.json` & `mobile/assets/rms_data.json`: XOR `image6034` with 0xFF to align with all other stores.
- `pkg/midp/graphics.go`: Correct `TRANS_MIRROR_ROT270` ($dx=sy, dy=sx$) and `TRANS_MIRROR_ROT90` ($dx=h-1-sy, dy=w-1-sx$).
- `mobile/lib/presentation/screens/game_screen.dart` & `mobile/lib/presentation/game_canvas.dart`: Enforce `fullscreen: false` with uniform 240x320 `AspectRatio` and `SafeArea`, restoring true pixel-art scaling and keeping HUD below the camera cutout.
- `mobile/lib/presentation/screens/game_screen.dart`: Add action/talk trigger (`KEY_FIRE` / key code -5) for interacting with focused NPCs.

**Tech Stack:** Go 1.24, gomobile, Flutter 3.47 / Dart, MIDP 2.0 / LCDUI Specification, Android SDK / emulator.

---

### Task 1: Fix `getRecord` Defensive Copying & Implement `listRecordStores`/`deleteRecordStore`

**Files:**
- Modify: `pkg/vm/interpreter.go`
- Test: `pkg/vm/rms_test.go`

- [x] **Step 1: Write failing test in `pkg/vm/rms_test.go`**
  - Verify that mutating a slice returned by `getRecord` does NOT alter subsequent calls to `getRecord`.
  - Verify `listRecordStores` returns store names.
  - Verify `deleteRecordStore` removes the store.

- [x] **Step 2: Implement defensive copying and missing methods in `pkg/vm/interpreter.go`**
  - In `getRecord(int)`:
    ```go
    recCopy := make([]byte, len(result))
    copy(recCopy, result)
    frame.OperandStack.PushRef(recCopy)
    ```
  - In `listRecordStores`:
    Extract keys from `ctx.RecordStores`, build String array object.
  - In `deleteRecordStore`:
    Delete store from `ctx.RecordStores`.

- [x] **Step 3: Run `go test ./pkg/vm -run TestRMS` and verify 100% pass**

---

### Task 2: Fix `image6034` XOR Inconsistency in RMS Stores

**Files:**
- Modify: `pkg/vm/default_rms.json`
- Modify: `mobile/assets/rms_data.json`

- [x] **Step 1: Write verification script to inspect all 87 image stores**
  - Verify `image6034` is XORed with 0xFF so all 87 stores decode cleanly with `89 50 4E 47`.

- [x] **Step 2: Update `image6034` in `pkg/vm/default_rms.json` and `mobile/assets/rms_data.json`**
  - XOR raw PNG bytes of `image6034` with 0xFF and base64-encode.

- [x] **Step 3: Run `go test ./pkg/vm/...` and verify preloading decodes 87 valid stores**

---

### Task 3: Fix Sprite Transform 4 & 7 Swapping in `pkg/midp/graphics.go`

**Files:**
- Modify: `pkg/midp/graphics.go`
- Test: `pkg/midp/graphics_test.go`

- [x] **Step 1: Write test for all 8 Sprite transforms in `pkg/midp/graphics_test.go`**
  - Test `TRANS_NONE` (0), `TRANS_MIRROR_ROT180` (1), `TRANS_MIRROR` (2), `TRANS_ROT180` (3), `TRANS_MIRROR_ROT270` (4), `TRANS_ROT90` (5), `TRANS_ROT270` (6), `TRANS_MIRROR_ROT90` (7).

- [x] **Step 2: Correct transform formulas in `pkg/midp/graphics.go`**
  - `case 4: dx, dy = sy, sx`
  - `case 7: dx, dy = h-1-sy, w-1-sx`

- [x] **Step 3: Run `go test ./pkg/midp/...` and verify 100% pass**

---

### Task 4: Fix Aspect Ratio, Screen Stretching & SafeArea in Flutter

**Files:**
- Modify: `mobile/lib/presentation/screens/game_screen.dart`
- Modify: `mobile/lib/presentation/game_canvas.dart`
- Modify: `mobile/lib/main.dart`
- Test: `mobile/test/game_screen_test.dart`
- Test: `mobile/test/game_canvas_test.dart`

- [x] **Step 1: Write failing tests in `mobile/test/game_screen_test.dart`**
  - Test that `GameScreen` defaults to `fullscreen: false`.
  - Test that canvas renders within `SafeArea` and `AspectRatio(aspectRatio: 240 / 320)`.

- [x] **Step 2: Update `game_screen.dart` and `main.dart`**
  - Set `fullscreen: false`.
  - Wrap in `SafeArea(bottom: false)` so top HUD is never behind the camera punch-hole.

- [x] **Step 3: Run `flutter test` and verify tests pass**

---

### Task 5: Add NPC Interaction Action Key Trigger (`KEY_FIRE` / Talk Button)

**Files:**
- Modify: `mobile/lib/presentation/screens/game_screen.dart`
- Modify: `mobile/lib/presentation/game_canvas.dart`
- Test: `mobile/test/game_screen_test.dart`

- [x] **Step 1: Implement Action / Talk trigger**
  - Provide an on-screen floating Talk / Action button (`KEY_FIRE`, key code -5) and/or handle double tap on canvas.
  - When pressed, sends `sendKeyEvent(KEY_FIRE, true)` followed by `sendKeyEvent(KEY_FIRE, false)`.

- [x] **Step 2: Verify with widget tests**

---

### Task 6: Recompile Gomobile AAR, Build APK, Deploy & Empirically Verify

**Files:**
- Run: `mobile/scripts/bind_native.sh`
- Build & Deploy: `flutter build apk --debug && adb install -r ...`

- [ ] **Step 1: Execute `bind_native.sh` to update native binary**
- [ ] **Step 2: Build and deploy debug APK to `emulator-5554`**
- [ ] **Step 3: Take screenshot of live gameplay, verify:**
  - Aspect ratio 3:4 pixel-perfect (no vertical stretch).
  - HUD and notifications clearly visible below camera hole.
  - All icons rendered correctly.
  - NPC talk dialogue opens upon interaction.
