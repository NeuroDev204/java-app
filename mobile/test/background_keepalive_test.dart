import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/services/background_keepalive.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUpAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
      const MethodChannel('xyz.luan/audioplayers.global'),
      (MethodCall methodCall) async => 1,
    );
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
      const MethodChannel('xyz.luan/audioplayers'),
      (MethodCall methodCall) async => 1,
    );
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
      const MethodChannel('dev.fluttercommunity.plus/wakelock_plus'),
      (MethodCall methodCall) async => true,
    );
  });

  group('BackgroundKeepAliveService Tests', () {
    test('BackgroundKeepAliveService initializes with active state tracking', () {
      final service = BackgroundKeepAliveService();
      expect(service.isKeepAliveActive, isFalse);
      expect(service.isWakelockEnabled, isFalse);
      expect(service.player, isNotNull);
    });

    test('startSilentAudioLoop activates keep alive state with fallback handling', () async {
      final service = BackgroundKeepAliveService();
      expect(service.isKeepAliveActive, isFalse);

      await service.startSilentAudioLoop();
      expect(service.isKeepAliveActive, isTrue);
    });

    test('stopSilentAudioLoop stops keep alive and updates state', () async {
      final service = BackgroundKeepAliveService();
      await service.startSilentAudioLoop();
      expect(service.isKeepAliveActive, isTrue);

      await service.stopSilentAudioLoop();
      expect(service.isKeepAliveActive, isFalse);
    });

    test('setWakelock toggles state gracefully', () async {
      final service = BackgroundKeepAliveService();
      expect(service.isWakelockEnabled, isFalse);

      await service.setWakelock(true);
      expect(service.isWakelockEnabled, isTrue);

      await service.setWakelock(false);
      expect(service.isWakelockEnabled, isFalse);
    });

    test('dispose stops active audio loop and safely tears down resources', () async {
      final service = BackgroundKeepAliveService();
      await service.startSilentAudioLoop();
      expect(service.isKeepAliveActive, isTrue);

      await service.dispose();
      expect(service.isKeepAliveActive, isFalse);
    });
  });
}
