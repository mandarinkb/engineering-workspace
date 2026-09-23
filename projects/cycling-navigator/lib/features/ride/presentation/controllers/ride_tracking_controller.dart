import 'dart:async';

import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../location/domain/entities/location_point.dart';
import '../../../location/presentation/controllers/location_controller.dart';
import '../../data/datasources/ride_database.dart';
import '../../data/repositories/drift_ride_repository.dart';
import '../../domain/entities/ride.dart';
import '../../domain/entities/track_point.dart';
import '../../domain/repositories/ride_repository.dart';
import '../../domain/services/gps_filter.dart';
import '../../domain/services/ride_stats_calculator.dart';

part 'ride_tracking_controller.g.dart';

@Riverpod(keepAlive: true)
RideDatabase rideDatabase(Ref ref) => RideDatabase();

@Riverpod(keepAlive: true)
RideRepository rideRepository(Ref ref) => DriftRideRepository(ref.watch(rideDatabaseProvider));

class RideTrackingState {
  const RideTrackingState({
    required this.isActive,
    required this.startTime,
    required this.points,
    required this.stats,
    required this.lastFinishedRide,
  });

  final bool isActive;
  final DateTime? startTime;
  final List<TrackPoint> points;
  final RideStats stats;
  final Ride? lastFinishedRide;

  static const initial = RideTrackingState(
    isActive: false,
    startTime: null,
    points: [],
    stats: RideStats.zero,
    lastFinishedRide: null,
  );
}

@riverpod
class RideTrackingController extends _$RideTrackingController {
  final _gpsFilter = const GpsFilter();
  final _statsCalculator = const RideStatsCalculator();

  StreamSubscription<LocationPoint>? _subscription;
  TrackPoint? _lastAcceptedPoint;

  @override
  RideTrackingState build() {
    ref.onDispose(() => _subscription?.cancel());
    return RideTrackingState.initial;
  }

  void startRide() {
    _lastAcceptedPoint = null;
    state = RideTrackingState(
      isActive: true,
      startTime: DateTime.now(),
      points: const [],
      stats: RideStats.zero,
      lastFinishedRide: null,
    );

    _subscription = ref.read(locationRepositoryProvider).watchLocation().listen(_onLocation);
  }

  void _onLocation(LocationPoint location) {
    final candidate = TrackPoint(
      timestamp: location.timestamp,
      latitude: location.latitude,
      longitude: location.longitude,
      altitude: location.altitude,
      speed: location.speed,
      accuracy: location.accuracy,
    );

    if (!_gpsFilter.isAcceptable(candidate, _lastAcceptedPoint)) return;
    _lastAcceptedPoint = candidate;

    final points = [...state.points, candidate];
    state = RideTrackingState(
      isActive: true,
      startTime: state.startTime,
      points: points,
      stats: _statsCalculator.calculate(points),
      lastFinishedRide: null,
    );
  }

  /// Persists the ride and returns it; the caller navigates to the summary
  /// screen, which reads it back from [state.lastFinishedRide].
  Future<Ride> stopRide() async {
    await _subscription?.cancel();
    _subscription = null;

    final points = state.points;
    final startTime = state.startTime ?? DateTime.now();
    final endTime = DateTime.now();
    final stats = _statsCalculator.calculate(points);

    final ride = Ride(
      id: DateTime.now().microsecondsSinceEpoch.toString(),
      startTime: startTime,
      endTime: endTime,
      duration: endTime.difference(startTime),
      movingTime: stats.movingTime,
      distanceMeters: stats.distanceMeters,
      averageSpeed: stats.averageSpeed,
      maxSpeed: stats.maxSpeed,
      elevationGain: stats.elevationGain,
      elevationLoss: stats.elevationLoss,
      startLatitude: points.isNotEmpty ? points.first.latitude : 0,
      startLongitude: points.isNotEmpty ? points.first.longitude : 0,
      endLatitude: points.isNotEmpty ? points.last.latitude : 0,
      endLongitude: points.isNotEmpty ? points.last.longitude : 0,
    );

    await ref.read(rideRepositoryProvider).saveRide(ride, points);

    state = RideTrackingState(
      isActive: false,
      startTime: null,
      points: const [],
      stats: RideStats.zero,
      lastFinishedRide: ride,
    );

    return ride;
  }
}
