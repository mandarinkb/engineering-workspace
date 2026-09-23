class MapRegion {
  const MapRegion({
    required this.id,
    required this.name,
    required this.minLat,
    required this.minLon,
    required this.maxLat,
    required this.maxLon,
  });

  final String id;
  final String name;

  /// Bounding box the offline tiles actually cover — outside of this there
  /// is no downloaded data, so the map camera must be constrained to it
  /// (see MapScreen's `cameraTargetBounds`).
  final double minLat;
  final double minLon;
  final double maxLat;
  final double maxLon;
}
