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
import org.json.JSONObject

/**
 * Phase 6: a thin native pipe into Valhalla's routing engine
 * (io.github.rallista:valhalla-mobile, confirmed working on Android in
 * the Phase 6a spike — see CYCLING-NAVIGATOR-TODO.md). `ValhallaActor`
 * (raw JSON in/out) is `internal` to that library, so this has to go
 * through the public `Valhalla` class's typed API instead — Kotlin builds
 * the typed request from plain lat/lon args and hands back a small JSON
 * object with only what the Dart side needs (status/summary/shape), via
 * `org.json` (already on the Android SDK, no extra dependency).
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
        )

        val response = valhalla.route(request)
        val json = JSONObject()
        when (response) {
            is ValhallaResponse.Json -> {
                val trip = response.jsonResponse.trip
                json.put("status", trip.status)
                json.put("statusMessage", trip.statusMessage)
                if (trip.status == 0) {
                    json.put("distanceKm", trip.summary.length)
                    json.put("timeSeconds", trip.summary.time)
                    json.put("shape", trip.legs.first().shape)
                }
            }
            else -> {
                json.put("status", -1)
                json.put("statusMessage", "Unexpected response type: $response")
            }
        }
        return json.toString()
    }
}
