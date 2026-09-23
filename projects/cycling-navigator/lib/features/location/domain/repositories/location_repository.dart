import '../entities/location_point.dart';

/// May throw [LocationServiceDisabledException] or
/// [LocationPermissionDeniedException] (see core/errors) — implementations
/// must check service/permission status before emitting any point.
abstract class LocationRepository {
  Stream<LocationPoint> watchLocation();

  Future<LocationPoint?> getCurrentLocation();
}
