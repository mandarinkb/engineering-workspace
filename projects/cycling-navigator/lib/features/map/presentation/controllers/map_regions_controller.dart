import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../data/repositories/bundled_asset_map_repository.dart';
import '../../domain/entities/map_region.dart';
import '../../domain/repositories/map_repository.dart';

part 'map_regions_controller.g.dart';

@Riverpod(keepAlive: true)
MapRepository mapRepository(Ref ref) => BundledAssetMapRepository();

@riverpod
Future<bool> isRegionDownloaded(Ref ref, MapRegion region) {
  return ref.watch(mapRepositoryProvider).isRegionAvailable(region);
}

@riverpod
Future<int> regionSizeBytes(Ref ref, MapRegion region) {
  return ref.watch(mapRepositoryProvider).regionSizeBytes(region);
}

/// The style to hand `MapLibreMap.styleString`, plus which region it came
/// from (used only to warn when "recenter" would land outside downloaded
/// data — Phase 4). Picks the first *downloaded* region in
/// [MapRepository.availableRegions] order, i.e. the most detailed one
/// available; [region] is null when falling back to the (world-coverage)
/// online demo style from Phase 3.
@riverpod
Future<({String styleString, MapRegion? region})> activeMapStyle(Ref ref) async {
  final repository = ref.watch(mapRepositoryProvider);
  for (final region in repository.availableRegions) {
    final localStylePath = await repository.stylePathForRegion(region);
    if (localStylePath != null) {
      return (styleString: localStylePath, region: region);
    }
  }
  return (styleString: 'https://demotiles.maplibre.org/style.json', region: null);
}
