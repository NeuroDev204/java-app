/// State management for the JAR runner feature.
///
/// [JarRunnerController] coordinates:
/// 1. Loading JAR bytes into the native Go VM via [GameBridge]
/// 2. Streaming RGBA frame-buffer updates at the target tick rate
/// 3. Forwarding touch events (canvas-space) to the native layer
library;

import 'dart:async';
import 'dart:convert';
import 'dart:ui' show Offset, Size;

import 'package:flutter/foundation.dart';

import '../../core/bridge/game_bridge.dart';
import '../../core/engine/viewport_scaler.dart';

/// Holds pending text input request data from J2ME TextBox.
class TextInputData {
  final String title;
  final String text;
  final int maxSize;
  final bool isPassword;

  const TextInputData({
    required this.title,
    required this.text,
    required this.maxSize,
    required this.isPassword,
  });
}

/// Possible states of the JAR runner lifecycle.
enum JarRunnerState {
  idle,
  loading,
  running,
  error,
}

/// [JarRunnerController] is a [ChangeNotifier] that drives the full JAR
/// loading → running pipeline and exposes frame-buffer updates to the UI.
class JarRunnerController extends ChangeNotifier {
  JarRunnerController({GameBridge? bridge, this.fullscreen = true})
      : _bridge = bridge ?? GameBridge();

  final GameBridge _bridge;
  final bool fullscreen;

  JarRunnerState _state = JarRunnerState.idle;
  String _errorMessage = '';
  bool _shouldPaint = true;

  /// The most recently decoded RGBA frame buffer, or null before first frame.
  Uint8List? _frameBuffer;

  Timer? _frameTimer;
  int _frameTickCounter = 0;
  TextInputData? _pendingTextInput;

  // --- Public getters ---

  JarRunnerState get state => _state;
  String get errorMessage => _errorMessage;
  TextInputData? get pendingTextInput => _pendingTextInput;

  /// Current RGBA pixel buffer (240×320×4 bytes), null while not yet running.
  Uint8List? get frameBuffer => _frameBuffer;

  bool get isRunning => _state == JarRunnerState.running;

  // --- Public API ---

  /// Loads [jarBytes] into the native VM.
  /// Used at launch for the bundled [KnightOnline_402.jar] from assets.
  Future<void> loadJARBytes(Uint8List jarBytes) async {
    if (_state == JarRunnerState.loading || _state == JarRunnerState.running) {
      return;
    }
    await _loadJAR(jarBytes);
  }

  /// Converts a screen-space touch [offset] into canvas-space coordinates
  /// and dispatches a pointer-down event to the native VM.
  ///
  /// [renderWidth] and [renderHeight] are the pixel dimensions of the
  /// GameCanvasWidget on screen (may differ from 240×320 due to scaling).
  Future<void> onTouchDown(
    double screenX,
    double screenY,
    double renderWidth,
    double renderHeight,
  ) async {
    if (!isRunning) return;
    final (int cx, int cy) =
        _toCanvasCoords(screenX, screenY, renderWidth, renderHeight);
    await _bridge.sendTouchDown(cx, cy);
  }

  /// Converts a screen-space touch offset into canvas-space and dispatches
  /// a pointer-up event to the native VM.
  Future<void> onTouchUp(
    double screenX,
    double screenY,
    double renderWidth,
    double renderHeight,
  ) async {
    if (!isRunning) return;
    final (int cx, int cy) =
        _toCanvasCoords(screenX, screenY, renderWidth, renderHeight);
    await _bridge.sendTouchUp(cx, cy);
    unawaited(Future.delayed(const Duration(milliseconds: 100), checkPendingTextInput));
  }

  /// Converts a screen-space drag offset into canvas-space and dispatches
  /// a pointer-drag event to the native VM.
  Future<void> onTouchMove(
    double screenX,
    double screenY,
    double renderWidth,
    double renderHeight,
  ) async {
    if (!isRunning) return;
    final (int cx, int cy) =
        _toCanvasCoords(screenX, screenY, renderWidth, renderHeight);
    await _bridge.sendTouchDrag(cx, cy);
  }

  /// Dispatches a pointer-down already mapped to 240×320 canvas space.
  Future<void> onCanvasTouchDown(int x, int y) async {
    if (!isRunning) return;
    await _bridge.sendTouchDown(x, y);
  }

  /// Dispatches a pointer-up already mapped to 240×320 canvas space.
  Future<void> onCanvasTouchUp(int x, int y) async {
    if (!isRunning) return;
    await _bridge.sendTouchUp(x, y);
    unawaited(Future.delayed(const Duration(milliseconds: 100), checkPendingTextInput));
  }

  /// Submits entered text to the pending J2ME TextBox.
  Future<void> submitTextInput(String text) async {
    _pendingTextInput = null;
    notifyListeners();
    await _bridge.submitTextInput(text);
    unawaited(checkPendingTextInput());
  }

