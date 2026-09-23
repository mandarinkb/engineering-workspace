// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'ride_database.dart';

// ignore_for_file: type=lint
class $RidesTable extends Rides with TableInfo<$RidesTable, RideRow> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $RidesTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _idMeta = const VerificationMeta('id');
  @override
  late final GeneratedColumn<String> id = GeneratedColumn<String>(
    'id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _startTimeMeta = const VerificationMeta(
    'startTime',
  );
  @override
  late final GeneratedColumn<DateTime> startTime = GeneratedColumn<DateTime>(
    'start_time',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _endTimeMeta = const VerificationMeta(
    'endTime',
  );
  @override
  late final GeneratedColumn<DateTime> endTime = GeneratedColumn<DateTime>(
    'end_time',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _durationMsMeta = const VerificationMeta(
    'durationMs',
  );
  @override
  late final GeneratedColumn<int> durationMs = GeneratedColumn<int>(
    'duration_ms',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _movingTimeMsMeta = const VerificationMeta(
    'movingTimeMs',
  );
  @override
  late final GeneratedColumn<int> movingTimeMs = GeneratedColumn<int>(
    'moving_time_ms',
    aliasedName,
    false,
    type: DriftSqlType.int,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _distanceMetersMeta = const VerificationMeta(
    'distanceMeters',
  );
  @override
  late final GeneratedColumn<double> distanceMeters = GeneratedColumn<double>(
    'distance_meters',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _averageSpeedMeta = const VerificationMeta(
    'averageSpeed',
  );
  @override
  late final GeneratedColumn<double> averageSpeed = GeneratedColumn<double>(
    'average_speed',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _maxSpeedMeta = const VerificationMeta(
    'maxSpeed',
  );
  @override
  late final GeneratedColumn<double> maxSpeed = GeneratedColumn<double>(
    'max_speed',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _elevationGainMeta = const VerificationMeta(
    'elevationGain',
  );
  @override
  late final GeneratedColumn<double> elevationGain = GeneratedColumn<double>(
    'elevation_gain',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _elevationLossMeta = const VerificationMeta(
    'elevationLoss',
  );
  @override
  late final GeneratedColumn<double> elevationLoss = GeneratedColumn<double>(
    'elevation_loss',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _startLatitudeMeta = const VerificationMeta(
    'startLatitude',
  );
  @override
  late final GeneratedColumn<double> startLatitude = GeneratedColumn<double>(
    'start_latitude',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _startLongitudeMeta = const VerificationMeta(
    'startLongitude',
  );
  @override
  late final GeneratedColumn<double> startLongitude = GeneratedColumn<double>(
    'start_longitude',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _endLatitudeMeta = const VerificationMeta(
    'endLatitude',
  );
  @override
  late final GeneratedColumn<double> endLatitude = GeneratedColumn<double>(
    'end_latitude',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _endLongitudeMeta = const VerificationMeta(
    'endLongitude',
  );
  @override
  late final GeneratedColumn<double> endLongitude = GeneratedColumn<double>(
    'end_longitude',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    id,
    startTime,
    endTime,
    durationMs,
    movingTimeMs,
    distanceMeters,
    averageSpeed,
    maxSpeed,
    elevationGain,
    elevationLoss,
    startLatitude,
    startLongitude,
    endLatitude,
    endLongitude,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'rides';
  @override
  VerificationContext validateIntegrity(
    Insertable<RideRow> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('id')) {
      context.handle(_idMeta, id.isAcceptableOrUnknown(data['id']!, _idMeta));
    } else if (isInserting) {
      context.missing(_idMeta);
    }
    if (data.containsKey('start_time')) {
      context.handle(
        _startTimeMeta,
        startTime.isAcceptableOrUnknown(data['start_time']!, _startTimeMeta),
      );
    } else if (isInserting) {
      context.missing(_startTimeMeta);
    }
    if (data.containsKey('end_time')) {
      context.handle(
        _endTimeMeta,
        endTime.isAcceptableOrUnknown(data['end_time']!, _endTimeMeta),
      );
    } else if (isInserting) {
      context.missing(_endTimeMeta);
    }
    if (data.containsKey('duration_ms')) {
      context.handle(
        _durationMsMeta,
        durationMs.isAcceptableOrUnknown(data['duration_ms']!, _durationMsMeta),
      );
    } else if (isInserting) {
      context.missing(_durationMsMeta);
    }
    if (data.containsKey('moving_time_ms')) {
      context.handle(
        _movingTimeMsMeta,
        movingTimeMs.isAcceptableOrUnknown(
          data['moving_time_ms']!,
          _movingTimeMsMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_movingTimeMsMeta);
    }
    if (data.containsKey('distance_meters')) {
      context.handle(
        _distanceMetersMeta,
        distanceMeters.isAcceptableOrUnknown(
          data['distance_meters']!,
          _distanceMetersMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_distanceMetersMeta);
    }
    if (data.containsKey('average_speed')) {
      context.handle(
        _averageSpeedMeta,
        averageSpeed.isAcceptableOrUnknown(
          data['average_speed']!,
          _averageSpeedMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_averageSpeedMeta);
    }
    if (data.containsKey('max_speed')) {
      context.handle(
        _maxSpeedMeta,
        maxSpeed.isAcceptableOrUnknown(data['max_speed']!, _maxSpeedMeta),
      );
    } else if (isInserting) {
      context.missing(_maxSpeedMeta);
    }
    if (data.containsKey('elevation_gain')) {
      context.handle(
        _elevationGainMeta,
        elevationGain.isAcceptableOrUnknown(
          data['elevation_gain']!,
          _elevationGainMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_elevationGainMeta);
    }
    if (data.containsKey('elevation_loss')) {
      context.handle(
        _elevationLossMeta,
        elevationLoss.isAcceptableOrUnknown(
          data['elevation_loss']!,
          _elevationLossMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_elevationLossMeta);
    }
    if (data.containsKey('start_latitude')) {
      context.handle(
        _startLatitudeMeta,
        startLatitude.isAcceptableOrUnknown(
          data['start_latitude']!,
          _startLatitudeMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_startLatitudeMeta);
    }
    if (data.containsKey('start_longitude')) {
      context.handle(
        _startLongitudeMeta,
        startLongitude.isAcceptableOrUnknown(
          data['start_longitude']!,
          _startLongitudeMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_startLongitudeMeta);
    }
    if (data.containsKey('end_latitude')) {
      context.handle(
        _endLatitudeMeta,
        endLatitude.isAcceptableOrUnknown(
          data['end_latitude']!,
          _endLatitudeMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_endLatitudeMeta);
    }
    if (data.containsKey('end_longitude')) {
      context.handle(
        _endLongitudeMeta,
        endLongitude.isAcceptableOrUnknown(
          data['end_longitude']!,
          _endLongitudeMeta,
        ),
      );
    } else if (isInserting) {
      context.missing(_endLongitudeMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {id};
  @override
  RideRow map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return RideRow(
      id: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}id'],
      )!,
      startTime: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}start_time'],
      )!,
      endTime: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}end_time'],
      )!,
      durationMs: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}duration_ms'],
      )!,
      movingTimeMs: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}moving_time_ms'],
      )!,
      distanceMeters: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}distance_meters'],
      )!,
      averageSpeed: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}average_speed'],
      )!,
      maxSpeed: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}max_speed'],
      )!,
      elevationGain: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}elevation_gain'],
      )!,
      elevationLoss: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}elevation_loss'],
      )!,
      startLatitude: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}start_latitude'],
      )!,
      startLongitude: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}start_longitude'],
      )!,
      endLatitude: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}end_latitude'],
      )!,
      endLongitude: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}end_longitude'],
      )!,
    );
  }

  @override
  $RidesTable createAlias(String alias) {
    return $RidesTable(attachedDatabase, alias);
  }
}

