import 'package:flutter/widgets.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/engine/lifecycle_manager.dart';
import 'package:knight_online/main.dart';

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
          return 'test-stub';
        case 'isStarted':
          return false;
        default:
          return null;
      }
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  testWidgets('paused lifecycle requests skip-paint on GameBridge',
      (WidgetTester tester) async {
    final AppLifecycleManager lifecycle = AppLifecycleManager();
    await tester.pumpWidget(KnightApp(lifecycleManager: lifecycle));
    await tester.pump();
    log.clear();

    lifecycle.updateState(AppLifecycleState.paused);
    await tester.pump();

    final Iterable<MethodCall> paintCalls =
        log.where((MethodCall c) => c.method == 'setShouldPaint');
    expect(paintCalls, isNotEmpty);
    expect(paintCalls.first.arguments, isFalse);

    lifecycle.updateState(AppLifecycleState.resumed);
    await tester.pump();
    expect(
      log.where((MethodCall c) => c.method == 'setShouldPaint').last.arguments,
      isTrue,
    );
  });

  testWidgets('paused lifecycle stops frame poll without calling stop',
      (WidgetTester tester) async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (MethodCall call) async {
      log.add(call);
      switch (call.method) {
        case 'loadJAR':
          return '';
        case 'getFrameBuffer':
          return null;
        default:
          return null;
      }
    });

    final AppLifecycleManager lifecycle = AppLifecycleManager();
    await tester.pumpWidget(KnightApp(lifecycleManager: lifecycle));
    await tester.pump();
    await tester.pump();

    lifecycle.updateState(AppLifecycleState.paused);
    await tester.pump();
    log.clear();

    await tester.pump(const Duration(milliseconds: 200));
    expect(
      log.where((MethodCall c) => c.method == 'getFrameBuffer'),
      isEmpty,
    );
    expect(log.where((MethodCall c) => c.method == 'stop'), isEmpty);
  });
}
