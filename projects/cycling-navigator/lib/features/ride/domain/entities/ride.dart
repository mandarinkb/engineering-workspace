class Ride {
  const Ride({
    required this.id,
    required this.startTime,
    required this.endTime,
    required this.duration,
    required this.movingTime,
    required this.distanceMeters,
    required this.averageSpeed,
    required this.maxSpeed,
    required this.elevationGain,
    required this.elevationLoss,
    required this.startLatitude,
    required this.startLongitude,
    required this.endLatitude,
    required this.endLongitude,
  });

  final String id;
  final DateTime startTime;
  final DateTime endTime;
  final Duration duration;
  final Duration movingTime;
  final double distanceMeters;

  /// Meters per second, same unit as [TrackPoint.speed].
  final double averageSpeed;
  final double maxSpeed;
  final double elevationGain;
  final double elevationLoss;
  final double startLatitude;
  final double startLongitude;
  final double endLatitude;
  final double endLongitude;
}
