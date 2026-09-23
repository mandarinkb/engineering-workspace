import '../entities/map_region.dart';

abstract class MapRepository {
  /// Regions the app knows how to fetch. For now this is a hardcoded
  /// single-entry catalog (see spec section 24) — becomes a real remote
  /// catalog once regions are hosted instead of bundled.
  List<MapRegion> get availableRegions;

  Future<void> downloadRegion(MapRegion region);

  Future<void> deleteRegion(MapRegion region);

  Future<bool> isRegionAvailable(MapRegion region);

  Future<int> regionSizeBytes(MapRegion region);

  /// Absolute path to the region's style JSON once downloaded, or null if
  /// not yet available. Suitable to pass directly as `MapLibreMap.styleString`.
  Future<String?> stylePathForRegion(MapRegion region);
}
