import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/bridge/game_bridge.dart';
import 'package:knight_online/features/autoupgrade/models.dart';
import 'package:knight_online/features/autoupgrade/controller.dart';

class _FakeGameBridge extends GameBridge {
  final List<String> startUpgradeCalls = <String>[];
  int stopUpgradeCalls = 0;
  String startUpgradeResult = '';
  String statusJSON =
      '{"running":false,"connected":true,"in_game":true,"attempts":0,"success":0,"fail":0,"item_level":6,"logs":[]}';

  @override
  Future<String> startUpgrade(String configJSON) async {
    startUpgradeCalls.add(configJSON);
    if (startUpgradeResult.isEmpty) {
      statusJSON =
          '{"running":true,"connected":true,"in_game":true,"attempts":0,"success":0,"fail":0,"item_level":6,"logs":["[12:00:00] Bat dau Auto-Upgrade +10"]}';
    }
    return startUpgradeResult;
  }

  @override
  Future<void> stopUpgrade() async {
    stopUpgradeCalls++;
    statusJSON =
        '{"running":false,"connected":true,"in_game":true,"attempts":0,"success":0,"fail":0,"item_level":6,"logs":[]}';
  }

  @override
  Future<String> getUpgradeStatus() async => statusJSON;
}

void main() {
  group('AutoUpgradeConfig & Clover Selection', () {
    test('AutoUpgradeConfig properly determines lucky clover to use', () {
      const config = AutoUpgradeConfig(
        targetLevel: 10,
        clover3Min: 4,
        clover3Max: 7,
        clover4Min: 8,
        clover4Max: 15,
        stopIfOutOfClover: true,
        delayMs: 500,
      );

      expect(config.getCloverTypeForLevel(3), equals(CloverType.none));
      expect(config.getCloverTypeForLevel(4), equals(CloverType.leaf3));
      expect(config.getCloverTypeForLevel(7), equals(CloverType.leaf3));
      expect(config.getCloverTypeForLevel(8), equals(CloverType.leaf4));
      expect(config.getCloverTypeForLevel(12), equals(CloverType.leaf4));
    });

    test('AutoUpgradeConfig default and boundaries', () {
      final config = AutoUpgradeConfig.defaultConfig();
      expect(config.targetLevel, equals(10));
      expect(config.clover3Min, equals(4));
      expect(config.clover3Max, equals(7));
      expect(config.clover4Min, equals(8));
      expect(config.clover4Max, equals(15));
      expect(config.stopIfOutOfClover, isTrue);
      expect(config.delayMs, equals(500));

      expect(config.getCloverTypeForLevel(0), equals(CloverType.none));
      expect(config.getCloverTypeForLevel(3), equals(CloverType.none));
      expect(config.getCloverTypeForLevel(4), equals(CloverType.leaf3));
      expect(config.getCloverTypeForLevel(5), equals(CloverType.leaf3));
      expect(config.getCloverTypeForLevel(7), equals(CloverType.leaf3));
      expect(config.getCloverTypeForLevel(8), equals(CloverType.leaf4));
      expect(config.getCloverTypeForLevel(15), equals(CloverType.leaf4));
      expect(config.getCloverTypeForLevel(16), equals(CloverType.none));

      final updated = config.copyWith(targetLevel: 12, delayMs: 250);
      expect(updated.targetLevel, equals(12));
      expect(updated.delayMs, equals(250));
      expect(updated.clover3Min, equals(4));
    });
  });

  group('EquipmentItem & AutoUpgradeStats', () {
    test('EquipmentItem updates level and copyWith works', () {
      final item = EquipmentItem(id: 101, name: 'Kiếm Huyết Thần', level: 6);
      expect(item.id, equals(101));
      expect(item.name, equals('Kiếm Huyết Thần'));
      expect(item.level, equals(6));

      item.level = 7;
      expect(item.level, equals(7));

      final clone = item.copyWith(level: 8);
      expect(clone.id, equals(101));
      expect(clone.name, equals('Kiếm Huyết Thần'));
      expect(clone.level, equals(8));
      expect(item.toString(), contains('Kiếm Huyết Thần'));
    });

    test(
        'AutoUpgradeStats tracks attempts, currency, and calculates success rate',
        () {
      final stats = AutoUpgradeStats();
      expect(stats.attemptsCount, equals(0));
      expect(stats.successCount, equals(0));
      expect(stats.failCount, equals(0));
      expect(stats.goldSpent, equals(0));
      expect(stats.gemsSpent, equals(0));
      expect(stats.successRate, equals(0.0));

      stats.attemptsCount = 4;
      stats.successCount = 3;
      stats.failCount = 1;
      stats.goldSpent = 12000;
      stats.gemsSpent = 50;
      expect(stats.successRate, equals(75.0));

      stats.reset();
      expect(stats.attemptsCount, equals(0));
      expect(stats.successCount, equals(0));
      expect(stats.failCount, equals(0));
      expect(stats.goldSpent, equals(0));
      expect(stats.gemsSpent, equals(0));
      expect(stats.successRate, equals(0.0));
    });
  });

  group('AutoUpgradeController Lifecycle & State Machine', () {
    test('AutoUpgradeController transitions state and tracks attempts',
        () async {
      final bridge = _FakeGameBridge();
      final controller = AutoUpgradeController(bridge: bridge);
      expect(controller.isRunning, isFalse);

      await controller.start(
        targetItem: EquipmentItem(id: 1, name: 'Kiếm Huyết Thần', level: 6),
        config: AutoUpgradeConfig.defaultConfig(),
      );
      expect(controller.isRunning, isTrue);
      expect(bridge.startUpgradeCalls, hasLength(1));
      expect(bridge.startUpgradeCalls.single, contains('"item_id":1'));
      expect(bridge.startUpgradeCalls.single, contains('"item_type":0'));
      expect(bridge.startUpgradeCalls.single, contains('"target_level":10'));
      expect(bridge.startUpgradeCalls.single, contains('"clover3_min":4'));
      expect(bridge.startUpgradeCalls.single, contains('"clover3_max":7'));
      expect(bridge.startUpgradeCalls.single, contains('"clover4_min":8'));
      expect(bridge.startUpgradeCalls.single, contains('"clover4_max":15'));
      expect(
        bridge.startUpgradeCalls.single,
        contains('"stop_if_out_of_clover":true'),
      );
      expect(bridge.startUpgradeCalls.single, contains('"delay_ms":500'));

      controller.recordResult(success: true, newLevel: 7);
      expect(controller.stats.successCount, equals(1));
      expect(controller.stats.attemptsCount, equals(1));

      controller.stop();
      expect(controller.isRunning, isFalse);
      expect(bridge.stopUpgradeCalls, equals(1));
    });

    test('NativeUpgradeStatus.parse reads Go HUD fields', () {
      final NativeUpgradeStatus status = NativeUpgradeStatus.parse(
        '{"running":true,"connected":true,"in_game":true,"attempts":3,"success":1,"fail":2,"item_level":7,"logs":["ok"]}',
      );
      expect(status.running, isTrue);
      expect(status.connected, isTrue);
      expect(status.inGame, isTrue);
      expect(status.attempts, 3);
      expect(status.success, 1);
      expect(status.fail, 2);
      expect(status.itemLevel, 7);
      expect(status.logs, <String>['ok']);
    });

    test('refreshNativeStatus copies Go stats and gates canStartNative',
        () async {
      final bridge = _FakeGameBridge()
        ..statusJSON =
            '{"running":false,"connected":false,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}';
      final controller = AutoUpgradeController(bridge: bridge);
      await controller.refreshNativeStatus();
      expect(controller.canStartNative, isFalse);

      bridge.statusJSON =
          '{"running":false,"connected":true,"in_game":true,"attempts":4,"success":1,"fail":3,"item_level":8,"logs":["go-log"]}';
      await controller.refreshNativeStatus();
      expect(controller.canStartNative, isTrue);
      expect(controller.stats.attemptsCount, 4);
      expect(controller.stats.successCount, 1);
      expect(controller.stats.failCount, 3);
      expect(controller.nativeStatus?.logs, <String>['go-log']);
    });

    test('start with GameBridge error keeps isRunning false and logs it',
        () async {
      final bridge = _FakeGameBridge()..startUpgradeResult = 'not in game';
      final controller = AutoUpgradeController(bridge: bridge);

      await controller.start(
        targetItem: EquipmentItem(id: 1, name: 'Kiếm Huyết Thần', level: 6),
        config: AutoUpgradeConfig.defaultConfig(),
      );

      expect(controller.isRunning, isFalse);
      expect(bridge.startUpgradeCalls, hasLength(1));
      expect(controller.logs, anyElement(contains('not in game')));
    });

    test('AutoUpgradeController handles failure results properly', () {
      final controller = AutoUpgradeController();
      final item = EquipmentItem(id: 2, name: 'Giáp Hiệp Sĩ', level: 5);
      controller.start(
        targetItem: item,
        config: AutoUpgradeConfig.defaultConfig(),
      );

      controller.recordResult(
          success: false, newLevel: 4, goldSpent: 2000, gemsSpent: 5);
      expect(controller.stats.attemptsCount, equals(1));
      expect(controller.stats.failCount, equals(1));
      expect(controller.stats.successCount, equals(0));
      expect(controller.stats.goldSpent, equals(2000));
      expect(controller.stats.gemsSpent, equals(5));
      expect(item.level, equals(4));
      expect(controller.isRunning, isTrue);

      controller.stop();
      expect(controller.isRunning, isFalse);
    });

    test(
        'AutoUpgradeController automatically stops when target level is reached',
        () {
      final controller = AutoUpgradeController();
      final item = EquipmentItem(id: 3, name: 'Cung Thần Sấm', level: 9);
      controller.start(
        targetItem: item,
        config: const AutoUpgradeConfig(
          targetLevel: 10,
          clover3Min: 4,
          clover3Max: 7,
          clover4Min: 8,
          clover4Max: 15,
          stopIfOutOfClover: true,
          delayMs: 500,
        ),
      );
      expect(controller.isRunning, isTrue);

      controller.recordResult(success: true, newLevel: 10);
      expect(item.level, equals(10));
      expect(controller.stats.successCount, equals(1));
      expect(controller.isRunning, isFalse);
      expect(
          controller.logs, anyElement(contains('Đã đạt cấp độ mục tiêu +10')));
    });

    test('AutoUpgradeController emits to logStream and enforces log capacity',
        () async {
      final controller = AutoUpgradeController();
      final emittedLogs = <String>[];
      final subscription = controller.logStream.listen(emittedLogs.add);

      controller.start(
        targetItem: EquipmentItem(id: 4, name: 'Gậy Ma Thuật', level: 1),
        config: AutoUpgradeConfig.defaultConfig(),
      );

      for (int i = 1; i <= 120; i++) {
        controller.recordResult(success: false, newLevel: 1);
      }

      await Future<void>.delayed(Duration.zero);
      expect(controller.logs.length, equals(100));
      expect(emittedLogs.length, equals(121));

      await subscription.cancel();
      controller.dispose();
      expect(controller.isRunning, isFalse);
    });
  });
}
