import 'package:cycling_navigator/features/ride/domain/entities/track_point.dart';
import 'package:cycling_navigator/features/ride/domain/services/gps_filter.dart';
import 'package:flutter_test/flutter_test.dart';

TrackPoint _point({
  required double lat,
  required double lon,
  required DateTime time,
  double accuracy = 5,
}) =>
    TrackPoint(
      timestamp: time,
      latitude: lat,
      longitude: lon,
      altitude: 10,
      speed: 0,
      accuracy: accuracy,
    );

void main() {
  const filter = GpsFilter();
  final t0 = DateTime(2026);

  test('rejects a fix with poor accuracy even with no previous point', () {
    final point = _point(lat: 13.7563, lon: 100.5018, time: t0, accuracy: 50);
    expect(filter.isAcceptable(point, null), isFalse);
  });

  test('accepts a fix with good accuracy and no previous point', () {
    final point = _point(lat: 13.7563, lon: 100.5018, time: t0, accuracy: 10);
    expect(filter.isAcceptable(point, null), isTrue);
  });

  test('accepts a plausible cycling-speed jump between two points', () {
    final previous = _point(lat: 13.7563, lon: 100.5018, time: t0);
    // ~28m north in 5s -> ~5.6 m/s (~20 km/h), a normal cycling speed.
    final candidate = _point(lat: 13.75655, lon: 100.5018, time: t0.add(const Duration(seconds: 5)));
    expect(filter.isAcceptable(candidate, previous), isTrue);
  });

  test('rejects an impossible speed jump between two points', () {
    final previous = _point(lat: 13.7563, lon: 100.5018, time: t0);
    // ~1.1km jump in 1s -> way beyond any real cycling speed.
    final candidate = _point(lat: 13.7663, lon: 100.5018, time: t0.add(const Duration(seconds: 1)));
    expect(filter.isAcceptable(candidate, previous), isFalse);
  });

  test('rejects a non-positive time delta (duplicate or out-of-order fix)', () {
    final previous = _point(lat: 13.7563, lon: 100.5018, time: t0);
    final candidate = _point(lat: 13.7564, lon: 100.5018, time: t0);
    expect(filter.isAcceptable(candidate, previous), isFalse);
  });
}
