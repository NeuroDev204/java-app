import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/presentation/game_canvas.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('GameCanvasWidget Widget Tests', () {
    testWidgets(
        'GameCanvasWidget defaults to fullscreen: false and renders AspectRatio(240 / 320)',
        (WidgetTester tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: GameCanvasWidget(),
          ),
        ),
      );

      final GameCanvasWidget canvas =
          tester.widget<GameCanvasWidget>(find.byType(GameCanvasWidget));
      expect(canvas.fullscreen, isFalse);

      expect(find.byType(AspectRatio), findsOneWidget);
      final AspectRatio aspectRatio =
          tester.widget<AspectRatio>(find.byType(AspectRatio));
      expect(aspectRatio.aspectRatio, closeTo(240 / 320, 0.001));
    });

    testWidgets(
        'GameCanvasWidget in fullscreen: false mode centers AspectRatio within available viewport',
        (WidgetTester tester) async {
      tester.view.physicalSize = const Size(360, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 360,
              height: 800,
              child: GameCanvasWidget(fullscreen: false),
            ),
          ),
        ),
      );

      expect(find.byType(AspectRatio), findsOneWidget);
      expect(find.byType(Center), findsWidgets);

      final Rect canvasRect = tester.getRect(find.byType(AspectRatio));
      expect(canvasRect.width, equals(360));
      expect(canvasRect.height, equals(480));
      // Centered vertically: (800 - 480) / 2 = 160
      expect(canvasRect.top, equals(160));
      expect(canvasRect.bottom, equals(640));
    });

    testWidgets(
        'GameCanvasWidget accurately maps touches to 0..239, 0..319 coordinates via toGamePoint',
        (WidgetTester tester) async {
      tester.view.physicalSize = const Size(360, 800);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);

      int? downX, downY;
      int? moveX, moveY;
      int? upX, upY;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 360,
              height: 800,
              child: GameCanvasWidget(
                fullscreen: false,
                onPointerDown: (int x, int y) {
                  downX = x;
                  downY = y;
                },
                onPointerMove: (int x, int y) {
                  moveX = x;
                  moveY = y;
                },
                onPointerUp: (int x, int y) {
                  upX = x;
                  upY = y;
                },
              ),
            ),
          ),
        ),
      );

      // Tap top letterbox margin (y = 50 < 160) -> clamped to y = 0
      final TestGesture gesture =
          await tester.startGesture(const Offset(180, 50));
      expect(downX, equals(120));
      expect(downY, equals(0));

      // Move inside canvas center (y = 400 => (400-160)/1.5 = 160)
      await gesture.moveTo(const Offset(180, 400));
      expect(moveX, equals(120));
      expect(moveY, equals(160));

      // Release in bottom letterbox margin (y = 750 > 640) -> clamped to y = 319
      await gesture.moveTo(const Offset(180, 750));
      await gesture.up();
      expect(upX, equals(120));
      expect(upY, equals(319));
    });

    testWidgets('GameCanvasWidget fullscreen: true omits AspectRatio',
        (WidgetTester tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: GameCanvasWidget(fullscreen: true),
          ),
        ),
      );

      expect(find.byType(AspectRatio), findsNothing);
    });
  });
}