class RideRow extends DataClass implements Insertable<RideRow> {
  final String id;
  final DateTime startTime;
  final DateTime endTime;
  final int durationMs;
  final int movingTimeMs;
  final double distanceMeters;
  final double averageSpeed;
  final double maxSpeed;
  final double elevationGain;
  final double elevationLoss;
  final double startLatitude;
  final double startLongitude;
  final double endLatitude;
  final double endLongitude;
  const RideRow({
    required this.id,
    required this.startTime,
    required this.endTime,
    required this.durationMs,
    required this.movingTimeMs,
    required this.distanceMeters,
    required this.averageSpeed,
    required this.maxSpeed,
    required this.elevationGain,
    required this.elevationLoss,
    required this.startLatitude,
    required this.startLongitude,
    required this.endLatitude,
    required this.endLongitude,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['id'] = Variable<String>(id);
    map['start_time'] = Variable<DateTime>(startTime);
    map['end_time'] = Variable<DateTime>(endTime);
    map['duration_ms'] = Variable<int>(durationMs);
    map['moving_time_ms'] = Variable<int>(movingTimeMs);
    map['distance_meters'] = Variable<double>(distanceMeters);
    map['average_speed'] = Variable<double>(averageSpeed);
    map['max_speed'] = Variable<double>(maxSpeed);
    map['elevation_gain'] = Variable<double>(elevationGain);
    map['elevation_loss'] = Variable<double>(elevationLoss);
    map['start_latitude'] = Variable<double>(startLatitude);
    map['start_longitude'] = Variable<double>(startLongitude);
    map['end_latitude'] = Variable<double>(endLatitude);
    map['end_longitude'] = Variable<double>(endLongitude);
    return map;
  }

  RidesCompanion toCompanion(bool nullToAbsent) {
    return RidesCompanion(
      id: Value(id),
      startTime: Value(startTime),
      endTime: Value(endTime),
      durationMs: Value(durationMs),
      movingTimeMs: Value(movingTimeMs),
      distanceMeters: Value(distanceMeters),
      averageSpeed: Value(averageSpeed),
      maxSpeed: Value(maxSpeed),
      elevationGain: Value(elevationGain),
      elevationLoss: Value(elevationLoss),
      startLatitude: Value(startLatitude),
      startLongitude: Value(startLongitude),
      endLatitude: Value(endLatitude),
      endLongitude: Value(endLongitude),
    );
  }

  factory RideRow.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return RideRow(
      id: serializer.fromJson<String>(json['id']),
      startTime: serializer.fromJson<DateTime>(json['startTime']),
      endTime: serializer.fromJson<DateTime>(json['endTime']),
      durationMs: serializer.fromJson<int>(json['durationMs']),
      movingTimeMs: serializer.fromJson<int>(json['movingTimeMs']),
      distanceMeters: serializer.fromJson<double>(json['distanceMeters']),
      averageSpeed: serializer.fromJson<double>(json['averageSpeed']),
      maxSpeed: serializer.fromJson<double>(json['maxSpeed']),
      elevationGain: serializer.fromJson<double>(json['elevationGain']),
      elevationLoss: serializer.fromJson<double>(json['elevationLoss']),
      startLatitude: serializer.fromJson<double>(json['startLatitude']),
      startLongitude: serializer.fromJson<double>(json['startLongitude']),
      endLatitude: serializer.fromJson<double>(json['endLatitude']),
      endLongitude: serializer.fromJson<double>(json['endLongitude']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'id': serializer.toJson<String>(id),
      'startTime': serializer.toJson<DateTime>(startTime),
      'endTime': serializer.toJson<DateTime>(endTime),
      'durationMs': serializer.toJson<int>(durationMs),
      'movingTimeMs': serializer.toJson<int>(movingTimeMs),
      'distanceMeters': serializer.toJson<double>(distanceMeters),
      'averageSpeed': serializer.toJson<double>(averageSpeed),
      'maxSpeed': serializer.toJson<double>(maxSpeed),
      'elevationGain': serializer.toJson<double>(elevationGain),
      'elevationLoss': serializer.toJson<double>(elevationLoss),
      'startLatitude': serializer.toJson<double>(startLatitude),
      'startLongitude': serializer.toJson<double>(startLongitude),
      'endLatitude': serializer.toJson<double>(endLatitude),
      'endLongitude': serializer.toJson<double>(endLongitude),
    };
  }

  RideRow copyWith({
    String? id,
    DateTime? startTime,
    DateTime? endTime,
    int? durationMs,
    int? movingTimeMs,
    double? distanceMeters,
    double? averageSpeed,
    double? maxSpeed,
    double? elevationGain,
    double? elevationLoss,
    double? startLatitude,
    double? startLongitude,
    double? endLatitude,
    double? endLongitude,
  }) => RideRow(
    id: id ?? this.id,
    startTime: startTime ?? this.startTime,
    endTime: endTime ?? this.endTime,
    durationMs: durationMs ?? this.durationMs,
    movingTimeMs: movingTimeMs ?? this.movingTimeMs,
    distanceMeters: distanceMeters ?? this.distanceMeters,
    averageSpeed: averageSpeed ?? this.averageSpeed,
    maxSpeed: maxSpeed ?? this.maxSpeed,
    elevationGain: elevationGain ?? this.elevationGain,
    elevationLoss: elevationLoss ?? this.elevationLoss,
    startLatitude: startLatitude ?? this.startLatitude,
    startLongitude: startLongitude ?? this.startLongitude,
    endLatitude: endLatitude ?? this.endLatitude,
    endLongitude: endLongitude ?? this.endLongitude,
  );
  RideRow copyWithCompanion(RidesCompanion data) {
    return RideRow(
      id: data.id.present ? data.id.value : this.id,
      startTime: data.startTime.present ? data.startTime.value : this.startTime,
      endTime: data.endTime.present ? data.endTime.value : this.endTime,
      durationMs: data.durationMs.present
          ? data.durationMs.value
          : this.durationMs,
      movingTimeMs: data.movingTimeMs.present
          ? data.movingTimeMs.value
          : this.movingTimeMs,
      distanceMeters: data.distanceMeters.present
          ? data.distanceMeters.value
          : this.distanceMeters,
      averageSpeed: data.averageSpeed.present
          ? data.averageSpeed.value
          : this.averageSpeed,
      maxSpeed: data.maxSpeed.present ? data.maxSpeed.value : this.maxSpeed,
      elevationGain: data.elevationGain.present
          ? data.elevationGain.value
          : this.elevationGain,
      elevationLoss: data.elevationLoss.present
          ? data.elevationLoss.value
          : this.elevationLoss,
      startLatitude: data.startLatitude.present
          ? data.startLatitude.value
          : this.startLatitude,
      startLongitude: data.startLongitude.present
          ? data.startLongitude.value
          : this.startLongitude,
      endLatitude: data.endLatitude.present
          ? data.endLatitude.value
          : this.endLatitude,
      endLongitude: data.endLongitude.present
          ? data.endLongitude.value
          : this.endLongitude,
    );
  }

  @override
  String toString() {
    return (StringBuffer('RideRow(')
          ..write('id: $id, ')
          ..write('startTime: $startTime, ')
          ..write('endTime: $endTime, ')
          ..write('durationMs: $durationMs, ')
          ..write('movingTimeMs: $movingTimeMs, ')
          ..write('distanceMeters: $distanceMeters, ')
          ..write('averageSpeed: $averageSpeed, ')
          ..write('maxSpeed: $maxSpeed, ')
          ..write('elevationGain: $elevationGain, ')
          ..write('elevationLoss: $elevationLoss, ')
          ..write('startLatitude: $startLatitude, ')
          ..write('startLongitude: $startLongitude, ')
          ..write('endLatitude: $endLatitude, ')
          ..write('endLongitude: $endLongitude')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    id,
    startTime,
    endTime,
    durationMs,
    movingTimeMs,
    distanceMeters,
    averageSpeed,
    maxSpeed,
    elevationGain,
    elevationLoss,
    startLatitude,
    startLongitude,
    endLatitude,
    endLongitude,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is RideRow &&
          other.id == this.id &&
          other.startTime == this.startTime &&
          other.endTime == this.endTime &&
          other.durationMs == this.durationMs &&
          other.movingTimeMs == this.movingTimeMs &&
          other.distanceMeters == this.distanceMeters &&
          other.averageSpeed == this.averageSpeed &&
          other.maxSpeed == this.maxSpeed &&
          other.elevationGain == this.elevationGain &&
          other.elevationLoss == this.elevationLoss &&
          other.startLatitude == this.startLatitude &&
          other.startLongitude == this.startLongitude &&
          other.endLatitude == this.endLatitude &&
          other.endLongitude == this.endLongitude);
}

class RidesCompanion extends UpdateCompanion<RideRow> {
  final Value<String> id;
  final Value<DateTime> startTime;
  final Value<DateTime> endTime;
  final Value<int> durationMs;
  final Value<int> movingTimeMs;
  final Value<double> distanceMeters;
  final Value<double> averageSpeed;
  final Value<double> maxSpeed;
  final Value<double> elevationGain;
  final Value<double> elevationLoss;
  final Value<double> startLatitude;
  final Value<double> startLongitude;
  final Value<double> endLatitude;
  final Value<double> endLongitude;
  final Value<int> rowid;
  const RidesCompanion({
    this.id = const Value.absent(),
    this.startTime = const Value.absent(),
    this.endTime = const Value.absent(),
    this.durationMs = const Value.absent(),
    this.movingTimeMs = const Value.absent(),
    this.distanceMeters = const Value.absent(),
    this.averageSpeed = const Value.absent(),
    this.maxSpeed = const Value.absent(),
    this.elevationGain = const Value.absent(),
    this.elevationLoss = const Value.absent(),
    this.startLatitude = const Value.absent(),
    this.startLongitude = const Value.absent(),
    this.endLatitude = const Value.absent(),
    this.endLongitude = const Value.absent(),
    this.rowid = const Value.absent(),
  });
  RidesCompanion.insert({
    required String id,
    required DateTime startTime,
    required DateTime endTime,
    required int durationMs,
    required int movingTimeMs,
    required double distanceMeters,
    required double averageSpeed,
    required double maxSpeed,
    required double elevationGain,
    required double elevationLoss,
    required double startLatitude,
    required double startLongitude,
    required double endLatitude,
    required double endLongitude,
    this.rowid = const Value.absent(),
  }) : id = Value(id),
       startTime = Value(startTime),
       endTime = Value(endTime),
       durationMs = Value(durationMs),
       movingTimeMs = Value(movingTimeMs),
       distanceMeters = Value(distanceMeters),
       averageSpeed = Value(averageSpeed),
       maxSpeed = Value(maxSpeed),
       elevationGain = Value(elevationGain),
       elevationLoss = Value(elevationLoss),
       startLatitude = Value(startLatitude),
       startLongitude = Value(startLongitude),
       endLatitude = Value(endLatitude),
       endLongitude = Value(endLongitude);
  static Insertable<RideRow> custom({
    Expression<String>? id,
    Expression<DateTime>? startTime,
    Expression<DateTime>? endTime,
    Expression<int>? durationMs,
    Expression<int>? movingTimeMs,
    Expression<double>? distanceMeters,
    Expression<double>? averageSpeed,
    Expression<double>? maxSpeed,
    Expression<double>? elevationGain,
    Expression<double>? elevationLoss,
    Expression<double>? startLatitude,
    Expression<double>? startLongitude,
    Expression<double>? endLatitude,
    Expression<double>? endLongitude,
    Expression<int>? rowid,
  }) {
    return RawValuesInsertable({
      if (id != null) 'id': id,
      if (startTime != null) 'start_time': startTime,
      if (endTime != null) 'end_time': endTime,
      if (durationMs != null) 'duration_ms': durationMs,
      if (movingTimeMs != null) 'moving_time_ms': movingTimeMs,
      if (distanceMeters != null) 'distance_meters': distanceMeters,
      if (averageSpeed != null) 'average_speed': averageSpeed,
      if (maxSpeed != null) 'max_speed': maxSpeed,
      if (elevationGain != null) 'elevation_gain': elevationGain,
      if (elevationLoss != null) 'elevation_loss': elevationLoss,
      if (startLatitude != null) 'start_latitude': startLatitude,
      if (startLongitude != null) 'start_longitude': startLongitude,
      if (endLatitude != null) 'end_latitude': endLatitude,
      if (endLongitude != null) 'end_longitude': endLongitude,
      if (rowid != null) 'rowid': rowid,
    });
  }

  RidesCompanion copyWith({
    Value<String>? id,
    Value<DateTime>? startTime,
    Value<DateTime>? endTime,
    Value<int>? durationMs,
    Value<int>? movingTimeMs,
    Value<double>? distanceMeters,
    Value<double>? averageSpeed,
    Value<double>? maxSpeed,
    Value<double>? elevationGain,
    Value<double>? elevationLoss,
    Value<double>? startLatitude,
    Value<double>? startLongitude,
    Value<double>? endLatitude,
    Value<double>? endLongitude,
    Value<int>? rowid,
  }) {
    return RidesCompanion(
      id: id ?? this.id,
      startTime: startTime ?? this.startTime,
      endTime: endTime ?? this.endTime,
      durationMs: durationMs ?? this.durationMs,
      movingTimeMs: movingTimeMs ?? this.movingTimeMs,
      distanceMeters: distanceMeters ?? this.distanceMeters,
      averageSpeed: averageSpeed ?? this.averageSpeed,
      maxSpeed: maxSpeed ?? this.maxSpeed,
      elevationGain: elevationGain ?? this.elevationGain,
      elevationLoss: elevationLoss ?? this.elevationLoss,
      startLatitude: startLatitude ?? this.startLatitude,
      startLongitude: startLongitude ?? this.startLongitude,
      endLatitude: endLatitude ?? this.endLatitude,
      endLongitude: endLongitude ?? this.endLongitude,
      rowid: rowid ?? this.rowid,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (id.present) {
      map['id'] = Variable<String>(id.value);
    }
    if (startTime.present) {
      map['start_time'] = Variable<DateTime>(startTime.value);
    }
    if (endTime.present) {
      map['end_time'] = Variable<DateTime>(endTime.value);
    }
    if (durationMs.present) {
      map['duration_ms'] = Variable<int>(durationMs.value);
    }
    if (movingTimeMs.present) {
      map['moving_time_ms'] = Variable<int>(movingTimeMs.value);
    }
    if (distanceMeters.present) {
      map['distance_meters'] = Variable<double>(distanceMeters.value);
    }
    if (averageSpeed.present) {
      map['average_speed'] = Variable<double>(averageSpeed.value);
    }
    if (maxSpeed.present) {
      map['max_speed'] = Variable<double>(maxSpeed.value);
    }
    if (elevationGain.present) {
      map['elevation_gain'] = Variable<double>(elevationGain.value);
    }
    if (elevationLoss.present) {
      map['elevation_loss'] = Variable<double>(elevationLoss.value);
    }
    if (startLatitude.present) {
      map['start_latitude'] = Variable<double>(startLatitude.value);
    }
    if (startLongitude.present) {
      map['start_longitude'] = Variable<double>(startLongitude.value);
    }
    if (endLatitude.present) {
      map['end_latitude'] = Variable<double>(endLatitude.value);
    }
    if (endLongitude.present) {
      map['end_longitude'] = Variable<double>(endLongitude.value);
    }
    if (rowid.present) {
      map['rowid'] = Variable<int>(rowid.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('RidesCompanion(')
          ..write('id: $id, ')
          ..write('startTime: $startTime, ')
          ..write('endTime: $endTime, ')
          ..write('durationMs: $durationMs, ')
          ..write('movingTimeMs: $movingTimeMs, ')
          ..write('distanceMeters: $distanceMeters, ')
          ..write('averageSpeed: $averageSpeed, ')
          ..write('maxSpeed: $maxSpeed, ')
          ..write('elevationGain: $elevationGain, ')
          ..write('elevationLoss: $elevationLoss, ')
          ..write('startLatitude: $startLatitude, ')
          ..write('startLongitude: $startLongitude, ')
          ..write('endLatitude: $endLatitude, ')
          ..write('endLongitude: $endLongitude, ')
          ..write('rowid: $rowid')
          ..write(')'))
        .toString();
  }
}

class $TrackPointRowsTable extends TrackPointRows
    with TableInfo<$TrackPointRowsTable, TrackPointRow> {
  @override
  final GeneratedDatabase attachedDatabase;
  final String? _alias;
  $TrackPointRowsTable(this.attachedDatabase, [this._alias]);
  static const VerificationMeta _rowIdMeta = const VerificationMeta('rowId');
  @override
  late final GeneratedColumn<int> rowId = GeneratedColumn<int>(
    'row_id',
    aliasedName,
    false,
    hasAutoIncrement: true,
    type: DriftSqlType.int,
    requiredDuringInsert: false,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'PRIMARY KEY AUTOINCREMENT',
    ),
  );
  static const VerificationMeta _rideIdMeta = const VerificationMeta('rideId');
  @override
  late final GeneratedColumn<String> rideId = GeneratedColumn<String>(
    'ride_id',
    aliasedName,
    false,
    type: DriftSqlType.string,
    requiredDuringInsert: true,
    defaultConstraints: GeneratedColumn.constraintIsAlways(
      'REFERENCES rides (id)',
    ),
  );
  static const VerificationMeta _timestampMeta = const VerificationMeta(
    'timestamp',
  );
  @override
  late final GeneratedColumn<DateTime> timestamp = GeneratedColumn<DateTime>(
    'timestamp',
    aliasedName,
    false,
    type: DriftSqlType.dateTime,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _latitudeMeta = const VerificationMeta(
    'latitude',
  );
  @override
  late final GeneratedColumn<double> latitude = GeneratedColumn<double>(
    'latitude',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _longitudeMeta = const VerificationMeta(
    'longitude',
  );
  @override
  late final GeneratedColumn<double> longitude = GeneratedColumn<double>(
    'longitude',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _altitudeMeta = const VerificationMeta(
    'altitude',
  );
  @override
  late final GeneratedColumn<double> altitude = GeneratedColumn<double>(
    'altitude',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _speedMeta = const VerificationMeta('speed');
  @override
  late final GeneratedColumn<double> speed = GeneratedColumn<double>(
    'speed',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  static const VerificationMeta _accuracyMeta = const VerificationMeta(
    'accuracy',
  );
  @override
  late final GeneratedColumn<double> accuracy = GeneratedColumn<double>(
    'accuracy',
    aliasedName,
    false,
    type: DriftSqlType.double,
    requiredDuringInsert: true,
  );
  @override
  List<GeneratedColumn> get $columns => [
    rowId,
    rideId,
    timestamp,
    latitude,
    longitude,
    altitude,
    speed,
    accuracy,
  ];
  @override
  String get aliasedName => _alias ?? actualTableName;
  @override
  String get actualTableName => $name;
  static const String $name = 'track_point_rows';
  @override
  VerificationContext validateIntegrity(
    Insertable<TrackPointRow> instance, {
    bool isInserting = false,
  }) {
    final context = VerificationContext();
    final data = instance.toColumns(true);
    if (data.containsKey('row_id')) {
      context.handle(
        _rowIdMeta,
        rowId.isAcceptableOrUnknown(data['row_id']!, _rowIdMeta),
      );
    }
    if (data.containsKey('ride_id')) {
      context.handle(
        _rideIdMeta,
        rideId.isAcceptableOrUnknown(data['ride_id']!, _rideIdMeta),
      );
    } else if (isInserting) {
      context.missing(_rideIdMeta);
    }
    if (data.containsKey('timestamp')) {
      context.handle(
        _timestampMeta,
        timestamp.isAcceptableOrUnknown(data['timestamp']!, _timestampMeta),
      );
    } else if (isInserting) {
      context.missing(_timestampMeta);
    }
    if (data.containsKey('latitude')) {
      context.handle(
        _latitudeMeta,
        latitude.isAcceptableOrUnknown(data['latitude']!, _latitudeMeta),
      );
    } else if (isInserting) {
      context.missing(_latitudeMeta);
    }
    if (data.containsKey('longitude')) {
      context.handle(
        _longitudeMeta,
        longitude.isAcceptableOrUnknown(data['longitude']!, _longitudeMeta),
      );
    } else if (isInserting) {
      context.missing(_longitudeMeta);
    }
    if (data.containsKey('altitude')) {
      context.handle(
        _altitudeMeta,
        altitude.isAcceptableOrUnknown(data['altitude']!, _altitudeMeta),
      );
    } else if (isInserting) {
      context.missing(_altitudeMeta);
    }
    if (data.containsKey('speed')) {
      context.handle(
        _speedMeta,
        speed.isAcceptableOrUnknown(data['speed']!, _speedMeta),
      );
    } else if (isInserting) {
      context.missing(_speedMeta);
    }
    if (data.containsKey('accuracy')) {
      context.handle(
        _accuracyMeta,
        accuracy.isAcceptableOrUnknown(data['accuracy']!, _accuracyMeta),
      );
    } else if (isInserting) {
      context.missing(_accuracyMeta);
    }
    return context;
  }

  @override
  Set<GeneratedColumn> get $primaryKey => {rowId};
  @override
  TrackPointRow map(Map<String, dynamic> data, {String? tablePrefix}) {
    final effectivePrefix = tablePrefix != null ? '$tablePrefix.' : '';
    return TrackPointRow(
      rowId: attachedDatabase.typeMapping.read(
        DriftSqlType.int,
        data['${effectivePrefix}row_id'],
      )!,
      rideId: attachedDatabase.typeMapping.read(
        DriftSqlType.string,
        data['${effectivePrefix}ride_id'],
      )!,
      timestamp: attachedDatabase.typeMapping.read(
        DriftSqlType.dateTime,
        data['${effectivePrefix}timestamp'],
      )!,
      latitude: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}latitude'],
      )!,
      longitude: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}longitude'],
      )!,
      altitude: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}altitude'],
      )!,
      speed: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}speed'],
      )!,
      accuracy: attachedDatabase.typeMapping.read(
        DriftSqlType.double,
        data['${effectivePrefix}accuracy'],
      )!,
    );
  }