  /// Cancels the pending J2ME TextBox.
  Future<void> cancelTextInput() async {
    _pendingTextInput = null;
    notifyListeners();
    await _bridge.cancelTextInput();
    unawaited(checkPendingTextInput());
  }

  /// Checks whether a J2ME TextBox is awaiting user text input.
  Future<void> checkPendingTextInput() async {
    if (!isRunning) return;
    try {
      final String jsonStr = await _bridge.getTextInputInfo();
      final Map<String, dynamic> data = jsonDecode(jsonStr);
      final bool active = data['active'] == true;
      if (active) {
        final TextInputData info = TextInputData(
          title: data['title'] as String? ?? 'Nhập văn bản',
          text: data['text'] as String? ?? '',
          maxSize: data['max_size'] as int? ?? 200,
          isPassword: data['is_password'] == true,
        );
        if (_pendingTextInput == null ||
            _pendingTextInput!.title != info.title ||
            _pendingTextInput!.isPassword != info.isPassword) {
          _pendingTextInput = info;
          notifyListeners();
        }
      } else if (_pendingTextInput != null) {
        _pendingTextInput = null;
        notifyListeners();
      }
    } catch (_) {}
  }

  /// Dispatches a pointer-drag already mapped to 240×320 canvas space.
  Future<void> onCanvasTouchMove(int x, int y) async {
    if (!isRunning) return;
    await _bridge.sendTouchDrag(x, y);
  }

  /// Dispatches a key-press event (press + release) with [keyCode] to the native VM.
  Future<void> sendKeyPress(int keyCode) async {
    await _bridge.sendKeyPress(keyCode);
  }

  /// Forwards skip-paint to the native VM and pauses Flutter frame polling.
  /// Does not stop the native tick — AFK / sockets keep running.
  Future<void> setShouldPaint(bool paint) async {
    _shouldPaint = paint;
    if (!paint) {
      _stopFrameLoop();
    }
    await _bridge.setShouldPaint(paint);
    if (paint && isRunning) {
      _startFrameLoop();
    }
  }

  /// Surfaces a boot/load failure to the canvas.
  void reportError(String message) {
    _stopFrameLoop();
    _setError(message);
  }

  @override
  void dispose() {
    _stopFrameLoop();
    _bridge.stop();
    super.dispose();
  }

  // --- Private helpers ---

  Future<void> _loadJAR(Uint8List jarBytes) async {
    _setState(JarRunnerState.loading);

    final String error = await _bridge.loadJAR(jarBytes);
    if (error.isNotEmpty) {
      _setError('VM boot failed: $error');
      return;
    }

    _setState(JarRunnerState.running);
    if (_shouldPaint) {
      _startFrameLoop();
    }
  }

  /// Polls the native frame buffer at ~20 fps and notifies listeners when
  /// a new frame is available.  The frame loop runs on the Dart event loop so
  /// it does not block the UI thread — the actual VM tick happens inside Go.
  void _startFrameLoop() {
    if (!_shouldPaint) return;
    _frameTimer?.cancel();
    _frameTickCounter = 0;
    _frameTimer = Timer.periodic(
      const Duration(milliseconds: 50), // 20 fps
      (_) async {
        if (_state != JarRunnerState.running || !_shouldPaint) {
          if (_state != JarRunnerState.running) {
            _stopFrameLoop();
          }
          return;
        }
        final Uint8List? buf = await _bridge.getFrameBuffer();
        if (buf != null && buf.isNotEmpty) {
          _frameBuffer = buf;
          notifyListeners();
        }
        _frameTickCounter++;
        if (_frameTickCounter % 6 == 0) { // ~300 ms
          unawaited(checkPendingTextInput());
        }
      },
    );
  }

  void _stopFrameLoop() {
    _frameTimer?.cancel();
    _frameTimer = null;
  }

  void _setState(JarRunnerState newState) {
    _state = newState;
    _errorMessage = '';
    notifyListeners();
  }

  void _setError(String message) {
    _state = JarRunnerState.error;
    _errorMessage = message;
    notifyListeners();
  }

  /// Maps screen-space (sx, sy) inside a render surface of [rw]×[rh] pixels
  /// to J2ME canvas-space via fullscreen viewport scaler.
  (int, int) _toCanvasCoords(
      double sx, double sy, double rw, double rh) {
    final ViewportScaler scaler = ViewportScaler(
      gameWidth: GameBridge.canvasWidth,
      gameHeight: GameBridge.canvasHeight,
      fullscreen: fullscreen,
    );
    scaler.updateViewport(Size(rw, rh));
    final Offset coords = scaler.toGameCoordinates(Offset(sx, sy));
    return (coords.dx.toInt(), coords.dy.toInt());
  }
}
