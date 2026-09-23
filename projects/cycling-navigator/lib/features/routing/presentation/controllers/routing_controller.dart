import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../data/datasources/valhalla_channel.dart';
import '../../data/provisioning/bundled_valhalla_tiles_provisioner.dart';
import '../../data/repositories/valhalla_routing_repository.dart';
import '../../domain/entities/cycling_route.dart';
import '../../domain/entities/geo_point.dart';
import '../../domain/repositories/routing_repository.dart';

part 'routing_controller.g.dart';

@Riverpod(keepAlive: true)
RoutingRepository routingRepository(Ref ref) => ValhallaRoutingRepository(
      const ValhallaChannel(),
      BundledValhallaTilesProvisioner(),
    );

@riverpod
class RouteToDestination extends _$RouteToDestination {
  @override
  Future<CyclingRoute?> build() async => null;

  Future<void> calculate(GeoPoint start, GeoPoint destination) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final routes = await ref.read(routingRepositoryProvider).calculateRoutes(start, destination);
      return routes.first;
    });
  }

  void clear() {
    state = const AsyncData(null);
  }
}
