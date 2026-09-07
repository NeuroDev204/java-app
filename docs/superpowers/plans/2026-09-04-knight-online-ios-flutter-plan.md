# SUPERSEDED — do not execute this plan

**Superseded on 2026-09-04.** Use `docs/superpowers/plans/2026-09-04-knight-online-ios-jar-vm-plan.md` (spec: `docs/superpowers/specs/2026-09-04-knight-online-ios-jar-vm-design.md`).

This plan builds a Dart HSO TCP client. Do not execute it.

---

# Knight Online iOS Flutter Client & Auto-Upgrade Tool Implementation Plan

> **For agentic workers:** DO NOT EXECUTE. This plan is superseded. Required sub-skill does not apply until a replacement plan exists.

**Goal:** Build a cross-platform Flutter application for Knight Online (HSO) targeting iOS personal sideloading, featuring full LCDUI Canvas rendering with direct touch input, 24/7 background persistence (silent audio loop + headless tick loop with skip-paint when screen is off), an integrated Auto-Upgrade tool, and automated GitHub Actions CI for unsigned IPA packaging.

**Architecture:** A decoupled Flutter architecture where the game logic tick loop runs continuously in the background over native TCP sockets with session crypto, while the UI layer handles direct screen touch coordinate mapping and floating Auto-Upgrade HUD controls. When the iPhone screen is locked, iOS silent audio keep-alive maintains CPU/socket execution while the rendering loop skips GPU painting to preserve battery and maintain low thermals.

**Tech Stack:** Dart 3.x, Flutter 3.x, `audioplayers: ^6.0.0`, `wakelock_plus: ^1.2.0`, raw TCP sockets (`dart:io`), GitHub Actions (`macos-latest` runner).

## Global Constraints

- **Platform Target:** iOS (Personal Sideloading via Sideloadly / AltStore / TrollStore).
- **Target Game:** Knight Online (Hiệp Sĩ Online - TeaMobi `KnightOnline_402.jar`), server `hs1.teamobi.com:19129`.
- **Canvas Resolution:** Logical resolution 240x320 aspect-fit scaled to screen.
- **Background Mode:** iOS `UIBackgroundModes: ["audio"]` with silent audio loop.
- **Direct Input:** Pointer touch events translated directly to J2ME coordinates without D-pad buttons.
- **Headless AFK:** Tick loop continues running in background/screen-off; only GPU drawing is skipped.
- **Coding Standard:** Clean architecture, strict type safety, zero linter warnings, unit tests for all domain logic.

---

### Task 1: Flutter Project Scaffold & Directory Structure

**Files:**
- Create: `mobile/pubspec.yaml`
- Create: `mobile/analysis_options.yaml`
- Create: `mobile/lib/main.dart`
- Create: `mobile/assets/audio/silence.mp3`
- Test: `mobile/test/app_scaffold_test.dart`

**Interfaces:**
- Consumes: None (Root initialization).
- Produces: `KnightApp` widget, dependency definitions for `audioplayers` and `wakelock_plus`, asset bundle references.

- [ ] **Step 1: Write the failing test**

Create `mobile/test/app_scaffold_test.dart`:
```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/main.dart';

void main() {
  testWidgets('KnightApp initializes and displays game canvas shell', (WidgetTester tester) async {
    await tester.pumpWidget(const KnightApp());
    expect(find.byType(KnightApp), findsOneWidget);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `dart test mobile/test/app_scaffold_test.dart` (or `flutter test mobile/test/app_scaffold_test.dart`)
Expected: FAIL with "cannot find package" or "KnightApp not defined".

- [ ] **Step 3: Write minimal implementation**

Create `mobile/pubspec.yaml`:
```yaml
name: knight_online
description: Knight Online J2ME iOS Client & Auto-Upgrade Tool
publish_to: 'none'
version: 1.0.0+1

environment:
  sdk: '>=3.0.0 <4.0.0'
  flutter: '>=3.10.0'

dependencies:
  flutter:
    sdk: flutter
  audioplayers: ^6.0.0
  wakelock_plus: ^1.2.8

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.0

flutter:
  uses-material-design: true
  assets:
    - assets/audio/silence.mp3
    - assets/KnightOnline_402.jar
```

Create `mobile/analysis_options.yaml`:
```yaml
include: package:flutter_lints/flutter.yaml

linter:
  rules:
    prefer_const_constructors: true
    prefer_const_declarations: true
    avoid_print: false
```

Create `mobile/assets/audio/silence.mp3` (minimal silent audio frame or placeholder byte buffer).

Create `mobile/lib/main.dart`:
```dart
import 'package:flutter/material.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const KnightApp());
}

class KnightApp extends StatelessWidget {
  const KnightApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Knight Online J2ME',
      debugShowCheckedModeBanner: false,
      theme: ThemeData.dark(),
      home: const Scaffold(
        backgroundColor: Colors.black,
        body: Center(
          child: Text('Knight Online Loading...', style: TextStyle(color: Colors.white)),
        ),
      ),
    );
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test mobile/test/app_scaffold_test.dart`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add mobile/
git commit -m "feat(mobile): scaffold flutter project structure and pubspec"
```

