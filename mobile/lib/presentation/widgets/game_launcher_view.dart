import 'package:flutter/material.dart';
import '../../features/game_session/game_session_controller.dart';

/// Interactive UI view rendered inside the game canvas that allows players to
/// configure server connection, login to their Knight Online account, monitor
/// live 24/7 AFK persistence state, and trigger the Auto-Upgrade tool.
class GameLauncherView extends StatefulWidget {
  final GameSessionController sessionController;
  final VoidCallback onOpenAutoUpgrade;

  const GameLauncherView({
    super.key,
    required this.sessionController,
    required this.onOpenAutoUpgrade,
  });

  @override
  State<GameLauncherView> createState() => _GameLauncherViewState();
}

class _GameLauncherViewState extends State<GameLauncherView> {
  final TextEditingController _hostController =
      TextEditingController(text: 'hs1.teamobi.com');
  final TextEditingController _portController =
      TextEditingController(text: '19129');
  final TextEditingController _userController = TextEditingController();
  final TextEditingController _passController = TextEditingController();
  final TextEditingController _charController = TextEditingController();

  bool _obscurePassword = true;

  @override
  void dispose() {
    _hostController.dispose();
    _portController.dispose();
    _userController.dispose();
    _passController.dispose();
    _charController.dispose();
    super.dispose();
  }

