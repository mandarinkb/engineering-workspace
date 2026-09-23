import '../entities/cycling_route.dart';
import '../entities/geo_point.dart';

abstract class RoutingRepository {
  /// May throw [RoutingException] (see domain/errors) — e.g. when either
  /// point falls outside the routing data currently available on-device.
  Future<List<CyclingRoute>> calculateRoutes(GeoPoint start, GeoPoint destination);
}
