import 'dart:io';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('CI workflow runs gomobile bind before flutter build', () {
    final yml = File('../../.github/workflows/build-ios.yml').readAsStringSync();
    expect(yml.contains('gomobile bind'), isTrue);
    expect(yml.contains('bind_native.sh'), isTrue);
  });

  test('Runner Xcode project links and embeds KnightBridge.xcframework', () {
    final pbx = File('ios/Runner.xcodeproj/project.pbxproj').readAsStringSync();
    expect(pbx.contains('KnightBridge.xcframework'), isTrue);
    expect(pbx.contains(r'$(PROJECT_DIR)/Frameworks'), isTrue);
    expect(pbx.contains('KnightBridge.xcframework in Frameworks'), isTrue);
    expect(pbx.contains('KnightBridge.xcframework in Embed Frameworks'), isTrue);
  });
}
