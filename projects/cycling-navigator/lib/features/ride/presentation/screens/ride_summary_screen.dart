import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../controllers/ride_tracking_controller.dart';
import '../ride_format.dart';

class RideSummaryScreen extends ConsumerWidget {
  const RideSummaryScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ride = ref.watch(rideTrackingControllerProvider).lastFinishedRide;

    return Scaffold(
      appBar: AppBar(title: const Text('Ride Summary')),
      body: ride == null
          ? const Center(child: Text('No ride data.'))
          : Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(formatDistanceKm(ride.distanceMeters), style: Theme.of(context).textTheme.displaySmall),
                  const SizedBox(height: 24),
                  _SummaryRow(label: 'Duration', value: formatDuration(ride.duration)),
                  _SummaryRow(label: 'Moving time', value: formatDuration(ride.movingTime)),
                  _SummaryRow(label: 'Average speed', value: formatSpeedKmh(ride.averageSpeed)),
                  _SummaryRow(label: 'Max speed', value: formatSpeedKmh(ride.maxSpeed)),
                  _SummaryRow(label: 'Elevation gain', value: '+${formatElevation(ride.elevationGain)}'),
                  _SummaryRow(label: 'Elevation loss', value: '-${formatElevation(ride.elevationLoss)}'),
                  const SizedBox(height: 32),
                  FilledButton(
                    onPressed: () => context.go('/'),
                    child: const Text('Done'),
                  ),
                ],
              ),
            ),
    );
  }
}

class _SummaryRow extends StatelessWidget {
  const _SummaryRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label),
          Text(value, style: const TextStyle(fontWeight: FontWeight.bold)),
        ],
      ),
    );
  }
}
