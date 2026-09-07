import 'dart:async';

import '../../core/bridge/game_bridge.dart';
import 'models.dart';

/// AutoUpgradeController coordinates the automated upgrade state machine,
/// clover rules, currency accounting, and transaction logging.
///
/// When a [GameBridge] is supplied, Start/Stop go through the native channel
/// only — this controller never opens a Dart game socket.
class AutoUpgradeController {
  AutoUpgradeController({GameBridge? bridge}) : _bridge = bridge;

  final GameBridge? _bridge;

  bool _isRunning = false;
  EquipmentItem? _targetItem;
  AutoUpgradeConfig? _config;
  NativeUpgradeStatus? nativeStatus;
  final AutoUpgradeStats stats = AutoUpgradeStats();
  final List<String> logs = <String>[];
  final StreamController<String> _logController =
      StreamController<String>.broadcast();

  /// Whether the auto-upgrade routine is currently active.
  bool get isRunning => _isRunning;

  /// Currently targeted equipment piece, or null if unassigned.
  EquipmentItem? get targetItem => _targetItem;

  /// Current active upgrade configuration.
  AutoUpgradeConfig? get config => _config;

  /// Broadcast stream emitting live log notifications for UI consumers.
  Stream<String> get logStream => _logController.stream;

  /// True when native status reports an in-game, connected session.
  bool get canStartNative {
    final NativeUpgradeStatus? status = nativeStatus;
    return status != null && status.inGame && status.connected;
  }

  /// Pulls [GameBridge.getUpgradeStatus] into [nativeStatus] and live stats.
  Future<void> refreshNativeStatus() async {
    if (_bridge == null) {
      nativeStatus = null;
      return;
    }
    try {
      final String raw = await _bridge!.getUpgradeStatus();
      final NativeUpgradeStatus parsed = NativeUpgradeStatus.parse(raw);
      nativeStatus = parsed;
      _isRunning = parsed.running;
      stats.attemptsCount = parsed.attempts;
      stats.successCount = parsed.success;
      stats.failCount = parsed.fail;
      if (_targetItem != null) {
        _targetItem!.level = parsed.itemLevel;
      }
    } catch (_) {
      return;
    }
  }

  /// Starts the auto-upgrade routine targeting [targetItem] with [config].
  ///
  /// With a [GameBridge], native `startUpgrade` must succeed (empty string)
  /// before [isRunning] becomes true. A non-empty error is logged and the
  /// controller stays stopped.
  Future<void> start({
    required EquipmentItem targetItem,
    required AutoUpgradeConfig config,
  }) async {
    _targetItem = targetItem;
    _config = config;
    final String json =
        '{"item_id":${targetItem.id},"item_type":${targetItem.type},"target_level":${config.targetLevel},"clover3_min":${config.clover3Min},"clover3_max":${config.clover3Max},"clover4_min":${config.clover4Min},"clover4_max":${config.clover4Max},"stop_if_out_of_clover":${config.stopIfOutOfClover},"delay_ms":${config.delayMs}}';
    if (_bridge != null) {
      final String err = await _bridge!.startUpgrade(json);
      if (err.isNotEmpty) {
        _isRunning = false;
        _addLog(err);
        await refreshNativeStatus();
        return;
      }
    }
    _isRunning = true;
    stats.reset();
    _addLog(
      'Bắt đầu Auto-Upgrade cho [${targetItem.name}] lên cấp +${config.targetLevel}',
    );
    if (_bridge != null) {
      await refreshNativeStatus();
    }
  }

  /// Records an upgrade attempt result and advances the controller state machine.
  void recordResult({
    required bool success,
    required int newLevel,
    int goldSpent = 0,
    int gemsSpent = 0,
  }) {
    stats.attemptsCount++;
    stats.goldSpent += goldSpent;
    stats.gemsSpent += gemsSpent;

    if (success) {
      stats.successCount++;
      if (_targetItem != null) {
        _targetItem!.level = newLevel;
      }
      _addLog('Thành công: Cấp độ hiện tại +$newLevel');
      if (_config != null && newLevel >= _config!.targetLevel) {
        _addLog('Đã đạt cấp độ mục tiêu +${_config!.targetLevel}! Dừng Auto.');
        stop();
      }
    } else {
      stats.failCount++;
      if (_targetItem != null && newLevel >= 0) {
        _targetItem!.level = newLevel;
      }
      _addLog('Thất bại: Giữ nguyên hoặc tụt cấp.');
    }
  }

  /// Halts the auto-upgrade loop.
  void stop() {
    _isRunning = false;
    _bridge?.stopUpgrade();
    _addLog('Đã dừng Auto-Upgrade.');
  }

  /// Disposes of active stream controllers and releases listeners.
  void dispose() {
    stop();
    _logController.close();
  }

  void _addLog(String msg) {
    logs.insert(
      0,
      '[${DateTime.now().toIso8601String().substring(11, 19)}] $msg',
    );
    if (logs.length > 100) {
      logs.removeLast();
    }
    if (!_logController.isClosed) {
      _logController.add(msg);
    }
  }
}
