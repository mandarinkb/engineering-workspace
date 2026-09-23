import 'package:cycling_navigator/features/routing/domain/services/polyline_codec.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('decodes Google\'s canonical documented example (precision 1e5)', () {
    // From https://developers.google.com/maps/documentation/utilities/polylinealgorithm
    final points = decodePolyline('_p~iF~ps|U_ulLnnqC_mqNvxq`@', precision: 1e5);

    expect(points, hasLength(3));
    expect(points[0].lat, closeTo(38.5, 0.00001));
    expect(points[0].lon, closeTo(-120.2, 0.00001));
    expect(points[1].lat, closeTo(40.7, 0.00001));
    expect(points[1].lon, closeTo(-120.95, 0.00001));
    expect(points[2].lat, closeTo(43.252, 0.00001));
    expect(points[2].lon, closeTo(-126.453, 0.00001));
  });

  test('empty string decodes to no points', () {
    expect(decodePolyline(''), isEmpty);
  });
}
