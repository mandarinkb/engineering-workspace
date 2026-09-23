import '../entities/geo_point.dart';

/// Decodes Google's encoded polyline algorithm format, which is what
/// Valhalla returns in `trip.legs[i].shape`. Valhalla's default precision
/// is 6 decimal places (1e6) — pass a different [precision] only if the
/// request explicitly asked for another `shape_format`.
List<GeoPoint> decodePolyline(String encoded, {double precision = 1e6}) {
  final points = <GeoPoint>[];
  var index = 0;
  var lat = 0;
  var lon = 0;

  while (index < encoded.length) {
    var shift = 0;
    var result = 0;
    int byte;
    do {
      byte = encoded.codeUnitAt(index++) - 63;
      result |= (byte & 0x1f) << shift;
      shift += 5;
    } while (byte >= 0x20);
    lat += (result & 1) != 0 ? ~(result >> 1) : (result >> 1);

    shift = 0;
    result = 0;
    do {
      byte = encoded.codeUnitAt(index++) - 63;
      result |= (byte & 0x1f) << shift;
      shift += 5;
    } while (byte >= 0x20);
    lon += (result & 1) != 0 ? ~(result >> 1) : (result >> 1);

    points.add(GeoPoint(lat: lat / precision, lon: lon / precision));
  }

  return points;
}
