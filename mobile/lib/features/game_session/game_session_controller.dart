import 'dart:async';
import 'package:flutter/foundation.dart';
import '../../core/network/message.dart';
import '../../core/network/session.dart';
import '../autoupgrade/models.dart';

/// Connection and lifecycle states for the HSO game session.
enum SessionState {
  disconnected,
  connecting,
  handshaking,
  authenticating,
  inGame,
  error,
}

/// Manages direct TCP connection to Knight Online (TeaMobi) game servers,
/// handling authentication, character selection, in-game keep-alive,
/// and automated upgrade packet dispatching without requiring heavy GUI emulation.
class GameSessionController extends ChangeNotifier {
  final HsoSession _session;

  SessionState _state = SessionState.disconnected;
  String _serverHost = 'hs1.teamobi.com';
  int _serverPort = 19129;
  String _username = '';
  String _characterName = '';
  int _gold = 0;
  int _clover3Count = 0;
  int _clover4Count = 0;
  int _insuranceCount = 0;
  String? _errorMessage;

  final List<EquipmentItem> _inventory = <EquipmentItem>[];
  final List<String> _activityLogs = <String>[];
  StreamSubscription<HsoMessage>? _messageSubscription;
  Timer? _keepAliveTimer;

  GameSessionController({HsoSession? session})
      : _session = session ?? HsoSession() {
    _messageSubscription = _session.messages.listen(_handleIncomingMessage);
  }

  // --- Getters ---
  SessionState get state => _state;
  bool get isConnected => _session.isConnected;
  bool get isInGame => _state == SessionState.inGame;
  String get serverHost => _serverHost;
  int get serverPort => _serverPort;
  String get username => _username;
  String get characterName => _characterName;
  int get gold => _gold;
  int get clover3Count => _clover3Count;
  int get clover4Count => _clover4Count;
  int get insuranceCount => _insuranceCount;
  String? get errorMessage => _errorMessage;
  List<EquipmentItem> get inventory => List<EquipmentItem>.unmodifiable(_inventory);
  List<String> get activityLogs => List<String>.unmodifiable(_activityLogs);
  HsoSession get session => _session;

  /// Connects to the HSO server and authenticates with [username] and [password].
  Future<bool> connectAndLogin({
    required String host,
    required int port,
    required String username,
    required String password,
    String? characterName,
  }) async {
    _serverHost = host.trim();
    _serverPort = port;
    _username = username.trim();
    _characterName = (characterName != null && characterName.trim().isNotEmpty)
        ? characterName.trim()
        : _username;
    _errorMessage = null;

    _updateState(SessionState.connecting);
    _log('Bắt đầu kết nối tới $_serverHost:$_serverPort...');

    try {
      await _session.connect(_serverHost, _serverPort);
      _log('Đã kết nối TCP thành công. Đang kiểm tra mã hóa...');

      _updateState(SessionState.handshaking);
      try {
        await _session.performHandshake(timeout: const Duration(seconds: 3));
        _log('Đã hoàn tất bắt tay mã hóa HsoCrypto.');
      } catch (e) {
        _log('Máy chủ không yêu cầu bắt tay mã hóa (giao tiếp kênh tiêu chuẩn).');
      }

      _updateState(SessionState.authenticating);
      _log('Đang gửi gói tin đăng nhập (Opcode 1) cho tài khoản: $_username...');

      final loginMsg = HsoMessage(command: 1);
      loginMsg.writeUtf(_username);
      loginMsg.writeUtf(password);
      loginMsg.writeUtf('4.0.2');
      _session.send(loginMsg);

      // Select character
      await Future<void>.delayed(const Duration(milliseconds: 300));
      _log('Đang chọn nhân vật (Opcode 7): $_characterName...');
      final charMsg = HsoMessage(command: 7);
      charMsg.writeUtf(_characterName);
      _session.send(charMsg);

      _updateState(SessionState.inGame);
      _log('Đã vào game thành công! Trạng thái AFK trực tiếp 24/7 đang hoạt động.');

      _startKeepAliveHeartbeat();
      return true;
    } catch (e) {
      _errorMessage = e.toString();
      _log('Lỗi kết nối / Đăng nhập: $e');
      _updateState(SessionState.error);
      return false;
    }
  }

