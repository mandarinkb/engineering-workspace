import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../controllers/ride_tracking_controller.dart';
import '../ride_format.dart';

class ActiveRideScreen extends ConsumerStatefulWidget {
  const ActiveRideScreen({super.key});

  @override
  ConsumerState<ActiveRideScreen> createState() => _ActiveRideScreenState();
}

class _ActiveRideScreenState extends ConsumerState<ActiveRideScreen> {
  Timer? _ticker;

  @override
  void dispose() {
    _ticker?.cancel();
    super.dispose();
  }

  /// GPS fixes only arrive every few meters moved (Phase 2's
  /// `distanceFilter`), so without this the elapsed-time display would
  /// freeze whenever the rider stops (red light, etc).
  void _ensureTicking(bool isActive) {
    if (isActive && _ticker == null) {
      _ticker = Timer.periodic(const Duration(seconds: 1), (_) => setState(() {}));
    } else if (!isActive && _ticker != null) {
      _ticker!.cancel();
      _ticker = null;
    }
  }

  @override
  Widget build(BuildContext context) {
    final trackingState = ref.watch(rideTrackingControllerProvider);
    _ensureTicking(trackingState.isActive);

    return Scaffold(
      appBar: AppBar(title: const Text('Active Ride')),
      body: Center(
        child: trackingState.isActive
            ? _ActiveRideStats(state: trackingState)
            : const Text('Not riding yet.'),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _onPressed(trackingState.isActive),
        icon: Icon(trackingState.isActive ? Icons.stop : Icons.play_arrow),
        label: Text(trackingState.isActive ? 'Stop Ride' : 'Start Ride'),
        backgroundColor: trackingState.isActive ? Colors.red : null,
      ),
    );
  }

  Future<void> _onPressed(bool isActive) async {
    final controller = ref.read(rideTrackingControllerProvider.notifier);
    if (isActive) {
      await controller.stopRide();
      if (mounted) context.push('/ride-summary');
    } else {
      controller.startRide();
    }
  }
}

class _ActiveRideStats extends StatelessWidget {
  const _ActiveRideStats({required this.state});

  final RideTrackingState state;

  @override
  Widget build(BuildContext context) {
    final elapsed = DateTime.now().difference(state.startTime!);
    final stats = state.stats;

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        _StatRow(label: 'Distance', value: formatDistanceKm(stats.distanceMeters)),
        _StatRow(label: 'Time', value: formatDuration(elapsed)),
        _StatRow(
          label: 'Speed',
          value: formatSpeedKmh(state.points.isEmpty ? 0 : state.points.last.speed),
        ),
        _StatRow(label: 'Average', value: formatSpeedKmh(stats.averageSpeed)),
        _StatRow(label: 'Elevation', value: '+${formatElevation(stats.elevationGain)}'),
      ],
    );
  }
}

class _StatRow extends StatelessWidget {
  const _StatRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        children: [
          Text(label, style: Theme.of(context).textTheme.labelLarge),
          Text(value, style: Theme.of(context).textTheme.headlineMedium),
        ],
      ),
    );
  }
}
