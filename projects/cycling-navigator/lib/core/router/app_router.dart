import 'package:go_router/go_router.dart';

import '../../features/map/presentation/screens/dev_menu_screen.dart';
import '../../features/map/presentation/screens/map_screen.dart';
import '../../features/map/presentation/screens/offline_maps_screen.dart';
import '../../features/navigation/presentation/screens/navigation_screen.dart';
import '../../features/places/presentation/screens/places_screen.dart';
import '../../features/ride/presentation/screens/active_ride_screen.dart';
import '../../features/ride/presentation/screens/ride_detail_screen.dart';
import '../../features/ride/presentation/screens/ride_history_screen.dart';
import '../../features/ride/presentation/screens/ride_summary_screen.dart';
import '../../features/routing/presentation/screens/route_planning_screen.dart';
import '../../features/settings/presentation/screens/settings_screen.dart';

final appRouter = GoRouter(
  initialLocation: '/',
  routes: [
    GoRoute(path: '/', builder: (context, state) => const MapScreen()),
    GoRoute(
      path: '/dev-menu',
      builder: (context, state) => const DevMenuScreen(),
    ),
    GoRoute(
      path: '/route-planning',
      builder: (context, state) => const RoutePlanningScreen(),
    ),
    GoRoute(
      path: '/navigation',
      builder: (context, state) => const NavigationScreen(),
    ),
    GoRoute(
      path: '/active-ride',
      builder: (context, state) => const ActiveRideScreen(),
    ),
    GoRoute(
      path: '/ride-summary',
      builder: (context, state) => const RideSummaryScreen(),
    ),
    GoRoute(
      path: '/ride-history',
      builder: (context, state) => const RideHistoryScreen(),
      routes: [
        GoRoute(
          path: ':id',
          builder: (context, state) =>
              RideDetailScreen(rideId: state.pathParameters['id']!),
        ),
      ],
    ),
    GoRoute(
      path: '/offline-maps',
      builder: (context, state) => const OfflineMapsScreen(),
    ),
    GoRoute(path: '/places', builder: (context, state) => const PlacesScreen()),
    GoRoute(
      path: '/settings',
      builder: (context, state) => const SettingsScreen(),
    ),
  ],
);
