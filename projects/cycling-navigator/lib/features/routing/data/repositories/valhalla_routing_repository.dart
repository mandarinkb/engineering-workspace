import 'dart:convert';

import '../../domain/entities/cycling_route.dart';
import '../../domain/entities/geo_point.dart';
import '../../domain/errors/routing_exceptions.dart';
import '../../domain/repositories/routing_repository.dart';
import '../../domain/services/polyline_codec.dart';
import '../datasources/valhalla_channel.dart';
import '../provisioning/bundled_valhalla_tiles_provisioner.dart';

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

    final decoded = jsonDecode(responseJson) as Map<String, dynamic>;

    if (decoded['status'] != 0) {
      throw RoutingException(decoded['statusMessage'] as String? ?? 'Unknown routing error');
    }

    return [
      CyclingRoute(
        start: start,
        destination: destination,
        // Valhalla's default units are kilometers.
        distanceMeters: (decoded['distanceKm'] as num).toDouble() * 1000,
        duration: Duration(milliseconds: ((decoded['timeSeconds'] as num) * 1000).round()),
        points: decodePolyline(decoded['shape'] as String),
      ),
    ];
  }
}
