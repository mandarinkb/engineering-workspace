import 'package:flutter/material.dart';

/// Stand-in body for screens whose feature isn't implemented yet.
/// [phaseNote] should point at the roadmap phase in CYCLING-NAVIGATOR-TODO.md
/// that will replace it, so it's obvious this isn't a forgotten TODO.
class PlaceholderScreen extends StatelessWidget {
  const PlaceholderScreen({required this.title, required this.phaseNote, super.key});

  final String title;
  final String phaseNote;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            phaseNote,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyLarge,
          ),
        ),
      ),
    );
  }
}
