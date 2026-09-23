import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/map_region.dart';
import '../controllers/map_regions_controller.dart';

class OfflineMapsScreen extends ConsumerWidget {
  const OfflineMapsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final regions = ref.watch(mapRepositoryProvider).availableRegions;

    return Scaffold(
      appBar: AppBar(title: const Text('Offline Maps')),
      body: ListView(
        children: [
          for (final region in regions) _RegionTile(region: region),
        ],
      ),
    );
  }
}

class _RegionTile extends ConsumerWidget {
  const _RegionTile({required this.region});

  final MapRegion region;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final downloaded = ref.watch(isRegionDownloadedProvider(region));
    final size = ref.watch(regionSizeBytesProvider(region));

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(region.name, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 4),
            Text(
              size.maybeWhen(
                data: (bytes) => bytes > 0 ? '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB on device' : 'Not downloaded',
                orElse: () => '...',
              ),
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            downloaded.when(
              data: (isDownloaded) => Row(
                children: [
                  if (!isDownloaded)
                    FilledButton(
                      onPressed: () => _download(ref),
                      child: const Text('Download'),
                    )
                  else
                    OutlinedButton(
                      onPressed: () => _delete(ref),
                      child: const Text('Delete'),
                    ),
                ],
              ),
              loading: () => const SizedBox(
                height: 36,
                width: 36,
                child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
              ),
              error: (error, _) => Text('Error: $error'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _download(WidgetRef ref) async {
    await ref.read(mapRepositoryProvider).downloadRegion(region);
    ref.invalidate(isRegionDownloadedProvider(region));
    ref.invalidate(regionSizeBytesProvider(region));
    ref.invalidate(activeMapStyleProvider);
  }

  Future<void> _delete(WidgetRef ref) async {
    await ref.read(mapRepositoryProvider).deleteRegion(region);
    ref.invalidate(isRegionDownloadedProvider(region));
    ref.invalidate(regionSizeBytesProvider(region));
    ref.invalidate(activeMapStyleProvider);
  }
}
