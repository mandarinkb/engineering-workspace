import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../../core/providers/shared_preferences_provider.dart';

part 'theme_mode_controller.g.dart';

const _themeModePrefsKey = 'theme_mode';

@riverpod
class ThemeModeController extends _$ThemeModeController {
  @override
  ThemeMode build() {
    final stored = ref.watch(sharedPreferencesProvider).getString(_themeModePrefsKey);
    return ThemeMode.values.firstWhere(
      (mode) => mode.name == stored,
      orElse: () => ThemeMode.system,
    );
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    await ref.read(sharedPreferencesProvider).setString(_themeModePrefsKey, mode.name);
    state = mode;
  }
}
