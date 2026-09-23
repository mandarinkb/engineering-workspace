import '../entities/track_point.dart';
import 'geo_math.dart';

/// Thresholds kept as plain config (spec section 13) instead of hardcoded
/// inside the filter logic, so they're easy to tune per accuracy profile
/// later (Phase 12) without touching the filtering algorithm itself.
class GpsFilterConfig {
  const GpsFilterConfig({
    this.maxAccuracyMeters = 30,
    this.maxPlausibleSpeedMetersPerSecond = 25,
  });

  final double maxAccuracyMeters;

  /// ~90 km/h — generous enough to cover a fast downhill without rejecting
  /// real data, but well below the impossible speeds a GPS glitch produces.
  final double maxPlausibleSpeedMetersPerSecond;
}

/// Rejects individual GPS fixes that are too imprecise to trust, or that
/// imply an impossible jump from the previous accepted point (spec
/// section 11) — e.g. a reflected signal placing you 200m away a second
/// later. Reset [previous] to null when starting a new ride.
class GpsFilter {
  const GpsFilter([this.config = const GpsFilterConfig()]);

  final GpsFilterConfig config;

  bool isAcceptable(TrackPoint candidate, TrackPoint? previous) {
    if (candidate.accuracy > config.maxAccuracyMeters) return false;
    if (previous == null) return true;

    final seconds = candidate.timestamp.difference(previous.timestamp).inMilliseconds / 1000;
    if (seconds <= 0) return false;

    final meters = haversineMeters(
      previous.latitude,
      previous.longitude,
      candidate.latitude,
      candidate.longitude,
    );
    return (meters / seconds) <= config.maxPlausibleSpeedMetersPerSecond;
  }
}
