/// A single GPS fix. Pure domain value object — no dependency on the
/// `geolocator` plugin's own `Position` type, so the data layer can swap
/// location providers without this type changing.
class LocationPoint {
  const LocationPoint({
    required this.latitude,
    required this.longitude,
    required this.timestamp,
    required this.accuracy,
    required this.altitude,
    required this.speed,
    required this.heading,
  });

  final double latitude;
  final double longitude;
  final DateTime timestamp;
  final double accuracy;
  final double altitude;
  final double speed;
  final double heading;
}
