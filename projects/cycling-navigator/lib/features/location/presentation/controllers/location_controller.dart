import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../data/repositories/geolocator_location_repository.dart';
import '../../domain/entities/location_point.dart';
import '../../domain/repositories/location_repository.dart';

part 'location_controller.g.dart';

@Riverpod(keepAlive: true)
LocationRepository locationRepository(Ref ref) => GeolocatorLocationRepository();

@riverpod
Stream<LocationPoint> currentLocationStream(Ref ref) {
  return ref.watch(locationRepositoryProvider).watchLocation();
}
