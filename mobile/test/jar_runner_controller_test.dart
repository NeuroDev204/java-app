import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/features/jar_runner/jar_runner_controller.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const MethodChannel channel =
      MethodChannel('com.example.knight_online/game_bridge');
  final List<MethodCall> log = <MethodCall>[];

  setUp(() {
    log.clear();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (MethodCall call) async {
      log.add(call);
      switch (call.method) {
        case 'loadJAR':
          return '';
        case 'getFrameBuffer':
          return null;
        case 'isStarted':
          return true;
        default:
          return null;
      }
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  test('onCanvasTouchDown sends canvas coords without remapping', () async {
    final JarRunnerController controller = JarRunnerController();
    addTearDown(controller.dispose);

    await controller.loadJARBytes(Uint8List.fromList(<int>[1, 2, 3]));
    expect(controller.isRunning, isTrue);
    log.clear();

    await controller.onCanvasTouchDown(12, 34);

    expect(log.single.method, 'sendTouchDown');
    expect(log.single.arguments, <String, int>{'x': 12, 'y': 34});
  });

  test('onTouchDown letterboxes a tall viewport when fullscreen is false',
      () async {
    final JarRunnerController controller =
        JarRunnerController(fullscreen: false);
    addTearDown(controller.dispose);

    await controller.loadJARBytes(Uint8List.fromList(<int>[1, 2, 3]));
    log.clear();

    await controller.onTouchDown(180, 50, 360, 800);

    expect(log.single.method, 'sendTouchDown');
    expect(log.single.arguments, <String, int>{'x': 120, 'y': 0});
  });

  test('onTouchDown maps to edge-to-edge canvas by default (fullscreen: true)',
      () async {
    final JarRunnerController controller = JarRunnerController();
    addTearDown(controller.dispose);

    await controller.loadJARBytes(Uint8List.fromList(<int>[1, 2, 3]));
    log.clear();

    await controller.onTouchDown(180, 50, 360, 800);

    expect(log.single.method, 'sendTouchDown');
    expect(log.single.arguments, <String, int>{'x': 120, 'y': 20});
  });

  test('onTouchMove forwards canvas coords as sendTouchDrag', () async {
    final JarRunnerController controller = JarRunnerController();
    addTearDown(controller.dispose);

    await controller.loadJARBytes(Uint8List.fromList(<int>[1, 2, 3]));
    expect(controller.isRunning, isTrue);
    log.clear();

    await controller.onTouchMove(120, 160, 240, 320);

    final Iterable<MethodCall> drags =
        log.where((MethodCall c) => c.method == 'sendTouchDrag');
    expect(drags, isNotEmpty);
    expect(drags.first.arguments, <String, int>{'x': 120, 'y': 160});
  });

  test('setShouldPaint(false) stops getFrameBuffer polling without stopping the VM',
      () async {
    final JarRunnerController controller = JarRunnerController();
    addTearDown(controller.dispose);

    await controller.loadJARBytes(Uint8List.fromList(<int>[1, 2, 3]));
    expect(controller.isRunning, isTrue);
    log.clear();

    await controller.setShouldPaint(false);
    await Future<void>.delayed(const Duration(milliseconds: 150));

    expect(
      log.where((MethodCall c) => c.method == 'getFrameBuffer'),
      isEmpty,
    );
    expect(log.where((MethodCall c) => c.method == 'stop'), isEmpty);
    expect(controller.isRunning, isTrue);

    log.clear();
    await controller.setShouldPaint(true);
    await Future<void>.delayed(const Duration(milliseconds: 150));
    expect(
      log.where((MethodCall c) => c.method == 'getFrameBuffer'),
      isNotEmpty,
    );
    expect(log.where((MethodCall c) => c.method == 'stop'), isEmpty);
  });
}
