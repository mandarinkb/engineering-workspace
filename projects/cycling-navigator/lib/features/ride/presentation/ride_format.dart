String formatDistanceKm(double meters) => '${(meters / 1000).toStringAsFixed(2)} km';

String formatSpeedKmh(double metersPerSecond) => '${(metersPerSecond * 3.6).toStringAsFixed(1)} km/h';

String formatElevation(double meters) => '${meters.toStringAsFixed(0)} m';

String _twoDigits(int n) => n.toString().padLeft(2, '0');

String formatDuration(Duration duration) {
  final hours = duration.inHours;
  final minutes = duration.inMinutes.remainder(60);
  final seconds = duration.inSeconds.remainder(60);
  if (hours > 0) {
    return '${_twoDigits(hours)}:${_twoDigits(minutes)}:${_twoDigits(seconds)}';
  }
  return '${_twoDigits(minutes)}:${_twoDigits(seconds)}';
}
