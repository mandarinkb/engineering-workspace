import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../location/presentation/widgets/location_status_card.dart';

/// Temporary shell reachable from the map's AppBar action so every stub
/// screen stays testable. Will shrink/disappear as real navigation flows
/// (e.g. Stop Ride -> Ride Summary -> Ride History) connect screens
/// directly instead of through this menu.
class DevMenuScreen extends StatelessWidget {
  const DevMenuScreen({super.key});

  static const _destinations = <(String label, String route)>[
    ('Route Planning', '/route-planning'),
    ('Navigation', '/navigation'),
    ('Active Ride', '/active-ride'),
    ('Ride Summary', '/ride-summary'),
    ('Ride History', '/ride-history'),
    ('Offline Maps', '/offline-maps'),
    ('Saved Places', '/places'),
    ('Settings', '/settings'),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Dev Menu')),
      body: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const LocationStatusCard(),
            for (final (label, route) in _destinations)
              ListTile(
                title: Text(label),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => context.push(route),
              ),
          ],
        ),
      ),
    );
  }
}
