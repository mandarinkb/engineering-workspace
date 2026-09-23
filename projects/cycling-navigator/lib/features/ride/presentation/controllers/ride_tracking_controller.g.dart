// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'ride_tracking_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(rideDatabase)
final rideDatabaseProvider = RideDatabaseProvider._();

final class RideDatabaseProvider
    extends $FunctionalProvider<RideDatabase, RideDatabase, RideDatabase>
    with $Provider<RideDatabase> {
  RideDatabaseProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'rideDatabaseProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$rideDatabaseHash();

  @$internal
  @override
  $ProviderElement<RideDatabase> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  RideDatabase create(Ref ref) {
    return rideDatabase(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RideDatabase value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RideDatabase>(value),
    );
  }
}

String _$rideDatabaseHash() => r'4ed62d2e2b80658df361156c4f33f12d25d832af';

@ProviderFor(rideRepository)
final rideRepositoryProvider = RideRepositoryProvider._();

final class RideRepositoryProvider
    extends $FunctionalProvider<RideRepository, RideRepository, RideRepository>
    with $Provider<RideRepository> {
  RideRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'rideRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$rideRepositoryHash();

  @$internal
  @override
  $ProviderElement<RideRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  RideRepository create(Ref ref) {
    return rideRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RideRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RideRepository>(value),
    );
  }
}

String _$rideRepositoryHash() => r'c699baa852388c52a37bc9a11720abde4be21e0b';

@ProviderFor(RideTrackingController)
final rideTrackingControllerProvider = RideTrackingControllerProvider._();

final class RideTrackingControllerProvider
    extends $NotifierProvider<RideTrackingController, RideTrackingState> {
  RideTrackingControllerProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'rideTrackingControllerProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$rideTrackingControllerHash();

  @$internal
  @override
  RideTrackingController create() => RideTrackingController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(RideTrackingState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<RideTrackingState>(value),
    );
  }
}

String _$rideTrackingControllerHash() =>
    r'211c56262569956cb7a0c733b6cb559af9f5ab41';

abstract class _$RideTrackingController extends $Notifier<RideTrackingState> {
  RideTrackingState build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<RideTrackingState, RideTrackingState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<RideTrackingState, RideTrackingState>,
              RideTrackingState,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