  @override
  $TrackPointRowsTable createAlias(String alias) {
    return $TrackPointRowsTable(attachedDatabase, alias);
  }
}

class TrackPointRow extends DataClass implements Insertable<TrackPointRow> {
  final int rowId;
  final String rideId;
  final DateTime timestamp;
  final double latitude;
  final double longitude;
  final double altitude;
  final double speed;
  final double accuracy;
  const TrackPointRow({
    required this.rowId,
    required this.rideId,
    required this.timestamp,
    required this.latitude,
    required this.longitude,
    required this.altitude,
    required this.speed,
    required this.accuracy,
  });
  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    map['row_id'] = Variable<int>(rowId);
    map['ride_id'] = Variable<String>(rideId);
    map['timestamp'] = Variable<DateTime>(timestamp);
    map['latitude'] = Variable<double>(latitude);
    map['longitude'] = Variable<double>(longitude);
    map['altitude'] = Variable<double>(altitude);
    map['speed'] = Variable<double>(speed);
    map['accuracy'] = Variable<double>(accuracy);
    return map;
  }

  TrackPointRowsCompanion toCompanion(bool nullToAbsent) {
    return TrackPointRowsCompanion(
      rowId: Value(rowId),
      rideId: Value(rideId),
      timestamp: Value(timestamp),
      latitude: Value(latitude),
      longitude: Value(longitude),
      altitude: Value(altitude),
      speed: Value(speed),
      accuracy: Value(accuracy),
    );
  }

  factory TrackPointRow.fromJson(
    Map<String, dynamic> json, {
    ValueSerializer? serializer,
  }) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return TrackPointRow(
      rowId: serializer.fromJson<int>(json['rowId']),
      rideId: serializer.fromJson<String>(json['rideId']),
      timestamp: serializer.fromJson<DateTime>(json['timestamp']),
      latitude: serializer.fromJson<double>(json['latitude']),
      longitude: serializer.fromJson<double>(json['longitude']),
      altitude: serializer.fromJson<double>(json['altitude']),
      speed: serializer.fromJson<double>(json['speed']),
      accuracy: serializer.fromJson<double>(json['accuracy']),
    );
  }
  @override
  Map<String, dynamic> toJson({ValueSerializer? serializer}) {
    serializer ??= driftRuntimeOptions.defaultSerializer;
    return <String, dynamic>{
      'rowId': serializer.toJson<int>(rowId),
      'rideId': serializer.toJson<String>(rideId),
      'timestamp': serializer.toJson<DateTime>(timestamp),
      'latitude': serializer.toJson<double>(latitude),
      'longitude': serializer.toJson<double>(longitude),
      'altitude': serializer.toJson<double>(altitude),
      'speed': serializer.toJson<double>(speed),
      'accuracy': serializer.toJson<double>(accuracy),
    };
  }

  TrackPointRow copyWith({
    int? rowId,
    String? rideId,
    DateTime? timestamp,
    double? latitude,
    double? longitude,
    double? altitude,
    double? speed,
    double? accuracy,
  }) => TrackPointRow(
    rowId: rowId ?? this.rowId,
    rideId: rideId ?? this.rideId,
    timestamp: timestamp ?? this.timestamp,
    latitude: latitude ?? this.latitude,
    longitude: longitude ?? this.longitude,
    altitude: altitude ?? this.altitude,
    speed: speed ?? this.speed,
    accuracy: accuracy ?? this.accuracy,
  );
  TrackPointRow copyWithCompanion(TrackPointRowsCompanion data) {
    return TrackPointRow(
      rowId: data.rowId.present ? data.rowId.value : this.rowId,
      rideId: data.rideId.present ? data.rideId.value : this.rideId,
      timestamp: data.timestamp.present ? data.timestamp.value : this.timestamp,
      latitude: data.latitude.present ? data.latitude.value : this.latitude,
      longitude: data.longitude.present ? data.longitude.value : this.longitude,
      altitude: data.altitude.present ? data.altitude.value : this.altitude,
      speed: data.speed.present ? data.speed.value : this.speed,
      accuracy: data.accuracy.present ? data.accuracy.value : this.accuracy,
    );
  }

  @override
  String toString() {
    return (StringBuffer('TrackPointRow(')
          ..write('rowId: $rowId, ')
          ..write('rideId: $rideId, ')
          ..write('timestamp: $timestamp, ')
          ..write('latitude: $latitude, ')
          ..write('longitude: $longitude, ')
          ..write('altitude: $altitude, ')
          ..write('speed: $speed, ')
          ..write('accuracy: $accuracy')
          ..write(')'))
        .toString();
  }

  @override
  int get hashCode => Object.hash(
    rowId,
    rideId,
    timestamp,
    latitude,
    longitude,
    altitude,
    speed,
    accuracy,
  );
  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      (other is TrackPointRow &&
          other.rowId == this.rowId &&
          other.rideId == this.rideId &&
          other.timestamp == this.timestamp &&
          other.latitude == this.latitude &&
          other.longitude == this.longitude &&
          other.altitude == this.altitude &&
          other.speed == this.speed &&
          other.accuracy == this.accuracy);
}

