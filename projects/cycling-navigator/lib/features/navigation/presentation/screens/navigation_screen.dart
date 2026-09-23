import 'package:flutter/material.dart';

import '../../../../core/widgets/placeholder_screen.dart';

class NavigationScreen extends StatelessWidget {
  const NavigationScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Navigation',
      phaseNote: 'Phase 8 — Navigation (route progress, off-route detection).',
    );
  }
}
