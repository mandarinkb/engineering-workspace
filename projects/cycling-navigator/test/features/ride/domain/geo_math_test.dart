import 'package:cycling_navigator/features/ride/domain/services/geo_math.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('one degree of latitude is approximately 111.32 km', () {
    final meters = haversineMeters(13.0, 100.0, 14.0, 100.0);
    expect(meters, closeTo(111320, 200));
  });

  test('same point has zero distance', () {
    final meters = haversineMeters(13.7563, 100.5018, 13.7563, 100.5018);
    expect(meters, closeTo(0, 0.001));
  });
}
