import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../ride/presentation/ride_format.dart';
import '../controllers/routing_controller.dart';

/// A row of tappable cards, one per alternate Valhalla returned, shown
/// above the map once a route has been calculated (Phase 7). Renders
/// nothing while there are no routes yet (still loading, or before the
/// first destination tap), so it doesn't reserve empty space on the
/// always-visible `MapScreen`.
class RouteOptionsPanel extends ConsumerWidget {
  const RouteOptionsPanel({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final routes = ref.watch(routeToDestinationProvider).value ?? const [];
    if (routes.isEmpty) return const SizedBox.shrink();

    final selectedIndex = ref.watch(selectedRouteIndexProvider);

    return SizedBox(
      height: 88,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        itemCount: routes.length,
        separatorBuilder: (_, _) => const SizedBox(width: 8),
        itemBuilder: (context, index) {
          final route = routes[index];
          final isSelected = index == selectedIndex;
          return _RouteCard(
            // Valhalla ranks alternates from most to least preferred, but
            // that isn't necessarily "shortest first" — label by rank,
            // not by a claim about which is fastest/shortest.
            label: index == 0 ? 'Suggested' : 'Alternate $index',
            distanceLabel: formatDistanceKm(route.distanceMeters),
            durationLabel: formatDuration(route.duration),
            isSelected: isSelected,
            onTap: () => ref.read(selectedRouteIndexProvider.notifier).select(index),
          );
        },
      ),
    );
  }
}

class _RouteCard extends StatelessWidget {
  const _RouteCard({
    required this.label,
    required this.distanceLabel,
    required this.durationLabel,
    required this.isSelected,
    required this.onTap,
  });

  final String label;
  final String distanceLabel;
  final String durationLabel;
  final bool isSelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: isSelected ? theme.colorScheme.primaryContainer : theme.colorScheme.surface,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Container(
          width: 140,
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: isSelected ? theme.colorScheme.primary : theme.colorScheme.outlineVariant,
              width: isSelected ? 2 : 1,
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(
                label,
                style: theme.textTheme.labelMedium?.copyWith(
                  color: isSelected ? theme.colorScheme.onPrimaryContainer : theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 4),
              Text('$distanceLabel · $durationLabel', style: theme.textTheme.titleSmall),
            ],
          ),
        ),
      ),
    );
  }
}