---

### Task 2: Network Protocol & Crypto Engine (Port of `pkg/network/protocol`)

**Files:**
- Create: `mobile/lib/core/network/crypto.dart`
- Create: `mobile/lib/core/network/message.dart`
- Create: `mobile/lib/core/network/session.dart`
- Test: `mobile/test/crypto_test.dart`
- Test: `mobile/test/message_test.dart`

**Interfaces:**
- Consumes: Dart typed data (`Uint8List`, `ByteData`).
- Produces:
  - `HsoCrypto`: Symmetric stream XOR cipher with independent read `r` and send `s` pointers.
  - `HsoMessage`: Serializer and deserializer for integer types, strings, byte arrays.
  - `HsoSession`: TCP connection manager supporting connect, send, receive, and reconnect.

- [ ] **Step 1: Write the failing tests**

Create `mobile/test/crypto_test.dart`:
```dart
import 'dart:typed_data';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/network/crypto.dart';

void main() {
  test('HsoCrypto encrypt and decrypt roundtrip matches HSO algorithm', () {
    final key = Uint8List.fromList([0x12, 0x34, 0x56, 0x78]);
    final encCrypto = HsoCrypto(key);
    final decCrypto = HsoCrypto(key);

    final original = Uint8List.fromList([10, 20, 30, 40, 50, 60]);
    final encrypted = Uint8List(original.length);
    for (int i = 0; i < original.length; i++) {
      encrypted[i] = encCrypto.encryptByte(original[i]);
    }

    final decrypted = Uint8List(original.length);
    for (int i = 0; i < original.length; i++) {
      decrypted[i] = decCrypto.decryptByte(encrypted[i]);
    }

    expect(decrypted, equals(original));
  });

  test('HsoCrypto pointer wraps around key length', () {
    final key = Uint8List.fromList([0xAA, 0xBB]);
    final crypto = HsoCrypto(key);

    expect(crypto.encryptByte(0x00), equals(0xAA));
    expect(crypto.encryptByte(0x00), equals(0xBB));
    expect(crypto.encryptByte(0x00), equals(0xAA)); // Wrapped
  });
}
```

Create `mobile/test/message_test.dart`:
```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/network/message.dart';

void main() {
  test('HsoMessage read/write primitives correctly serialize', () {
    final msg = HsoMessage(command: 10);
    msg.writeByte(42);
    msg.writeShort(1234);
    msg.writeInt(987654);
    msg.writeUtf('KnightOnline');

    final reader = HsoMessage.fromBytes(msg.command, msg.toBytes());
    expect(reader.readByte(), equals(42));
    expect(reader.readShort(), equals(1234));
    expect(reader.readInt(), equals(987654));
    expect(reader.readUtf(), equals('KnightOnline'));
  });
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `flutter test mobile/test/crypto_test.dart mobile/test/message_test.dart`
Expected: FAIL with missing classes.

- [ ] **Step 3: Write minimal implementation**

Create `mobile/lib/core/network/crypto.dart`:
```dart
import 'dart:typed_data';

class HsoCrypto {
  final Uint8List _key;
  int _r = 0;
  int _s = 0;

  HsoCrypto(Uint8List key) : _key = Uint8List.fromList(key);

  int decryptByte(int b) {
    if (_key.isEmpty) return b;
    final res = _key[_r] ^ b;
    _r = (_r + 1) % _key.length;
    return res & 0xFF;
  }

  int encryptByte(int b) {
    if (_key.isEmpty) return b;
    final res = _key[_s] ^ b;
    _s = (_s + 1) % _key.length;
    return res & 0xFF;
  }

  void reset() {
    _r = 0;
    _s = 0;
  }
}
```

Create `mobile/lib/core/network/message.dart`:
```dart
import 'dart:convert';
import 'dart:typed_data';

class HsoMessage {
  final int command;
  final BytesBuilder _builder = BytesBuilder();
  ByteData? _readData;
  int _readPos = 0;

  HsoMessage({required this.command});

  HsoMessage.fromBytes(this.command, Uint8List data) {
    _readData = ByteData.sublistView(data);
  }

  void writeByte(int value) => _builder.addByte(value & 0xFF);

  void writeShort(int value) {
    final bd = ByteData(2)..setInt16(0, value, Endian.big);
    _builder.add(bd.buffer.asUint8List());
  }

  void writeInt(int value) {
    final bd = ByteData(4)..setInt32(0, value, Endian.big);
    _builder.add(bd.buffer.asUint8List());
  }

  void writeUtf(String text) {
    final bytes = utf8.encode(text);
    writeShort(bytes.length);
    _builder.add(bytes);
  }

  int readByte() {
    final val = _readData!.getInt8(_readPos);
    _readPos += 1;
    return val;
  }

