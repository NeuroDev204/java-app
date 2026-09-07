import 'dart:async';

import 'package:flutter/material.dart';

import '../../features/autoupgrade/controller.dart';
import '../../features/autoupgrade/models.dart';

/// Modal bottom sheet providing interactive controls, lucky clover policies,
/// live execution metrics, and real-time transaction event logs for the Auto-Upgrade tool.
class AutoUpgradeSheet extends StatefulWidget {
  final AutoUpgradeController controller;

  const AutoUpgradeSheet({super.key, required this.controller});

  @override
  State<AutoUpgradeSheet> createState() => _AutoUpgradeSheetState();
}

class _AutoUpgradeSheetState extends State<AutoUpgradeSheet> {
  int _targetLevel = 10;
  bool _stopIfOutOfClover = true;
  StreamSubscription<String>? _logSubscription;
  Timer? _statusTimer;

  @override
  void initState() {
    super.initState();
    _targetLevel = widget.controller.config?.targetLevel ?? 10;
    _stopIfOutOfClover = widget.controller.config?.stopIfOutOfClover ?? true;
    _logSubscription = widget.controller.logStream.listen((_) {
      if (mounted) {
        setState(() {});
      }
    });
    _statusTimer = Timer.periodic(const Duration(milliseconds: 300), (_) {
      unawaited(_pollStatus());
    });
    unawaited(_pollStatus());
  }

  Future<void> _pollStatus() async {
    await widget.controller.refreshNativeStatus();
    if (mounted) {
      setState(() {});
    }
  }

  @override
  void dispose() {
    _statusTimer?.cancel();
    _logSubscription?.cancel();
    super.dispose();
  }

