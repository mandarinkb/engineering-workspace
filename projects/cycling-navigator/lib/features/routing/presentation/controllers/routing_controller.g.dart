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

@ProviderFor(RouteToDestination)
final routeToDestinationProvider = RouteToDestinationProvider._();

final class RouteToDestinationProvider
    extends $AsyncNotifierProvider<RouteToDestination, CyclingRoute?> {
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
    r'e32d9b196740b3a5fb33d4339b13fe6a666aa15b';

abstract class _$RouteToDestination extends $AsyncNotifier<CyclingRoute?> {
  FutureOr<CyclingRoute?> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<CyclingRoute?>, CyclingRoute?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<CyclingRoute?>, CyclingRoute?>,
              AsyncValue<CyclingRoute?>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
