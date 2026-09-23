import 'package:flutter/material.dart';

import '../../../../core/widgets/placeholder_screen.dart';

class RideDetailScreen extends StatelessWidget {
  const RideDetailScreen({required this.rideId, super.key});

  final String rideId;

  @override
  Widget build(BuildContext context) {
    return PlaceholderScreen(
      title: 'Ride Detail',
      phaseNote: 'Phase 11 — Ride History detail view.\n(ride id: $rideId)',
    );
  }
}
