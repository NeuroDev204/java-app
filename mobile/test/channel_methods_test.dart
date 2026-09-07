import 'dart:io';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('iOS AppDelegate and Android MainActivity mention every channel method', () {
    const methods = [
      'loadJAR', 'getFrameBuffer', 'sendTouchDown', 'sendTouchUp',
      'sendTouchDrag', 'stop', 'isStarted', 'setShouldPaint',
      'startUpgrade', 'stopUpgrade', 'getUpgradeStatus',
    ];
    final swift = File('ios/Runner/AppDelegate.swift').readAsStringSync();
    final kt = File('android/app/src/main/kotlin/com/example/knight_online/MainActivity.kt').readAsStringSync();
    for (final m in methods) {
      expect(swift.contains('"$m"') || swift.contains("\"$m\""), isTrue, reason: 'swift $m');
      expect(kt.contains('"$m"'), isTrue, reason: 'kotlin $m');
    }
  });

  test('iOS linked VM calls hop off the platform thread before result', () {
    final swift = File('ios/Runner/AppDelegate.swift').readAsStringSync();
    expect(swift.contains('DispatchQueue(label:'), isTrue, reason: 'serial off-main queue');
    expect(swift.contains('DispatchQueue.main'), isTrue, reason: 'hop back before result');
    for (final method in ['loadJAR', 'getFrameBuffer', 'stop']) {
      expect(swift.contains('"$method"'), isTrue, reason: 'swift $method');
    }
  });

  test('unlinked getUpgradeStatus returns empty JSON object', () {
    const empty =
        '{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}';
    final swift = File('ios/Runner/AppDelegate.swift').readAsStringSync();
    final kt = File('android/app/src/main/kotlin/com/example/knight_online/MainActivity.kt')
        .readAsStringSync();
    expect(swift.contains(empty), isTrue, reason: 'swift empty upgrade JSON');
    expect(kt.contains(empty), isTrue, reason: 'kotlin empty upgrade JSON');
  });
}
