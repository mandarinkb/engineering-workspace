import '../../domain/entities/ride.dart';
import '../../domain/entities/track_point.dart';
import '../../domain/repositories/ride_repository.dart';
import '../datasources/ride_database.dart';

class DriftRideRepository implements RideRepository {
  DriftRideRepository(this._db);

  final RideDatabase _db;

  @override
  Future<void> saveRide(Ride ride, List<TrackPoint> trackPoints) async {
    await _db.transaction(() async {
      await _db.managers.rides.create(
        (o) => o(
          id: ride.id,
          startTime: ride.startTime,
          endTime: ride.endTime,
          durationMs: ride.duration.inMilliseconds,
          movingTimeMs: ride.movingTime.inMilliseconds,
          distanceMeters: ride.distanceMeters,
          averageSpeed: ride.averageSpeed,
          maxSpeed: ride.maxSpeed,
          elevationGain: ride.elevationGain,
          elevationLoss: ride.elevationLoss,
          startLatitude: ride.startLatitude,
          startLongitude: ride.startLongitude,
          endLatitude: ride.endLatitude,
          endLongitude: ride.endLongitude,
        ),
      );

      for (final point in trackPoints) {
        await _db.managers.trackPointRows.create(
          (o) => o(
            rideId: ride.id,
            timestamp: point.timestamp,
            latitude: point.latitude,
            longitude: point.longitude,
            altitude: point.altitude,
            speed: point.speed,
            accuracy: point.accuracy,
          ),
        );
      }
    });
  }

  @override
  Future<List<Ride>> getRides() async {
    final rows = await _db.managers.rides.get();
    final rides = rows.map(_toRide).toList()
      ..sort((a, b) => b.startTime.compareTo(a.startTime));
    return rides;
  }

  @override
  Future<Ride?> getRide(String id) async {
    final row = await _db.managers.rides.filter((f) => f.id.equals(id)).getSingleOrNull();
    return row == null ? null : _toRide(row);
  }

  @override
  Future<List<TrackPoint>> getTrackPoints(String rideId) async {
    final rows =
        await _db.managers.trackPointRows.filter((f) => f.rideId.id.equals(rideId)).get();
    final points = rows.map(_toTrackPoint).toList()
      ..sort((a, b) => a.timestamp.compareTo(b.timestamp));
    return points;
  }

  Ride _toRide(RideRow row) => Ride(
        id: row.id,
        startTime: row.startTime,
        endTime: row.endTime,
        duration: Duration(milliseconds: row.durationMs),
        movingTime: Duration(milliseconds: row.movingTimeMs),
        distanceMeters: row.distanceMeters,
        averageSpeed: row.averageSpeed,
        maxSpeed: row.maxSpeed,
        elevationGain: row.elevationGain,
        elevationLoss: row.elevationLoss,
        startLatitude: row.startLatitude,
        startLongitude: row.startLongitude,
        endLatitude: row.endLatitude,
        endLongitude: row.endLongitude,
      );

  TrackPoint _toTrackPoint(TrackPointRow row) => TrackPoint(
        timestamp: row.timestamp,
        latitude: row.latitude,
        longitude: row.longitude,
        altitude: row.altitude,
        speed: row.speed,
        accuracy: row.accuracy,
      );
}
