import 'dart:math';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/engine/viewport_scaler.dart';
import 'package:knight_online/presentation/game_canvas.dart';

void main() {
  group('ViewportScaler Unit Tests', () {
    test('ViewportScaler scales coordinates accurately with letterboxing when fullscreen is false', () {
      final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320, fullscreen: false);
      // Viewport 480x640 (exact 2x scale, zero offset)
      scaler.updateViewport(const Size(480, 640));

      final p1 = scaler.toGameCoordinates(const Offset(240, 320));
      expect(p1.dx, equals(120));
      expect(p1.dy, equals(160));

      // Viewport with horizontal letterboxing / pillarboxing (width 600, height 640)
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

    test('ViewportScaler handles vertical letterboxing when fullscreen is false (iPhone tall aspect ratio)', () {
      final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320, fullscreen: false);
      // Viewport 360 x 800:
      // scaleX = 360 / 240 = 1.5, scaleY = 800 / 320 = 2.5 => scale = 1.5
      // scaledW = 360, scaledH = 480
      // offsetX = 0.0, offsetY = (800 - 480) / 2 = 160.0
      scaler.updateViewport(const Size(360, 800));

      expect(scaler.scale, equals(1.5));
      expect(scaler.offsetX, equals(0.0));
      expect(scaler.offsetY, equals(160.0));

      // Top letterbox margin (y < 160)
      final pTop = scaler.toGameCoordinates(const Offset(180, 50));
      expect(pTop.dx, equals(120));
      expect(pTop.dy, equals(0)); // Clamped to 0

      // Bottom letterbox margin (y > 640)
      final pBottom = scaler.toGameCoordinates(const Offset(180, 750));
      expect(pBottom.dx, equals(120));
      expect(pBottom.dy, equals(319)); // Clamped to 319

      // Exact canvas center
      final pCenter = scaler.toGameCoordinates(const Offset(180, 400));
      expect(pCenter.dx, equals(120));
      expect(pCenter.dy, equals(160));
    });

    test('ViewportScaler scales coordinates accurately in fullscreen mode without letterbox margins', () {
      final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320, fullscreen: true);
      // Viewport 360 x 800:
      // scaleX = 360 / 240 = 1.5, scaleY = 800 / 320 = 2.5
      // offsetX = 0.0, offsetY = 0.0
      scaler.updateViewport(const Size(360, 800));

      expect(scaler.scaleX, equals(1.5));
      expect(scaler.scaleY, equals(2.5));
      expect(scaler.offsetX, equals(0.0));
      expect(scaler.offsetY, equals(0.0));

      // Top-left (0, 0) -> (0, 0)
      final pTopLeft = scaler.toGameCoordinates(const Offset(0, 0));
      expect(pTopLeft.dx, equals(0));
      expect(pTopLeft.dy, equals(0));

      // Center (180, 400) -> 180 / 1.5 = 120, 400 / 2.5 = 160
      final pCenter = scaler.toGameCoordinates(const Offset(180, 400));
      expect(pCenter.dx, equals(120));
      expect(pCenter.dy, equals(160));

      // Bottom-right edge (359, 799)
      final pBottomRight = scaler.toGameCoordinates(const Offset(359, 799));
      expect(pBottomRight.dx, equals(239));
      expect(pBottomRight.dy, equals(319));

      // Bounds detection
      expect(scaler.isInGameBounds(const Offset(0, 0)), isTrue);
      expect(scaler.isInGameBounds(const Offset(180, 400)), isTrue);
      expect(scaler.isInGameBounds(const Offset(359, 799)), isTrue);
      expect(scaler.isInGameBounds(const Offset(360, 800)), isFalse);

      // gameRect spans entire 360x800 viewport
      expect(scaler.gameRect, equals(const Rect.fromLTWH(0.0, 0.0, 360.0, 800.0)));
    });

    test('ViewportScaler boundary touches map to exact canvas edges', () {
      final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320);
      scaler.updateViewport(const Size(480, 640)); // scale = 2.0

      final pTopLeft = scaler.toGameCoordinates(const Offset(0, 0));
      expect(pTopLeft.dx, equals(0));
      expect(pTopLeft.dy, equals(0));

      final pBottomRight = scaler.toGameCoordinates(const Offset(479, 639));
      expect(pBottomRight.dx, equals(239));
      expect(pBottomRight.dy, equals(319));
    });

    test('ViewportScaler safely handles degenerate or zero viewport sizes', () {
      final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320);
      scaler.updateViewport(Size.zero);
      expect(scaler.scale, equals(1.0));

      final p = scaler.toGameCoordinates(const Offset(100, 100));
      expect(p.dx, equals(100));
      expect(p.dy, equals(100));

      scaler.updateViewport(const Size(-100, -100));
      expect(scaler.scale, equals(1.0));

      scaler.updateViewport(const Size(double.infinity, 100));
      expect(scaler.scale, equals(1.0));

      scaler.updateViewport(const Size(100, double.nan));
      expect(scaler.scale, equals(1.0));
    });

    test('ViewportScaler isInGameBounds detects active area vs letterbox when fullscreen is false', () {
      final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320, fullscreen: false);
      // Viewport with pillarbox: 600x640, scale = 2.0, offsetX = 60, offsetY = 0
      scaler.updateViewport(const Size(600, 640));

      expect(scaler.isInGameBounds(const Offset(30, 320)), isFalse); // Left pillarbox
      expect(scaler.isInGameBounds(const Offset(550, 320)), isFalse); // Right pillarbox
      expect(scaler.isInGameBounds(const Offset(60, 0)), isTrue); // Canvas top-left
      expect(scaler.isInGameBounds(const Offset(300, 320)), isTrue); // Inside
      expect(scaler.isInGameBounds(const Offset(539, 639)), isTrue); // Near bottom-right
      expect(scaler.isInGameBounds(const Offset(540, 640)), isFalse); // Past border
    });

    test('ViewportScaler toGamePoint returns Point<int>', () {
      final scaler = ViewportScaler(gameWidth: 240, gameHeight: 320);
      scaler.updateViewport(const Size(480, 640));

      final pt = scaler.toGamePoint(const Offset(240, 320));
      expect(pt, equals(const Point<int>(120, 160)));
    });
  });

  group('GameCanvasWidget Widget Tests', () {
    testWidgets('GameCanvasWidget renders placeholder text and captures touches', (WidgetTester tester) async {
      tester.view.physicalSize = const Size(480, 640);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);

      int? downX, downY;
      int? moveX, moveY;
      int? upX, upY;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 480,
              height: 640,
              child: GameCanvasWidget(
                onPointerDown: (x, y) {
                  downX = x;
                  downY = y;
                },
                onPointerMove: (x, y) {
                  moveX = x;
                  moveY = y;
                },
                onPointerUp: (x, y) {
                  upX = x;
                  upY = y;
                },
              ),
            ),
          ),
        ),
      );

      expect(find.text('Màn hình Game J2ME\n(240x320 Direct Touch)'), findsOneWidget);

      final TestGesture gesture = await tester.startGesture(const Offset(240, 320));
      expect(downX, equals(120));
      expect(downY, equals(160));

      await gesture.moveTo(const Offset(242, 322));
      expect(moveX, equals(121));
      expect(moveY, equals(161));

      await gesture.up();
      expect(upX, equals(121));
      expect(upY, equals(161));
    });

    testWidgets('GameCanvasWidget renders custom child widget when provided', (WidgetTester tester) async {
      tester.view.physicalSize = const Size(480, 640);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 480,
              height: 640,
              child: GameCanvasWidget(
                child: Text('Custom Game View'),
              ),
            ),
          ),
        ),
      );

      expect(find.text('Custom Game View'), findsOneWidget);
      expect(find.text('Màn hình Game J2ME\n(240x320 Direct Touch)'), findsNothing);
    });

    testWidgets(
        'GameCanvasWidget letterboxes a tall viewport instead of stretch-fill',
        (WidgetTester tester) async {
      tester.view.physicalSize = const Size(360, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);

      int? downX, downY;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 360,
              height: 800,
              child: GameCanvasWidget(
                fullscreen: false,
                onPointerDown: (x, y) {
                  downX = x;
                  downY = y;
                },
              ),
            ),
          ),
        ),
      );

      await tester.startGesture(const Offset(180, 50));
      expect(downX, equals(120));
      expect(downY, equals(0));
    });

    testWidgets('GameCanvasWidget skips painting when shouldPaint is false',
        (WidgetTester tester) async {
      tester.view.physicalSize = const Size(480, 640);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 480,
              height: 640,
              child: GameCanvasWidget(shouldPaint: false),
            ),
          ),
        ),
      );

      expect(
        find.text('Màn hình Game J2ME\n(240x320 Direct Touch)'),
        findsNothing,
      );
    });
  });
}
