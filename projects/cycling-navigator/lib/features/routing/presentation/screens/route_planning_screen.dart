import 'package:flutter/material.dart';

import '../../../../core/widgets/placeholder_screen.dart';

class RoutePlanningScreen extends StatelessWidget {
  const RoutePlanningScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Route Planning',
      phaseNote: 'Phase 7 — Multiple Routes (after the local routing engine from Phase 6 is ready).',
    );
  }
}
