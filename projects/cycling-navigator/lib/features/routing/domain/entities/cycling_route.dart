import 'geo_point.dart';

class CyclingRoute {
  const CyclingRoute({
    required this.start,
    required this.destination,
    required this.distanceMeters,
    required this.duration,
    required this.points,
  });

  /// The exact points that were requested — Valhalla snaps [points]'s
  /// first/last entries to the nearest road, which rarely matches these
  /// exactly (e.g. GPS drift, or a destination pinned mid-block). Kept so
  /// presentation can render a route that visually still touches both.
  final GeoPoint start;
  final GeoPoint destination;

  final double distanceMeters;
  final Duration duration;

  /// Decoded, road-snapped route geometry, in order from start to
  /// destination.
  final List<GeoPoint> points;
}