  Future<void> _handleLogin() async {
    final username = _userController.text.trim();
    final password = _passController.text.trim();

    if (username.isEmpty || password.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Vui lòng nhập đầy đủ Tài khoản và Mật khẩu!'),
          backgroundColor: Colors.redAccent,
        ),
      );
      return;
    }

    final host = _hostController.text.trim();
    final port = int.tryParse(_portController.text.trim()) ?? 19129;
    final charName = _charController.text.trim();

    await widget.sessionController.connectAndLogin(
      host: host,
      port: port,
      username: username,
      password: password,
      characterName: charName.isNotEmpty ? charName : null,
    );
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: widget.sessionController,
      builder: (BuildContext context, _) {
        final state = widget.sessionController.state;

        if (state == SessionState.connecting ||
            state == SessionState.handshaking ||
            state == SessionState.authenticating) {
          return _buildConnectingView(state);
        }

        if (state == SessionState.inGame) {
          return _buildInGameDashboard();
        }

        return _buildLoginView();
      },
    );
  }

  Widget _buildConnectingView(SessionState state) {
    String stepText = 'Đang kết nối tới máy chủ TeaMobi...';
    if (state == SessionState.handshaking) {
      stepText = 'Đang kiểm tra bắt tay mã hóa HsoCrypto...';
    } else if (state == SessionState.authenticating) {
      stepText = 'Đang xác thực tài khoản & chọn nhân vật...';
    }

    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 24.0),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const CircularProgressIndicator(color: Colors.amberAccent),
            const SizedBox(height: 20),
            Text(
              stepText,
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 15,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              '${widget.sessionController.serverHost}:${widget.sessionController.serverPort}',
              style: const TextStyle(color: Colors.white54, fontSize: 13),
            ),
            const SizedBox(height: 24),
            OutlinedButton.icon(
              onPressed: () => widget.sessionController.disconnect(),
              icon: const Icon(Icons.close, color: Colors.white70),
              label: const Text('Hủy kết nối', style: TextStyle(color: Colors.white70)),
              style: OutlinedButton.styleFrom(
                side: const BorderSide(color: Colors.white24),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildInGameDashboard() {
    return SingleChildScrollView(
      padding: const EdgeInsets.symmetric(horizontal: 20.0, vertical: 32.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Header
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: const Color(0xFF1E1E38),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.tealAccent.withValues(alpha: 0.3)),
            ),
            child: Row(
              children: [
                const Icon(Icons.shield, color: Colors.tealAccent, size: 36),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Text(
                            widget.sessionController.characterName.isNotEmpty
                                ? widget.sessionController.characterName
                                : widget.sessionController.username,
                            style: const TextStyle(
                              color: Colors.white,
                              fontSize: 18,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(
                                horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: Colors.teal.withValues(alpha: 0.2),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: const Text(
                              'ONLINE',
                              style: TextStyle(
                                color: Colors.tealAccent,
                                fontSize: 10,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Máy chủ: ${widget.sessionController.serverHost}:${widget.sessionController.serverPort}',
                        style: const TextStyle(color: Colors.white60, fontSize: 12),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),

          // 24/7 Keep-Alive Badge
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: const Color(0xFF142820),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: Colors.greenAccent.withValues(alpha: 0.4)),
            ),
            child: const Row(
              children: [
                Icon(Icons.bolt, color: Colors.greenAccent, size: 20),
                SizedBox(width: 8),
                Expanded(
                  child: Text(
                    'Treo máy tắt màn hình 24/7: ĐANG HOẠT ĐỘNG (Tiết kiệm pin & Không dừng app)',
                    style: TextStyle(
                      color: Colors.greenAccent,
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 20),

          // Primary Actions
          FilledButton.icon(
            onPressed: widget.onOpenAutoUpgrade,
            icon: const Icon(Icons.auto_awesome, color: Colors.white),
            label: const Text(
              'Mở Tiện Ích Auto-Upgrade',
              style: TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
            ),
            style: FilledButton.styleFrom(
              backgroundColor: const Color(0xFF00BFA5),
              padding: const EdgeInsets.symmetric(vertical: 14),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
            ),
          ),
          const SizedBox(height: 12),

          OutlinedButton.icon(
            onPressed: () => widget.sessionController.disconnect(),
            icon: const Icon(Icons.logout, color: Colors.redAccent, size: 18),
            label: const Text(
              'Đăng Xuất / Ngắt Kết Nối',
              style: TextStyle(color: Colors.redAccent),
            ),
            style: OutlinedButton.styleFrom(
              side: BorderSide(color: Colors.redAccent.withValues(alpha: 0.5)),
              padding: const EdgeInsets.symmetric(vertical: 12),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
            ),
          ),
          const SizedBox(height: 20),

          // Live Activity Logs
          const Text(
            'Nhật ký kết nối thời gian thực:',
            style: TextStyle(
              color: Colors.white70,
              fontSize: 13,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 8),
          Container(
            height: 180,
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.black45,
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: Colors.white12),
            ),
            child: widget.sessionController.activityLogs.isEmpty
                ? const Center(
                    child: Text(
                      'Đang chờ dữ liệu gói tin...',
                      style: TextStyle(color: Colors.white30, fontSize: 12),
                    ),
                  )
                : ListView.builder(
                    itemCount: widget.sessionController.activityLogs.length,
                    itemBuilder: (BuildContext context, int index) {
                      return Padding(
                        padding: const EdgeInsets.symmetric(vertical: 2.0),
                        child: Text(
                          widget.sessionController.activityLogs[index],
                          style: const TextStyle(
                            color: Colors.white70,
                            fontSize: 11,
                            fontFamily: 'monospace',
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }

  Widget _buildLoginView() {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 24.0, vertical: 24.0),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Game Logo / Title
            const Icon(Icons.videogame_asset, size: 54, color: Colors.amberAccent),
            const SizedBox(height: 8),
            const Text(
              'Knight Online J2ME',
              textAlign: TextAlign.center,
              style: TextStyle(
                color: Colors.white,
                fontSize: 22,
                fontWeight: FontWeight.bold,
                letterSpacing: 0.5,
              ),
            ),
            const SizedBox(height: 4),
            const Text(
              'Hiệp Sĩ Online — Treo Máy AFK 24/7',
              textAlign: TextAlign.center,
              style: TextStyle(color: Colors.white60, fontSize: 13),
            ),
            const SizedBox(height: 24),

            if (widget.sessionController.errorMessage != null)
              Container(
                margin: const EdgeInsets.only(bottom: 16),
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: Colors.red.withValues(alpha: 0.2),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: Colors.redAccent.withValues(alpha: 0.4)),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.error_outline, color: Colors.redAccent, size: 20),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        widget.sessionController.errorMessage!,
                        style: const TextStyle(color: Colors.redAccent, fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),

            // Credentials Form
            TextField(
              controller: _userController,
              style: const TextStyle(color: Colors.white),
              decoration: InputDecoration(
                labelText: 'Tài khoản / Email',
                labelStyle: const TextStyle(color: Colors.white70),
                prefixIcon: const Icon(Icons.person, color: Colors.white70),
                filled: true,
                fillColor: const Color(0xFF1E1E38),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide.none,
                ),
              ),
            ),
            const SizedBox(height: 12),

            TextField(
              controller: _passController,
              obscureText: _obscurePassword,
              style: const TextStyle(color: Colors.white),
              decoration: InputDecoration(
                labelText: 'Mật khẩu',
                labelStyle: const TextStyle(color: Colors.white70),
                prefixIcon: const Icon(Icons.lock, color: Colors.white70),
                suffixIcon: IconButton(
                  icon: Icon(
                    _obscurePassword ? Icons.visibility_off : Icons.visibility,
                    color: Colors.white60,
                  ),
                  onPressed: () {
                    setState(() {
                      _obscurePassword = !_obscurePassword;
                    });
                  },
                ),
                filled: true,
                fillColor: const Color(0xFF1E1E38),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide.none,
                ),
              ),
            ),
            const SizedBox(height: 12),

            TextField(
              controller: _charController,
              style: const TextStyle(color: Colors.white),
              decoration: InputDecoration(
                labelText: 'Tên nhân vật (Tùy chọn)',
                labelStyle: const TextStyle(color: Colors.white70),
                prefixIcon: const Icon(Icons.badge, color: Colors.white70),
                filled: true,
                fillColor: const Color(0xFF1E1E38),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide.none,
                ),
              ),
            ),
            const SizedBox(height: 20),

            // Submit Button
            FilledButton.icon(
              onPressed: _handleLogin,
              icon: const Icon(Icons.login, color: Colors.black87),
              label: const Text(
                'VÀO GAME & KẾT NỐI AFK',
                style: TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.bold,
                  color: Colors.black87,
                ),
              ),
              style: FilledButton.styleFrom(
                backgroundColor: Colors.amberAccent,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              ),
            ),
            const SizedBox(height: 12),

            // Auto-upgrade drawer quick link
            TextButton.icon(
              onPressed: widget.onOpenAutoUpgrade,
              icon: const Icon(Icons.build_circle_outlined, color: Colors.white70, size: 18),
              label: const Text(
                'Cấu hình Auto-Upgrade trước khi vào',
                style: TextStyle(color: Colors.white70, fontSize: 13),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
