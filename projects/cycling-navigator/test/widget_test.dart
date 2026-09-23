import 'package:cycling_navigator/core/providers/shared_preferences_provider.dart';
import 'package:cycling_navigator/core/theme/app_theme.dart';
import 'package:cycling_navigator/features/location/domain/entities/location_point.dart';
import 'package:cycling_navigator/features/location/domain/repositories/location_repository.dart';
import 'package:cycling_navigator/features/location/presentation/controllers/location_controller.dart';
import 'package:cycling_navigator/features/map/domain/entities/map_region.dart';
import 'package:cycling_navigator/features/map/domain/repositories/map_repository.dart';
import 'package:cycling_navigator/features/map/presentation/controllers/map_regions_controller.dart';
import 'package:cycling_navigator/features/map/presentation/screens/dev_menu_screen.dart';
import 'package:cycling_navigator/features/map/presentation/screens/offline_maps_screen.dart';
import 'package:cycling_navigator/features/settings/presentation/controllers/theme_mode_controller.dart';
import 'package:cycling_navigator/features/settings/presentation/screens/settings_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Widget tests never talk to the real `geolocator` plugin (there's no
/// platform channel host in the test binary) — the repository is faked so
/// the GPS card resolves instantly instead of spinning forever.
class _FakeLocationRepository implements LocationRepository {
  static final _point = LocationPoint(
    latitude: 13.7563,
    longitude: 100.5018,
    timestamp: DateTime(2026),
    accuracy: 5,
    altitude: 1.5,
    speed: 4.2,
    heading: 90,
  );

  @override
  Stream<LocationPoint> watchLocation() => Stream.value(_point);

  @override
  Future<LocationPoint?> getCurrentLocation() async => _point;
}

/// Fakes file I/O and asset bundling entirely (no real `path_provider`/asset
/// plugin channel in the test binary) — just tracks which region ids have
/// been "downloaded" in memory.
class _FakeMapRepository implements MapRepository {
  static const region = MapRegion(
    id: 'thailand',
    name: 'Thailand',
    minLat: 5.6,
    minLon: 97.3,
    maxLat: 20.5,
    maxLon: 105.7,
  );

  final Set<String> _downloadedIds = {};

  @override
  List<MapRegion> get availableRegions => const [region];

  @override
  Future<void> downloadRegion(MapRegion region) async => _downloadedIds.add(region.id);

  @override
  Future<void> deleteRegion(MapRegion region) async => _downloadedIds.remove(region.id);

  @override
  Future<bool> isRegionAvailable(MapRegion region) async => _downloadedIds.contains(region.id);

  @override
  Future<int> regionSizeBytes(MapRegion region) async =>
      _downloadedIds.contains(region.id) ? 44000000 : 0;

  @override
  Future<String?> stylePathForRegion(MapRegion region) async =>
      _downloadedIds.contains(region.id) ? '/fake/thailand.style.json' : null;
}

/// `MapLibreMap` (the real home route, '/') creates a native platform view
/// that has no host in `flutter test`'s headless environment — mounting or
/// disposing it there throws `LateInitializationError` from the plugin's
/// own channel teardown. That's a limitation of testing native map SDKs
/// this way, not a bug in this app, so these tests start from '/dev-menu'
/// instead and never construct the real MapScreen. The map itself is
/// verified on a physical device (see CYCLING-NAVIGATOR-TODO.md Phase 3).
Future<Widget> buildTestApp() async {
  SharedPreferences.setMockInitialValues({});
  final prefs = await SharedPreferences.getInstance();

  final router = GoRouter(
    initialLocation: '/dev-menu',
    routes: [
      GoRoute(path: '/dev-menu', builder: (context, state) => const DevMenuScreen()),
      GoRoute(path: '/settings', builder: (context, state) => const SettingsScreen()),
      GoRoute(path: '/offline-maps', builder: (context, state) => const OfflineMapsScreen()),
    ],
  );

  return ProviderScope(
    overrides: [
      sharedPreferencesProvider.overrideWithValue(prefs),
      locationRepositoryProvider.overrideWithValue(_FakeLocationRepository()),
      mapRepositoryProvider.overrideWithValue(_FakeMapRepository()),
    ],
    child: Consumer(
      builder: (context, ref, _) {
        final themeMode = ref.watch(themeModeControllerProvider);
        return MaterialApp.router(
          theme: AppTheme.light,
          darkTheme: AppTheme.dark,
          themeMode: themeMode,
          routerConfig: router,
        );
      },
    ),
  );
}

void main() {
  testWidgets('dev menu lists every stub destination and shows GPS fix', (tester) async {
    await tester.pumpWidget(await buildTestApp());
    await tester.pumpAndSettle();

    expect(find.text('Dev Menu'), findsOneWidget);
    expect(find.text('Settings'), findsOneWidget);
    expect(find.text('Ride History'), findsOneWidget);
    expect(find.textContaining('13.756300'), findsOneWidget);
  });

  testWidgets('navigating to settings and switching theme mode persists', (tester) async {
    await tester.pumpWidget(await buildTestApp());
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Settings'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Settings'));
    await tester.pumpAndSettle();

    expect(find.text('Theme'), findsOneWidget);

    await tester.tap(find.text('Dark'));
    await tester.pumpAndSettle();

    final material = tester.widget<MaterialApp>(find.byType(MaterialApp));
    expect(material.themeMode, ThemeMode.dark);
  });

  testWidgets('offline map region can be downloaded then deleted', (tester) async {
    await tester.pumpWidget(await buildTestApp());
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Offline Maps'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Offline Maps'));
    await tester.pumpAndSettle();

    expect(find.text('Thailand'), findsOneWidget);
    expect(find.text('Not downloaded'), findsOneWidget);
    expect(find.text('Download'), findsOneWidget);

    await tester.tap(find.text('Download'));
    await tester.pumpAndSettle();

    expect(find.text('42.0 MB on device'), findsOneWidget);
    expect(find.text('Delete'), findsOneWidget);

    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();

    expect(find.text('Not downloaded'), findsOneWidget);
    expect(find.text('Download'), findsOneWidget);
  });
}
