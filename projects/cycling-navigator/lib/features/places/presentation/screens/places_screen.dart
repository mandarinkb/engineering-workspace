import 'package:flutter/material.dart';

import '../../../../core/widgets/placeholder_screen.dart';

class PlacesScreen extends StatelessWidget {
  const PlacesScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Saved Places',
      phaseNote: 'Not yet scheduled to a numbered phase — see spec section 17. Likely lands alongside Phase 4 or 11.',
    );
  }
}
