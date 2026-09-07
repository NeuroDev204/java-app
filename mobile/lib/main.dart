import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'core/bridge/game_bridge.dart';
import 'core/engine/game_loop.dart';
import 'core/engine/lifecycle_manager.dart';
import 'core/services/background_keepalive.dart';
import 'features/autoupgrade/controller.dart';
import 'features/jar_runner/jar_runner_controller.dart';
import 'presentation/screens/game_screen.dart';
import 'presentation/widgets/autoupgrade_sheet.dart';
import 'presentation/widgets/floating_hud.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  // Immersive full-screen: hides status bar and navigation bar permanently.
  SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky);
  SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]);
  runApp(const KnightApp());
}

/// Root widget for the Knight Online J2ME mobile client.
class KnightApp extends StatelessWidget {
  final AutoUpgradeController? controller;
  final JarRunnerController? jarController;
  final BackgroundKeepAliveService? keepAliveService;
  final GameLoop? gameLoop;
  final AppLifecycleManager? lifecycleManager;
  final GameBridge? bridge;

  const KnightApp({
    super.key,
    this.controller,
    this.jarController,
    this.keepAliveService,
    this.gameLoop,
    this.lifecycleManager,
    this.bridge,
  });

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Knight Online J2ME',
      debugShowCheckedModeBanner: false,
      theme: ThemeData.dark(),
      home: GameHomeScreen(
        controller: controller,
        jarController: jarController,
        keepAliveService: keepAliveService,
        gameLoop: gameLoop,
        lifecycleManager: lifecycleManager,
        bridge: bridge,
      ),
    );
  }
}

/// Primary interactive screen managing the game lifecycle, background services,
/// and the auto-upgrade floating HUD overlay.
class GameHomeScreen extends StatefulWidget {
  final AutoUpgradeController? controller;
  final JarRunnerController? jarController;
  final BackgroundKeepAliveService? keepAliveService;
  final GameLoop? gameLoop;
  final AppLifecycleManager? lifecycleManager;
  final GameBridge? bridge;

  const GameHomeScreen({
    super.key,
    this.controller,
    this.jarController,
    this.keepAliveService,
    this.gameLoop,
    this.lifecycleManager,
    this.bridge,
  });

  @override
  State<GameHomeScreen> createState() => _GameHomeScreenState();
}

class _GameHomeScreenState extends State<GameHomeScreen> {
  late final GameBridge _bridge;
  late final AutoUpgradeController _controller;
  late final JarRunnerController _jarController;
  late final AppLifecycleManager _lifecycleManager;
  late final BackgroundKeepAliveService _keepAliveService;
  late final GameLoop _gameLoop;
  StreamSubscription<String>? _logSubscription;

  @override
  void initState() {
    super.initState();
    _bridge = widget.bridge ?? GameBridge();
    _controller = widget.controller ?? AutoUpgradeController(bridge: _bridge);
    _jarController = widget.jarController ?? JarRunnerController(bridge: _bridge);

    // Keep-alive must activate when either the game is running or AFK is active.
    _jarController.addListener(_syncKeepAliveState);

    _lifecycleManager = widget.lifecycleManager ?? AppLifecycleManager();
    _lifecycleManager.attach();
    _lifecycleManager.addListener(_onLifecycleChanged);

    _keepAliveService =
        widget.keepAliveService ?? BackgroundKeepAliveService();

    // Headless game tick — continues regardless of screen state so the AFK
    // bot logic keeps running even when the app is in the background.
    _gameLoop = widget.gameLoop ??
        GameLoop(
          tickRate: const Duration(milliseconds: 50),
          lifecycleManager: _lifecycleManager,
          onTick: () {
            // Headless tick placeholder — actual game loop is inside Go bridge.
          },
        );
    _gameLoop.start();

    _logSubscription = _controller.logStream.listen((_) {
      _syncKeepAliveState();
    });

    unawaited(_bootBundledJar());
  }

  /// Boots the shipped Knight Online JAR — there is no picker on the play path.
  Future<void> _bootBundledJar() async {
    try {
      final ByteData data =
          await rootBundle.load('assets/KnightOnline_402.jar');
      if (!mounted) return;
      await _jarController.loadJARBytes(
        data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes),
      );
    } catch (e, st) {
      debugPrint('Failed to boot bundled JAR: $e\n$st');
      if (mounted) {
        _jarController.reportError('Could not load bundled JAR: $e');
      }
    }
  }

  void _onLifecycleChanged(AppLifecycleState state) {
    unawaited(
      _jarController.setShouldPaint(state == AppLifecycleState.resumed),
    );
    if (mounted) {
      setState(() {});
    }
  }

  void _syncKeepAliveState() {
    // Keep the silent audio loop running whenever the game VM is active OR
    // the auto-upgrade bot is running — prevents iOS from suspending the app.
    final bool shouldKeepAlive =
        _controller.isRunning || _jarController.isRunning;
    if (shouldKeepAlive) {
      if (!_keepAliveService.isKeepAliveActive) {
        _keepAliveService.startSilentAudioLoop();
        _keepAliveService.setWakelock(true);
      }
    } else {
      if (_keepAliveService.isKeepAliveActive) {
        _keepAliveService.stopSilentAudioLoop();
        _keepAliveService.setWakelock(false);
      }
    }
  }

  @override
  void dispose() {
    _lifecycleManager.removeListener(_onLifecycleChanged);
    _jarController.removeListener(_syncKeepAliveState);
    _logSubscription?.cancel();
    _gameLoop.dispose();
    _keepAliveService.dispose();
    _lifecycleManager.dispose();
    _controller.dispose();
    _jarController.dispose();
    super.dispose();
  }

  void _openAutoUpgradeSheet() {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (BuildContext sheetContext) {
        return AutoUpgradeSheet(controller: _controller);
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      body: Stack(
        children: [
          // Game canvas (fullscreen: false, aspect-fit with SafeArea). JAR boots from assets.
          Positioned.fill(
            child: GameScreen(
              controller: _jarController,
              shouldPaint: _lifecycleManager.shouldPaint,
              fullscreen: false,
            ),
          ),
          // Floating AFK / auto-upgrade HUD overlay
          FloatingHudButton(
            controller: _controller,
            onOpen: _openAutoUpgradeSheet,
          ),
        ],
      ),
    );
  }
}
