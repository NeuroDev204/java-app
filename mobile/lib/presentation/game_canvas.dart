import 'dart:math';
import 'package:flutter/material.dart';
import '../core/engine/viewport_scaler.dart';

/// Interactive Flutter widget that renders the J2ME game canvas (240x320)
/// scaled to fit any iPhone or mobile screen while preserving aspect ratio,
/// with pillarbox or letterbox black margins.
///
/// Directly maps touch and pointer gestures to 240x320 integer game coordinates
/// without requiring on-screen D-pads or virtual buttons.
class GameCanvasWidget extends StatefulWidget {
  /// Base canvas width (defaults to 240).
  final int gameWidth;

  /// Base canvas height (defaults to 320).
  final int gameHeight;

  /// Whether to render the game canvas full-bleed edge-to-edge
  /// or maintain 240x320 aspect ratio with letterboxing. Defaults to false
  /// (uniform scale + letterbox) for the play path.
  final bool fullscreen;

  /// Optional child widget displayed inside the active game viewport.
  final Widget? child;

  /// Callback invoked when a pointer touches down on the canvas.
  final void Function(int x, int y)? onPointerDown;

  /// Callback invoked when a pointer moves across the canvas.
  final void Function(int x, int y)? onPointerMove;

  /// Callback invoked when a pointer releases contact with the canvas.
  final void Function(int x, int y)? onPointerUp;

  /// Callback invoked when a pointer interaction is canceled.
  final void Function(int x, int y)? onPointerCancel;

  /// Callback invoked when double-tapping on the canvas.
  final VoidCallback? onDoubleTap;

  /// When false, [CustomPaint] draws nothing and the game child is omitted.
  final bool shouldPaint;

  const GameCanvasWidget({
    super.key,
    this.gameWidth = 240,
    this.gameHeight = 320,
    this.fullscreen = false,
    this.child,
    this.onPointerDown,
    this.onPointerMove,
    this.onPointerUp,
    this.onPointerCancel,
    this.onDoubleTap,
    this.shouldPaint = true,
  });

  @override
  State<GameCanvasWidget> createState() => _GameCanvasWidgetState();
}

class _GameCanvasWidgetState extends State<GameCanvasWidget> {
  late ViewportScaler _scaler;

  @override
  void initState() {
    super.initState();
    _scaler = ViewportScaler(
      gameWidth: widget.gameWidth,
      gameHeight: widget.gameHeight,
      fullscreen: widget.fullscreen,
    );
  }

  @override
  void didUpdateWidget(covariant GameCanvasWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.gameWidth != widget.gameWidth ||
        oldWidget.gameHeight != widget.gameHeight ||
        oldWidget.fullscreen != widget.fullscreen) {
      _scaler = ViewportScaler(
        gameWidth: widget.gameWidth,
        gameHeight: widget.gameHeight,
        fullscreen: widget.fullscreen,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        _scaler.updateViewport(Size(constraints.maxWidth, constraints.maxHeight));

        final Widget canvasBox = Container(
          width: double.infinity,
          height: double.infinity,
          color: Colors.black,
          child: widget.child ??
              const Center(
                child: Text(
                  'Màn hình Game J2ME\n(240x320 Direct Touch)',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Colors.white70, fontSize: 13),
                ),
              ),
        );

        final Widget painted = CustomPaint(
          painter: _SkipPaintPainter(shouldPaint: widget.shouldPaint),
          child: widget.shouldPaint
              ? (widget.fullscreen
                  ? canvasBox
                  : Center(
                      child: AspectRatio(
                        aspectRatio: widget.gameWidth / widget.gameHeight,
                        child: canvasBox,
                      ),
                    ))
              : const SizedBox.expand(),
        );

        return GestureDetector(
          behavior: HitTestBehavior.opaque,
          onDoubleTap: widget.onDoubleTap,
          child: Listener(
            behavior: HitTestBehavior.opaque,
            onPointerDown: (PointerDownEvent event) {
              final Point<int> gamePos = _scaler.toGamePoint(event.localPosition);
              widget.onPointerDown?.call(gamePos.x, gamePos.y);
            },
            onPointerMove: (PointerMoveEvent event) {
              final Point<int> gamePos = _scaler.toGamePoint(event.localPosition);
              widget.onPointerMove?.call(gamePos.x, gamePos.y);
            },
            onPointerUp: (PointerUpEvent event) {
              final Point<int> gamePos = _scaler.toGamePoint(event.localPosition);
              widget.onPointerUp?.call(gamePos.x, gamePos.y);
            },
            onPointerCancel: (PointerCancelEvent event) {
              final Point<int> gamePos = _scaler.toGamePoint(event.localPosition);
              widget.onPointerCancel?.call(gamePos.x, gamePos.y);
            },
            child: Container(
              color: Colors.black,
              width: double.infinity,
              height: double.infinity,
              child: painted,
            ),
          ),
        );
      },
    );
  }
}

/// No-op painter used so skip-paint is a [CustomPaint] early-return.
class _SkipPaintPainter extends CustomPainter {
  const _SkipPaintPainter({required this.shouldPaint});

  final bool shouldPaint;

  @override
  void paint(Canvas canvas, Size size) {
    if (!shouldPaint) return;
  }

  @override
  bool shouldRepaint(covariant _SkipPaintPainter oldDelegate) =>
      oldDelegate.shouldPaint != shouldPaint;
}
