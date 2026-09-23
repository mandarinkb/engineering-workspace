import 'package:flutter/material.dart';

import '../../../../core/widgets/placeholder_screen.dart';

class RideHistoryScreen extends StatelessWidget {
  const RideHistoryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Ride History',
      phaseNote: 'Phase 11 — Ride History (list, stats, elevation profile).',
    );
  }
}
