// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_regions_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(mapRepository)
final mapRepositoryProvider = MapRepositoryProvider._();

final class MapRepositoryProvider
    extends $FunctionalProvider<MapRepository, MapRepository, MapRepository>
    with $Provider<MapRepository> {
  MapRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'mapRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$mapRepositoryHash();

  @$internal
  @override
  $ProviderElement<MapRepository> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  MapRepository create(Ref ref) {
    return mapRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(MapRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<MapRepository>(value),
    );
  }
}

String _$mapRepositoryHash() => r'181b6aa317aa59e6b64bebd196cc82ddce1303b7';

@ProviderFor(isRegionDownloaded)
final isRegionDownloadedProvider = IsRegionDownloadedFamily._();

final class IsRegionDownloadedProvider
    extends $FunctionalProvider<AsyncValue<bool>, bool, FutureOr<bool>>
    with $FutureModifier<bool>, $FutureProvider<bool> {
  IsRegionDownloadedProvider._({
    required IsRegionDownloadedFamily super.from,
    required MapRegion super.argument,
  }) : super(
         retry: null,
         name: r'isRegionDownloadedProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$isRegionDownloadedHash();

  @override
  String toString() {
    return r'isRegionDownloadedProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<bool> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<bool> create(Ref ref) {
    final argument = this.argument as MapRegion;
    return isRegionDownloaded(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is IsRegionDownloadedProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$isRegionDownloadedHash() =>
    r'3479b773d81bd5678a30979f0f7b7314fec141d1';

final class IsRegionDownloadedFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<bool>, MapRegion> {
  IsRegionDownloadedFamily._()
    : super(
        retry: null,
        name: r'isRegionDownloadedProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  IsRegionDownloadedProvider call(MapRegion region) =>
      IsRegionDownloadedProvider._(argument: region, from: this);

  @override
  String toString() => r'isRegionDownloadedProvider';
}

@ProviderFor(regionSizeBytes)
final regionSizeBytesProvider = RegionSizeBytesFamily._();

final class RegionSizeBytesProvider
    extends $FunctionalProvider<AsyncValue<int>, int, FutureOr<int>>
    with $FutureModifier<int>, $FutureProvider<int> {
  RegionSizeBytesProvider._({
    required RegionSizeBytesFamily super.from,
    required MapRegion super.argument,
  }) : super(
         retry: null,
         name: r'regionSizeBytesProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$regionSizeBytesHash();

  @override
  String toString() {
    return r'regionSizeBytesProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<int> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<int> create(Ref ref) {
    final argument = this.argument as MapRegion;
    return regionSizeBytes(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is RegionSizeBytesProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$regionSizeBytesHash() => r'1a15e3d50ddd5064258115da68c9ae90dae78571';

final class RegionSizeBytesFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<int>, MapRegion> {
  RegionSizeBytesFamily._()
    : super(
        retry: null,
        name: r'regionSizeBytesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  RegionSizeBytesProvider call(MapRegion region) =>
      RegionSizeBytesProvider._(argument: region, from: this);

  @override
  String toString() => r'regionSizeBytesProvider';
}

/// The style to hand `MapLibreMap.styleString`, plus which region it came
/// from (used only to warn when "recenter" would land outside downloaded
/// data — Phase 4). Picks the first *downloaded* region in
/// [MapRepository.availableRegions] order, i.e. the most detailed one
/// available; [region] is null when falling back to the (world-coverage)
/// online demo style from Phase 3.

@ProviderFor(activeMapStyle)
final activeMapStyleProvider = ActiveMapStyleProvider._();

/// The style to hand `MapLibreMap.styleString`, plus which region it came
/// from (used only to warn when "recenter" would land outside downloaded
/// data — Phase 4). Picks the first *downloaded* region in
/// [MapRepository.availableRegions] order, i.e. the most detailed one
/// available; [region] is null when falling back to the (world-coverage)
/// online demo style from Phase 3.

final class ActiveMapStyleProvider
    extends
        $FunctionalProvider<
          AsyncValue<({MapRegion? region, String styleString})>,
          ({MapRegion? region, String styleString}),
          FutureOr<({MapRegion? region, String styleString})>
        >
    with
        $FutureModifier<({MapRegion? region, String styleString})>,
        $FutureProvider<({MapRegion? region, String styleString})> {
  /// The style to hand `MapLibreMap.styleString`, plus which region it came
  /// from (used only to warn when "recenter" would land outside downloaded
  /// data — Phase 4). Picks the first *downloaded* region in
  /// [MapRepository.availableRegions] order, i.e. the most detailed one
  /// available; [region] is null when falling back to the (world-coverage)
  /// online demo style from Phase 3.
  ActiveMapStyleProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'activeMapStyleProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$activeMapStyleHash();

  @$internal
  @override
  $FutureProviderElement<({MapRegion? region, String styleString})>
  $createElement($ProviderPointer pointer) => $FutureProviderElement(pointer);

  @override
  FutureOr<({MapRegion? region, String styleString})> create(Ref ref) {
    return activeMapStyle(ref);
  }
}

String _$activeMapStyleHash() => r'1e7214da07b8b28ed38c6c32080e235f1d6c19ea';