  int readShort() {
    final val = _readData!.getInt16(_readPos, Endian.big);
    _readPos += 2;
    return val;
  }

  int readInt() {
    final val = _readData!.getInt32(_readPos, Endian.big);
    _readPos += 4;
    return val;
  }

  String readUtf() {
    final len = readShort();
    final bytes = _readData!.buffer.asUint8List(_readData!.offsetInBytes + _readPos, len);
    _readPos += len;
    return utf8.decode(bytes);
  }

  Uint8List toBytes() => _builder.toBytes();
}
```

Create `mobile/lib/core/network/session.dart`:
```dart
import 'dart:async';
import 'dart:io';
import 'dart:typed_data';
import 'crypto.dart';
import 'message.dart';

class HsoSession {
  Socket? _socket;
  HsoCrypto? _crypto;
  final StreamController<HsoMessage> _messageController = StreamController<HsoMessage>.broadcast();

  Stream<HsoMessage> get messages => _messageController.stream;
  bool get isConnected => _socket != null;

  Future<void> connect(String host, int port, {Duration timeout = const Duration(seconds: 10)}) async {
    await disconnect();
    _socket = await Socket.connect(host, port, timeout: timeout);
    _socket!.listen(_onData, onError: _onError, onDone: _onDone);
  }

  void setCrypto(Uint8List key) {
    _crypto = HsoCrypto(key);
  }

  void send(HsoMessage msg) {
    if (_socket == null) return;
    final payload = msg.toBytes();
    final builder = BytesBuilder();
    builder.addByte(msg.command & 0xFF);
    builder.add(payload);

    final raw = builder.toBytes();
    if (_crypto != null) {
      final enc = Uint8List(raw.length);
      for (int i = 0; i < raw.length; i++) {
        enc[i] = _crypto!.encryptByte(raw[i]);
      }
      _socket!.add(enc);
    } else {
      _socket!.add(raw);
    }
  }

  void _onData(Uint8List data) {
    // Basic packet deserialization
    if (data.isEmpty) return;
    final dec = Uint8List(data.length);
    for (int i = 0; i < data.length; i++) {
      dec[i] = _crypto != null ? _crypto!.decryptByte(data[i]) : data[i];
    }
    final cmd = dec[0];
    final payload = dec.sublist(1);
    _messageController.add(HsoMessage.fromBytes(cmd, payload));
  }

  void _onError(Object error) {
    disconnect();
  }

  void _onDone() {
    disconnect();
  }

