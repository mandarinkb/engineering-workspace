// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'location_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(locationRepository)
final locationRepositoryProvider = LocationRepositoryProvider._();

final class LocationRepositoryProvider
    extends
        $FunctionalProvider<
          LocationRepository,
          LocationRepository,
          LocationRepository
        >
    with $Provider<LocationRepository> {
  LocationRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'locationRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$locationRepositoryHash();

  @$internal
  @override
  $ProviderElement<LocationRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  LocationRepository create(Ref ref) {
    return locationRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(LocationRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<LocationRepository>(value),
    );
  }
}

String _$locationRepositoryHash() =>
    r'ecaa298f0185d5b69eb8bc2e4a07c705df726b4f';

@ProviderFor(currentLocationStream)
final currentLocationStreamProvider = CurrentLocationStreamProvider._();

final class CurrentLocationStreamProvider
    extends
        $FunctionalProvider<
          AsyncValue<LocationPoint>,
          LocationPoint,
          Stream<LocationPoint>
        >
    with $FutureModifier<LocationPoint>, $StreamProvider<LocationPoint> {
  CurrentLocationStreamProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'currentLocationStreamProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$currentLocationStreamHash();

  @$internal
  @override
  $StreamProviderElement<LocationPoint> $createElement(
    $ProviderPointer pointer,
  ) => $StreamProviderElement(pointer);

  @override
  Stream<LocationPoint> create(Ref ref) {
    return currentLocationStream(ref);
  }
}

String _$currentLocationStreamHash() =>
    r'cb5f5f8810d52b5cf3a182e2961be93f3cf680d1';
