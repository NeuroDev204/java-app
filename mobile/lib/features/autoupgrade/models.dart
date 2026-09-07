import 'dart:convert';

/// Snapshot of native [GameBridge.getUpgradeStatus] JSON.
class NativeUpgradeStatus {
  const NativeUpgradeStatus({
    this.running = false,
    this.connected = false,
    this.inGame = false,
    this.attempts = 0,
    this.success = 0,
    this.fail = 0,
    this.itemLevel = 0,
    this.logs = const <String>[],
  });

  final bool running;
  final bool connected;
  final bool inGame;
  final int attempts;
  final int success;
  final int fail;
  final int itemLevel;
  final List<String> logs;

  factory NativeUpgradeStatus.parse(String raw) {
    if (raw.isEmpty) {
      return const NativeUpgradeStatus();
    }
    try {
      final Object? decoded = jsonDecode(raw);
      if (decoded is! Map) {
        return const NativeUpgradeStatus();
      }
      final Map<String, dynamic> m = Map<String, dynamic>.from(decoded);
      return NativeUpgradeStatus(
        running: m['running'] == true,
        connected: m['connected'] == true,
        inGame: m['in_game'] == true,
        attempts: (m['attempts'] as num?)?.toInt() ?? 0,
        success: (m['success'] as num?)?.toInt() ?? 0,
        fail: (m['fail'] as num?)?.toInt() ?? 0,
        itemLevel: (m['item_level'] as num?)?.toInt() ?? 0,
        logs: (m['logs'] as List<dynamic>?)
                ?.map((dynamic e) => e.toString())
                .toList() ??
            const <String>[],
      );
    } catch (_) {
      return const NativeUpgradeStatus();
    }
  }
}

/// CloverType represents the type of lucky clover used for upgrade enhancement.
enum CloverType {
  none,
  leaf3,
  leaf4,
}

/// EquipmentItem represents an upgradeable piece of equipment in the player's inventory.
class EquipmentItem {
  final int id;
  final String name;
  int level;
  final int type;

  EquipmentItem({
    required this.id,
    required this.name,
    required this.level,
    this.type = 0,
  });

  /// Creates a copy of this equipment item with optional updated fields.
  EquipmentItem copyWith({
    int? id,
    String? name,
    int? level,
    int? type,
  }) {
    return EquipmentItem(
      id: id ?? this.id,
      name: name ?? this.name,
      level: level ?? this.level,
      type: type ?? this.type,
    );
  }

  @override
  String toString() => 'EquipmentItem(id: $id, name: $name, level: +$level, type: $type)';
}

/// AutoUpgradeConfig holds rule parameters for automating equipment upgrades.
class AutoUpgradeConfig {
  final int targetLevel;
  final int clover3Min;
  final int clover3Max;
  final int clover4Min;
  final int clover4Max;
  final bool stopIfOutOfClover;
  final int delayMs;

  const AutoUpgradeConfig({
    required this.targetLevel,
    required this.clover3Min,
    required this.clover3Max,
    required this.clover4Min,
    required this.clover4Max,
    required this.stopIfOutOfClover,
    required this.delayMs,
  });

  /// Factory providing default configuration matching the J2ME client rules.
  factory AutoUpgradeConfig.defaultConfig() {
    return const AutoUpgradeConfig(
      targetLevel: 10,
      clover3Min: 4,
      clover3Max: 7,
      clover4Min: 8,
      clover4Max: 15,
      stopIfOutOfClover: true,
      delayMs: 500,
    );
  }

  /// Determines the lucky clover required for the given [currentLevel].
  CloverType getCloverTypeForLevel(int currentLevel) {
    if (currentLevel >= clover3Min && currentLevel <= clover3Max) {
      return CloverType.leaf3;
    }
    if (currentLevel >= clover4Min && currentLevel <= clover4Max) {
      return CloverType.leaf4;
    }
    return CloverType.none;
  }

  /// Creates a copy of this config with optional replaced properties.
  AutoUpgradeConfig copyWith({
    int? targetLevel,
    int? clover3Min,
    int? clover3Max,
    int? clover4Min,
    int? clover4Max,
    bool? stopIfOutOfClover,
    int? delayMs,
  }) {
    return AutoUpgradeConfig(
      targetLevel: targetLevel ?? this.targetLevel,
      clover3Min: clover3Min ?? this.clover3Min,
      clover3Max: clover3Max ?? this.clover3Max,
      clover4Min: clover4Min ?? this.clover4Min,
      clover4Max: clover4Max ?? this.clover4Max,
      stopIfOutOfClover: stopIfOutOfClover ?? this.stopIfOutOfClover,
      delayMs: delayMs ?? this.delayMs,
    );
  }
}

/// AutoUpgradeStats tracks transactional metrics and spent currencies during an auto-upgrade session.
class AutoUpgradeStats {
  int attemptsCount = 0;
  int successCount = 0;
  int failCount = 0;
  int goldSpent = 0;
  int gemsSpent = 0;

  /// Percentage of successful upgrade attempts (0.0 - 100.0).
  double get successRate =>
      attemptsCount > 0 ? (successCount / attemptsCount) * 100.0 : 0.0;

  /// Resets all counters to initial zero values.
  void reset() {
    attemptsCount = 0;
    successCount = 0;
    failCount = 0;
    goldSpent = 0;
    gemsSpent = 0;
  }
}
