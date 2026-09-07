/// Legacy JAR picker screen — not on the play path.
///
/// Home now boots `assets/KnightOnline_402.jar` directly into [GameScreen].
/// This widget remains only so tests can assert it is absent.
library;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../features/jar_runner/jar_runner_controller.dart';
import 'game_screen.dart';

/// [JarPickerScreen] is the home/entry screen of the app.
class JarPickerScreen extends StatefulWidget {
  final JarRunnerController controller;

  const JarPickerScreen({super.key, required this.controller});

  @override
  State<JarPickerScreen> createState() => _JarPickerScreenState();
}

class _JarPickerScreenState extends State<JarPickerScreen> {
  @override
  void initState() {
    super.initState();
    // Listen to controller state changes so we can auto-navigate when running.
    widget.controller.addListener(_onControllerUpdate);
  }

  @override
  void dispose() {
    widget.controller.removeListener(_onControllerUpdate);
    super.dispose();
  }

  void _onControllerUpdate() {
    if (!mounted) return;
    if (widget.controller.state == JarRunnerState.running) {
      // Navigate to the game screen as a full-screen route (no back button).
      Navigator.of(context).pushReplacement(
        MaterialPageRoute<void>(
          builder: (_) => GameScreen(controller: widget.controller),
        ),
      );
    } else if (widget.controller.state == JarRunnerState.error) {
      _showError(widget.controller.errorMessage);
    }
  }

  void _showError(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: Colors.red.shade800,
        duration: const Duration(seconds: 5),
      ),
    );
  }

  /// Loads the bundled JAR from Flutter assets.
  Future<void> _onLoadBundled() async {
    try {
      final ByteData data =
          await rootBundle.load('assets/KnightOnline_402.jar');
      final Uint8List bytes = data.buffer.asUint8List();
      await widget.controller.loadJARBytes(bytes);
    } catch (e) {
      _showError('Could not load bundled JAR: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 32),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                // Game logo / title
                const Icon(Icons.sports_esports,
                    size: 80, color: Colors.amber),
                const SizedBox(height: 16),
                const Text(
                  'Knight Online',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 28,
                    fontWeight: FontWeight.bold,
                    letterSpacing: 2,
                  ),
                ),
                const SizedBox(height: 8),
                const Text(
                  'J2ME Mobile Emulator',
                  style: TextStyle(color: Colors.white54, fontSize: 14),
                ),
                const SizedBox(height: 48),

                // Loading indicator while booting VM
                ListenableBuilder(
                  listenable: widget.controller,
                  builder: (BuildContext ctx, _) {
                    final JarRunnerState state = widget.controller.state;

                    if (state == JarRunnerState.loading) {
                      return const Column(
                        children: [
                          CircularProgressIndicator(color: Colors.amber),
                          SizedBox(height: 16),
                          Text(
                            'Starting J2ME VM…',
                            style: TextStyle(color: Colors.white70),
                          ),
                        ],
                      );
                    }

                    return Column(
                      children: [
                        // Primary CTA: load bundled JAR
                        _ActionButton(
                          icon: Icons.play_arrow_rounded,
                          label: 'Start Game (Bundled)',
                          onPressed: _onLoadBundled,
                          primary: true,
                        ),
                        const SizedBox(height: 24),
                        const Text(
                          'Server: hs1.teamobi.com:19129',
                          style: TextStyle(
                              color: Colors.white30, fontSize: 11),
                        ),
                      ],
                    );
                  },
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.icon,
    required this.label,
    required this.onPressed,
    required this.primary,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final bool primary;

  @override
  Widget build(BuildContext context) {
    if (primary) {
      return SizedBox(
        width: double.infinity,
        height: 56,
        child: ElevatedButton.icon(
          icon: Icon(icon),
          label: Text(label),
          style: ElevatedButton.styleFrom(
            backgroundColor: Colors.amber,
            foregroundColor: Colors.black,
            textStyle: const TextStyle(
                fontSize: 16, fontWeight: FontWeight.bold),
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12)),
          ),
          onPressed: onPressed,
        ),
      );
    }

    return SizedBox(
      width: double.infinity,
      height: 52,
      child: OutlinedButton.icon(
        icon: Icon(icon),
        label: Text(label),
        style: OutlinedButton.styleFrom(
          foregroundColor: Colors.white70,
          side: const BorderSide(color: Colors.white24),
          textStyle: const TextStyle(fontSize: 15),
          shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(12)),
        ),
        onPressed: onPressed,
      ),
    );
  }
}
