import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/bridge/game_bridge.dart';
import 'package:knight_online/features/autoupgrade/controller.dart';
import 'package:knight_online/features/autoupgrade/models.dart';
import 'package:knight_online/presentation/widgets/floating_hud.dart';
import 'package:knight_online/presentation/widgets/autoupgrade_sheet.dart';

class _FakeGameBridge extends GameBridge {
  String startUpgradeResult = '';
  String statusJSON =
      '{"running":false,"connected":true,"in_game":true,"attempts":0,"success":0,"fail":0,"item_level":6,"logs":[]}';

  @override
  Future<String> startUpgrade(String configJSON) async {
    if (startUpgradeResult.isEmpty) {
      statusJSON =
          '{"running":true,"connected":true,"in_game":true,"attempts":0,"success":0,"fail":0,"item_level":6,"logs":["[12:00:00] Bat dau Auto-Upgrade +10"]}';
    }
    return startUpgradeResult;
  }

  @override
  Future<void> stopUpgrade() async {
    statusJSON =
        '{"running":false,"connected":true,"in_game":true,"attempts":0,"success":0,"fail":0,"item_level":6,"logs":[]}';
  }

  @override
  Future<String> getUpgradeStatus() async => statusJSON;
}

void main() {
  group('FloatingHudButton Tests', () {
    testWidgets('FloatingHudButton renders and triggers onOpen callback', (WidgetTester tester) async {
      bool opened = false;
      final controller = AutoUpgradeController();
      addTearDown(controller.dispose);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Stack(
              children: [
                FloatingHudButton(
                  controller: controller,
                  onOpen: () => opened = true,
                ),
              ],
            ),
          ),
        ),
      );

      final buttonFinder = find.byType(FloatingHudButton);
      expect(buttonFinder, findsOneWidget);

      await tester.tap(buttonFinder);
      expect(opened, isTrue);
    });

    testWidgets('FloatingHudButton updates position on pan drag', (WidgetTester tester) async {
      final controller = AutoUpgradeController();
      addTearDown(controller.dispose);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Stack(
              children: [
                FloatingHudButton(
                  controller: controller,
                  onOpen: () {},
                ),
              ],
            ),
          ),
        ),
      );

      final Finder buttonFinder = find.byType(FloatingActionButton);
      expect(buttonFinder, findsOneWidget);

      final Offset initialPos = tester.getTopLeft(buttonFinder);

      // Drag by (50, 60)
      await tester.drag(buttonFinder, const Offset(50, 60));
      await tester.pumpAndSettle();

      final Offset newPos = tester.getTopLeft(buttonFinder);
      expect(newPos.dx, greaterThan(initialPos.dx));
      expect(newPos.dy, greaterThan(initialPos.dy));
    });

    testWidgets('FloatingHudButton updates color when controller starts and stops', (WidgetTester tester) async {
      final controller = AutoUpgradeController();
      addTearDown(controller.dispose);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Stack(
              children: [
                FloatingHudButton(
                  controller: controller,
                  onOpen: () {},
                ),
              ],
            ),
          ),
        ),
      );

      FloatingActionButton fab = tester.widget(find.byType(FloatingActionButton));
      expect(fab.backgroundColor, equals(const Color(0xB237474F)));

      controller.start(
        targetItem: EquipmentItem(id: 1, name: 'Kiếm thần', level: 5),
        config: AutoUpgradeConfig.defaultConfig(),
      );
      await tester.pumpAndSettle();

      fab = tester.widget(find.byType(FloatingActionButton));
      expect(fab.backgroundColor, equals(const Color(0xCC2E7D32)));

      controller.stop();
      await tester.pumpAndSettle();

      fab = tester.widget(find.byType(FloatingActionButton));
      expect(fab.backgroundColor, equals(const Color(0xB237474F)));
    });
  });

  group('AutoUpgradeSheet Tests', () {
    testWidgets('AutoUpgradeSheet Start is disabled unless in_game and connected',
        (WidgetTester tester) async {
      final bridge = _FakeGameBridge()
        ..statusJSON =
            '{"running":false,"connected":true,"in_game":false,"attempts":0,"success":0,"fail":0,"item_level":0,"logs":[]}';
      final controller = AutoUpgradeController(bridge: bridge);
      addTearDown(controller.dispose);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: AutoUpgradeSheet(controller: controller),
          ),
        ),
      );
      await tester.pump();
      await tester.pump();

      expect(find.textContaining('Chưa vào game'), findsOneWidget);
      final ElevatedButton startBtn = tester.widget<ElevatedButton>(
        find.widgetWithText(ElevatedButton, 'Bắt Đầu Treo Auto'),
      );
      expect(startBtn.onPressed, isNull);
    });

    testWidgets('AutoUpgradeSheet polls Go status and start/stop',
        (WidgetTester tester) async {
      final bridge = _FakeGameBridge();
      final controller = AutoUpgradeController(bridge: bridge);
      addTearDown(controller.dispose);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: AutoUpgradeSheet(controller: controller),
          ),
        ),
      );
      await tester.pump();
      await tester.pump();

      expect(find.text('Auto-Upgrade Tiện Ích'), findsOneWidget);
      expect(find.text('Cấp độ đích: '), findsOneWidget);
      expect(find.text('+10'), findsOneWidget);
      expect(find.text('Dừng nếu hết cỏ bảo hiểm'), findsOneWidget);
      expect(find.textContaining('Trong game'), findsOneWidget);
      expect(find.text('Bắt Đầu Treo Auto'), findsOneWidget);

      await tester.tap(find.text('Bắt Đầu Treo Auto'));
      await tester.pump();
      await tester.pump();

      expect(controller.isRunning, isTrue);
      expect(find.text('Dừng Auto'), findsOneWidget);
      expect(find.textContaining('Bat dau Auto-Upgrade'), findsOneWidget);

      bridge.statusJSON =
          '{"running":true,"connected":true,"in_game":true,"attempts":4,"success":1,"fail":3,"item_level":7,"logs":["[12:01:05] Thanh cong: cap +7"]}';
      await tester.pump(const Duration(milliseconds: 350));

      expect(find.text('4'), findsOneWidget);
      expect(find.text('1'), findsWidgets);
      expect(find.text('3'), findsOneWidget);
      expect(find.text('+7'), findsOneWidget);
      expect(find.textContaining('Thanh cong: cap +7'), findsOneWidget);

      await tester.tap(find.text('Dừng Auto'));
      await tester.pump();
      await tester.pump();

      expect(controller.isRunning, isFalse);
      expect(find.text('Bắt Đầu Treo Auto'), findsOneWidget);
    });

    testWidgets('AutoUpgradeSheet target level dropdown changes value', (WidgetTester tester) async {
      final controller = AutoUpgradeController();
      addTearDown(controller.dispose);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: AutoUpgradeSheet(controller: controller),
          ),
        ),
      );

      // Open dropdown
      await tester.tap(find.text('+10'));
      await tester.pumpAndSettle();

      // Select +12
      await tester.tap(find.text('+12').last);
      await tester.pumpAndSettle();

      expect(find.text('+12'), findsOneWidget);
    });
  });
}