class TrackPointRowsCompanion extends UpdateCompanion<TrackPointRow> {
  final Value<int> rowId;
  final Value<String> rideId;
  final Value<DateTime> timestamp;
  final Value<double> latitude;
  final Value<double> longitude;
  final Value<double> altitude;
  final Value<double> speed;
  final Value<double> accuracy;
  const TrackPointRowsCompanion({
    this.rowId = const Value.absent(),
    this.rideId = const Value.absent(),
    this.timestamp = const Value.absent(),
    this.latitude = const Value.absent(),
    this.longitude = const Value.absent(),
    this.altitude = const Value.absent(),
    this.speed = const Value.absent(),
    this.accuracy = const Value.absent(),
  });
  TrackPointRowsCompanion.insert({
    this.rowId = const Value.absent(),
    required String rideId,
    required DateTime timestamp,
    required double latitude,
    required double longitude,
    required double altitude,
    required double speed,
    required double accuracy,
  }) : rideId = Value(rideId),
       timestamp = Value(timestamp),
       latitude = Value(latitude),
       longitude = Value(longitude),
       altitude = Value(altitude),
       speed = Value(speed),
       accuracy = Value(accuracy);
  static Insertable<TrackPointRow> custom({
    Expression<int>? rowId,
    Expression<String>? rideId,
    Expression<DateTime>? timestamp,
    Expression<double>? latitude,
    Expression<double>? longitude,
    Expression<double>? altitude,
    Expression<double>? speed,
    Expression<double>? accuracy,
  }) {
    return RawValuesInsertable({
      if (rowId != null) 'row_id': rowId,
      if (rideId != null) 'ride_id': rideId,
      if (timestamp != null) 'timestamp': timestamp,
      if (latitude != null) 'latitude': latitude,
      if (longitude != null) 'longitude': longitude,
      if (altitude != null) 'altitude': altitude,
      if (speed != null) 'speed': speed,
      if (accuracy != null) 'accuracy': accuracy,
    });
  }