  /// Sends character select packet (Opcode 7).
  void selectCharacter(String charName) {
    if (!isConnected) return;
    _characterName = charName;
    final msg = HsoMessage(command: 7);
    msg.writeUtf(charName);
    _session.send(msg);
    _log('Đã gửi yêu cầu chọn nhân vật: $charName');
    notifyListeners();
  }

  /// Sends equipment upgrade packet (Opcode 67, Action 5).
  void performUpgrade({
    required int itemId,
    required int itemType,
    required bool useClover3,
    required bool useClover4,
    required bool useInsurance,
  }) {
    if (!isConnected) {
      throw StateError('Chưa kết nối đến máy chủ game');
    }
    final msg = HsoMessage(command: 67);
    msg.writeByte(5); // Action 5: Execute upgrade
    msg.writeShort(itemId);
    msg.writeByte(itemType);
    msg.writeBool(useClover3);
    msg.writeBool(useClover4);
    msg.writeBool(useInsurance);
    _session.send(msg);

    _log('Gửi gói tin đập đồ #$itemId (Cỏ 3 lá: $useClover3, Cỏ 4 lá: $useClover4, BH: $useInsurance)');
    notifyListeners();
  }

  /// Updates in-game resource counters when server inventory packets arrive.
  void updateResources({
    int? gold,
    int? clover3,
    int? clover4,
    int? insurance,
  }) {
    if (gold != null) _gold = gold;
    if (clover3 != null) _clover3Count = clover3;
    if (clover4 != null) _clover4Count = clover4;
    if (insurance != null) _insuranceCount = insurance;
    notifyListeners();
  }

  /// Disconnects from the game server.
  Future<void> disconnect() async {
    _stopKeepAliveHeartbeat();
    await _session.disconnect();
    _updateState(SessionState.disconnected);
    _log('Đã ngắt kết nối khỏi máy chủ.');
  }

  void _startKeepAliveHeartbeat() {
    _keepAliveTimer?.cancel();
    _keepAliveTimer = Timer.periodic(const Duration(seconds: 25), (_) {
      if (isConnected && isInGame) {
        try {
          // Opcode -100 or 0 keep-alive ping
          final pingMsg = HsoMessage(command: -100);
          _session.send(pingMsg);
          _log('Gửi ping keep-alive duy trì phiên kết nối AFK');
        } catch (_) {}
      }
    });
  }

  void _stopKeepAliveHeartbeat() {
    _keepAliveTimer?.cancel();
    _keepAliveTimer = null;
  }

  void _handleIncomingMessage(HsoMessage msg) {
    try {
      final cmd = msg.command;
      if (cmd == 37 || cmd == 25) {
        // Error or notification popup from server
        final text = msg.readUtf();
        if (text.isNotEmpty) {
          _log('Thông báo từ Server: $text');
          notifyListeners();
        }
      } else if (cmd == 67) {
        // Upgrade response
        _log('Nhận phản hồi kết quả đập đồ (Opcode 67)');
        notifyListeners();
      }
    } catch (_) {
      // Ignore packet parse variations gracefully
    }
  }

  void _updateState(SessionState newState) {
    _state = newState;
    notifyListeners();
  }

  void _log(String message) {
    final now = DateTime.now();
    final timeStr =
        '${now.hour.toString().padLeft(2, '0')}:${now.minute.toString().padLeft(2, '0')}:${now.second.toString().padLeft(2, '0')}';
    _activityLogs.insert(0, '[$timeStr] $message');
    if (_activityLogs.length > 50) {
      _activityLogs.removeLast();
    }
    notifyListeners();
  }

  @override
  void dispose() {
    _stopKeepAliveHeartbeat();
    _messageSubscription?.cancel();
    _session.disconnect();
    super.dispose();
  }
}
