/// Flutter abstraction layer over the native Go game bridge.
///
/// Communication uses a Flutter MethodChannel that maps 1:1 to the
/// Go [gamebridge.Bridge] API built into the native layer:
///
/// - loadJAR(jarBytes) → String (empty = success)
/// - getFrameBuffer()  → Uint8List (RGBA bytes)
/// - sendTouchDown(x, y)
/// - sendTouchUp(x, y)
/// - sendTouchDrag(x, y)
/// - sendKeyPress(keyCode)
/// - setShouldPaint(paint)
/// - startUpgrade(configJSON) → String (empty = success)
/// - stopUpgrade()
/// - getUpgradeStatus() → String (JSON)
/// - stop()
/// - isStarted()       → bool
///
/// On iOS, this channel is wired to the gomobile-generated KnightBridge.xcframework.
/// On Android, it is wired to the Kotlin side that loads the Go .so via JNI.
library;

import 'package:flutter/services.dart';

/// [GameBridge] wraps the platform-specific MethodChannel calls and exposes
/// a clean, typed Dart API to the rest of the Flutter codebase.
class GameBridge {
  static const MethodChannel _channel =
      MethodChannel('com.example.knight_online/game_bridge');

  /// Canvas width as declared by the native bridge (240 px).
  static const int canvasWidth = 240;

  /// Canvas height as declared by the native bridge (320 px).
  static const int canvasHeight = 320;

  /// Loads the JAR file passed as raw bytes into the Go VM.
  ///
  /// Returns an empty string on success, or a human-readable error message.
  Future<String> loadJAR(Uint8List jarBytes) async {
    final String? result = await _channel.invokeMethod<String>(
      'loadJAR',
      jarBytes,
    );
    return result ?? '';
  }

  /// Retrieves the current RGBA frame buffer from the native canvas.
  ///
  /// Returns [canvasWidth] × [canvasHeight] × 4 bytes.
  /// Returns null if the native side is not yet initialised.
  Future<Uint8List?> getFrameBuffer() async {
    final Uint8List? buf =
        await _channel.invokeMethod<Uint8List>('getFrameBuffer');
    return buf;
  }

  /// Sends a pointer-down event at the given canvas-space coordinates.
  Future<void> sendTouchDown(int x, int y) async {
    await _channel.invokeMethod<void>('sendTouchDown', {'x': x, 'y': y});
  }

  /// Sends a pointer-up event at the given canvas-space coordinates.
  Future<void> sendTouchUp(int x, int y) async {
    await _channel.invokeMethod<void>('sendTouchUp', {'x': x, 'y': y});
  }

  /// Sends a pointer-drag event at the given canvas-space coordinates.
  Future<void> sendTouchDrag(int x, int y) async {
    await _channel.invokeMethod<void>('sendTouchDrag', {'x': x, 'y': y});
  }

  /// Sends a key-press event (press + release) with [keyCode] to the active canvas.
  Future<void> sendKeyPress(int keyCode) async {
    await _channel.invokeMethod<void>('sendKeyPress', {'keyCode': keyCode});
  }

  /// Tells the native layer whether Flutter still wants painted frames.
  Future<void> setShouldPaint(bool paint) async {
    await _channel.invokeMethod<void>('setShouldPaint', paint);
  }

  /// Starts native auto-upgrade with the spec JSON payload.
  ///
  /// Returns an empty string on success, or a human-readable error message.
  Future<String> startUpgrade(String configJSON) async {
    final String? result = await _channel.invokeMethod<String>(
      'startUpgrade',
      configJSON,
    );
    return result ?? '';
  }

  /// Stops the native auto-upgrade loop.
  Future<void> stopUpgrade() async {
    await _channel.invokeMethod<void>('stopUpgrade');
  }

  /// Returns native auto-upgrade status as a JSON string.
  Future<String> getUpgradeStatus() async {
    final String? result =
        await _channel.invokeMethod<String>('getUpgradeStatus');
    return result ?? '';
  }

  /// Halts the background game loop in the native layer.
  Future<void> stop() async {
    await _channel.invokeMethod<void>('stop');
  }

  /// Returns true if loadJAR completed successfully.
  Future<bool> isStarted() async {
    final bool? started = await _channel.invokeMethod<bool>('isStarted');
    return started ?? false;
  }

  /// Retrieves JSON describing pending text input (if any).
  Future<String> getTextInputInfo() async {
    final String? result =
        await _channel.invokeMethod<String>('getTextInputInfo');
    return result ?? '{"active":false}';
  }

  /// Submits text from native dialog to pending J2ME TextBox.
  Future<void> submitTextInput(String text) async {
    await _channel.invokeMethod<void>('submitTextInput', text);
  }

  /// Cancels pending J2ME TextBox.
  Future<void> cancelTextInput() async {
    await _channel.invokeMethod<void>('cancelTextInput');
  }
}

