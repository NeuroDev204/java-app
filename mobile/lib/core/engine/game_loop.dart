import 'dart:async';
import 'lifecycle_manager.dart';

/// Decoupled game tick loop running at a fixed frequency (e.g. 50ms / 20 Hz).
///
/// Unlike Flutter's vsync-bound [Ticker] or [AnimationController], which pauses
/// frame callbacks when the screen turns off or backgrounded, [GameLoop] uses
/// [Timer.periodic] to drive continuous background simulation and state machine
/// progression (AFK grinding, combat, auto-upgrades).
class GameLoop {
  /// Time interval between consecutive game logic ticks.
  final Duration tickRate;

  /// Callback executed on each tick.
  final void Function() onTick;

  /// Lifecycle manager reference tracking screen/app foreground status.
  final AppLifecycleManager lifecycleManager;

  /// Optional error handler for defensive error isolation.
  final void Function(Object error, StackTrace stackTrace)? onError;

  Timer? _timer;
  bool _isRunning = false;
  int _ticks = 0;

  /// Creates a new [GameLoop] instance.
  GameLoop({
    required this.tickRate,
    required this.onTick,
    required this.lifecycleManager,
    this.onError,
  });

  /// Whether the game loop is currently running.
  bool get isRunning => _isRunning;

  /// Total count of completed ticks executed by this loop instance.
  int get ticks => _ticks;

  /// Starts the game loop.
  ///
  /// If the loop is already running, cancels any existing timer and restarts
  /// periodic ticking cleanly.
  void start() {
    _isRunning = true;
    _timer?.cancel();
    _timer = Timer.periodic(tickRate, (_) {
      if (_isRunning) {
        _ticks++;
        try {
          onTick();
        } catch (error, stackTrace) {
          if (onError != null) {
            onError!(error, stackTrace);
          } else {
            rethrow;
          }
        }
      }
    });
  }

  /// Stops the game loop and cancels active periodic timers.
  void stop() {
    _isRunning = false;
    _timer?.cancel();
    _timer = null;
  }

  /// Disposes the game loop and releases all timer resources.
  void dispose() {
    stop();
  }
}