  TrackPointRowsCompanion copyWith({
    Value<int>? rowId,
    Value<String>? rideId,
    Value<DateTime>? timestamp,
    Value<double>? latitude,
    Value<double>? longitude,
    Value<double>? altitude,
    Value<double>? speed,
    Value<double>? accuracy,
  }) {
    return TrackPointRowsCompanion(
      rowId: rowId ?? this.rowId,
      rideId: rideId ?? this.rideId,
      timestamp: timestamp ?? this.timestamp,
      latitude: latitude ?? this.latitude,
      longitude: longitude ?? this.longitude,
      altitude: altitude ?? this.altitude,
      speed: speed ?? this.speed,
      accuracy: accuracy ?? this.accuracy,
    );
  }

  @override
  Map<String, Expression> toColumns(bool nullToAbsent) {
    final map = <String, Expression>{};
    if (rowId.present) {
      map['row_id'] = Variable<int>(rowId.value);
    }
    if (rideId.present) {
      map['ride_id'] = Variable<String>(rideId.value);
    }
    if (timestamp.present) {
      map['timestamp'] = Variable<DateTime>(timestamp.value);
    }
    if (latitude.present) {
      map['latitude'] = Variable<double>(latitude.value);
    }
    if (longitude.present) {
      map['longitude'] = Variable<double>(longitude.value);
    }
    if (altitude.present) {
      map['altitude'] = Variable<double>(altitude.value);
    }
    if (speed.present) {
      map['speed'] = Variable<double>(speed.value);
    }
    if (accuracy.present) {
      map['accuracy'] = Variable<double>(accuracy.value);
    }
    return map;
  }

  @override
  String toString() {
    return (StringBuffer('TrackPointRowsCompanion(')
          ..write('rowId: $rowId, ')
          ..write('rideId: $rideId, ')
          ..write('timestamp: $timestamp, ')
          ..write('latitude: $latitude, ')
          ..write('longitude: $longitude, ')
          ..write('altitude: $altitude, ')
          ..write('speed: $speed, ')
          ..write('accuracy: $accuracy')
          ..write(')'))
        .toString();
  }
}

abstract class _$RideDatabase extends GeneratedDatabase {
  _$RideDatabase(QueryExecutor e) : super(e);
  $RideDatabaseManager get managers => $RideDatabaseManager(this);
  late final $RidesTable rides = $RidesTable(this);
  late final $TrackPointRowsTable trackPointRows = $TrackPointRowsTable(this);
  @override
  Iterable<TableInfo<Table, Object?>> get allTables =>
      allSchemaEntities.whereType<TableInfo<Table, Object?>>();
  @override
  List<DatabaseSchemaEntity> get allSchemaEntities => [rides, trackPointRows];
}

