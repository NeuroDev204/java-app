/// The main game rendering screen.
///
/// Shows an aspect-fit 240×320 [GameCanvasWidget] that draws the RGBA pixel
/// buffer received from the native Go VM every ~50 ms. Touch events are mapped
/// through [ViewportScaler] then forwarded to [JarRunnerController].
library;

import 'dart:async';
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import '../../core/bridge/game_bridge.dart';
import '../../features/jar_runner/jar_runner_controller.dart';
import '../game_canvas.dart';

/// [GameScreen] is a fullscreen widget that renders the J2ME game frame buffer.
/// It is the home canvas; the bundled JAR boots on launch.
class GameScreen extends StatefulWidget {
  final JarRunnerController controller;

  /// When false, [CustomPaint] draws nothing so backgrounding skips GPU work.
  final bool shouldPaint;

  /// When true, scales canvas to fill entire viewport edge-to-edge. Defaults to false.
  final bool fullscreen;

  const GameScreen({
    super.key,
    required this.controller,
    this.shouldPaint = true,
    this.fullscreen = false,
  });

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen> {
  /// Decoded [ui.Image] ready for [Canvas.drawImage].
  ui.Image? _currentImage;

  bool _decoding = false;
  bool _showingTextInputDialog = false;

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_onControllerUpdate);
  }

  @override
  void dispose() {
    widget.controller.removeListener(_onControllerUpdate);
    _currentImage?.dispose();
    super.dispose();
  }

  Uint8List? _lastFrameBuffer;

  /// Rebuilds on error; decodes frames only while [GameScreen.shouldPaint] and when pixels change.
  void _onControllerUpdate() {
    if (!mounted) return;
    if (widget.controller.state == JarRunnerState.error) {
      setState(() {});
      return;
    }

    // Check for pending text input modal prompt
    final TextInputData? inputData = widget.controller.pendingTextInput;
    if (inputData != null && !_showingTextInputDialog) {
      _showTextInputDialog(inputData);
    }

    if (!widget.shouldPaint || _decoding) return;
    final Uint8List? buf = widget.controller.frameBuffer;
    if (buf == null || buf.length != GameBridge.canvasWidth * GameBridge.canvasHeight * 4) return;

    if (identical(_lastFrameBuffer, buf)) {
      return;
    }
    if (_lastFrameBuffer != null && _bufferEquals(_lastFrameBuffer!, buf)) {
      _lastFrameBuffer = buf;
      return;
    }

    _lastFrameBuffer = buf;
    _decoding = true;
    ui.decodeImageFromPixels(
      buf,
      GameBridge.canvasWidth,
      GameBridge.canvasHeight,
      ui.PixelFormat.rgba8888,
      (ui.Image img) {
        if (!mounted) {
          img.dispose();
          return;
        }
        final ui.Image? oldImage = _currentImage;
        setState(() {
          _currentImage = img;
          _decoding = false;
        });
        WidgetsBinding.instance.addPostFrameCallback((_) {
          oldImage?.dispose();
        });
      },
    );
  }

  void _showTextInputDialog(TextInputData info) {
    _showingTextInputDialog = true;
    showDialog<String>(
      context: context,
      barrierDismissible: false,
      builder: (BuildContext ctx) {
        final TextEditingController textCtrl = TextEditingController(text: info.text);
        bool obscure = info.isPassword;
        return StatefulBuilder(
          builder: (BuildContext context, StateSetter setDialogState) {
            return AlertDialog(
              backgroundColor: const Color(0xFF1E1E2C),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
              title: Text(
                info.title.isEmpty ? 'Nhập thông tin' : info.title,
                style: const TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold),
              ),
              content: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(
                    controller: textCtrl,
                    autofocus: true,
                    maxLength: info.maxSize,
                    obscureText: obscure,
                    style: const TextStyle(color: Colors.white, fontSize: 16),
                    decoration: InputDecoration(
                      filled: true,
                      fillColor: const Color(0xFF2A2A3E),
                      counterStyle: const TextStyle(color: Colors.white54),
                      border: OutlineInputBorder(borderRadius: BorderRadius.circular(10), borderSide: BorderSide.none),
                      suffixIcon: info.isPassword
                          ? IconButton(
                              icon: Icon(obscure ? Icons.visibility : Icons.visibility_off, color: Colors.white70),
                              onPressed: () {
                                setDialogState(() {
                                  obscure = !obscure;
                                });
                              },
                            )
                          : null,
                    ),
                    onSubmitted: (String val) {
                      Navigator.of(ctx).pop(val);
                    },
                  ),
                ],
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.of(ctx).pop(null),
                  child: const Text('Hủy', style: TextStyle(color: Colors.white60)),
                ),
                ElevatedButton(
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFF4C6EF5),
                    foregroundColor: Colors.white,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                  ),
                  onPressed: () => Navigator.of(ctx).pop(textCtrl.text),
                  child: const Text('Đồng ý'),
                ),
              ],
            );
          },
        );
      },
    ).then((String? result) {
      _showingTextInputDialog = false;
      if (result != null) {
        widget.controller.submitTextInput(result);
      } else {
        widget.controller.cancelTextInput();
      }
    });
  }

  static bool _bufferEquals(Uint8List a, Uint8List b) {
    if (identical(a, b)) return true;
    if (a.lengthInBytes != b.lengthInBytes) return false;
    try {
      final Int64List a64 = a.buffer.asInt64List(a.offsetInBytes, a.lengthInBytes ~/ 8);
      final Int64List b64 = b.buffer.asInt64List(b.offsetInBytes, b.lengthInBytes ~/ 8);
      for (int i = 0; i < a64.length; i++) {
        if (a64[i] != b64[i]) return false;
      }
      return true;
    } catch (_) {
      for (int i = 0; i < a.length; i++) {
        if (a[i] != b[i]) return false;
      }
      return true;
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.controller.state == JarRunnerState.error) {
      return Scaffold(
        backgroundColor: Colors.black,
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text(
              widget.controller.errorMessage,
              textAlign: TextAlign.center,
              style: const TextStyle(color: Colors.redAccent, fontSize: 14),
            ),
          ),
        ),
      );
    }

    Widget? child;
    if (_currentImage != null) {
      child = CustomPaint(
        painter: _FramePainter(
          _currentImage!,
          shouldPaint: widget.shouldPaint,
        ),
        child: const SizedBox.expand(),
      );
    } else if (widget.shouldPaint) {
      child = const SizedBox.expand();
    }

    return Scaffold(
      backgroundColor: Colors.black,
      body: SafeArea(
        bottom: false,
        child: GameCanvasWidget(
          fullscreen: widget.fullscreen,
          shouldPaint: widget.shouldPaint,
          onDoubleTap: () {
            unawaited(widget.controller.sendKeyPress(-5));
          },
          onPointerDown: (int x, int y) {
            unawaited(widget.controller.onCanvasTouchDown(x, y));
          },
          onPointerMove: (int x, int y) {
            unawaited(widget.controller.onCanvasTouchMove(x, y));
          },
          onPointerUp: (int x, int y) {
            unawaited(widget.controller.onCanvasTouchUp(x, y));
          },
          child: child,
        ),
      ),
    );
  }
}

/// Paints a single [ui.Image] into the fullscreen edge-to-edge canvas.
class _FramePainter extends CustomPainter {
  const _FramePainter(this._image, {this.shouldPaint = true});

  final ui.Image _image;
  final bool shouldPaint;

  @override
  void paint(Canvas canvas, Size size) {
    if (!shouldPaint) return;
    final Rect src = Rect.fromLTWH(
      0,
      0,
      _image.width.toDouble(),
      _image.height.toDouble(),
    );
    final Rect dst = Rect.fromLTWH(0, 0, size.width, size.height);
    canvas.drawImageRect(
      _image,
      src,
      dst,
      Paint()..filterQuality = FilterQuality.none,
    );
  }

  @override
  bool shouldRepaint(_FramePainter old) =>
      old._image != _image || old.shouldPaint != shouldPaint;
}
