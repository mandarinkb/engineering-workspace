import '../entities/ride.dart';
import '../entities/track_point.dart';

abstract class RideRepository {
  Future<void> saveRide(Ride ride, List<TrackPoint> trackPoints);

  Future<List<Ride>> getRides();

  Future<Ride?> getRide(String id);

  Future<List<TrackPoint>> getTrackPoints(String rideId);
}