  Future<void> _toggleUpgradeState() async {
    if (widget.controller.isRunning) {
      widget.controller.stop();
      await widget.controller.refreshNativeStatus();
      if (mounted) setState(() {});
      return;
    }
    await widget.controller.start(
      targetItem: widget.controller.targetItem ??
          EquipmentItem(id: 1, name: 'Trang bị chọn', level: 6),
      config: AutoUpgradeConfig(
        targetLevel: _targetLevel,
        clover3Min: 4,
        clover3Max: 7,
        clover4Min: 8,
        clover4Max: 15,
        stopIfOutOfClover: _stopIfOutOfClover,
        delayMs: 500,
      ),
    );
    await widget.controller.refreshNativeStatus();
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final NativeUpgradeStatus? native = widget.controller.nativeStatus;
    final stats = widget.controller.stats;
    final bool isRunning = native?.running ?? widget.controller.isRunning;
    final int attempts = native?.attempts ?? stats.attemptsCount;
    final int success = native?.success ?? stats.successCount;
    final int fail = native?.fail ?? stats.failCount;
    final int itemLevel = native?.itemLevel ?? widget.controller.targetItem?.level ?? 0;
    final List<String> logs =
        (native != null && native.logs.isNotEmpty) ? native.logs : widget.controller.logs;
    final bool connected = native?.connected ?? false;
    final bool inGame = native?.inGame ?? false;
    final bool canPress = isRunning || widget.controller.canStartNative;
    final double successRate =
        attempts > 0 ? (success / attempts) * 100.0 : 0.0;

    return Material(
      color: const Color(0xFF1E1E2C),
      borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
      child: Container(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(context).size.height * 0.75,
        ),
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 20),
        child: SafeArea(
          top: false,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Header bar
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  const Row(
                    children: [
                      Icon(Icons.auto_awesome, color: Colors.amber, size: 20),
                      SizedBox(width: 8),
                      Text(
                        'Auto-Upgrade Tiện Ích',
                        style: TextStyle(
                          color: Colors.white,
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ],
                  ),
                  IconButton(
                    icon: const Icon(Icons.close, color: Colors.white70),
                    onPressed: () => Navigator.of(context).pop(),
                  ),
                ],
              ),
              const Divider(color: Colors.white24, height: 1),
              const SizedBox(height: 12),

              // Controls: Target Level & Safety Clover
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  const Text(
                    'Cấp độ đích: ',
                    style: TextStyle(color: Colors.white70, fontSize: 15),
                  ),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                    decoration: BoxDecoration(
                      color: const Color(0xFF2A2A3E),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: Colors.white24),
                    ),
                    child: DropdownButton<int>(
                      value: _targetLevel,
                      underline: const SizedBox(),
                      dropdownColor: const Color(0xFF2A2A3E),
                      items: const [7, 8, 9, 10, 11, 12, 13, 14, 15]
                          .map((int lvl) => DropdownMenuItem<int>(
                                value: lvl,
                                child: Text(
                                  '+$lvl',
                                  style: const TextStyle(
                                    color: Colors.amberAccent,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ))
                          .toList(),
                      onChanged: isRunning
                          ? null
                          : (int? val) {
                              if (val != null) {
                                setState(() => _targetLevel = val);
                              }
                            },
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text(
                  'Dừng nếu hết cỏ bảo hiểm',
                  style: TextStyle(color: Colors.white70, fontSize: 14),
                ),
                subtitle: const Text(
                  'Tự động ngắt khi thiếu cỏ 3 lá (+4..+7) hoặc 4 lá (+8..+15)',
                  style: TextStyle(color: Colors.white38, fontSize: 12),
                ),
                value: _stopIfOutOfClover,
                activeThumbColor: Colors.tealAccent,
                onChanged: isRunning
                    ? null
                    : (bool val) => setState(() => _stopIfOutOfClover = val),
              ),
              const SizedBox(height: 12),
              Text(
                '${connected ? 'Đã kết nối' : 'Mất kết nối'} · ${inGame ? 'Trong game' : 'Chưa vào game'}',
                style: TextStyle(
                  color: inGame && connected ? Colors.tealAccent : Colors.white38,
                  fontSize: 12,
                ),
              ),
              const SizedBox(height: 8),

              // Start / Stop Button
              ElevatedButton.icon(
                style: ElevatedButton.styleFrom(
                  backgroundColor: isRunning ? Colors.redAccent : Colors.teal,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(10),
                  ),
                ),
                icon: Icon(
                  isRunning ? Icons.stop : Icons.play_arrow,
                  color: Colors.white,
                ),
                label: Text(
                  isRunning ? 'Dừng Auto' : 'Bắt Đầu Treo Auto',
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 16,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                onPressed: canPress ? _toggleUpgradeState : null,
              ),
              const SizedBox(height: 16),

              // Live Metrics / Statistics Bar
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: const Color(0xFF141422),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: Colors.white12),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceAround,
                  children: [
                    _buildStatItem('Lần đập', '$attempts'),
                    _buildStatItem('Thành công', '$success', color: Colors.greenAccent),
                    _buildStatItem('Thất bại', '$fail', color: Colors.redAccent),
                    _buildStatItem('Tỉ lệ', '${successRate.toStringAsFixed(1)}%', color: Colors.amberAccent),
                    _buildStatItem('Cấp', '+$itemLevel'),
                  ],
                ),
              ),
              const SizedBox(height: 12),

              // Live Transaction Logs
              const Text(
                'Nhật ký giao dịch:',
                style: TextStyle(
                  color: Colors.white70,
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 6),
              Expanded(
                child: Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Colors.black45,
                    borderRadius: BorderRadius.circular(6),
                    border: Border.all(color: Colors.white10),
                  ),
                  child: logs.isEmpty
                      ? const Center(
                          child: Text(
                            'Chưa có dữ liệu nâng cấp.',
                            style: TextStyle(color: Colors.white38, fontSize: 12),
                          ),
                        )
                      : ListView.builder(
                          itemCount: logs.length,
                          itemBuilder: (context, index) {
                            final log = logs[index];
                            return Padding(
                              padding: const EdgeInsets.symmetric(vertical: 2),
                              child: Text(
                                log,
                                style: TextStyle(
                                  color: log.contains('Thành công')
                                      ? Colors.greenAccent
                                      : log.contains('Thất bại')
                                          ? Colors.redAccent
                                          : Colors.white70,
                                  fontSize: 11,
                                  fontFamily: 'monospace',
                                ),
                              ),
                            );
                          },
                        ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildStatItem(String label, String value, {Color color = Colors.white}) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          value,
          style: TextStyle(
            color: color,
            fontWeight: FontWeight.bold,
            fontSize: 13,
          ),
        ),
        const SizedBox(height: 2),
        Text(
          label,
          style: const TextStyle(
            color: Colors.white54,
            fontSize: 10,
          ),
        ),
      ],
    );
  }
}
