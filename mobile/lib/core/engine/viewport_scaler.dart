import 'dart:math';
import 'dart:ui';

/// Handles uniform scaling of the original 240x320 J2ME canvas to any display/screen
/// viewport, calculating aspect ratio preservation offsets (pillarboxing/letterboxing)
/// and mapping local touch/pointer coordinates to scaled 240x320 integer game coordinates.
class ViewportScaler {
  /// Base canvas width of the original J2ME client (typically 240).
  final int gameWidth;

  /// Base canvas height of the original J2ME client (typically 320).
  final int gameHeight;

  /// Whether to stretch to fill the entire viewport (edge-to-edge fullscreen)
  /// or preserve original aspect ratio with letterboxing/pillarboxing.
  final bool fullscreen;

  /// The uniform scale factor applied to the game canvas (in aspect-fit mode).
  double scale = 1.0;

  /// Horizontal scale factor.
  double scaleX = 1.0;

  /// Vertical scale factor.
  double scaleY = 1.0;

  /// Horizontal letterbox/pillarbox offset in viewport pixels.
  double offsetX = 0.0;

  /// Vertical letterbox/pillarbox offset in viewport pixels.
  double offsetY = 0.0;

  ViewportScaler({
    this.gameWidth = 240,
    this.gameHeight = 320,
    this.fullscreen = true,
  });

  /// Updates viewport dimensions and recalculates uniform scale and letterbox offsets.
  void updateViewport(Size viewSize) {
    if (!viewSize.width.isFinite || !viewSize.height.isFinite ||
        viewSize.width <= 0 || viewSize.height <= 0 ||
        gameWidth <= 0 || gameHeight <= 0) {
      scale = 1.0;
      scaleX = 1.0;
      scaleY = 1.0;
      offsetX = 0.0;
      offsetY = 0.0;
      return;
    }

    scaleX = viewSize.width / gameWidth;
    scaleY = viewSize.height / gameHeight;

    if (fullscreen) {
      scale = min(scaleX, scaleY);
      offsetX = 0.0;
      offsetY = 0.0;
    } else {
      scale = min(scaleX, scaleY);
      offsetX = (viewSize.width - (gameWidth * scale)) / 2.0;
      offsetY = (viewSize.height - (gameHeight * scale)) / 2.0;
    }
  }

  /// Maps a viewport touch/pointer offset to scaled game canvas coordinates (0..gameWidth-1, 0..gameHeight-1).
  ///
  /// Out-of-bounds touches (e.g. within letterbox margins) are clamped defensively
  /// to the nearest valid canvas coordinate.
  Offset toGameCoordinates(Offset touch) {
    if (fullscreen) {
      if (scaleX <= 0.0 || scaleY <= 0.0) {
        return Offset.zero;
      }
      final double rawX = (touch.dx / scaleX).floorToDouble();
      final double rawY = (touch.dy / scaleY).floorToDouble();
      final double clampedX = rawX.clamp(0.0, (gameWidth - 1).toDouble());
      final double clampedY = rawY.clamp(0.0, (gameHeight - 1).toDouble());
      return Offset(clampedX, clampedY);
    }

    if (scale <= 0.0) {
      return Offset.zero;
    }
    final double rawX = ((touch.dx - offsetX) / scale).floorToDouble();
    final double rawY = ((touch.dy - offsetY) / scale).floorToDouble();
    final double clampedX = rawX.clamp(0.0, (gameWidth - 1).toDouble());
    final double clampedY = rawY.clamp(0.0, (gameHeight - 1).toDouble());
    return Offset(clampedX, clampedY);
  }

  /// Convenience method returning game coordinates as an integer [Point].
  Point<int> toGamePoint(Offset touch) {
    final Offset coords = toGameCoordinates(touch);
    return Point<int>(coords.dx.toInt(), coords.dy.toInt());
  }

  /// Returns whether a touch point falls strictly within the active game canvas
  /// (i.e. not in the pillarbox or letterbox black margins).
  bool isInGameBounds(Offset touch) {
    if (fullscreen) {
      if (scaleX <= 0.0 || scaleY <= 0.0) return false;
      return touch.dx >= 0 &&
          touch.dx < (gameWidth * scaleX) &&
          touch.dy >= 0 &&
          touch.dy < (gameHeight * scaleY);
    }

    if (scale <= 0.0) return false;
    final double scaledW = gameWidth * scale;
    final double scaledH = gameHeight * scale;
    return touch.dx >= offsetX &&
        touch.dx < (offsetX + scaledW) &&
        touch.dy >= offsetY &&
        touch.dy < (offsetY + scaledH);
  }

  /// The active rendered rectangle for the game canvas within the viewport.
  Rect get gameRect {
    if (fullscreen) {
      return Rect.fromLTWH(0.0, 0.0, gameWidth * scaleX, gameHeight * scaleY);
    }
    return Rect.fromLTWH(
      offsetX,
      offsetY,
      gameWidth * scale,
      gameHeight * scale,
    );
  }
}
