import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart' show rootBundle;
import 'package:path_provider/path_provider.dart';

import '../../domain/entities/map_region.dart';
import '../../domain/repositories/map_repository.dart';

/// Phase 4 MVP: two bundled regions, both shipped inside the app itself
/// (see CYCLING-NAVIGATOR-TODO.md section 5) — a whole-Thailand overview
/// (zoom 0-11, ~44MB, city/highway-level only, no small streets) and a
/// Bangkok Metro detail package (zoom 0-16, ~42MB, full street/soi detail
/// but Bangkok-only). "Downloading" is really copying bundled asset bytes
/// onto the real filesystem, which is what makes the style loadable via a
/// plain absolute path and the pmtiles source loadable via
/// `pmtiles://file://`. A future `HttpDownloadMapRepository` fetching from
/// a real host implements the same [MapRepository] interface without any
/// caller needing to change — bundling only works because these two
/// regions together are still small enough (~86MB) to ship in the APK;
/// more regions or higher zoom for a bigger area would need that real host.
class BundledAssetMapRepository implements MapRepository {
  static const _bangkokMetro = MapRegion(
    id: 'bangkok-metro',
    name: 'Bangkok Metro (detailed)',
    minLat: 13.55,
    minLon: 100.32,
    maxLat: 13.97,
    maxLon: 100.94,
  );

  static const _thailand = MapRegion(
    id: 'thailand',
    name: 'Thailand (overview)',
    minLat: 5.6,
    minLon: 97.3,
    maxLat: 20.5,
    maxLon: 105.7,
  );

  /// Order matters for [MapRepository.availableRegions] consumers that want
  /// "the best available style" (see `activeMapStyleProvider`) — most
  /// detailed first.
  @override
  List<MapRegion> get availableRegions => const [_bangkokMetro, _thailand];

  @override
  Future<void> downloadRegion(MapRegion region) async {
    final dir = await _regionDirectory();
    await dir.create(recursive: true);

    final pmtilesFile = await _pmtilesFile(region);
    final styleFile = await _styleFile(region);

    final pmtilesBytes = await rootBundle.load('assets/offline_regions/${region.id}.pmtiles');
    await pmtilesFile.writeAsBytes(pmtilesBytes.buffer.asUint8List(), flush: true);

    final styleTemplate =
        await rootBundle.loadString('assets/offline_regions/basemap-style-template.json');
    final style = jsonDecode(styleTemplate) as Map<String, dynamic>;
    final sources = style['sources'] as Map<String, dynamic>;
    final protomapsSource = sources['protomaps'] as Map<String, dynamic>;
    protomapsSource['url'] = 'pmtiles://file://${pmtilesFile.path}';
    await styleFile.writeAsString(jsonEncode(style), flush: true);
  }

  @override
  Future<void> deleteRegion(MapRegion region) async {
    final pmtilesFile = await _pmtilesFile(region);
    final styleFile = await _styleFile(region);
    if (await pmtilesFile.exists()) await pmtilesFile.delete();
    if (await styleFile.exists()) await styleFile.delete();
  }

  @override
  Future<bool> isRegionAvailable(MapRegion region) async {
    final pmtilesFile = await _pmtilesFile(region);
    final styleFile = await _styleFile(region);
    return pmtilesFile.exists().then((p) async => p && await styleFile.exists());
  }

  @override
  Future<int> regionSizeBytes(MapRegion region) async {
    final pmtilesFile = await _pmtilesFile(region);
    final styleFile = await _styleFile(region);
    var total = 0;
    if (await pmtilesFile.exists()) total += await pmtilesFile.length();
    if (await styleFile.exists()) total += await styleFile.length();
    return total;
  }

  @override
  Future<String?> stylePathForRegion(MapRegion region) async {
    final styleFile = await _styleFile(region);
    return await styleFile.exists() ? styleFile.path : null;
  }

  Future<Directory> _regionDirectory() async {
    final supportDir = await getApplicationSupportDirectory();
    return Directory('${supportDir.path}/offline_regions');
  }

  Future<File> _pmtilesFile(MapRegion region) async {
    final dir = await _regionDirectory();
    return File('${dir.path}/${region.id}.pmtiles');
  }

  Future<File> _styleFile(MapRegion region) async {
    final dir = await _regionDirectory();
    return File('${dir.path}/${region.id}.style.json');
  }
}
