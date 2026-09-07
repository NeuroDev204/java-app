import 'dart:async';
import 'package:flutter/material.dart';
import '../../features/autoupgrade/controller.dart';

/// Draggable translucent Floating HUD button overlaying the game canvas.
///
/// Indicates the active status of the automated upgrade background system
/// (emerald green when running, translucent blue-grey when idle) and provides
/// a quick gesture trigger to launch the configuration bottom sheet modal.
class FloatingHudButton extends StatefulWidget {
  final AutoUpgradeController controller;
  final VoidCallback onOpen;
  final Offset initialPosition;

  const FloatingHudButton({
    super.key,
    required this.controller,
    required this.onOpen,
    this.initialPosition = const Offset(20, 80),
  });

  @override
  State<FloatingHudButton> createState() => _FloatingHudButtonState();
}

class _FloatingHudButtonState extends State<FloatingHudButton> {
  late Offset _position;
  StreamSubscription<String>? _logSubscription;

  @override
  void initState() {
    super.initState();
    _position = widget.initialPosition;
    _subscribeToController();
  }

  @override
  void didUpdateWidget(covariant FloatingHudButton oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      _logSubscription?.cancel();
      _subscribeToController();
    }
  }

  void _subscribeToController() {
    _logSubscription = widget.controller.logStream.listen((_) {
      if (mounted) {
        setState(() {});
      }
    });
  }

  @override
  void dispose() {
    _logSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final bool isRunning = widget.controller.isRunning;
    final Color buttonColor = isRunning
        ? const Color(0xCC2E7D32) // Emerald translucent green
        : const Color(0xB237474F); // Blue-grey translucent

    return Positioned(
      left: _position.dx,
      top: _position.dy,
      child: GestureDetector(
        onPanUpdate: (DragUpdateDetails details) {
          setState(() {
            _position += details.delta;
          });
        },
        child: FloatingActionButton.small(
          heroTag: 'floating_hud_button',
          backgroundColor: buttonColor,
          elevation: 4,
          onPressed: widget.onOpen,
          child: Icon(
            isRunning ? Icons.autorenew : Icons.build_circle,
            color: Colors.white,
          ),
        ),
      ),
    );
  }
}
