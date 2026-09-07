import 'package:flutter/widgets.dart';

/// Manages the application lifecycle state and determines whether the rendering
/// pipeline should execute painting frames.
///
/// When the app enters the background or screen is turned off ([AppLifecycleState.paused],
/// [AppLifecycleState.inactive], etc.), [shouldPaint] returns `false`, allowing
/// the rendering pipeline (CustomPainter/Canvas) to skip painting, preserving
/// 100% of GPU resources and battery on iOS while the game logic continues headless.
class AppLifecycleManager with WidgetsBindingObserver {
  AppLifecycleState _state = AppLifecycleState.resumed;
  final List<void Function(AppLifecycleState state)> _listeners = [];

  /// Current application lifecycle state.
  AppLifecycleState get state => _state;

  /// Whether the rendering layer should paint frames.
  ///
  /// Returns `true` exclusively when the application is [AppLifecycleState.resumed]
  /// and actively visible to the user.
  bool get shouldPaint => _state == AppLifecycleState.resumed;

  /// Registers a listener callback triggered when lifecycle state changes.
  void addListener(void Function(AppLifecycleState state) listener) {
    if (!_listeners.contains(listener)) {
      _listeners.add(listener);
    }
  }

  /// Unregisters a previously registered lifecycle listener callback.
  void removeListener(void Function(AppLifecycleState state) listener) {
    _listeners.remove(listener);
  }

  /// Updates the internal lifecycle state and notifies registered listeners.
  void updateState(AppLifecycleState newState) {
    if (_state != newState) {
      _state = newState;
      for (final listener in List.of(_listeners)) {
        listener(_state);
      }
    }
  }

  /// Subscribes to Flutter's [WidgetsBinding] observer notifications.
  void attach() {
    WidgetsBinding.instance.addObserver(this);
  }

  /// Unsubscribes from Flutter's [WidgetsBinding] and cleans up listeners.
  void dispose() {
    try {
      WidgetsBinding.instance.removeObserver(this);
    } catch (_) {
      // Ignored if binding was not initialized (e.g. unit tests)
    }
    _listeners.clear();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    updateState(state);
  }
}