typedef $$RidesTableCreateCompanionBuilder = RidesCompanion Function({
  required String id,
  required DateTime startTime,
  required DateTime endTime,
  required int durationMs,
  required int movingTimeMs,
  required double distanceMeters,
  required double averageSpeed,
  required double maxSpeed,
  required double elevationGain,
  required double elevationLoss,
  required double startLatitude,
  required double startLongitude,
  required double endLatitude,
  required double endLongitude,
  Value<int> rowid,
});
typedef $$RidesTableUpdateCompanionBuilder = RidesCompanion Function({
  Value<String> id,
  Value<DateTime> startTime,
  Value<DateTime> endTime,
  Value<int> durationMs,
  Value<int> movingTimeMs,
  Value<double> distanceMeters,
  Value<double> averageSpeed,
  Value<double> maxSpeed,
  Value<double> elevationGain,
  Value<double> elevationLoss,
  Value<double> startLatitude,
  Value<double> startLongitude,
  Value<double> endLatitude,
  Value<double> endLongitude,
  Value<int> rowid,
});

final class $$RidesTableReferences
    extends BaseReferences<_$RideDatabase, $RidesTable, RideRow> {
  $$RidesTableReferences(super.$_db, super.$_table, super.$_typedResult);

  static MultiTypedResultKey<$TrackPointRowsTable, List<TrackPointRow>>
  _trackPointRowsRefsTable(_$RideDatabase db) => MultiTypedResultKey.fromTable(
    db.trackPointRows,
    aliasName: 'rides__id__track_point_rows__ride_id',
  );

  $$TrackPointRowsTableProcessedTableManager get trackPointRowsRefs {
    final manager = $$TrackPointRowsTableTableManager(
      $_db,
      $_db.trackPointRows,
    ).filter((f) => f.rideId.id.sqlEquals($_itemColumn<String>('id')!));

    final cache = $_typedResult.readTableOrNull(_trackPointRowsRefsTable($_db));
    return ProcessedTableManager(
      manager.$state.copyWith(prefetchedData: cache),
    );
  }
}

