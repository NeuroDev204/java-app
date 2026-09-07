import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/network/message.dart';
import 'package:knight_online/core/network/session.dart';
import 'package:knight_online/features/game_session/game_session_controller.dart';

void main() {
  group('GameSessionController Tests', () {
    test('initial state is disconnected with default server settings', () {
      final session = HsoSession();
      final controller = GameSessionController(session: session);

      expect(controller.state, equals(SessionState.disconnected));
      expect(controller.isConnected, isFalse);
      expect(controller.isInGame, isFalse);
      expect(controller.serverHost, equals('hs1.teamobi.com'));
      expect(controller.serverPort, equals(19129));
      expect(controller.activityLogs, isEmpty);
      expect(controller.errorMessage, isNull);

      controller.dispose();
    });

    test('disconnect resets session state and logs event', () async {
      final session = HsoSession();
      final controller = GameSessionController(session: session);

      await controller.disconnect();
      expect(controller.state, equals(SessionState.disconnected));
      expect(controller.activityLogs.first, contains('Đã ngắt kết nối'));

      controller.dispose();
    });

    test('handles incoming server notification packets', () async {
      final session = HsoSession();
      final controller = GameSessionController(session: session);

      // Simulate incoming opcode 25 (Notification/Chat) with UTF message
      final msg = HsoMessage(command: 25);
      msg.writeUtf('Chào mừng bạn đến với Hiệp Sĩ Online!');
      final packet = HsoSession.encodePacket(msg);

      session.handleData(packet);
      await Future<void>.delayed(Duration.zero);

      expect(controller.activityLogs, isNotEmpty);
      expect(controller.activityLogs.first, contains('Chào mừng bạn đến với Hiệp Sĩ Online!'));

      controller.dispose();
    });

    test('handles incoming upgrade response packet (Opcode 67)', () async {
      final session = HsoSession();
      final controller = GameSessionController(session: session);

      final msg = HsoMessage(command: 67);
      msg.writeByte(1); // Success code
      final packet = HsoSession.encodePacket(msg);

      session.handleData(packet);
      await Future<void>.delayed(Duration.zero);

      expect(controller.activityLogs, isNotEmpty);
      expect(controller.activityLogs.first, contains('kết quả đập đồ'));

      controller.dispose();
    });

    test('connectAndLogin handles connection error gracefully', () async {
      final session = HsoSession();
      final controller = GameSessionController(session: session);

      // Connect to unreachable local port
      final result = await controller.connectAndLogin(
        host: '127.0.0.1',
        port: 59999,
        username: 'test_player',
        password: 'secret_password',
      );

      expect(result, isFalse);
      expect(controller.state, equals(SessionState.error));
      expect(controller.errorMessage, isNotNull);
      expect(controller.activityLogs.first, contains('Lỗi kết nối'));

      controller.dispose();
    });
  });
}
