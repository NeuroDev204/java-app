import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/features/jar_runner/jar_runner_controller.dart';
import 'package:knight_online/presentation/game_canvas.dart';
import 'package:knight_online/presentation/screens/game_screen.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const MethodChannel channel =
      MethodChannel('com.example.knight_online/game_bridge');

  setUp(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (MethodCall call) async {
      switch (call.method) {
        case 'loadJAR':
          return 'boot-failed';
        default:
          return null;
      }
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  testWidgets('GameScreen shows boot error instead of spinner',
      (WidgetTester tester) async {
    final JarRunnerController controller = JarRunnerController();
    addTearDown(controller.dispose);

    await controller.loadJARBytes(Uint8List.fromList(<int>[1]));
    expect(controller.state, JarRunnerState.error);

    await tester.pumpWidget(
      MaterialApp(home: GameScreen(controller: controller)),
    );

    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.textContaining('boot-failed'), findsOneWidget);
  });

  testWidgets(
      'GameScreen play path uses aspect-fit GameCanvasWidget with SafeArea and fullscreen false by default',
      (WidgetTester tester) async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (MethodCall call) async {
      if (call.method == 'loadJAR') {
        return '';
      }
      return null;
    });

    final JarRunnerController controller = JarRunnerController();
    await controller.loadJARBytes(Uint8List.fromList(<int>[1]));

    await tester.pumpWidget(
      MaterialApp(home: GameScreen(controller: controller)),
    );

    // Verify SafeArea wraps GameCanvasWidget with bottom: false
    expect(find.byType(SafeArea), findsOneWidget);
    final SafeArea safeArea = tester.widget<SafeArea>(find.byType(SafeArea));
    expect(safeArea.bottom, isFalse);

    expect(find.byType(GameCanvasWidget), findsOneWidget);
    final GameCanvasWidget canvas =
        tester.widget<GameCanvasWidget>(find.byType(GameCanvasWidget));
    expect(canvas.fullscreen, isFalse);

    // Verify AspectRatio is 240 / 320
    expect(find.byType(AspectRatio), findsOneWidget);
    final AspectRatio aspectRatio =
        tester.widget<AspectRatio>(find.byType(AspectRatio));
    expect(aspectRatio.aspectRatio, closeTo(240 / 320, 0.001));

    await tester.pumpWidget(const SizedBox.shrink());
    controller.dispose();
  });

  testWidgets('GameScreen allows fullscreen true override',
      (WidgetTester tester) async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (MethodCall call) async {
      if (call.method == 'loadJAR') {
        return '';
      }
      return null;
    });

    final JarRunnerController controller =
        JarRunnerController(fullscreen: true);
    await controller.loadJARBytes(Uint8List.fromList(<int>[1]));

    await tester.pumpWidget(
      MaterialApp(home: GameScreen(controller: controller, fullscreen: true)),
    );

    expect(find.byType(GameCanvasWidget), findsOneWidget);
    final GameCanvasWidget canvas =
        tester.widget<GameCanvasWidget>(find.byType(GameCanvasWidget));
    expect(canvas.fullscreen, isTrue);
    expect(find.byType(AspectRatio), findsNothing);

    await tester.pumpWidget(const SizedBox.shrink());
    controller.dispose();
  });



  testWidgets('GameScreen double-tap on canvas triggers sendKeyPress(-5)',
      (WidgetTester tester) async {
    int? pressedKeyCode;
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (MethodCall call) async {
      if (call.method == 'loadJAR') {
        return '';
      } else if (call.method == 'sendKeyPress') {
        final Map<dynamic, dynamic>? args =
            call.arguments as Map<dynamic, dynamic>?;
        pressedKeyCode = args?['keyCode'] as int?;
        return null;
      }
      return null;
    });

    final JarRunnerController controller = JarRunnerController();
    await controller.loadJARBytes(Uint8List.fromList(<int>[1]));

    await tester.pumpWidget(
      MaterialApp(home: GameScreen(controller: controller)),
    );

    // Double tap on GameCanvasWidget
    await tester.tap(find.byType(GameCanvasWidget));
    await tester.pump(const Duration(milliseconds: 50));
    await tester.tap(find.byType(GameCanvasWidget));
    await tester.pump(const Duration(milliseconds: 400));

    expect(pressedKeyCode, equals(-5));

    await tester.pumpWidget(const SizedBox.shrink());
    controller.dispose();
  });
}
