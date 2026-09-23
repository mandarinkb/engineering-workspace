class LocationServiceDisabledException implements Exception {
  const LocationServiceDisabledException();

  @override
  String toString() => 'Location services are turned off on this device.';
}

class LocationPermissionDeniedException implements Exception {
  const LocationPermissionDeniedException({required this.isPermanent});

  /// True when the OS will no longer show the permission prompt — the user
  /// must grant it from the system app settings screen instead.
  final bool isPermanent;

  @override
  String toString() => isPermanent
      ? 'Location permission was permanently denied. Open app settings to grant it.'
      : 'Location permission was denied.';
}
