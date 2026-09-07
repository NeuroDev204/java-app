import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/main.dart';
import 'package:knight_online/presentation/screens/game_screen.dart';
import 'package:knight_online/presentation/screens/jar_picker_screen.dart';
import 'package:knight_online/presentation/widgets/autoupgrade_sheet.dart';
import 'package:knight_online/presentation/widgets/floating_hud.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const MethodChannel channel =
      MethodChannel('com.example.knight_online/game_bridge');

  setUp(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (MethodCall call) async {
      switch (call.method) {
        case 'loadJAR':
          return 'test-stub';
        case 'isStarted':
          return false;
        default:
          return null;
      }
    });
  });

  tearDown(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null);
  });

  testWidgets('KnightApp initializes game canvas shell without JAR picker',
      (WidgetTester tester) async {
    await tester.pumpWidget(const KnightApp());
    expect(find.byType(KnightApp), findsOneWidget);
    expect(find.byType(JarPickerScreen), findsNothing);
    expect(find.byType(GameScreen), findsOneWidget);
    expect(find.byType(FloatingHudButton), findsOneWidget);
  });

  testWidgets(
      'KnightApp tapping FloatingHudButton opens AutoUpgradeSheet and closes on dismiss',
      (WidgetTester tester) async {
    await tester.pumpWidget(const KnightApp());

    // AutoUpgradeSheet should not be visible initially.
    expect(find.byType(AutoUpgradeSheet), findsNothing);

    // Tap FloatingHudButton. Bounded pumps — GameScreen's boot spinner
    // is an infinite animation so pumpAndSettle never completes.
    await tester.tap(find.byType(FloatingHudButton));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));

    // AutoUpgradeSheet should now be visible.
    expect(find.byType(AutoUpgradeSheet), findsOneWidget);
    expect(find.text('Auto-Upgrade Tiện Ích'), findsOneWidget);

    // Tap Close icon button.
    await tester.tap(find.byIcon(Icons.close));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));

    // AutoUpgradeSheet should be dismissed.
    expect(find.byType(AutoUpgradeSheet), findsNothing);
  });
}
