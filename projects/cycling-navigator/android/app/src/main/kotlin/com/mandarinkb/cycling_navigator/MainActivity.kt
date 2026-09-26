package com.mandarinkb.cycling_navigator

import com.valhalla.api.models.CostingModel
import com.valhalla.api.models.RouteRequest
import com.valhalla.api.models.RoutingWaypoint
import com.valhalla.config.ValhallaConfigBuilder
import com.valhalla.valhalla.Valhalla
import com.valhalla.valhalla.ValhallaResponse
import com.valhalla.valhalla.config.ValhallaConfigManager
import com.valhalla.valhalla.files.ValhallaFile
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import org.json.JSONArray
import org.json.JSONObject

/**
 * Phase 6/7: a thin native pipe into Valhalla's routing engine
 * (io.github.rallista:valhalla-mobile, confirmed working on Android in
 * the Phase 6a spike — see CYCLING-NAVIGATOR-TODO.md). `ValhallaActor`
 * (raw JSON in/out) is `internal` to that library, so this has to go
 * through the public `Valhalla` class's typed API instead — Kotlin builds
 * the typed request from plain lat/lon args and hands back a small JSON
 * object with only what the Dart side needs, via `org.json` (already on
 * the Android SDK, no extra dependency).
 *
 * Phase 7: switched `format` from Valhalla's native `json` to `osrm`.
 * Confirmed by inspecting the library's compiled classes (javap) since
 * this isn't documented in its README: the native-`json`-format
 * `RouteResponse` model this library deserializes into only carries a
 * single `trip` — there is no field for the alternate routes Valhalla
 * itself computes. The `osrm`-format response's `routes: List<Route>`
 * does carry them. `shapeFormat = polyline6` keeps every route's geometry
 * decodable with the same `decodePolyline()` already used for Phase 6
 * (Valhalla's own default precision), instead of OSRM's own default of
 * polyline5.
 */
class MainActivity : FlutterActivity() {
    private val channelName = "valhalla_routing"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName).setMethodCallHandler { call, result ->
            if (call.method == "calculateRoute") {
                val args = call.arguments as? Map<*, *>
                val tilesDirName = args?.get("tilesDir") as? String
                val startLat = (args?.get("startLat") as? Double)
                val startLon = (args?.get("startLon") as? Double)
                val destLat = (args?.get("destLat") as? Double)
                val destLon = (args?.get("destLon") as? Double)

                if (tilesDirName == null || startLat == null || startLon == null || destLat == null || destLon == null) {
                    result.error("INVALID_ARGS", "tilesDir, startLat, startLon, destLat, destLon are required", null)
                    return@setMethodCallHandler
                }

                try {
                    result.success(runRoute(tilesDirName, startLat, startLon, destLat, destLon))
                } catch (e: Exception) {
                    result.error("VALHALLA_ERROR", e.message, e.stackTraceToString())
                }
            } else {
                result.notImplemented()
            }
        }
    }

    private fun runRoute(
        tilesDirName: String,
        startLat: Double,
        startLon: Double,
        destLat: Double,
        destLon: Double,
    ): String {
        val tilesDir = getExternalFilesDir(null)!!.resolve(tilesDirName)
        val config = ValhallaConfigBuilder().withTileDir(tilesDir.absolutePath).build()

        val configFile = ValhallaFile(applicationContext, "valhalla_$tilesDirName.json", filesDir)
        val manager = ValhallaConfigManager(applicationContext, configFile)
        manager.writeConfig(config)

        val valhalla = Valhalla(applicationContext, config, manager)
        val request = RouteRequest(
            locations = listOf(
                RoutingWaypoint(lat = startLat, lon = startLon),
                RoutingWaypoint(lat = destLat, lon = destLon),
            ),
            costing = CostingModel.bicycle,
            // Up to 2 alternates + the primary route — Valhalla's own cap;
            // asking for more doesn't guarantee more come back, since it
            // only returns alternates that are meaningfully different from
            // the primary route.
            alternates = 2,
            format = RouteRequest.Format.osrm,
            shapeFormat = RouteRequest.ShapeFormat.polyline6,
        )

        val json = JSONObject()
        when (val response = valhalla.route(request)) {
            is ValhallaResponse.Osrm -> {
                val osrm = response.osrmResponse
                if (osrm.code != com.osrm.api.models.RouteResponse.Code.Ok) {
                    json.put("status", -1)
                    json.put("statusMessage", osrm.message ?: osrm.code.toString())
                    return json.toString()
                }

                json.put("status", 0)
                json.put("statusMessage", "")
                val routesJson = JSONArray()
                for (route in osrm.routes.orEmpty()) {
                    // `geometry` is typed `Any` in this library's model
                    // because OSRM format can also return it as a GeoJSON
                    // object — always a String here since the request
                    // above always sets shapeFormat to polyline6.
                    val shape = route.geometry as? String ?: continue
                    val routeJson = JSONObject()
                    // OSRM format reports distance in meters and duration
                    // in seconds directly — unlike Valhalla's native `json`
                    // format (Phase 6), whose `summary.length` was in
                    // kilometers. No *1000 conversion needed here.
                    routeJson.put("distanceMeters", route.distance ?: 0.0)
                    routeJson.put("durationSeconds", route.duration ?: 0.0)
                    routeJson.put("shape", shape)
                    routesJson.put(routeJson)
                }
                json.put("routes", routesJson)
            }
            else -> {
                json.put("status", -1)
                json.put("statusMessage", "Unexpected response type from Valhalla: $response")
            }
        }
        return json.toString()
    }
}
