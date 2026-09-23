import 'package:cycling_navigator/features/ride/domain/entities/track_point.dart';
import 'package:cycling_navigator/features/ride/domain/services/ride_stats_calculator.dart';
import 'package:flutter_test/flutter_test.dart';

TrackPoint _point({
  required double lat,
  required double lon,
  required DateTime time,
  double altitude = 10,
  double speed = 0,
}) =>
    TrackPoint(timestamp: time, latitude: lat, longitude: lon, altitude: altitude, speed: speed, accuracy: 5);

void main() {
  const calculator = RideStatsCalculator();
  final t0 = DateTime(2026);

  test('fewer than two points yields zero stats', () {
    expect(calculator.calculate([]), RideStats.zero);
    expect(calculator.calculate([_point(lat: 0, lon: 0, time: t0)]), RideStats.zero);
  });

  test('distance, moving time and average speed for a steady 100s ride', () {
    final points = [
      _point(lat: 13.7563, lon: 100.5018, time: t0, speed: 10),
      // ~0.009 deg north ~= 1001.88 m
      _point(lat: 13.7653, lon: 100.5018, time: t0.add(const Duration(seconds: 100)), speed: 10),
    ];

    final stats = calculator.calculate(points);

    expect(stats.distanceMeters, closeTo(1001.88, 5));
    expect(stats.movingTime, const Duration(seconds: 100));
    expect(stats.averageSpeed, closeTo(10.02, 0.1));
    expect(stats.maxSpeed, 10);
  });

  test('time spent below the moving-speed threshold does not count as moving', () {
    final points = [
      // ~1.1m apart over 60s -> ~0.018 m/s, well under the 0.5 m/s threshold.
      _point(lat: 13.7563, lon: 100.5018, time: t0),
      _point(lat: 13.75631, lon: 100.5018, time: t0.add(const Duration(seconds: 60))),
    ];

    final stats = calculator.calculate(points);

    expect(stats.movingTime, Duration.zero);
    expect(stats.averageSpeed, 0);
  });

  test('elevation gain/loss only commits once accumulated delta clears the noise threshold', () {
    final points = [
      _point(lat: 13.7563, lon: 100.5018, time: t0, altitude: 10),
      _point(lat: 13.7563, lon: 100.5018, time: t0.add(const Duration(seconds: 10)), altitude: 12),
      _point(lat: 13.7563, lon: 100.5018, time: t0.add(const Duration(seconds: 20)), altitude: 9),
    ];

    final stats = calculator.calculate(points);

    expect(stats.elevationGain, closeTo(2, 0.001));
    expect(stats.elevationLoss, closeTo(3, 0.001));
  });

  test('small altitude jitter under the noise threshold is ignored', () {
    final points = [
      _point(lat: 13.7563, lon: 100.5018, time: t0, altitude: 10),
      _point(lat: 13.7563, lon: 100.5018, time: t0.add(const Duration(seconds: 10)), altitude: 10.4),
      _point(lat: 13.7563, lon: 100.5018, time: t0.add(const Duration(seconds: 20)), altitude: 9.8),
    ];

    final stats = calculator.calculate(points);

    expect(stats.elevationGain, 0);
    expect(stats.elevationLoss, 0);
  });
}
