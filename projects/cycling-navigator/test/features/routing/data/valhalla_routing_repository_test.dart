import 'package:cycling_navigator/features/routing/data/repositories/valhalla_routing_repository.dart';
import 'package:cycling_navigator/features/routing/domain/entities/geo_point.dart';
import 'package:cycling_navigator/features/routing/domain/errors/routing_exceptions.dart';
import 'package:flutter_test/flutter_test.dart';

/// Mirrors `decodePolyline()`'s algorithm in reverse, at the same default
/// precision (1e6) — just enough to build realistic `shape` fixtures for
/// these tests (points that land at exact/known distances from a given
/// start/destination), not a general-purpose encoder.
String _encodePolyline6(List<(double lat, double lon)> points) {
  final buffer = StringBuffer();
  var prevLat = 0;
  var prevLon = 0;
  for (final (lat, lon) in points) {
    final lat6 = (lat * 1e6).round();
    final lon6 = (lon * 1e6).round();
    _encodeValue(lat6 - prevLat, buffer);
    _encodeValue(lon6 - prevLon, buffer);
    prevLat = lat6;
    prevLon = lon6;
  }
  return buffer.toString();
}

void _encodeValue(int value, StringBuffer buffer) {
  var v = value < 0 ? ~(value << 1) : (value << 1);
  while (v >= 0x20) {
    buffer.writeCharCode((0x20 | (v & 0x1f)) + 63);
    v >>= 5;
  }
  buffer.writeCharCode(v + 63);
}

void main() {
  const start = GeoPoint(lat: 13.7563, lon: 100.5018);
  const destination = GeoPoint(lat: 13.76, lon: 100.51);
  // A route whose decoded shape starts/ends exactly on start/destination
  // — the common case for these tests, where snap-distance filtering
  // (Phase 7) should never reject anything.
  final onPointShape = _encodePolyline6([(start.lat, start.lon), (destination.lat, destination.lon)]);

  group('parseValhallaRoutesResponse', () {
    test('parses multiple alternates in Valhalla-ranked order', () {
      final routes = parseValhallaRoutesResponse(
        '{"status":0,"statusMessage":"","routes":['
        '{"distanceMeters":2300.0,"durationSeconds":540.0,"shape":"$onPointShape"},'
        '{"distanceMeters":2600.0,"durationSeconds":480.0,"shape":"$onPointShape"}'
        ']}',
        start,
        destination,
      );

      expect(routes, hasLength(2));
      expect(routes[0].distanceMeters, 2300.0);
      expect(routes[0].duration, const Duration(seconds: 540));
      expect(routes[1].distanceMeters, 2600.0);
      expect(routes[1].duration, const Duration(seconds: 480));
    });

    // Regression test for the exact unit-mismatch bug this parser has to
    // avoid: OSRM-format routes (Phase 7) report distance directly in
    // meters, unlike Valhalla's native `json` format (Phase 6) which used
    // kilometers and needed a *1000 conversion. Carrying that conversion
    // over here would silently make every distance 1000x too large.
    test('does not convert distanceMeters — OSRM format is already meters', () {
      final routes = parseValhallaRoutesResponse(
        '{"status":0,"statusMessage":"","routes":[{"distanceMeters":1500.0,"durationSeconds":300.0,"shape":"$onPointShape"}]}',
        start,
        destination,
      );

      expect(routes.single.distanceMeters, 1500.0);
    });

    test('carries the requested start/destination onto every route', () {
      final routes = parseValhallaRoutesResponse(
        '{"status":0,"statusMessage":"","routes":[{"distanceMeters":100.0,"durationSeconds":10.0,"shape":"$onPointShape"}]}',
        start,
        destination,
      );

      expect(routes.single.start, start);
      expect(routes.single.destination, destination);
    });

    test('throws RoutingException on a non-zero status', () {
      expect(
        () => parseValhallaRoutesResponse(
          '{"status":-1,"statusMessage":"No suitable edges near location","routes":[]}',
          start,
          destination,
        ),
        throwsA(isA<RoutingException>()),
      );
    });

    test('throws RoutingException when routes come back empty despite status 0', () {
      expect(
        () => parseValhallaRoutesResponse('{"status":0,"statusMessage":"","routes":[]}', start, destination),
        throwsA(isA<RoutingException>()),
      );
    });

    // Regression test for the on-device finding this session: tapping a
    // destination outside the downloaded routing tiles did NOT throw —
    // Valhalla silently snapped to the nearest reachable road, however
    // far away, and the app drew a straight line across the gap as if it
    // were a real route. A ~55m snap gap (destination shifted by 0.0005°,
    // ~1 degree latitude ≈ 111.32km) is well within the normal "GPS
    // drift / off-road pin" range this shouldn't reject.
    test('keeps a route whose snapped destination is only a normal small gap away', () {
      final shape = _encodePolyline6([(start.lat, start.lon), (destination.lat + 0.0005, destination.lon)]);
      final routes = parseValhallaRoutesResponse(
        '{"status":0,"statusMessage":"","routes":[{"distanceMeters":100.0,"durationSeconds":10.0,"shape":"$shape"}]}',
        start,
        destination,
      );

      expect(routes, hasLength(1));
    });

    test('drops a route whose snapped destination is kilometers away, throwing if none qualify', () {
      // destination shifted by 0.05° (~5.6km) — well outside any
      // plausible "off-road pin" gap.
      final shape = _encodePolyline6([(start.lat, start.lon), (destination.lat + 0.05, destination.lon)]);

      expect(
        () => parseValhallaRoutesResponse(
          '{"status":0,"statusMessage":"","routes":[{"distanceMeters":100.0,"durationSeconds":10.0,"shape":"$shape"}]}',
          start,
          destination,
        ),
        throwsA(isA<RoutingException>()),
      );
    });

    test('keeps the alternates that snap close while dropping the one that snaps kilometers away', () {
      final farShape = _encodePolyline6([(start.lat, start.lon), (destination.lat + 0.05, destination.lon)]);
      final routes = parseValhallaRoutesResponse(
        '{"status":0,"statusMessage":"","routes":['
        '{"distanceMeters":100.0,"durationSeconds":10.0,"shape":"$onPointShape"},'
        '{"distanceMeters":200.0,"durationSeconds":20.0,"shape":"$farShape"}'
        ']}',
        start,
        destination,
      );

      expect(routes, hasLength(1));
      expect(routes.single.distanceMeters, 100.0);
    });
  });
}
