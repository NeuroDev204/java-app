import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:knight_online/core/engine/game_loop.dart';
import 'package:knight_online/core/engine/lifecycle_manager.dart';

void main() {
  group('GameLoop & LifecycleManager Tests', () {
    test('GameLoop continues ticking even when lifecycle is paused/background', () async {
      final lifecycle = AppLifecycleManager();
      int tickCounter = 0;

      final gameLoop = GameLoop(
        tickRate: const Duration(milliseconds: 10),
        onTick: () => tickCounter++,
        lifecycleManager: lifecycle,
      );

      gameLoop.start();
      await Future.delayed(const Duration(milliseconds: 35));
      expect(tickCounter, greaterThanOrEqualTo(2));
      expect(lifecycle.shouldPaint, isTrue);

      // Simulate screen off / background
      lifecycle.updateState(AppLifecycleState.paused);
      expect(lifecycle.shouldPaint, isFalse);

      final beforeBackground = tickCounter;
      await Future.delayed(const Duration(milliseconds: 35));
      // Must still tick during background!
      expect(tickCounter, greaterThan(beforeBackground));

      gameLoop.stop();
      expect(gameLoop.isRunning, isFalse);
    });

    test('GameLoop start, stop, and dispose control execution state', () async {
      final lifecycle = AppLifecycleManager();
      int tickCounter = 0;

      final gameLoop = GameLoop(
        tickRate: const Duration(milliseconds: 10),
        onTick: () => tickCounter++,
        lifecycleManager: lifecycle,
      );

      expect(gameLoop.isRunning, isFalse);

      gameLoop.start();
      expect(gameLoop.isRunning, isTrue);

      await Future.delayed(const Duration(milliseconds: 25));
      final ticksAfterStart = tickCounter;
      expect(ticksAfterStart, greaterThanOrEqualTo(1));

      gameLoop.stop();
      expect(gameLoop.isRunning, isFalse);

      await Future.delayed(const Duration(milliseconds: 25));
      expect(tickCounter, equals(ticksAfterStart));

      // Test dispose also ensures stopped state
      gameLoop.start();
      expect(gameLoop.isRunning, isTrue);
      gameLoop.dispose();
      expect(gameLoop.isRunning, isFalse);
    });

    test('GameLoop multiple starts cancel prior timer without leaking', () async {
      final lifecycle = AppLifecycleManager();
      int tickCounter = 0;

      final gameLoop = GameLoop(
        tickRate: const Duration(milliseconds: 15),
        onTick: () => tickCounter++,
        lifecycleManager: lifecycle,
      );

      gameLoop.start();
      gameLoop.start(); // Second start
      await Future.delayed(const Duration(milliseconds: 40));

      // If timer doubled, tickCounter would be disproportionately high.
      // 40ms / 15ms is ~2-3 ticks.
      expect(tickCounter, inInclusiveRange(1, 4));
      gameLoop.stop();
    });

    test('GameLoop handles errors via onError callback defensively', () async {
      final lifecycle = AppLifecycleManager();
      Object? capturedError;
      StackTrace? capturedStackTrace;

      final gameLoop = GameLoop(
        tickRate: const Duration(milliseconds: 10),
        onTick: () {
          throw Exception('Simulated tick failure');
        },
        onError: (err, st) {
          capturedError = err;
          capturedStackTrace = st;
        },
        lifecycleManager: lifecycle,
      );

      gameLoop.start();
      await Future.delayed(const Duration(milliseconds: 25));
      gameLoop.stop();

      expect(capturedError, isNotNull);
      expect(capturedError.toString(), contains('Simulated tick failure'));
      expect(capturedStackTrace, isNotNull);
    });

    test('AppLifecycleManager updates state and triggers listeners', () {
      final lifecycle = AppLifecycleManager();

      expect(lifecycle.state, equals(AppLifecycleState.resumed));
      expect(lifecycle.shouldPaint, isTrue);

      AppLifecycleState? observedState;
      lifecycle.addListener((state) {
        observedState = state;
      });

      // Transition to inactive
      lifecycle.updateState(AppLifecycleState.inactive);
      expect(lifecycle.state, equals(AppLifecycleState.inactive));
      expect(lifecycle.shouldPaint, isFalse);
      expect(observedState, equals(AppLifecycleState.inactive));

      // Transition to paused via didChangeAppLifecycleState
      lifecycle.didChangeAppLifecycleState(AppLifecycleState.paused);
      expect(lifecycle.state, equals(AppLifecycleState.paused));
      expect(lifecycle.shouldPaint, isFalse);
      expect(observedState, equals(AppLifecycleState.paused));

      // Transition to detached
      lifecycle.updateState(AppLifecycleState.detached);
      expect(lifecycle.state, equals(AppLifecycleState.detached));
      expect(lifecycle.shouldPaint, isFalse);
      expect(observedState, equals(AppLifecycleState.detached));

      // Transition back to resumed
      lifecycle.updateState(AppLifecycleState.resumed);
      expect(lifecycle.state, equals(AppLifecycleState.resumed));
      expect(lifecycle.shouldPaint, isTrue);
      expect(observedState, equals(AppLifecycleState.resumed));
    });
  });
}
