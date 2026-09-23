/// Wraps a failure from the routing engine (e.g. Valhalla's own error
/// codes — "No suitable edges near location" when a point is outside the
/// currently loaded routing tiles) in a domain type, so callers never need
/// to know which engine produced it.
class RoutingException implements Exception {
  const RoutingException(this.message);

  final String message;

  @override
  String toString() => 'RoutingException: $message';
}
