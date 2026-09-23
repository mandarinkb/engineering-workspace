import '../entities/track_point.dart';
import 'geo_math.dart';

class RideStats {
  const RideStats({
    required this.distanceMeters,
    required this.movingTime,
    required this.averageSpeed,
    required this.maxSpeed,
    required this.elevationGain,
    required this.elevationLoss,
  });

  final double distanceMeters;
  final Duration movingTime;
  final double averageSpeed;
  final double maxSpeed;
  final double elevationGain;
  final double elevationLoss;

  static const zero = RideStats(
    distanceMeters: 0,
    movingTime: Duration.zero,
    averageSpeed: 0,
    maxSpeed: 0,
    elevationGain: 0,
    elevationLoss: 0,
  );
}

/// Derives ride-level stats from an already GPS-filtered track. Two noise
/// sources need their own smoothing here (spec sections 10/19):
/// - "moving time" excludes segments slower than [movingSpeedThreshold]
///   (red lights, photo stops) so average speed isn't dragged down by them.
/// - elevation gain/loss accumulates altitude deltas and only commits once
///   they cross [minElevationDeltaMeters], since raw GPS altitude jitters
///   by a few meters even standing still.
class RideStatsCalculator {
  const RideStatsCalculator({
    this.movingSpeedThreshold = 0.5,
    this.minElevationDeltaMeters = 1.5,
  });

  final double movingSpeedThreshold;
  final double minElevationDeltaMeters;

  RideStats calculate(List<TrackPoint> points) {
    if (points.length < 2) return RideStats.zero;

    var distance = 0.0;
    var movingSeconds = 0.0;
    var maxSpeed = 0.0;
    var gain = 0.0;
    var loss = 0.0;
    var pendingElevationDelta = 0.0;
    var lastAltitude = points.first.altitude;

    for (var i = 1; i < points.length; i++) {
      final previous = points[i - 1];
      final current = points[i];

      final segmentMeters = haversineMeters(
        previous.latitude,
        previous.longitude,
        current.latitude,
        current.longitude,
      );
      distance += segmentMeters;

      final seconds = current.timestamp.difference(previous.timestamp).inMilliseconds / 1000;
      if (seconds > 0 && (segmentMeters / seconds) >= movingSpeedThreshold) {
        movingSeconds += seconds;
      }

      if (current.speed > maxSpeed) maxSpeed = current.speed;

      pendingElevationDelta += current.altitude - lastAltitude;
      if (pendingElevationDelta.abs() >= minElevationDeltaMeters) {
        if (pendingElevationDelta > 0) {
          gain += pendingElevationDelta;
        } else {
          loss += -pendingElevationDelta;
        }
        pendingElevationDelta = 0;
      }
      lastAltitude = current.altitude;
    }

    return RideStats(
      distanceMeters: distance,
      movingTime: Duration(milliseconds: (movingSeconds * 1000).round()),
      averageSpeed: movingSeconds > 0 ? distance / movingSeconds : 0,
      maxSpeed: maxSpeed,
      elevationGain: gain,
      elevationLoss: loss,
    );
  }
}
