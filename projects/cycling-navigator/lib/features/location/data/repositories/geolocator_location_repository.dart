import 'package:geolocator/geolocator.dart' hide LocationServiceDisabledException;

import '../../../../core/errors/location_exceptions.dart';
import '../../domain/entities/location_point.dart';
import '../../domain/repositories/location_repository.dart';

class GeolocatorLocationRepository implements LocationRepository {
  static const _settings = LocationSettings(
    accuracy: LocationAccuracy.high,
    distanceFilter: 5,
  );

  @override
  Stream<LocationPoint> watchLocation() async* {
    await _ensurePermission();
    yield* Geolocator.getPositionStream(locationSettings: _settings)
        .map(_toLocationPoint);
  }

  @override
  Future<LocationPoint?> getCurrentLocation() async {
    await _ensurePermission();
    final position = await Geolocator.getCurrentPosition(locationSettings: _settings);
    return _toLocationPoint(position);
  }

  Future<void> _ensurePermission() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      throw const LocationServiceDisabledException();
    }

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }

    if (permission == LocationPermission.denied) {
      throw const LocationPermissionDeniedException(isPermanent: false);
    }
    if (permission == LocationPermission.deniedForever) {
      throw const LocationPermissionDeniedException(isPermanent: true);
    }
  }

  LocationPoint _toLocationPoint(Position position) => LocationPoint(
        latitude: position.latitude,
        longitude: position.longitude,
        timestamp: position.timestamp,
        accuracy: position.accuracy,
        altitude: position.altitude,
        speed: position.speed,
        heading: position.heading,
      );
}
