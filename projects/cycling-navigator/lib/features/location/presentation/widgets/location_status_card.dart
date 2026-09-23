import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/location_point.dart';
import '../controllers/location_controller.dart';

/// Phase 2 deliverable: prove the GPS stream works end-to-end before any map
/// exists to draw it on (that's Phase 3). Shown raw on Home for now.
class LocationStatusCard extends ConsumerWidget {
  const LocationStatusCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final location = ref.watch(currentLocationStreamProvider);

    return Card(
      margin: const EdgeInsets.all(16),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: location.when(
          data: (point) => _LocationDetails(point: point),
          loading: () => const Row(
            children: [
              SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)),
              SizedBox(width: 12),
              Text('Getting GPS fix...'),
            ],
          ),
          error: (error, _) => _LocationError(
            message: error.toString(),
            onRetry: () => ref.invalidate(currentLocationStreamProvider),
          ),
        ),
      ),
    );
  }
}

class _LocationDetails extends StatelessWidget {
  const _LocationDetails({required this.point});

  final LocationPoint point;

  @override
  Widget build(BuildContext context) {
    final speedKmh = point.speed * 3.6;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Current position', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        Text('Lat/Lng: ${point.latitude.toStringAsFixed(6)}, ${point.longitude.toStringAsFixed(6)}'),
        Text('Accuracy: ±${point.accuracy.toStringAsFixed(1)} m'),
        Text('Altitude: ${point.altitude.toStringAsFixed(1)} m'),
        Text('Speed: ${speedKmh.toStringAsFixed(1)} km/h'),
        Text('Heading: ${point.heading.toStringAsFixed(0)}°'),
      ],
    );
  }
}

class _LocationError extends StatelessWidget {
  const _LocationError({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(message, style: TextStyle(color: Theme.of(context).colorScheme.error)),
        const SizedBox(height: 8),
        OutlinedButton(onPressed: onRetry, child: const Text('Retry')),
      ],
    );
  }
}
