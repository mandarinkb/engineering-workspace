// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'routing_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(routingRepository)
final routingRepositoryProvider = RoutingRepositoryProvider._();

final class RoutingRepositoryProvider
    extends
        $FunctionalProvider<
          RoutingRepository,
          RoutingRepository,
          RoutingRepository
        >
    with $Provider<RoutingRepository> {
  RoutingRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'routingRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$routingRepositoryHash();

  @$internal
  @override
  $ProviderElement<RoutingRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  RoutingRepository create(Ref ref) {
    return routingRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RoutingRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RoutingRepository>(value),
    );
  }
}

String _$routingRepositoryHash() => r'eb397c1cef8d0d3c362c479f290070564ecc3682';

/// All alternates Valhalla returned for the current start/destination pair
/// (Phase 7 — up to 3: 1 primary + 2 alternates, see `MainActivity.kt`),
/// in the order Valhalla ranked them. [SelectedRouteIndex] tracks which one
/// the user picked to actually ride.

@ProviderFor(RouteToDestination)
final routeToDestinationProvider = RouteToDestinationProvider._();

/// All alternates Valhalla returned for the current start/destination pair
/// (Phase 7 — up to 3: 1 primary + 2 alternates, see `MainActivity.kt`),
/// in the order Valhalla ranked them. [SelectedRouteIndex] tracks which one
/// the user picked to actually ride.
final class RouteToDestinationProvider
    extends $AsyncNotifierProvider<RouteToDestination, List<CyclingRoute>> {
  /// All alternates Valhalla returned for the current start/destination pair
  /// (Phase 7 — up to 3: 1 primary + 2 alternates, see `MainActivity.kt`),
  /// in the order Valhalla ranked them. [SelectedRouteIndex] tracks which one
  /// the user picked to actually ride.
  RouteToDestinationProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'routeToDestinationProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$routeToDestinationHash();

  @$internal
  @override
  RouteToDestination create() => RouteToDestination();
}

String _$routeToDestinationHash() =>
    r'7997463fc49adc9bac95973d49f9b2b67bbf5253';

/// All alternates Valhalla returned for the current start/destination pair
/// (Phase 7 — up to 3: 1 primary + 2 alternates, see `MainActivity.kt`),
/// in the order Valhalla ranked them. [SelectedRouteIndex] tracks which one
/// the user picked to actually ride.

abstract class _$RouteToDestination extends $AsyncNotifier<List<CyclingRoute>> {
  FutureOr<List<CyclingRoute>> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<List<CyclingRoute>>, List<CyclingRoute>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<List<CyclingRoute>>, List<CyclingRoute>>,
              AsyncValue<List<CyclingRoute>>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

/// Index into [RouteToDestination]'s list — which alternate is currently
/// highlighted/selected for riding. Plain `int` state kept separate from
/// the route list itself (rather than bundling both into one state object)
/// because selection changes on every tap while the route list only
/// changes once per `calculate()` call.

@ProviderFor(SelectedRouteIndex)
final selectedRouteIndexProvider = SelectedRouteIndexProvider._();

/// Index into [RouteToDestination]'s list — which alternate is currently
/// highlighted/selected for riding. Plain `int` state kept separate from
/// the route list itself (rather than bundling both into one state object)
/// because selection changes on every tap while the route list only
/// changes once per `calculate()` call.
final class SelectedRouteIndexProvider
    extends $NotifierProvider<SelectedRouteIndex, int> {
  /// Index into [RouteToDestination]'s list — which alternate is currently
  /// highlighted/selected for riding. Plain `int` state kept separate from
  /// the route list itself (rather than bundling both into one state object)
  /// because selection changes on every tap while the route list only
  /// changes once per `calculate()` call.
  SelectedRouteIndexProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'selectedRouteIndexProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$selectedRouteIndexHash();

  @$internal
  @override
  SelectedRouteIndex create() => SelectedRouteIndex();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(int value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<int>(value),
    );
  }
}

String _$selectedRouteIndexHash() =>
    r'418a798d6bf74ae7365dd5d72371061a4162258c';

/// Index into [RouteToDestination]'s list — which alternate is currently
/// highlighted/selected for riding. Plain `int` state kept separate from
/// the route list itself (rather than bundling both into one state object)
/// because selection changes on every tap while the route list only
/// changes once per `calculate()` call.

abstract class _$SelectedRouteIndex extends $Notifier<int> {
  int build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<int, int>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<int, int>,
              int,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

/// Convenience read of "the route the user would actually ride right now",
/// clamped against the current route list so a stale index (e.g. right
/// after [RouteToDestination.clear]) can never index out of range.

@ProviderFor(selectedRoute)
final selectedRouteProvider = SelectedRouteProvider._();

/// Convenience read of "the route the user would actually ride right now",
/// clamped against the current route list so a stale index (e.g. right
/// after [RouteToDestination.clear]) can never index out of range.

final class SelectedRouteProvider
    extends $FunctionalProvider<CyclingRoute?, CyclingRoute?, CyclingRoute?>
    with $Provider<CyclingRoute?> {
  /// Convenience read of "the route the user would actually ride right now",
  /// clamped against the current route list so a stale index (e.g. right
  /// after [RouteToDestination.clear]) can never index out of range.
  SelectedRouteProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'selectedRouteProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$selectedRouteHash();

  @$internal
  @override
  $ProviderElement<CyclingRoute?> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  CyclingRoute? create(Ref ref) {
    return selectedRoute(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(CyclingRoute? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<CyclingRoute?>(value),
    );
  }
}

String _$selectedRouteHash() => r'0d4bc640fe8decc31072a595124eb64d70e5c302';
