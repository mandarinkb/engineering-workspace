import 'package:drift/drift.dart';
import 'package:drift_flutter/drift_flutter.dart';

part 'ride_database.g.dart';

@DataClassName('RideRow')
class Rides extends Table {
  TextColumn get id => text()();
  DateTimeColumn get startTime => dateTime()();
  DateTimeColumn get endTime => dateTime()();
  IntColumn get durationMs => integer()();
  IntColumn get movingTimeMs => integer()();
  RealColumn get distanceMeters => real()();
  RealColumn get averageSpeed => real()();
  RealColumn get maxSpeed => real()();
  RealColumn get elevationGain => real()();
  RealColumn get elevationLoss => real()();
  RealColumn get startLatitude => real()();
  RealColumn get startLongitude => real()();
  RealColumn get endLatitude => real()();
  RealColumn get endLongitude => real()();

  @override
  Set<Column> get primaryKey => {id};
}

@DataClassName('TrackPointRow')
class TrackPointRows extends Table {
  IntColumn get rowId => integer().autoIncrement()();
  TextColumn get rideId => text().references(Rides, #id)();
  DateTimeColumn get timestamp => dateTime()();
  RealColumn get latitude => real()();
  RealColumn get longitude => real()();
  RealColumn get altitude => real()();
  RealColumn get speed => real()();
  RealColumn get accuracy => real()();
}

@DriftDatabase(tables: [Rides, TrackPointRows])
class RideDatabase extends _$RideDatabase {
  RideDatabase() : super(driftDatabase(name: 'cycling_navigator'));

  @override
  int get schemaVersion => 1;
}