class $$RidesTableFilterComposer extends Composer<_$RideDatabase, $RidesTable> {
  $$RidesTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get startTime => $composableBuilder(
    column: $table.startTime,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get endTime => $composableBuilder(
    column: $table.endTime,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get durationMs => $composableBuilder(
    column: $table.durationMs,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<int> get movingTimeMs => $composableBuilder(
    column: $table.movingTimeMs,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get distanceMeters => $composableBuilder(
    column: $table.distanceMeters,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get averageSpeed => $composableBuilder(
    column: $table.averageSpeed,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get maxSpeed => $composableBuilder(
    column: $table.maxSpeed,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get elevationGain => $composableBuilder(
    column: $table.elevationGain,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get elevationLoss => $composableBuilder(
    column: $table.elevationLoss,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get startLatitude => $composableBuilder(
    column: $table.startLatitude,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get startLongitude => $composableBuilder(
    column: $table.startLongitude,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get endLatitude => $composableBuilder(
    column: $table.endLatitude,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get endLongitude => $composableBuilder(
    column: $table.endLongitude,
    builder: (column) => ColumnFilters(column),
  );

  Expression<bool> trackPointRowsRefs(
    Expression<bool> Function($$TrackPointRowsTableFilterComposer f) f,
  ) {
    final $$TrackPointRowsTableFilterComposer composer = $composerBuilder(
      composer: this,
      getCurrentColumn: (t) => t.id,
      referencedTable: $db.trackPointRows,
      getReferencedColumn: (t) => t.rideId,
      builder:
          (
            joinBuilder, {
            $addJoinBuilderToRootComposer,
            $removeJoinBuilderFromRootComposer,
          }) => $$TrackPointRowsTableFilterComposer(
            $db: $db,
            $table: $db.trackPointRows,
            $addJoinBuilderToRootComposer: $addJoinBuilderToRootComposer,
            joinBuilder: joinBuilder,
            $removeJoinBuilderFromRootComposer:
                $removeJoinBuilderFromRootComposer,
          ),
    );
    return f(composer);
  }
}

class $$RidesTableOrderingComposer
    extends Composer<_$RideDatabase, $RidesTable> {
  $$RidesTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<String> get id => $composableBuilder(
    column: $table.id,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get startTime => $composableBuilder(
    column: $table.startTime,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get endTime => $composableBuilder(
    column: $table.endTime,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get durationMs => $composableBuilder(
    column: $table.durationMs,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<int> get movingTimeMs => $composableBuilder(
    column: $table.movingTimeMs,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get distanceMeters => $composableBuilder(
    column: $table.distanceMeters,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get averageSpeed => $composableBuilder(
    column: $table.averageSpeed,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get maxSpeed => $composableBuilder(
    column: $table.maxSpeed,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get elevationGain => $composableBuilder(
    column: $table.elevationGain,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get elevationLoss => $composableBuilder(
    column: $table.elevationLoss,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get startLatitude => $composableBuilder(
    column: $table.startLatitude,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get startLongitude => $composableBuilder(
    column: $table.startLongitude,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get endLatitude => $composableBuilder(
    column: $table.endLatitude,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get endLongitude => $composableBuilder(
    column: $table.endLongitude,
    builder: (column) => ColumnOrderings(column),
  );
}

class $$RidesTableAnnotationComposer
    extends Composer<_$RideDatabase, $RidesTable> {
  $$RidesTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<String> get id =>
      $composableBuilder(column: $table.id, builder: (column) => column);

  GeneratedColumn<DateTime> get startTime =>
      $composableBuilder(column: $table.startTime, builder: (column) => column);

  GeneratedColumn<DateTime> get endTime =>
      $composableBuilder(column: $table.endTime, builder: (column) => column);

  GeneratedColumn<int> get durationMs => $composableBuilder(
    column: $table.durationMs,
    builder: (column) => column,
  );

  GeneratedColumn<int> get movingTimeMs => $composableBuilder(
    column: $table.movingTimeMs,
    builder: (column) => column,
  );

  GeneratedColumn<double> get distanceMeters => $composableBuilder(
    column: $table.distanceMeters,
    builder: (column) => column,
  );

  GeneratedColumn<double> get averageSpeed => $composableBuilder(
    column: $table.averageSpeed,
    builder: (column) => column,
  );

  GeneratedColumn<double> get maxSpeed =>
      $composableBuilder(column: $table.maxSpeed, builder: (column) => column);

  GeneratedColumn<double> get elevationGain => $composableBuilder(
    column: $table.elevationGain,
    builder: (column) => column,
  );

  GeneratedColumn<double> get elevationLoss => $composableBuilder(
    column: $table.elevationLoss,
    builder: (column) => column,
  );

  GeneratedColumn<double> get startLatitude => $composableBuilder(
    column: $table.startLatitude,
    builder: (column) => column,
  );

  GeneratedColumn<double> get startLongitude => $composableBuilder(
    column: $table.startLongitude,
    builder: (column) => column,
  );

  GeneratedColumn<double> get endLatitude => $composableBuilder(
    column: $table.endLatitude,
    builder: (column) => column,
  );

  GeneratedColumn<double> get endLongitude => $composableBuilder(
    column: $table.endLongitude,
    builder: (column) => column,
  );

  Expression<T> trackPointRowsRefs<T extends Object>(
    Expression<T> Function($$TrackPointRowsTableAnnotationComposer a) f,
  ) {
    final $$TrackPointRowsTableAnnotationComposer composer = $composerBuilder(
      composer: this,
      getCurrentColumn: (t) => t.id,
      referencedTable: $db.trackPointRows,
      getReferencedColumn: (t) => t.rideId,
      builder:
          (
            joinBuilder, {
            $addJoinBuilderToRootComposer,
            $removeJoinBuilderFromRootComposer,
          }) => $$TrackPointRowsTableAnnotationComposer(
            $db: $db,
            $table: $db.trackPointRows,
            $addJoinBuilderToRootComposer: $addJoinBuilderToRootComposer,
            joinBuilder: joinBuilder,
            $removeJoinBuilderFromRootComposer:
                $removeJoinBuilderFromRootComposer,
          ),
    );
    return f(composer);
  }
}

class $$RidesTableTableManager
    extends
        RootTableManager<
          _$RideDatabase,
          $RidesTable,
          RideRow,
          $$RidesTableFilterComposer,
          $$RidesTableOrderingComposer,
          $$RidesTableAnnotationComposer,
          $$RidesTableCreateCompanionBuilder,
          $$RidesTableUpdateCompanionBuilder,
          (RideRow, $$RidesTableReferences),
          RideRow,
          PrefetchHooks Function({bool trackPointRowsRefs})
        > {
  $$RidesTableTableManager(_$RideDatabase db, $RidesTable table)
    : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$RidesTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$RidesTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$RidesTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<String> id = const Value.absent(),
                Value<DateTime> startTime = const Value.absent(),
                Value<DateTime> endTime = const Value.absent(),
                Value<int> durationMs = const Value.absent(),
                Value<int> movingTimeMs = const Value.absent(),
                Value<double> distanceMeters = const Value.absent(),
                Value<double> averageSpeed = const Value.absent(),
                Value<double> maxSpeed = const Value.absent(),
                Value<double> elevationGain = const Value.absent(),
                Value<double> elevationLoss = const Value.absent(),
                Value<double> startLatitude = const Value.absent(),
                Value<double> startLongitude = const Value.absent(),
                Value<double> endLatitude = const Value.absent(),
                Value<double> endLongitude = const Value.absent(),
                Value<int> rowid = const Value.absent(),
              }) => RidesCompanion(
                id: id,
                startTime: startTime,
                endTime: endTime,
                durationMs: durationMs,
                movingTimeMs: movingTimeMs,
                distanceMeters: distanceMeters,
                averageSpeed: averageSpeed,
                maxSpeed: maxSpeed,
                elevationGain: elevationGain,
                elevationLoss: elevationLoss,
                startLatitude: startLatitude,
                startLongitude: startLongitude,
                endLatitude: endLatitude,
                endLongitude: endLongitude,
                rowid: rowid,
              ),
          createCompanionCallback:
              ({
                required String id,
                required DateTime startTime,
                required DateTime endTime,
                required int durationMs,
                required int movingTimeMs,
                required double distanceMeters,
                required double averageSpeed,
                required double maxSpeed,
                required double elevationGain,
                required double elevationLoss,
                required double startLatitude,
                required double startLongitude,
                required double endLatitude,
                required double endLongitude,
                Value<int> rowid = const Value.absent(),
              }) => RidesCompanion.insert(
                id: id,
                startTime: startTime,
                endTime: endTime,
                durationMs: durationMs,
                movingTimeMs: movingTimeMs,
                distanceMeters: distanceMeters,
                averageSpeed: averageSpeed,
                maxSpeed: maxSpeed,
                elevationGain: elevationGain,
                elevationLoss: elevationLoss,
                startLatitude: startLatitude,
                startLongitude: startLongitude,
                endLatitude: endLatitude,
                endLongitude: endLongitude,
                rowid: rowid,
              ),
          withReferenceMapper: (p0) => p0
              .map(
                (e) => (
                  e.readTable<$RidesTable, RideRow>(table),
                  $$RidesTableReferences(db, table, e),
                ),
              )
              .toList(),
          prefetchHooksCallback: ({trackPointRowsRefs = false}) {
            return PrefetchHooks(
              db: db,
              explicitlyWatchedTables: [
                if (trackPointRowsRefs) db.trackPointRows,
              ],
              addJoins: null,
              getPrefetchedDataCallback: (items) async {
                return [
                  if (trackPointRowsRefs)
                    await $_getPrefetchedData<
                      RideRow,
                      $RidesTable,
                      TrackPointRow
                    >(
                      currentTable: table,
                      referencedTable: $$RidesTableReferences
                          ._trackPointRowsRefsTable(db),
                      managerFromTypedResult: (p0) => $$RidesTableReferences(
                        db,
                        table,
                        p0,
                      ).trackPointRowsRefs,
                      referencedItemsForCurrentItem: (item, referencedItems) =>
                          referencedItems.where((e) => e.rideId == item.id),
                      typedResults: items,
                    ),
                ];
              },
            );
          },
        ),
      );
}

typedef $$RidesTableProcessedTableManager =
    ProcessedTableManager<
      _$RideDatabase,
      $RidesTable,
      RideRow,
      $$RidesTableFilterComposer,
      $$RidesTableOrderingComposer,
      $$RidesTableAnnotationComposer,
      $$RidesTableCreateCompanionBuilder,
      $$RidesTableUpdateCompanionBuilder,
      (RideRow, $$RidesTableReferences),
      RideRow,
      PrefetchHooks Function({bool trackPointRowsRefs})
    >;
typedef $$TrackPointRowsTableCreateCompanionBuilder =
    TrackPointRowsCompanion Function({
      Value<int> rowId,
      required String rideId,
      required DateTime timestamp,
      required double latitude,
      required double longitude,
      required double altitude,
      required double speed,
      required double accuracy,
    });
typedef $$TrackPointRowsTableUpdateCompanionBuilder =
    TrackPointRowsCompanion Function({
      Value<int> rowId,
      Value<String> rideId,
      Value<DateTime> timestamp,
      Value<double> latitude,
      Value<double> longitude,
      Value<double> altitude,
      Value<double> speed,
      Value<double> accuracy,
    });

final class $$TrackPointRowsTableReferences
    extends
        BaseReferences<_$RideDatabase, $TrackPointRowsTable, TrackPointRow> {
  $$TrackPointRowsTableReferences(
    super.$_db,
    super.$_table,
    super.$_typedResult,
  );

  static $RidesTable _rideIdTable(_$RideDatabase db) =>
      db.rides.createAlias('track_point_rows__ride_id__rides__id');

  $$RidesTableProcessedTableManager get rideId {
    final $_column = $_itemColumn<String>('ride_id')!;

    final manager = $$RidesTableTableManager(
      $_db,
      $_db.rides,
    ).filter((f) => f.id.sqlEquals($_column));
    final item = $_typedResult.readTableOrNull(_rideIdTable($_db));
    if (item == null) return manager;
    return ProcessedTableManager(
      manager.$state.copyWith(prefetchedData: [item]),
    );
  }
}

class $$TrackPointRowsTableFilterComposer
    extends Composer<_$RideDatabase, $TrackPointRowsTable> {
  $$TrackPointRowsTableFilterComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnFilters<int> get rowId => $composableBuilder(
    column: $table.rowId,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<DateTime> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get latitude => $composableBuilder(
    column: $table.latitude,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get longitude => $composableBuilder(
    column: $table.longitude,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get altitude => $composableBuilder(
    column: $table.altitude,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get speed => $composableBuilder(
    column: $table.speed,
    builder: (column) => ColumnFilters(column),
  );

  ColumnFilters<double> get accuracy => $composableBuilder(
    column: $table.accuracy,
    builder: (column) => ColumnFilters(column),
  );

  $$RidesTableFilterComposer get rideId {
    final $$RidesTableFilterComposer composer = $composerBuilder(
      composer: this,
      getCurrentColumn: (t) => t.rideId,
      referencedTable: $db.rides,
      getReferencedColumn: (t) => t.id,
      builder:
          (
            joinBuilder, {
            $addJoinBuilderToRootComposer,
            $removeJoinBuilderFromRootComposer,
          }) => $$RidesTableFilterComposer(
            $db: $db,
            $table: $db.rides,
            $addJoinBuilderToRootComposer: $addJoinBuilderToRootComposer,
            joinBuilder: joinBuilder,
            $removeJoinBuilderFromRootComposer:
                $removeJoinBuilderFromRootComposer,
          ),
    );
    return composer;
  }
}

class $$TrackPointRowsTableOrderingComposer
    extends Composer<_$RideDatabase, $TrackPointRowsTable> {
  $$TrackPointRowsTableOrderingComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  ColumnOrderings<int> get rowId => $composableBuilder(
    column: $table.rowId,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<DateTime> get timestamp => $composableBuilder(
    column: $table.timestamp,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get latitude => $composableBuilder(
    column: $table.latitude,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get longitude => $composableBuilder(
    column: $table.longitude,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get altitude => $composableBuilder(
    column: $table.altitude,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get speed => $composableBuilder(
    column: $table.speed,
    builder: (column) => ColumnOrderings(column),
  );

  ColumnOrderings<double> get accuracy => $composableBuilder(
    column: $table.accuracy,
    builder: (column) => ColumnOrderings(column),
  );

  $$RidesTableOrderingComposer get rideId {
    final $$RidesTableOrderingComposer composer = $composerBuilder(
      composer: this,
      getCurrentColumn: (t) => t.rideId,
      referencedTable: $db.rides,
      getReferencedColumn: (t) => t.id,
      builder:
          (
            joinBuilder, {
            $addJoinBuilderToRootComposer,
            $removeJoinBuilderFromRootComposer,
          }) => $$RidesTableOrderingComposer(
            $db: $db,
            $table: $db.rides,
            $addJoinBuilderToRootComposer: $addJoinBuilderToRootComposer,
            joinBuilder: joinBuilder,
            $removeJoinBuilderFromRootComposer:
                $removeJoinBuilderFromRootComposer,
          ),
    );
    return composer;
  }
}

class $$TrackPointRowsTableAnnotationComposer
    extends Composer<_$RideDatabase, $TrackPointRowsTable> {
  $$TrackPointRowsTableAnnotationComposer({
    required super.$db,
    required super.$table,
    super.joinBuilder,
    super.$addJoinBuilderToRootComposer,
    super.$removeJoinBuilderFromRootComposer,
  });
  GeneratedColumn<int> get rowId =>
      $composableBuilder(column: $table.rowId, builder: (column) => column);

  GeneratedColumn<DateTime> get timestamp =>
      $composableBuilder(column: $table.timestamp, builder: (column) => column);

  GeneratedColumn<double> get latitude =>
      $composableBuilder(column: $table.latitude, builder: (column) => column);

  GeneratedColumn<double> get longitude =>
      $composableBuilder(column: $table.longitude, builder: (column) => column);

  GeneratedColumn<double> get altitude =>
      $composableBuilder(column: $table.altitude, builder: (column) => column);

  GeneratedColumn<double> get speed =>
      $composableBuilder(column: $table.speed, builder: (column) => column);

  GeneratedColumn<double> get accuracy =>
      $composableBuilder(column: $table.accuracy, builder: (column) => column);

  $$RidesTableAnnotationComposer get rideId {
    final $$RidesTableAnnotationComposer composer = $composerBuilder(
      composer: this,
      getCurrentColumn: (t) => t.rideId,
      referencedTable: $db.rides,
      getReferencedColumn: (t) => t.id,
      builder:
          (
            joinBuilder, {
            $addJoinBuilderToRootComposer,
            $removeJoinBuilderFromRootComposer,
          }) => $$RidesTableAnnotationComposer(
            $db: $db,
            $table: $db.rides,
            $addJoinBuilderToRootComposer: $addJoinBuilderToRootComposer,
            joinBuilder: joinBuilder,
            $removeJoinBuilderFromRootComposer:
                $removeJoinBuilderFromRootComposer,
          ),
    );
    return composer;
  }
}

class $$TrackPointRowsTableTableManager
    extends
        RootTableManager<
          _$RideDatabase,
          $TrackPointRowsTable,
          TrackPointRow,
          $$TrackPointRowsTableFilterComposer,
          $$TrackPointRowsTableOrderingComposer,
          $$TrackPointRowsTableAnnotationComposer,
          $$TrackPointRowsTableCreateCompanionBuilder,
          $$TrackPointRowsTableUpdateCompanionBuilder,
          (TrackPointRow, $$TrackPointRowsTableReferences),
          TrackPointRow,
          PrefetchHooks Function({bool rideId})
        > {
  $$TrackPointRowsTableTableManager(
    _$RideDatabase db,
    $TrackPointRowsTable table,
  ) : super(
        TableManagerState(
          db: db,
          table: table,
          createFilteringComposer: () =>
              $$TrackPointRowsTableFilterComposer($db: db, $table: table),
          createOrderingComposer: () =>
              $$TrackPointRowsTableOrderingComposer($db: db, $table: table),
          createComputedFieldComposer: () =>
              $$TrackPointRowsTableAnnotationComposer($db: db, $table: table),
          updateCompanionCallback:
              ({
                Value<int> rowId = const Value.absent(),
                Value<String> rideId = const Value.absent(),
                Value<DateTime> timestamp = const Value.absent(),
                Value<double> latitude = const Value.absent(),
                Value<double> longitude = const Value.absent(),
                Value<double> altitude = const Value.absent(),
                Value<double> speed = const Value.absent(),
                Value<double> accuracy = const Value.absent(),
              }) => TrackPointRowsCompanion(
                rowId: rowId,
                rideId: rideId,
                timestamp: timestamp,
                latitude: latitude,
                longitude: longitude,
                altitude: altitude,
                speed: speed,
                accuracy: accuracy,
              ),
          createCompanionCallback:
              ({
                Value<int> rowId = const Value.absent(),
                required String rideId,
                required DateTime timestamp,
                required double latitude,
                required double longitude,
                required double altitude,
                required double speed,
                required double accuracy,
              }) => TrackPointRowsCompanion.insert(
                rowId: rowId,
                rideId: rideId,
                timestamp: timestamp,
                latitude: latitude,
                longitude: longitude,
                altitude: altitude,
                speed: speed,
                accuracy: accuracy,
              ),
          withReferenceMapper: (p0) => p0
              .map(
                (e) => (
                  e.readTable<$TrackPointRowsTable, TrackPointRow>(table),
                  $$TrackPointRowsTableReferences(db, table, e),
                ),
              )
              .toList(),
          prefetchHooksCallback: ({rideId = false}) {
            return PrefetchHooks(
              db: db,
              explicitlyWatchedTables: [],
              addJoins:
                  <
                    T extends TableManagerState<
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic,
                      dynamic
                    >
                  >(state) {
                    if (rideId) {
                      state = state.withJoin(
                        currentTable: table,
                        currentColumn: table.rideId,
                        referencedTable: $$TrackPointRowsTableReferences
                            ._rideIdTable(db),
                        referencedColumn: $$TrackPointRowsTableReferences
                            ._rideIdTable(db)
                            .id,
                      ) as T;
                    }

                    return state;
                  },
              getPrefetchedDataCallback: (items) async {
                return [];
              },
            );
          },
        ),
      );
}

typedef $$TrackPointRowsTableProcessedTableManager =
    ProcessedTableManager<
      _$RideDatabase,
      $TrackPointRowsTable,
      TrackPointRow,
      $$TrackPointRowsTableFilterComposer,
      $$TrackPointRowsTableOrderingComposer,
      $$TrackPointRowsTableAnnotationComposer,
      $$TrackPointRowsTableCreateCompanionBuilder,
      $$TrackPointRowsTableUpdateCompanionBuilder,
      (TrackPointRow, $$TrackPointRowsTableReferences),
      TrackPointRow,
      PrefetchHooks Function({bool rideId})
    >;

class $RideDatabaseManager {
  final _$RideDatabase _db;
  $RideDatabaseManager(this._db);
  $$RidesTableTableManager get rides =>
      $$RidesTableTableManager(_db, _db.rides);
  $$TrackPointRowsTableTableManager get trackPointRows =>
      $$TrackPointRowsTableTableManager(_db, _db.trackPointRows);
}
