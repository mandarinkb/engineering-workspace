import 'dart:math';

const _earthRadiusMeters = 6371000.0;

/// Great-circle distance between two points, in meters.
double haversineMeters(double lat1, double lon1, double lat2, double lon2) {
  final dLat = _degToRad(lat2 - lat1);
  final dLon = _degToRad(lon2 - lon1);

  final a = sin(dLat / 2) * sin(dLat / 2) +
      cos(_degToRad(lat1)) * cos(_degToRad(lat2)) * sin(dLon / 2) * sin(dLon / 2);
  final c = 2 * atan2(sqrt(a), sqrt(1 - a));

  return _earthRadiusMeters * c;
}

double _degToRad(double deg) => deg * pi / 180;
