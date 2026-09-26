import 'dart:convert';

import '../../../ride/domain/services/geo_math.dart';
import '../../domain/entities/cycling_route.dart';
import '../../domain/entities/geo_point.dart';
import '../../domain/errors/routing_exceptions.dart';
import '../../domain/repositories/routing_repository.dart';
import '../../domain/services/polyline_codec.dart';
import '../datasources/valhalla_channel.dart';
import '../provisioning/bundled_valhalla_tiles_provisioner.dart';

/// Beyond this, a route's snapped endpoint is treated as "not actually
/// near what was requested" rather than a normal snap gap — see the
/// filtering loop in [parseValhallaRoutesResponse] below for why this
/// exists at all.
const _maxSnapDistanceMeters = 500.0;

class ValhallaRoutingRepository implements RoutingRepository {
  ValhallaRoutingRepository(this._channel, this._provisioner);

  final ValhallaChannel _channel;
  final BundledValhallaTilesProvisioner _provisioner;

  @override
  Future<List<CyclingRoute>> calculateRoutes(GeoPoint start, GeoPoint destination) async {
    final tilesDirName = await _provisioner.ensureProvisioned();

    final responseJson = await _channel.calculateRoute(
      tilesDirName: tilesDirName,
      start: start,
      destination: destination,
    );

    return parseValhallaRoutesResponse(responseJson, start, destination);
  }
}

/// Parses the JSON handed back by the native side (see `MainActivity.kt`)
/// into domain [CyclingRoute]s. Kept as a standalone function, separate
/// from the platform-channel call itself, so this parsing/unit-conversion
/// logic — the part actually worth testing — can be unit tested without
/// touching a real `MethodChannel` (this project's convention for
/// plugin-backed repositories: platform-channel wiring itself is verified
/// manually on-device instead, see `GeolocatorLocationRepository`).
List<CyclingRoute> parseValhallaRoutesResponse(
  String responseJson,
  GeoPoint start,
  GeoPoint destination,
) {
  final decoded = jsonDecode(responseJson) as Map<String, dynamic>;

  if (decoded['status'] != 0) {
    throw RoutingException(decoded['statusMessage'] as String? ?? 'Unknown routing error');
  }

  final routesJson = decoded['routes'] as List<dynamic>? ?? const [];
  if (routesJson.isEmpty) {
    throw const RoutingException('Valhalla returned no routes');
  }

  final routes = <CyclingRoute>[];
  for (final routeJson in routesJson.cast<Map<String, dynamic>>()) {
    final points = decodePolyline(routeJson['shape'] as String);
    if (points.isEmpty) continue;

    // Valhalla always "succeeds" by snapping start/destination to the
    // nearest road it can reach in the currently loaded tiles — even when
    // that's kilometers away outside them. It does not throw in that
    // case (verified on-device: a destination tapped outside the
    // downloaded bbox came back as a normal-looking route that just
    // silently jumped to a straight line for the final stretch). A huge
    // snap gap here is the only signal available to tell that apart from
    // the ordinary few-meters-to-low-hundreds gap (GPS drift, a
    // destination pinned mid-block) that `MapScreen._redrawRoutes`'s
    // start/destination "stitch" is meant to paper over cosmetically —
    // stitching over a multi-kilometer gap instead misrepresents it as a
    // real route.
    final startSnapMeters = haversineMeters(start.lat, start.lon, points.first.lat, points.first.lon);
    final destinationSnapMeters = haversineMeters(destination.lat, destination.lon, points.last.lat, points.last.lon);
    if (startSnapMeters > _maxSnapDistanceMeters || destinationSnapMeters > _maxSnapDistanceMeters) continue;

    routes.add(
      CyclingRoute(
        start: start,
        destination: destination,
        // OSRM-format routes (Phase 7) report distance directly in
        // meters — unlike Valhalla's native `json` format (Phase 6),
        // whose `distanceKm` needed a *1000 conversion here. Do not add
        // that conversion back, or every distance comes out 1000x too
        // large.
        distanceMeters: (routeJson['distanceMeters'] as num).toDouble(),
        duration: Duration(milliseconds: ((routeJson['durationSeconds'] as num) * 1000).round()),
        points: points,
      ),
    );
  }

  if (routes.isEmpty) {
    throw const RoutingException(
      'No routable road found near the start or destination — likely outside the downloaded routing area',
    );
  }

  return routes;
}
