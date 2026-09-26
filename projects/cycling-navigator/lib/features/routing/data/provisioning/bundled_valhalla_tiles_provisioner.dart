import 'dart:io';

import 'package:archive/archive.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:path_provider/path_provider.dart';

/// Extracts the bundled Valhalla routing tiles (a zip asset — see
/// CYCLING-NAVIGATOR-TODO.md Phase 7) onto the real filesystem so the
/// native `ValhallaActor` can read them by path. Not yet unified with the
/// Phase 4 `MapRepository` region system — this is its own small, fixed
/// region until that's worth generalizing.
///
/// Bundled as a single zip, extracted at runtime, rather than listing
/// every `.gph` file in `pubspec.yaml` (Phase 6a's original tiny test
/// bbox only had 6 files and could get away with that — this region's 42
/// files nested up to 4 levels deep can't).
class BundledValhallaTilesProvisioner {
  static const regionName = 'bangkok-metro';

  /// Returns the on-device directory name to pass to the routing channel
  /// (relative to the app's external files dir, matching Kotlin's
  /// `getExternalFilesDir(null)!!.resolve(tilesDirName)`).
  Future<String> ensureProvisioned() async {
    final regionDir = await _regionDirectory();

    // Written only after every entry below extracts successfully —
    // checking for this file (rather than just "does the directory
    // exist") means a previous run that got killed mid-extraction is
    // retried from scratch instead of being mistaken for done with
    // missing tiles.
    final marker = File('${regionDir.path}/.provisioned');
    if (await marker.exists()) return regionName;

    final zipBytes = await rootBundle.load('assets/routing_regions/$regionName.zip');
    final archive = ZipDecoder().decodeBytes(zipBytes.buffer.asUint8List());

    for (final entry in archive) {
      if (!entry.isFile) continue;
      final file = File('${regionDir.path}/${entry.name}');
      await file.parent.create(recursive: true);
      await file.writeAsBytes(entry.content as List<int>, flush: true);
    }

    await marker.writeAsString('done', flush: true);
    return regionName;
  }

  Future<Directory> _regionDirectory() async {
    final externalDir = await getExternalStorageDirectory();
    if (externalDir == null) {
      throw StateError('No external storage directory available (Android only for now).');
    }
    return Directory('${externalDir.path}/$regionName');
  }
}
