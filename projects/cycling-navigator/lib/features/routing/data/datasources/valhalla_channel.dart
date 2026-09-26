import 'package:flutter/services.dart';

import '../../domain/entities/geo_point.dart';

/// Thin adapter over the Kotlin bridge (see android/.../MainActivity.kt) —
/// `Valhalla.route()`'s Kotlin API is typed, not raw JSON (the raw-JSON
/// `ValhallaActor` is `internal` to that library), so Kotlin builds the
/// typed request itself and hands back a small custom JSON object with
/// status plus a `routes` array (one entry per alternate — Phase 7).
/// Android only for now (Phase 6a didn't verify iOS — no Xcode on this
/// dev machine).
class ValhallaChannel {
  const ValhallaChannel();

  static const _channel = MethodChannel('valhalla_routing');

  Future<String> calculateRoute({
    required String tilesDirName,
    required GeoPoint start,
    required GeoPoint destination,
  }) async {
    final result = await _channel.invokeMethod<String>('calculateRoute', {
      'tilesDir': tilesDirName,
      'startLat': start.lat,
      'startLon': start.lon,
      'destLat': destination.lat,
      'destLon': destination.lon,
    });
    if (result == null) {
      throw StateError('valhalla_routing channel returned no result');
    }
    return result;
  }
}
