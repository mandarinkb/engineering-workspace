import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre_gl/maplibre_gl.dart';

import '../../../location/domain/entities/location_point.dart';
import '../../../location/presentation/controllers/location_controller.dart';
import '../../../routing/domain/entities/cycling_route.dart';
import '../../../routing/domain/entities/geo_point.dart';
import '../../../routing/domain/errors/routing_exceptions.dart';
import '../../../routing/presentation/controllers/routing_controller.dart';
import '../../../routing/presentation/widgets/route_options_panel.dart';
import '../../domain/entities/map_region.dart';
import '../controllers/map_regions_controller.dart';

/// Line style for the currently-selected route vs. its alternates (Phase
/// 7) — selected is bolder/opaque so it reads as "this is the one you'd
/// actually ride", alternates stay visible but visually secondary.
const _selectedRouteColor = '#1565C0';
const _alternateRouteColor = '#90A4AE';

/// Phase 4: renders the downloaded offline Bangkok style if available
/// (`activeMapStyleProvider`), falling back to the online demo style
/// otherwise — a live position marker and a tap-to-pin destination sit on
/// top either way (Phase 3).
class MapScreen extends ConsumerStatefulWidget {
  const MapScreen({super.key});

  @override
  ConsumerState<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends ConsumerState<MapScreen> {
  static const _initialCamera = CameraPosition(
    target: LatLng(13.7563, 100.5018),
    zoom: 15,
  );

  MapLibreMapController? _mapController;
  Circle? _currentLocationCircle;
  Circle? _destinationCircle;
  // One Line per alternate Valhalla returned (Phase 7) — index-aligned
  // with routeToDestinationProvider's list, so e.g. _routeLines[0] is
  // always the same route as routes[0], even though its on-screen style
  // (color/width) changes depending on whether it's the selected one.
  List<Line> _routeLines = [];
  bool _hasCenteredOnFirstFix = false;

  @override
  Widget build(BuildContext context) {
    ref.listen<AsyncValue<LocationPoint>>(currentLocationStreamProvider, (previous, next) {
      final point = next.value;
      if (point != null) _onLocationUpdate(point);
    });

    ref.listen<AsyncValue<List<CyclingRoute>>>(routeToDestinationProvider, (previous, next) {
      next.when(
        data: (routes) {
          if (routes.isEmpty) return;
          _redrawRoutes(routes, ref.read(selectedRouteIndexProvider));
          // Phase 6 proof stage: confirm success explicitly, since a
          // silently-successful call that just doesn't render looks
          // identical to a silent failure otherwise.
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: Text(
                  routes.length == 1
                      ? 'Route OK: ${(routes.first.distanceMeters / 1000).toStringAsFixed(2)}km'
                      : 'Found ${routes.length} routes',
                ),
              ),
            );
          }
        },
        loading: () {},
        error: (error, _) {
          final message = error is RoutingException ? error.message : error.toString();
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Routing failed: $message')));
          }
        },
      );
    });

    // Re-style (not re-fetch) when the user taps a different route card in
    // RouteOptionsPanel — the alternates themselves haven't changed, only
    // which one is drawn as "selected".
    ref.listen<int>(selectedRouteIndexProvider, (previous, next) {
      final routes = ref.read(routeToDestinationProvider).value ?? const [];
      if (routes.isNotEmpty) _redrawRoutes(routes, next);
    });

    final styleAsync = ref.watch(activeMapStyleProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Cycling Navigator'),
        actions: [
          IconButton(
            icon: const Icon(Icons.list),
            tooltip: 'Dev menu',
            onPressed: () => context.push('/dev-menu'),
          ),
        ],
      ),
      body: styleAsync.when(
        data: (result) {
          // Deliberately NOT using cameraTargetBounds/minMaxZoomPreference
          // to lock the camera to a downloaded region: MapLibre Native's
          // hard camera-bounds constraint recalculates every gesture frame
          // and made pinch-zoom/pan feel stuck even with generous margin
          // (a known rough edge of bounded cameras on this SDK, not a
          // margin-sizing problem — tried minZoom=15 with plenty of slack
          // and it was still janky). Panning/zooming outside the small
          // downloaded bbox just shows blank space instead, which is an
          // honest reflection of a deliberately tiny test extract and
          // keeps gestures native-smooth everywhere else.
          return Stack(
            children: [
              MapLibreMap(
                key: ValueKey(result.styleString),
                styleString: result.styleString,
                initialCameraPosition: _initialCamera,
                onMapCreated: _onMapCreated,
                onMapClick: (_, coordinates) => _onMapTapped(coordinates),
              ),
              // Sits above the FAB's usual bottom-right spot (default ~72
              // tall including margin) so a route's alternates never cover
              // "recenter" — pending real-device check like every other
              // layout choice on this screen (Phase 4's bounds removal
              // above was itself one of those on-device-only findings).
              const Positioned(left: 0, right: 0, bottom: 84, child: RouteOptionsPanel()),
            ],
          );
        },
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(child: Text('Failed to load map style: $error')),
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _recenter,
        tooltip: 'Recenter on my location',
        child: const Icon(Icons.my_location),
      ),
    );
  }

  void _onMapCreated(MapLibreMapController controller) {
    _mapController = controller;
    // A new controller means a new style/platform view — old Circle/Line
    // handles from the previous one are no longer valid.
    _currentLocationCircle = null;
    _destinationCircle = null;
    _routeLines = [];
    _hasCenteredOnFirstFix = false;
  }

  Future<void> _onLocationUpdate(LocationPoint point) async {
    final controller = _mapController;
    if (controller == null) return;
    final position = LatLng(point.latitude, point.longitude);

    final existing = _currentLocationCircle;
    if (existing == null) {
      _currentLocationCircle = await controller.addCircle(
        CircleOptions(
          geometry: position,
          circleRadius: 8,
          circleColor: '#1565C0',
          circleStrokeColor: '#FFFFFF',
          circleStrokeWidth: 2,
        ),
      );
    } else {
      await controller.updateCircle(existing, CircleOptions(geometry: position));
    }

    if (!_hasCenteredOnFirstFix) {
      _hasCenteredOnFirstFix = true;
      await _moveCameraIfWithinBounds(position);
    }
  }

  Future<void> _onMapTapped(LatLng coordinates) async {
    final controller = _mapController;
    if (controller == null) return;

    final existing = _destinationCircle;
    if (existing == null) {
      _destinationCircle = await controller.addCircle(
        CircleOptions(
          geometry: coordinates,
          circleRadius: 8,
          circleColor: '#D32F2F',
          circleStrokeColor: '#FFFFFF',
          circleStrokeWidth: 2,
        ),
      );
    } else {
      await controller.updateCircle(existing, CircleOptions(geometry: coordinates));
    }

    final currentPoint = ref.read(currentLocationStreamProvider).value;
    if (currentPoint == null) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Still waiting for a GPS fix — try again in a moment.')),
        );
      }
      return;
    }

    await ref.read(routeToDestinationProvider.notifier).calculate(
          GeoPoint(lat: currentPoint.latitude, lon: currentPoint.longitude),
          GeoPoint(lat: coordinates.latitude, lon: coordinates.longitude),
        );
  }

  /// Draws every alternate in [routes], styling [selectedIndex] as the
  /// bold/selected line and the rest as dimmed alternates. Always removes
  /// and re-adds every `Line` (rather than diffing against the previous
  /// draw) — simplest correct option for at most 3 short-lived lines, and
  /// it guarantees the selected route's `Line` is added *last* so
  /// MapLibre layers it on top of the others instead of possibly under
  /// one, which `updateLine` alone can't do (it can't reorder layers).
  Future<void> _redrawRoutes(List<CyclingRoute> routes, int selectedIndex) async {
    final controller = _mapController;
    if (controller == null) return;

    for (final line in _routeLines) {
      await controller.removeLine(line);
    }
    _routeLines = [];

    final drawOrder = [
      for (var i = 0; i < routes.length; i++)
        if (i != selectedIndex) i,
      if (selectedIndex >= 0 && selectedIndex < routes.length) selectedIndex,
    ];

    final newLines = <int, Line>{};
    for (final i in drawOrder) {
      final route = routes[i];
      // Valhalla snaps both ends to the nearest road, which rarely matches
      // the exact requested points (GPS drift, a destination pinned
      // mid-block) — stitch the raw points onto both ends so the line
      // visually reaches the current-location dot and the destination pin
      // exactly, instead of stopping short at the snapped road point.
      final geometry = [
        LatLng(route.start.lat, route.start.lon),
        for (final point in route.points) LatLng(point.lat, point.lon),
        LatLng(route.destination.lat, route.destination.lon),
      ];
      final isSelected = i == selectedIndex;
      newLines[i] = await controller.addLine(
        LineOptions(
          geometry: geometry,
          lineColor: isSelected ? _selectedRouteColor : _alternateRouteColor,
          lineWidth: isSelected ? 5 : 3,
        ),
      );
    }
    // Restore index alignment with `routes` (draw order was reordered
    // above only to control on-screen stacking).
    _routeLines = [for (var i = 0; i < routes.length; i++) newLines[i]!];
  }

  Future<void> _recenter() async {
    final point = ref.read(currentLocationStreamProvider).value;
    if (point == null) return;
    await _moveCameraIfWithinBounds(LatLng(point.latitude, point.longitude));
  }

  /// `cameraTargetBounds` silently blocks any camera move whose target
  /// falls outside a downloaded region — that reads as a dead button
  /// unless we check first and explain why nothing happened.
  Future<void> _moveCameraIfWithinBounds(LatLng position) async {
    final controller = _mapController;
    if (controller == null) return;

    final region = ref.read(activeMapStyleProvider).value?.region;
    if (region != null && !_isWithinRegion(position, region)) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Your current location is outside the downloaded offline area.'),
          ),
        );
      }
      return;
    }

    await controller.animateCamera(CameraUpdate.newLatLngZoom(position, 16));
  }

  bool _isWithinRegion(LatLng position, MapRegion region) {
    return position.latitude >= region.minLat &&
        position.latitude <= region.maxLat &&
        position.longitude >= region.minLon &&
        position.longitude <= region.maxLon;
  }
}
