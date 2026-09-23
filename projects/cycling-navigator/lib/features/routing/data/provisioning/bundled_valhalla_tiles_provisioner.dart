import 'dart:io';

import 'package:flutter/services.dart' show rootBundle;
import 'package:path_provider/path_provider.dart';

/// Copies the bundled Valhalla routing tiles (Phase 6a's test bbox — see
/// CYCLING-NAVIGATOR-TODO.md Phase 6a) onto the real filesystem so the
/// native `ValhallaActor` can read them by path. Not yet unified with the
/// Phase 4 `MapRepository` region system — this is its own small, fixed
/// region until that's worth generalizing.
class BundledValhallaTilesProvisioner {
  static const regionName = 'bangkok-tiny';

  static const _relativeFilePaths = [
    '0/002/320.gph',
    '1/037/360.gph',
    '2/000/597/281.gph',
    '2/000/597/282.gph',
    '2/000/598/721.gph',
    '2/000/598/722.gph',
  ];

  /// Returns the on-device directory name to pass to the routing channel
  /// (relative to the app's external files dir, matching Kotlin's
  /// `getExternalFilesDir(null)!!.resolve(tilesDirName)`).
  Future<String> ensureProvisioned() async {
    final baseDir = await _regionDirectory();

    for (final relativePath in _relativeFilePaths) {
      final file = File('${baseDir.path}/$relativePath');
      if (await file.exists()) continue;

      await file.parent.create(recursive: true);
      final bytes = await rootBundle.load('assets/routing_regions/$regionName/$relativePath');
      await file.writeAsBytes(bytes.buffer.asUint8List(), flush: true);
    }

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
