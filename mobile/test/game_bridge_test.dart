import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/bridge/game_bridge.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.example.knight_online/game_bridge');
  final log = <MethodCall>[];

  setUp(() {
    log.clear();
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
      log.add(call);
      switch (call.method) {
        case 'startUpgrade':
          return 'not in game';
        case 'getUpgradeStatus':
          return '{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}';
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

  test('startUpgrade sends spec JSON', () async {
    final bridge = GameBridge();
    final err = await bridge.startUpgrade(
      '{"item_id":1,"item_type":0,"target_level":10,"clover3_min":4,"clover3_max":7,"clover4_min":8,"clover4_max":15,"stop_if_out_of_clover":true,"delay_ms":500}',
    );
    expect(err, 'not in game');
    expect(log.single.method, 'startUpgrade');
  });

  test('setShouldPaint and sendTouchDrag', () async {
    final bridge = GameBridge();
    await bridge.setShouldPaint(false);
    await bridge.sendTouchDrag(12, 34);
    expect(log[0].method, 'setShouldPaint');
    expect(log[1].method, 'sendTouchDrag');
  });
}