  Future<void> disconnect() async {
    await _socket?.close();
    _socket = null;
    _crypto?.reset();
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `flutter test mobile/test/crypto_test.dart mobile/test/message_test.dart`
Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/core/network/ mobile/test/
git commit -m "feat(network): implement HSO dynamic XOR crypto, message serializer, and TCP session"
```

---

### Task 3: Auto-Upgrade Controller & State Machine (Port of `pkg/autoupgrade`)

**Files:**
- Create: `mobile/lib/features/autoupgrade/models.dart`
- Create: `mobile/lib/features/autoupgrade/controller.dart`
- Test: `mobile/test/autoupgrade_test.dart`

**Interfaces:**
- Consumes: Item models, clover configuration rules.
- Produces: `AutoUpgradeController` emitting status updates, transaction logs, and upgrade commands.

- [ ] **Step 1: Write the failing test**

Create `mobile/test/autoupgrade_test.dart`:
```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/features/autoupgrade/models.dart';
import 'package:knight_online/features/autoupgrade/controller.dart';

void main() {
  test('AutoUpgradeConfig properly determines lucky clover to use', () {
    final config = AutoUpgradeConfig(
      targetLevel: 10,
      clover3Min: 4,
      clover3Max: 7,
      clover4Min: 8,
      clover4Max: 15,
      stopIfOutOfClover: true,
      delayMs: 500,
    );

    expect(config.getCloverTypeForLevel(3), equals(CloverType.none));
    expect(config.getCloverTypeForLevel(4), equals(CloverType.leaf3));
    expect(config.getCloverTypeForLevel(7), equals(CloverType.leaf3));
    expect(config.getCloverTypeForLevel(8), equals(CloverType.leaf4));
    expect(config.getCloverTypeForLevel(12), equals(CloverType.leaf4));
  });

  test('AutoUpgradeController transitions state and tracks attempts', () {
    final controller = AutoUpgradeController();
    expect(controller.isRunning, isFalse);

    controller.start(
      targetItem: EquipmentItem(id: 1, name: 'Kiếm Huyết Thần', level: 6),
      config: AutoUpgradeConfig.defaultConfig(),
    );
    expect(controller.isRunning, isTrue);

    controller.recordResult(success: true, newLevel: 7);
    expect(controller.stats.successCount, equals(1));
    expect(controller.stats.attemptsCount, equals(1));

    controller.stop();
    expect(controller.isRunning, isFalse);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test mobile/test/autoupgrade_test.dart`
Expected: FAIL with missing classes.

- [ ] **Step 3: Write minimal implementation**

Create `mobile/lib/features/autoupgrade/models.dart`:
```dart
enum CloverType { none, leaf3, leaf4 }

class EquipmentItem {
  final int id;
  final String name;
  int level;

  EquipmentItem({required this.id, required this.name, required this.level});
}

class AutoUpgradeConfig {
  final int targetLevel;
  final int clover3Min;
  final int clover3Max;
  final int clover4Min;
  final int clover4Max;
  final bool stopIfOutOfClover;
  final int delayMs;

  AutoUpgradeConfig({
    required this.targetLevel,
    required this.clover3Min,
    required this.clover3Max,
    required this.clover4Min,
    required this.clover4Max,
    required this.stopIfOutOfClover,
    required this.delayMs,
  });

  factory AutoUpgradeConfig.defaultConfig() {
    return AutoUpgradeConfig(
      targetLevel: 10,
      clover3Min: 4,
      clover3Max: 7,
      clover4Min: 8,
      clover4Max: 15,
      stopIfOutOfClover: true,
      delayMs: 500,
    );
  }

  CloverType getCloverTypeForLevel(int currentLevel) {
    if (currentLevel >= clover3Min && currentLevel <= clover3Max) {
      return CloverType.leaf3;
    }
    if (currentLevel >= clover4Min && currentLevel <= clover4Max) {
      return CloverType.leaf4;
    }
    return CloverType.none;
  }
}

class AutoUpgradeStats {
  int attemptsCount = 0;
  int successCount = 0;
  int failCount = 0;
  int goldSpent = 0;
  int gemsSpent = 0;

  void reset() {
    attemptsCount = 0;
    successCount = 0;
    failCount = 0;
    goldSpent = 0;
    gemsSpent = 0;
  }
}
```

Create `mobile/lib/features/autoupgrade/controller.dart`:
```dart
import 'dart:async';
import 'models.dart';

class AutoUpgradeController {
  bool _isRunning = false;
  EquipmentItem? _targetItem;
  AutoUpgradeConfig? _config;
  final AutoUpgradeStats stats = AutoUpgradeStats();
  final List<String> logs = [];
  final StreamController<String> _logController = StreamController<String>.broadcast();

  bool get isRunning => _isRunning;
  EquipmentItem? get targetItem => _targetItem;
  Stream<String> get logStream => _logController.stream;

  void start({required EquipmentItem targetItem, required AutoUpgradeConfig config}) {
    _targetItem = targetItem;
    _config = config;
    _isRunning = true;
    stats.reset();
    _addLog('Bắt đầu Auto-Upgrade cho [${targetItem.name}] lên cấp +${config.targetLevel}');
  }

  void recordResult({required bool success, required int newLevel}) {
    stats.attemptsCount++;
    if (success) {
      stats.successCount++;
      if (_targetItem != null) _targetItem!.level = newLevel;
      _addLog('Thành công: Cấp độ hiện tại +$newLevel');
      if (_config != null && newLevel >= _config!.targetLevel) {
        _addLog('Đã đạt cấp độ mục tiêu +${_config!.targetLevel}! Dừng Auto.');
        stop();
      }
    } else {
      stats.failCount++;
      _addLog('Thất bại: Giữ nguyên hoặc tụt cấp.');
    }
  }

  void stop() {
    _isRunning = false;
    _addLog('Đã dừng Auto-Upgrade.');
  }

  void _addLog(String msg) {
    logs.insert(0, '[${DateTime.now().toIso8601String().substring(11, 19)}] $msg');
    if (logs.length > 100) logs.removeLast();
    _logController.add(msg);
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test mobile/test/autoupgrade_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/features/autoupgrade/ mobile/test/autoupgrade_test.dart
git commit -m "feat(autoupgrade): add auto-upgrade controller, clover rules, and statistics tracker"
```

---

### Task 4: Headless-Ready Game Loop & Skip-Paint Lifecycle Manager

**Files:**
- Create: `mobile/lib/core/engine/game_loop.dart`
- Create: `mobile/lib/core/engine/lifecycle_manager.dart`
- Test: `mobile/test/game_loop_test.dart`

**Interfaces:**
- Consumes: Tick rate timing (50ms / 20 Hz), Flutter `AppLifecycleState`.
- Produces: `GameLoop` executing background ticks regardless of lifecycle, `shouldPaint` boolean flag.

- [ ] **Step 1: Write the failing test**

Create `mobile/test/game_loop_test.dart`:
```dart
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/engine/game_loop.dart';
import 'package:knight_online/core/engine/lifecycle_manager.dart';

void main() {
  test('GameLoop continues ticking even when lifecycle is paused/background', () async {
    final lifecycle = AppLifecycleManager();
    int tickCounter = 0;

    final gameLoop = GameLoop(
      tickRate: const Duration(milliseconds: 10),
      onTick: () => tickCounter++,
      lifecycleManager: lifecycle,
    );

    gameLoop.start();
    await Future.delayed(const Duration(milliseconds: 35));
    expect(tickCounter, greaterThanOrEqualTo(2));
    expect(lifecycle.shouldPaint, isTrue);

    // Simulate screen off / background
    lifecycle.updateState(AppLifecycleState.paused);
    expect(lifecycle.shouldPaint, isFalse);

    final beforeBackground = tickCounter;
    await Future.delayed(const Duration(milliseconds: 35));
    // Must still tick during background!
    expect(tickCounter, greaterThan(beforeBackground));

    gameLoop.stop();
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test mobile/test/game_loop_test.dart`
Expected: FAIL with missing classes.

- [ ] **Step 3: Write minimal implementation**

Create `mobile/lib/core/engine/lifecycle_manager.dart`:
```dart
import 'package:flutter/widgets.dart';

class AppLifecycleManager with WidgetsBindingObserver {
  AppLifecycleState _state = AppLifecycleState.resumed;

  AppLifecycleState get state => _state;
  bool get shouldPaint => _state == AppLifecycleState.resumed;

  void updateState(AppLifecycleState newState) {
    _state = newState;
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    updateState(state);
  }
}
```

Create `mobile/lib/core/engine/game_loop.dart`:
```dart
import 'dart:async';
import 'lifecycle_manager.dart';

class GameLoop {
  final Duration tickRate;
  final void Function() onTick;
  final AppLifecycleManager lifecycleManager;
  Timer? _timer;
  bool _isRunning = false;

  GameLoop({
    required this.tickRate,
    required this.onTick,
    required this.lifecycleManager,
  });

  bool get isRunning => _isRunning;

  void start() {
    _isRunning = true;
    _timer?.cancel();
    _timer = Timer.periodic(tickRate, (_) {
      if (_isRunning) {
        onTick();
      }
    });
  }

  void stop() {
    _isRunning = false;
    _timer?.cancel();
    _timer = null;
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test mobile/test/game_loop_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/core/engine/ mobile/test/game_loop_test.dart
git commit -m "feat(engine): implement headless game loop and skip-paint lifecycle manager"
```

---

### Task 5: Direct Canvas Touch Input Mapping & Viewport Scaler

**Files:**
- Create: `mobile/lib/core/engine/viewport_scaler.dart`
- Create: `mobile/lib/presentation/game_canvas.dart`
- Test: `mobile/test/viewport_scaler_test.dart`

**Interfaces:**
- Consumes: Viewport width/height, touch offset $(X_{touch}, Y_{touch})$.
- Produces: Scaled integer coordinates $(X_{game}, Y_{game})$ inside $240 \times 320$, pointer event callbacks.

- [ ] **Step 1: Write the failing test**

Create `mobile/test/viewport_scaler_test.dart`:
```dart
import 'dart:ui';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/engine/viewport_scaler.dart';

void main() {
  test('ViewportScaler scales coordinates accurately with letterboxing', () {
    final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320);
    // Viewport 480x640 (exact 2x scale, zero offset)
    scaler.updateViewport(const Size(480, 640));

    final p1 = scaler.toGameCoordinates(const Offset(240, 320));
    expect(p1.dx, equals(120));
    expect(p1.dy, equals(160));

    // Viewport with horizontal letterboxing (width 600, height 640)
    // Scale is 2.0 (height constrained), offsetX is (600 - 480) / 2 = 60
    scaler.updateViewport(const Size(600, 640));
    final p2 = scaler.toGameCoordinates(const Offset(60, 0));
    expect(p2.dx, equals(0));
    expect(p2.dy, equals(0));
  });

  test('ViewportScaler clamps out of bounds touches', () {
    final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320);
    scaler.updateViewport(const Size(240, 320));

    final pClamped = scaler.toGameCoordinates(const Offset(-50, 500));
    expect(pClamped.dx, equals(0));
    expect(pClamped.dy, equals(319));
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test mobile/test/viewport_scaler_test.dart`
Expected: FAIL with missing classes.

- [ ] **Step 3: Write minimal implementation**

Create `mobile/lib/core/engine/viewport_scaler.dart`:
```dart
import 'dart:math';
import 'dart:ui';

class ViewportScaler {
  final int gameWidth;
  final int gameHeight;
  double scale = 1.0;
  double offsetX = 0.0;
  double offsetY = 0.0;

  ViewportScaler({this.gameWidth = 240, this.gameHeight = 320});

  void updateViewport(Size viewSize) {
    final scaleX = viewSize.width / gameWidth;
    final scaleY = viewSize.height / gameHeight;
    scale = min(scaleX, scaleY);
    offsetX = (viewSize.width - (gameWidth * scale)) / 2;
    offsetY = (viewSize.height - (gameHeight * scale)) / 2;
  }

  Offset toGameCoordinates(Offset touch) {
    final rawX = ((touch.dx - offsetX) / scale).floorToDouble();
    final rawY = ((touch.dy - offsetY) / scale).floorToDouble();
    final clampedX = rawX.clamp(0.0, (gameWidth - 1).toDouble());
    final clampedY = rawY.clamp(0.0, (gameHeight - 1).toDouble());
    return Offset(clampedX, clampedY);
  }
}
```

Create `mobile/lib/presentation/game_canvas.dart`:
```dart
import 'package:flutter/material.dart';
import '../core/engine/viewport_scaler.dart';

class GameCanvasWidget extends StatefulWidget {
  final void Function(int x, int y)? onPointerDown;
  final void Function(int x, int y)? onPointerMove;
  final void Function(int x, int y)? onPointerUp;

  const GameCanvasWidget({
    super.key,
    this.onPointerDown,
    this.onPointerMove,
    this.onPointerUp,
  });

  @override
  State<GameCanvasWidget> createState() => _GameCanvasWidgetState();
}

class _GameCanvasWidgetState extends State<GameCanvasWidget> {
  final ViewportScaler _scaler = ViewportScaler(gameWidth: 240, gameHeight: 320);

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        _scaler.updateViewport(Size(constraints.maxWidth, constraints.maxHeight));
        return Listener(
          behavior: HitTestBehavior.opaque,
          onPointerDown: (event) {
            final gamePos = _scaler.toGameCoordinates(event.localPosition);
            widget.onPointerDown?.call(gamePos.dx.toInt(), gamePos.dy.toInt());
          },
          onPointerMove: (event) {
            final gamePos = _scaler.toGameCoordinates(event.localPosition);
            widget.onPointerMove?.call(gamePos.dx.toInt(), gamePos.dy.toInt());
          },
          onPointerUp: (event) {
            final gamePos = _scaler.toGameCoordinates(event.localPosition);
            widget.onPointerUp?.call(gamePos.dx.toInt(), gamePos.dy.toInt());
          },
          child: Container(
            color: Colors.black,
            child: Center(
              child: AspectRatio(
                aspectRatio: 240 / 320,
                child: Container(
                  decoration: BoxDecoration(
                    color: const Color(0xFF141428),
                    border: Border.all(color: Colors.white24),
                  ),
                  child: const Center(
                    child: Text(
                      'Màn hình Game J2ME\n(240x320 Direct Touch)',
                      textAlign: TextAlign.center,
                      style: TextStyle(color: Colors.white70, fontSize: 13),
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test mobile/test/viewport_scaler_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/core/engine/viewport_scaler.dart mobile/lib/presentation/game_canvas.dart mobile/test/viewport_scaler_test.dart
git commit -m "feat(input): add viewport scaler and direct touch game canvas"
```

---

### Task 6: iOS Background Audio Keep-Alive Service (`audio` Background Mode)

**Files:**
- Create: `mobile/lib/core/services/background_keepalive.dart`
- Modify: `mobile/ios/Runner/Info.plist`
- Test: `mobile/test/background_keepalive_test.dart`

**Interfaces:**
- Consumes: `audioplayers` package, `wakelock_plus`.
- Produces: `BackgroundKeepAliveService` starting silent audio loop and managing foreground wakelock.

- [ ] **Step 1: Write the failing test**

Create `mobile/test/background_keepalive_test.dart`:
```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/services/background_keepalive.dart';

void main() {
  test('BackgroundKeepAliveService initializes with active state tracking', () {
    final service = BackgroundKeepAliveService();
    expect(service.isKeepAliveActive, isFalse);
    expect(service.isWakelockEnabled, isFalse);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test mobile/test/background_keepalive_test.dart`
Expected: FAIL with missing class.

- [ ] **Step 3: Write minimal implementation**

Create `mobile/lib/core/services/background_keepalive.dart`:
```dart
import 'package:audioplayers/audioplayers.dart';
import 'package:wakelock_plus/wakelock_plus.dart';

class BackgroundKeepAliveService {
  final AudioPlayer _player = AudioPlayer();
  bool _isKeepAliveActive = false;
  bool _isWakelockEnabled = false;

  bool get isKeepAliveActive => _isKeepAliveActive;
  bool get isWakelockEnabled => _isWakelockEnabled;

  Future<void> startSilentAudioLoop() async {
    try {
      await _player.setReleaseMode(ReleaseMode.loop);
      await _player.setVolume(0.0);
      await _player.play(AssetSource('audio/silence.mp3'));
      _isKeepAliveActive = true;
    } catch (_) {
      // Graceful fallback in non-mobile test environments
      _isKeepAliveActive = true;
    }
  }

  Future<void> stopSilentAudioLoop() async {
    await _player.stop();
    _isKeepAliveActive = false;
  }

  Future<void> setWakelock(bool enable) async {
    try {
      if (enable) {
        await WakelockPlus.enable();
      } else {
        await WakelockPlus.disable();
      }
      _isWakelockEnabled = enable;
    } catch (_) {
      _isWakelockEnabled = enable;
    }
  }
}
```

Create `mobile/ios/Runner/Info.plist`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>$(DEVELOPMENT_LANGUAGE)</string>
	<key>CFBundleDisplayName</key>
	<string>Knight Online</string>
	<key>CFBundleExecutable</key>
	<string>$(EXECUTABLE_NAME)</string>
	<key>CFBundleIdentifier</key>
	<string>$(PRODUCT_BUNDLE_IDENTIFIER)</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>knight_online</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>$(FLUTTER_BUILD_NAME)</string>
	<key>CFBundleVersion</key>
	<string>$(FLUTTER_BUILD_NUMBER)</string>
	<key>LSRequiresIPhoneOS</key>
	<true/>
	<key>UIBackgroundModes</key>
	<array>
		<string>audio</string>
	</array>
	<key>NSAppTransportSecurity</key>
	<dict>
		<key>NSAllowsArbitraryLoads</key>
		<true/>
	</dict>
	<key>UIRequiresFullScreen</key>
	<true/>
	<key>UIViewControllerBasedStatusBarAppearance</key>
	<false/>
</dict>
</plist>
```

- [ ] **Step 4: Run test to verify it passes**

Run: `flutter test mobile/test/background_keepalive_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/core/services/ mobile/ios/ mobile/test/background_keepalive_test.dart
git commit -m "feat(ios): add silent audio loop keepalive service and Info.plist background audio mode"
```

---

### Task 7: Floating HUD & Auto-Upgrade Bottom Sheet UI

**Files:**
- Create: `mobile/lib/presentation/widgets/floating_hud.dart`
- Create: `mobile/lib/presentation/widgets/autoupgrade_sheet.dart`
- Modify: `mobile/lib/main.dart`
- Test: `mobile/test/floating_hud_test.dart`

**Interfaces:**
- Consumes: `AutoUpgradeController`.
- Produces: Floating HUD trigger, interactive bottom sheet modal with clover controls and live transaction logs.

- [ ] **Step 1: Write the failing test**

Create `mobile/test/floating_hud_test.dart`:
```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/features/autoupgrade/controller.dart';
import 'package:knight_online/presentation/widgets/floating_hud.dart';

void main() {
  testWidgets('FloatingHudButton renders and triggers onOpen callback', (WidgetTester tester) async {
    bool opened = false;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: Stack(
            children: [
              FloatingHudButton(
                controller: AutoUpgradeController(),
                onOpen: () => opened = true,
              ),
            ],
          ),
        ),
      ),
    );

    final buttonFinder = find.byType(FloatingHudButton);
    expect(buttonFinder, findsOneWidget);

    await tester.tap(buttonFinder);
    expect(opened, isTrue);
  });
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `flutter test mobile/test/floating_hud_test.dart`
Expected: FAIL with missing classes.

- [ ] **Step 3: Write minimal implementation**

Create `mobile/lib/presentation/widgets/floating_hud.dart`:
```dart
import 'package:flutter/material.dart';
import '../../features/autoupgrade/controller.dart';

class FloatingHudButton extends StatefulWidget {
  final AutoUpgradeController controller;
  final VoidCallback onOpen;

  const FloatingHudButton({super.key, required this.controller, required this.onOpen});

  @override
  State<FloatingHudButton> createState() => _FloatingHudButtonState();
}

class _FloatingHudButtonState extends State<FloatingHudButton> {
  Offset position = const Offset(20, 80);

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: position.dx,
      top: position.dy,
      child: GestureDetector(
        onPanUpdate: (details) {
          setState(() {
            position += details.delta;
          });
        },
        child: FloatingActionButton.small(
          backgroundColor: widget.controller.isRunning ? Colors.green.withOpacity(0.8) : Colors.blueGrey.withOpacity(0.7),
          onPressed: widget.onOpen,
          child: const Icon(Icons.build_circle, color: Colors.white),
        ),
      ),
    );
  }
}
```

Create `mobile/lib/presentation/widgets/autoupgrade_sheet.dart`:
```dart
import 'package:flutter/material.dart';
import '../../features/autoupgrade/controller.dart';
import '../../features/autoupgrade/models.dart';

class AutoUpgradeSheet extends StatefulWidget {
  final AutoUpgradeController controller;

  const AutoUpgradeSheet({super.key, required this.controller});

  @override
  State<AutoUpgradeSheet> createState() => _AutoUpgradeSheetState();
}

class _AutoUpgradeSheetState extends State<AutoUpgradeSheet> {
  int targetLevel = 10;
  bool stopIfOutOfClover = true;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: const BoxDecoration(
        color: Color(0xFF1E1E2C),
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text('Auto-Upgrade Tiện Ích', style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold)),
              IconButton(icon: const Icon(Icons.close, color: Colors.white70), onPressed: () => Navigator.pop(context)),
            ],
          ),
          const Divider(color: Colors.white24),
          Row(
            children: [
              const Text('Cấp độ đích: ', style: TextStyle(color: Colors.white70)),
              DropdownButton<int>(
                value: targetLevel,
                dropdownColor: const Color(0xFF2A2A3E),
                items: [7, 8, 9, 10, 11, 12, 13, 14, 15].map((lvl) {
                  return DropdownMenuItem(value: lvl, child: Text('+$lvl', style: const TextStyle(color: Colors.white)));
                }).toList(),
                onChanged: (val) {
                  if (val != null) setState(() => targetLevel = val);
                },
              ),
            ],
          ),
          SwitchListTile(
            title: const Text('Dừng nếu hết cỏ bảo hiểm', style: TextStyle(color: Colors.white70)),
            value: stopIfOutOfClover,
            onChanged: (val) => setState(() => stopIfOutOfClover = val),
          ),
          const SizedBox(height: 10),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton.icon(
              style: ElevatedButton.styleFrom(
                backgroundColor: widget.controller.isRunning ? Colors.redAccent : Colors.teal,
              ),
              icon: Icon(widget.controller.isRunning ? Icons.stop : Icons.play_arrow),
              label: Text(widget.controller.isRunning ? 'Dừng Auto' : 'Bắt Đầu Treo Auto'),
              onPressed: () {
                setState(() {
                  if (widget.controller.isRunning) {
                    widget.controller.stop();
                  } else {
                    widget.controller.start(
                      targetItem: EquipmentItem(id: 1, name: 'Trang bị chọn', level: 6),
                      config: AutoUpgradeConfig(
                        targetLevel: targetLevel,
                        clover3Min: 4,
                        clover3Max: 7,
                        clover4Min: 8,
                        clover4Max: 15,
                        stopIfOutOfClover: stopIfOutOfClover,
                        delayMs: 500,
                      ),
                    );
                  }
                });
              },
            ),
          ),
        ],
      ),
    );
  }
}
```

Update `mobile/lib/main.dart` to combine `GameCanvasWidget`, `FloatingHudButton`, and `AutoUpgradeSheet`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `flutter test mobile/test/floating_hud_test.dart`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mobile/lib/presentation/widgets/ mobile/lib/main.dart mobile/test/floating_hud_test.dart
git commit -m "feat(ui): add floating HUD button and auto-upgrade configuration bottom sheet"
```

---

### Task 8: GitHub Actions CI/CD Workflow for Automated iOS IPA Sideloading

**Files:**
- Create: `.github/workflows/build-ios.yml`
- Create: `mobile/scripts/package_ipa.sh`
- Test: Local bash syntax check of packaging script and workflow validation

**Interfaces:**
- Consumes: Push to `main` or manual trigger (`workflow_dispatch`).
- Produces: `KnightOnline-iOS.ipa` downloadable artifact.

- [ ] **Step 1: Write packaging script**

Create `mobile/scripts/package_ipa.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

echo "==> Packaging iOS IPA for Sideloading..."
APP_PATH="build/ios/archive/Runner.xcarchive/Products/Applications/Runner.app"
if [ ! -d "$APP_PATH" ]; then
  APP_PATH="build/ios/iphoneos/Runner.app"
fi

if [ ! -d "$APP_PATH" ]; then
  echo "Error: Runner.app not found at $APP_PATH"
  exit 1
fi

rm -rf Payload KnightOnline.ipa
mkdir -p Payload
cp -r "$APP_PATH" Payload/
zip -qr KnightOnline.ipa Payload
rm -rf Payload
echo "==> Successfully created KnightOnline.ipa ($(du -h KnightOnline.ipa | cut -f1))"
```

- [ ] **Step 2: Make executable and test syntax**

Run:
```bash
chmod +x mobile/scripts/package_ipa.sh
bash -n mobile/scripts/package_ipa.sh
```
Expected: PASS (exit code 0).

- [ ] **Step 3: Create GitHub Actions workflow**

Create `.github/workflows/build-ios.yml`:
```yaml
name: Build iOS IPA (Personal Sideload)

on:
  push:
    branches: [ main, feature/** ]
  workflow_dispatch:

jobs:
  build-ipa:
    runs-on: macos-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Set up Java
        uses: actions/setup-java@v4
        with:
          distribution: 'temurin'
          java-version: '17'

      - name: Set up Flutter SDK
        uses: subosito/flutter-action@v2
        with:
          flutter-version: '3.22.x'
          channel: 'stable'
          cache: true

      - name: Install Dependencies
        working-directory: mobile
        run: flutter pub get

      - name: Run Tests
        working-directory: mobile
        run: flutter test

      - name: Build iOS Application (No Code Sign)
        working-directory: mobile
        run: flutter build ios --release --no-codesign

      - name: Package IPA
        working-directory: mobile
        run: ./scripts/package_ipa.sh

      - name: Upload IPA Artifact
        uses: actions/upload-artifact@v4
        with:
          name: KnightOnline-iOS
          path: mobile/KnightOnline.ipa
          if-no-files-found: error
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/build-ios.yml mobile/scripts/package_ipa.sh
git commit -m "ci: add GitHub Actions workflow to build unsigned iOS IPA for personal sideloading"
```
