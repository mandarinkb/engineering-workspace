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

/// All alternates Valhalla returned for the current start/destination pair
/// (Phase 7 — up to 3: 1 primary + 2 alternates, see `MainActivity.kt`),
/// in the order Valhalla ranked them. [SelectedRouteIndex] tracks which one
/// the user picked to actually ride.
@riverpod
class RouteToDestination extends _$RouteToDestination {
  @override
  Future<List<CyclingRoute>> build() async => const [];

  Future<void> calculate(GeoPoint start, GeoPoint destination) async {
    // A fresh calculation invalidates whatever the user had selected from
    // the previous set of alternates — reset to Valhalla's top pick before
    // the new list even arrives, so the map never briefly highlights a
    // route index that doesn't exist yet (e.g. old selection was index 2,
    // new result only has 1 route).
    ref.read(selectedRouteIndexProvider.notifier).select(0);
    state = const AsyncLoading();
    state = await AsyncValue.guard(
      () => ref.read(routingRepositoryProvider).calculateRoutes(start, destination),
    );
  }

  void clear() {
    state = const AsyncData([]);
    ref.read(selectedRouteIndexProvider.notifier).select(0);
  }
}

/// Index into [RouteToDestination]'s list — which alternate is currently
/// highlighted/selected for riding. Plain `int` state kept separate from
/// the route list itself (rather than bundling both into one state object)
/// because selection changes on every tap while the route list only
/// changes once per `calculate()` call.
@riverpod
class SelectedRouteIndex extends _$SelectedRouteIndex {
  @override
  int build() => 0;

  void select(int index) => state = index;
}

/// Convenience read of "the route the user would actually ride right now",
/// clamped against the current route list so a stale index (e.g. right
/// after [RouteToDestination.clear]) can never index out of range.
@riverpod
CyclingRoute? selectedRoute(Ref ref) {
  final routes = ref.watch(routeToDestinationProvider).value ?? const [];
  final index = ref.watch(selectedRouteIndexProvider);
  if (index < 0 || index >= routes.length) return null;
  return routes[index];
}
