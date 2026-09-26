# TODO: Cycling Navigator — แอพนำทาง GPS จักรยาน Offline-first (Design เต็ม ยังไม่ Implement)

> **สถานะ**: Phase 0 (architecture/analysis) — เสร็จตามที่ spec ต้นฉบับขอ ("First Task" ห้ามเขียนโค้ดจำนวนมากจนกว่าจะ approve) รอ**อนุมัติจากเจ้าของ repo ก่อนเข้า Phase 1**
>
> **ความสัมพันธ์กับ BOP**: โปรเจกต์แยกอิสระ ไม่ผูกกับ `bot-orchestration-platform` เลย — ดู [REQUIREMENTS.md](REQUIREMENTS.md) เรื่อง priority/time-boxing

## สารบัญ

1. [Reality Check](#1-reality-check)
2. [A. Final Stack](#2-a-final-stack)
3. [B. Architecture Diagram](#3-b-architecture-diagram)
4. [C. Routing Engine Analysis](#4-c-routing-engine-analysis)
5. [D. Offline Map Strategy](#5-d-offline-map-strategy)
6. [E. Storage Architecture](#6-e-storage-architecture)
7. [Database Schema](#7-database-schema)
8. [Domain Interfaces](#8-domain-interfaces)
9. [Folder Structure](#9-folder-structure)
10. [Dependency List](#10-dependency-list)
11. [Licensing](#11-licensing)
12. [F. Development Roadmap](#12-f-development-roadmap)
13. [G. Technical Risks](#13-g-technical-risks)
14. [Open Questions](#14-open-questions)
15. [Reference](#15-reference)

---

## 1. Reality Check

พูดตรงๆ ตามหลักที่ตกลงกันไว้เสมอในทุกโปรเจกต์ ("ไม่ปลอบใจเกินจริง"):

- **Offline bicycle routing บนมือถือทั้ง iOS+Android พร้อมกันคือส่วนที่ยากที่สุดของทั้งแอพ ไม่ใช่แค่ "รายละเอียดทางเทคนิคอันหนึ่ง"** แอพที่ทำสำเร็จแนวนี้ (OsmAnd, Organic Maps, Komoot) ใช้เวลาเป็นปีและมีทีมสร้าง routing engine ของตัวเอง หรือ port engine C++ เข้าทั้งสอง platform เอง — ไม่มี off-the-shelf SDK ที่ "ติดตั้งแล้วใช้ได้เลยทั้ง 2 platform" ในระดับ production ณ ตอนที่เขียนเอกสารนี้ (ดูหัวข้อ 4)
- **แผนที่ (map tiles) กับ routing (road graph) เป็นข้อมูลคนละชุดคนละ pipeline** คนมักเข้าใจผิดว่าดาวน์โหลด "แผนที่ออฟไลน์" แล้วคำนวณเส้นทางได้เลย — ความจริงคือต้อง build ทั้งสองชุดจาก OSM extract เดียวกันแยกกัน (หัวข้อ 5, 6)
- **timeline ที่ realistic**: อย่าประเมิน Phase 6-9 (routing + navigation + rerouting) ว่าจะเร็วกว่า Phase 1-5 (foundation/GPS/map/tracking) รวมกัน — ในทางปฏิบัติมักจะกลับกัน ควรกันเวลาไว้มากสุดสำหรับช่วงนี้ และเริ่มจากพื้นที่ทดสอบเล็กมากก่อน (เช่นแค่เขต 1-2 เขตในกรุงเทพฯ) ไม่ใช่ทั้งประเทศ
- **นี่เป็นโปรเจกต์ personal ที่ทำคู่ขนานกับ BOP** — ถ้า routing spike (Phase 6a) กินเวลาเกิน timebox ที่ตั้งไว้มาก ให้ยอมรับ scope ที่เล็กลง (เช่น ปล่อย iOS แบบไม่มี on-device routing ก่อน) ดีกว่าปล่อยให้กินเวลาจนกระทบ BOP

ไม่ได้บอกว่าห้ามทำ — แค่ต้องตั้งความคาดหวังให้ตรงกับความเป็นจริงตั้งแต่ต้น โดยเฉพาะเรื่อง routing engine

---

## 2. A. Final Stack

| Layer | เลือก | เหตุผล |
|---|---|---|
| Framework | Flutter (stable channel) + Dart 3.x, null safety | ตาม spec, cross-platform เดียวจบทั้ง UI/logic |
| State management | **Riverpod** (`flutter_riverpod` + `riverpod_generator`) | มาตรฐาน Flutter ปี 2026, testable, ไม่ผูก `BuildContext`, แยก UI/Controller/UseCase/Repository ได้สะอาด |
| Map rendering | **`maplibre_gl`** (จาก `maplibre/flutter-maplibre-gl`, fork ของ `flutter-mapbox-gl` เดิม) | official plugin, รองรับ **PMTiles + offline region ในตัว** — ไม่ต้องเขียน local tile server เอง (ยืนยันจาก doc ทางการ `maplibre.org/flutter-maplibre-gl/advanced/pmtiles/`) |
| Tile format | **PMTiles** | ไฟล์เดียวต่อภูมิภาค ไม่ต้องพก SQLite tile server, plugin รองรับตรงๆ |
| Local DB | **Drift** (SQL, บน `sqlite3`) | ปี 2026 นี่คือ "boring correct answer" — **Isar/Hive ผู้เขียนเดิมเลิกดูแลแล้ว เหลือ community fork maintain เท่านั้น** เสี่ยงระยะยาวเกินไปสำหรับข้อมูล ride history ที่ต้องอยู่ได้นานหลายปี Drift ให้ SQL ตรงไปตรงมา + compile-time safety + migration ชัดเจน ตรงกับ schema แบบ relational ใน spec (Ride ↔ TrackPoints ↔ Place) |
| Routing engine (Android — **confirmed working, Phase 6a**) | **Valhalla** ผ่าน `io.github.rallista:valhalla-mobile` (Kotlin) + **MethodChannel** (ไม่ใช่ Dart FFI ดิบๆตามที่ประเมินไว้ตอน Phase 0) | prebuilt AAR บน Maven Central — ไม่ต้อง cross-compile C++ เอง, มี bicycle costing model จริงจัง, MIT license — MethodChannel bridge แบบเดียวกับ plugin อื่นๆที่ใช้อยู่แล้ว (geolocator/maplibre_gl) ง่ายกว่า raw FFI/JNI มาก |
| Routing engine (iOS — ยังไม่ verify) | Valhalla ผ่าน Swift Package (`valhalla-mobile`) | โครงสร้างเดียวกับ Android ตาม README ของโปรเจกต์ แต่ต้องมี Xcode เต็มถึงจะ build/test ได้ — เครื่อง dev นี้ไม่มี |
| Routing engine (fallback ถ้า iOS ไปต่อไม่ได้) | **GraphHopper** (JVM) เฉพาะ Android | เสถียรบน Android เพราะรันบน JVM ตรงๆ ได้ — iOS ต้องพึ่ง community fork ที่ไม่ official |
| GPS | `geolocator` ครอบด้วย `LocationService` interface เอง (ตาม spec หัวข้อ 9) | มาตรฐาน ครอบคลุม lat/lng/altitude/speed/heading/accuracy ทั้ง 2 platform |
| Background tracking | `geolocator` background mode (Android foreground service ในตัว) — iOS ใช้ background location update ผ่าน entitlement ปกติ | เริ่มจาก foreground-first ใน MVP (ดู risk หัวข้อ 13) |
| GPX | `gpx` (pub.dev) + mapping เองเข้า domain model | เขียน parser เองไม่มีประโยชน์ มี package มาตรฐานที่ maintained อยู่แล้ว |
| File/region storage | `path_provider` (application support directory) | แยก map/routing/gpx/db ตาม storage architecture (หัวข้อ 6) |

---

## 3. B. Architecture Diagram

```text
                         Flutter Application (Dart)
                                    │
        ┌───────────────┬──────────┼──────────┬───────────────┐
        │               │          │          │               │
        ▼               ▼          ▼          ▼               ▼
  LocationService   MapLibre    Drift DB   GPX (import/    Riverpod
  (geolocator)      (maplibre_gl)          export)         Controllers
        │               │          │          │
        │          PMTiles files   │          │
        │        (app support dir) │          │
        │                          │          │
        └──────────────┬───────────┴──────────┘
                        │
                        ▼
              Local Navigation Engine
                        │
           ┌────────────┴────────────┐
           │                         │
           ▼                         ▼
   RoutingRepository          Off-route detection
           │                  + local reroute
           ▼
   ┌───────────────────┐
   │ Valhalla (FFI)     │  ← primary, ต้อง spike ก่อน (Phase 6a)
   │ GraphHopper (JVM,   │  ← fallback, Android only
   │   Android only)     │
   └───────────────────┘
           │
           ▼
   Routing graph files
   (app support dir, แยกจาก PMTiles)
```

ไม่มีจุดใดใน core flow (GPS → Map → Routing → Tracking → History → GPX) ที่พึ่ง network — network ใช้เฉพาะตอน "ดาวน์โหลด region package" (map tiles + routing graph) เท่านั้น ตรงตาม spec section 23.

---

## 4. C. Routing Engine Analysis

| Criteria | **Valhalla** | **GraphHopper** | OSRM | BRouter |
|---|---|---|---|---|
| iOS | ไม่ official — ต้อง cross-compile C++ เอง (ARM64), มี GitHub Discussion เปิดอยู่เรื่อง official mobile artifact ยังไม่ปิด | มี **community fork สำหรับ iOS** เท่านั้น ไม่ official | ไม่พบ pattern มือถือ mainstream ที่ยืนยันได้ | **ไม่มีเลย** — Android only |
| Android | ต้อง build เอง (NDK) | native ผ่าน JVM ตรงๆ ได้ (demo อย่างเป็นทางการมีแค่ถึง GraphHopper 1.0 ของใหม่ต้องทำเอง) | ทำได้ทางเทคนิคแต่ไม่มี pattern มือถือ mainstream | ทำได้ แต่ architecture เป็น **AIDL service ที่พึ่งแอปอื่น** (OsmAnd/Locus/OruxMaps ต้องติดตั้งด้วย) |
| Bicycle costing model | ดีมาก — มี bicycle costing โดยเฉพาะ | ดี — มี bike profile | จำกัดกว่า (ออกแบบเพื่อรถเป็นหลักในยุคแรก) | **ดีที่สุดในกลุ่ม** — ออกแบบเพื่อ bike/elevation ตั้งแต่ day 1 |
| Route alternatives | รองรับ | รองรับ | รองรับ | รองรับ |
| ขนาด graph/runtime | กลาง-ใหญ่ (ต้อง build จาก .osm.pbf เอง) | กลาง (มี JVM heap overhead) | ใหญ่ — contraction hierarchy ออกแบบเพื่อ dataset ระดับประเทศ/ทวีปบน server | เล็กสุด — ออกแบบเพื่อโทรศัพท์ตั้งแต่ต้น |
| License | **MIT** | **Apache 2.0** (core) | BSD-2-Clause | LGPL/GPL family — ต้องเช็คให้ชัดก่อนใช้เชิงพาณิชย์ |
| ความเสี่ยงหลัก | ไม่มี official mobile build → ต้อง spike | iOS ต้องพึ่ง fork ที่ไม่ official/ไม่มีคนดูแลระยะยาวแน่นอน | ไม่เหมาะกับมือถือโดยธรรมชาติของ data structure (ต้องการ RAM มาก) | **ตัดออกทันที** — ไม่มี iOS ขัดกับ requirement ที่ต้องมีทั้ง 2 platform |

**สรุป/ตัดสินใจ (อัปเดตหลัง Phase 6a spike จริง — ดูหัวข้อ 12)**: เลือก **Valhalla** — spike ยืนยันแล้วว่า**ทำงานได้จริงบน Android** ผ่าน `io.github.rallista:valhalla-mobile` (prebuilt AAR บน Maven Central, ไม่ต้อง cross-compile C++ เองเลย ต่างจากที่ประเมินไว้ตอน Phase 0) — คำนวณ bicycle route จริงบนเครื่อง Samsung A32 ได้ตัวเลขตรงกับ Python reference (`distance=0.891km time=198.55s`) เป๊ะ **iOS ยังไม่ verify** (ไม่มี Xcode เต็มในเครื่อง dev นี้) ตามที่ประเมินความเสี่ยงไว้แต่แรก — ยังไม่ต้อง fallback ไป GraphHopper สำหรับ Android

---

## 5. D. Offline Map Strategy

```text
OSM extract ภูมิภาค (.osm.pbf จาก Geofabrik)
        ↓
สร้าง vector tiles ผ่าน pipeline ฝั่ง dev/server
(เช่น Planetiler หรือ tilemaker — รันครั้งเดียวต่อภูมิภาค ไม่ใช่บนมือถือ)
        ↓
ไฟล์ .pmtiles ต่อภูมิภาค (single file)
        ↓
Host เป็น static file (S3 / Cloudflare R2 / GitHub Release ก็พอ เพราะเป็นแค่ไฟล์นิ่ง)
        ↓
ผู้ใช้ดาวน์โหลดผ่าน Map Download Manager (spec section 24)
        ↓
เก็บใน app support directory ของมือถือ
        ↓
maplibre_gl เปิดผ่าน pmtiles:// scheme ตรงตาม official doc
        ↓
ใช้งานได้ 100% แบบ airplane mode ทันทีที่ดาวน์โหลดเสร็จ
```

Internet ใช้เฉพาะขั้น "ดาวน์โหลด region package" เท่านั้น — ไม่มีจุดไหนใน core flow ที่ยิง tile request ออนไลน์แบบ dynamic.

---

## 6. E. Storage Architecture

| ประเภทข้อมูล | เก็บที่ | Format |
|---|---|---|
| Map tiles | `<app support dir>/maps/<region>.pmtiles` | PMTiles (ไฟล์เดียว) |
| Routing graph | `<app support dir>/routing/<region>/` | Valhalla tile set (คนละชุดกับ map tiles — ดู reality check หัวข้อ 1) |
| Ride + TrackPoint history | `<app support dir>/db/cycling_navigator.sqlite` | Relational ผ่าน Drift |
| Places (saved pins) | ตารางใน DB เดียวกัน | Relational |
| GPX import/export | `<app documents dir>/gpx/*.gpx` | ไฟล์ .gpx ตรงๆ |
| Settings | key-value table ใน Drift (หรือ `shared_preferences` ถ้าง่ายกว่า) | Key-value |

แยกทั้ง 4 หมวดชัดเจนตาม spec section 41 — ไม่ผสม map/routing/ride/settings เข้าด้วยกัน เพื่อให้เปลี่ยน format ไหนก็ได้ในอนาคตโดยไม่กระทบส่วนอื่น (เช่น เปลี่ยนจาก PMTiles ไปเป็น format ใหม่ทีหลัง ไม่กระทบ Drift schema เลย)

---

## 7. Database Schema

Drift table (แนวคิด ยังไม่ใช่โค้ดจริง — implement ตอน Phase 5):

```text
Ride
 ├── id (pk)
 ├── startTime
 ├── endTime
 ├── duration
 ├── movingTime
 ├── distance
 ├── averageSpeed
 ├── maxSpeed
 ├── elevationGain
 ├── elevationLoss
 ├── startLatitude / startLongitude
 └── endLatitude / endLongitude

TrackPoint  (many-to-one → Ride)
 ├── id (pk)
 ├── rideId (fk → Ride.id)
 ├── timestamp
 ├── latitude / longitude
 ├── altitude
 ├── speed
 └── accuracy

Place
 ├── id (pk)
 ├── name
 ├── latitude / longitude
 ├── category
 └── createdAt
```

TrackPoint แยกตารางจาก Ride ตาม spec section 15 — เลี่ยงเก็บ JSON blob ก้อนใหญ่ ใช้ index บน `rideId` สำหรับ query ตอนแสดง ride detail/elevation profile

---

## 8. Domain Interfaces + Use Cases (Clean Architecture — เหมือน BOP ฝั่ง Go)

โครงสร้างนี้คือ **Clean Architecture แบบเดียวกับที่ใช้ใน BOP (Go)** แค่คนละภาษา — mapping ตรงๆ:

| Clean Architecture layer | BOP (Go) | Cycling Navigator (Flutter) |
|---|---|---|
| Entities | `internal/job/job.go` (struct `Job`) | `domain/entities/ride.dart` (`Ride`, `TrackPoint`) |
| Repository interface | `job.Repository` ใน `job.go` | `domain/repositories/*.dart` (`abstract class`) |
| Use Case | ปนอยู่ใน `internal/worker/pool.go`/handler (Go ไม่แยกชัด) | `domain/usecases/*.dart` — แยก class เดี่ยวชัดเจนกว่า Go |
| Repository implementation | `internal/repository/memory/job_repository.go` | `data/repositories/*.dart` (implements interface, คุยกับ data source ข้างใน) |
| Controller / Interface Adapter | `internal/api/http/handler.go` | `presentation/controllers/*.dart` (Riverpod `Notifier`) |
| Frameworks & Drivers (ชั้นนอกสุด) | net/http, PostgreSQL driver | Flutter widgets, `maplibre_gl`, `drift`, `geolocator`, Valhalla FFI |
| Dependency wiring | constructor injection ใน `main.go` | Riverpod `Provider` override ใน `ProviderScope` |

**กฎเดียวกันทุกภาษา**: dependency ชี้เข้าด้านในเสมอ (`presentation → domain ← data`) — domain เป็น pure Dart ล้วนๆ ห้าม import Flutter/Drift/MapLibre ตรงๆ เหมือนที่ `job.go` ไม่ import `net/http`/`database/sql` ตรงๆ

**ข้อควรระวัง (จาก karpathy-guidelines — อย่า over-engineer)**: ไม่ต้องสร้าง UseCase class แยกทุกการกระทำเล็กๆ (เช่น getter ธรรมดา) — แยกเป็น use case เดี่ยวเฉพาะ logic ที่มีนัยสำคัญ/ถูก reuse ข้ามหลาย controller เท่านั้น ถ้า logic สั้นแค่บรรทัดเดียวใส่ตรงใน controller ได้เลย เหมือนที่ handler บางตัวใน BOP ก็เรียก repository ตรงๆ ไม่ผ่าน service layer เพิ่มถ้าไม่จำเป็น

### Repository interfaces (domain layer)

```dart
abstract class LocationRepository {
  Stream<LocationPoint> watchLocation();
  Future<LocationPoint?> getCurrentLocation();
}

abstract class RoutingRepository {
  Future<List<CyclingRoute>> calculateRoutes(
    GeoPoint start,
    GeoPoint destination,
  );
}

abstract class RideRepository {
  Future<void> saveRide(Ride ride);
  Future<List<Ride>> getRides();
  Future<Ride?> getRide(String id);
}

abstract class MapRepository {
  Future<void> downloadRegion(MapRegion region);
  Future<bool> isRegionAvailable(MapRegion region);
}

abstract class PlacesRepository {
  Future<void> savePlace(Place place);
  Future<List<Place>> getPlaces();
  Future<void> deletePlace(String id);
}

abstract class GpxRepository {
  Future<Ride> importGpx(String filePath);
  Future<String> exportRideToGpx(Ride ride);
}
```

หลักการ: implementation ทั้งหมด (`ValhallaRoutingRepository`, `LocalPmtilesMapRepository`, `DriftRideRepository`, ...) ต้อง replace ได้โดยไม่แก้ interface — เผื่อ Phase 6a spike ล้มเหลวแล้วต้องสลับจาก Valhalla → GraphHopper ไม่ต้องแก้ controller ชั้นบน

### ตัวอย่าง Use Case (เฉพาะ logic ที่มีนัยสำคัญพอจะแยก — ดูข้อควรระวังด้านบน)

```dart
// domain/usecases/start_ride_usecase.dart
class StartRideUseCase {
  StartRideUseCase(this._locationRepository);
  final LocationRepository _locationRepository;

  Stream<LocationPoint> execute() => _locationRepository.watchLocation();
}

// domain/usecases/calculate_route_usecase.dart
class CalculateRouteUseCase {
  CalculateRouteUseCase(this._routingRepository);
  final RoutingRepository _routingRepository;

  Future<List<CyclingRoute>> execute(GeoPoint start, GeoPoint destination) =>
      _routingRepository.calculateRoutes(start, destination);
}

// data/repositories/drift_ride_repository.dart — impl จริง เทียบ memory.JobRepository ของ BOP
class DriftRideRepository implements RideRepository {
  DriftRideRepository(this._db);
  final AppDatabase _db;

  @override
  Future<void> saveRide(Ride ride) => _db.insertRide(ride);
  @override
  Future<List<Ride>> getRides() => _db.selectAllRides();
  @override
  Future<Ride?> getRide(String id) => _db.selectRideById(id);
}
```

---

## 9. Folder Structure

```text
lib/
├── core/
│   ├── config/
│   ├── errors/
│   ├── utils/
│   └── constants/
├── features/
│   ├── ride/
│   │   ├── data/
│   │   │   ├── datasources/       # DriftRideDataSource
│   │   │   └── repositories/      # DriftRideRepository (implements domain interface)
│   │   ├── domain/
│   │   │   ├── entities/          # Ride, TrackPoint
│   │   │   ├── repositories/      # RideRepository (abstract)
│   │   │   └── usecases/          # StartRideUseCase, StopRideUseCase (เฉพาะที่มีนัยสำคัญ)
│   │   └── presentation/
│   │       ├── controllers/       # RideController (Riverpod Notifier)
│   │       └── screens/           # RideScreen, ActiveRideScreen
│   ├── routing/    (data / domain / presentation เหมือนกัน)
│   ├── map/
│   ├── location/
│   ├── navigation/
│   ├── gpx/
│   ├── places/
│   └── settings/
└── main.dart
```

Feature-oriented + Clean Architecture ต่อฟีเจอร์ (ดูข้อ 8) — ไม่แตกเป็น generic folder เยอะเกินจำเป็น และไม่ต้องมี `usecases/` ครบทุกฟีเจอร์ตั้งแต่ Phase 1 ถ้าฟีเจอร์นั้น logic ยังสั้นพอใส่ใน controller ตรงได้ (เช่น `settings/` อาจไม่ต้องมี usecases layer เลยก็ได้)

---

## 10. Dependency List

| Package | ใช้สำหรับ |
|---|---|
| `flutter_riverpod`, `riverpod_annotation`, `riverpod_generator` | state management |
| `go_router` | declarative navigation (official Flutter team package) — เพิ่มเข้ามาใน Phase 1 ตอน implement จริง ไม่ได้อยู่ใน dependency list เดิมของ spec |
| `maplibre_gl` | map rendering + PMTiles/offline region |
| `drift`, `drift_dev`, `drift_flutter` | local database — **ไม่ใช้ `sqlite3_flutter_libs`** (ตัวนั้น EOL แล้ว, maintainer บอกเป็น no-op ตั้งแต่ `sqlite3` v3.x ที่ handle native build เองผ่าน Dart native-assets/hooks toolchain แล้ว) `drift_flutter`'s `driftDatabase()` helper คือวิธีเปิด DB ที่แนะนำปัจจุบัน |
| `geolocator` | GPS location + background mode |
| `gpx` | GPX parse/generate — ยังไม่ได้เพิ่มจริง (Phase 10) |
| `path_provider` | app storage directory ต่างๆ |
| `archive` | extract bundled routing tiles จาก zip asset ตอน runtime (Phase 7, `BundledValhallaTilesProvisioner`) — เพิ่มเข้ามาตอน implement จริงตอนพื้นที่ routing tiles โตเกินกว่าจะ list ไฟล์ทีละไฟล์ใน `pubspec.yaml` ไหว ไม่ได้อยู่ใน dependency list เดิมของ spec |
| `permission_handler` | location permission flow (iOS/Android ต่างกัน) — ยังไม่ได้เพิ่มจริง ตอนนี้ geolocator ขอ permission เองผ่าน native prompt |
| `share_plus` | export GPX ผ่าน native share sheet — ยังไม่ได้เพิ่มจริง (Phase 10) |
| **Android**: `io.github.rallista:valhalla-mobile:0.6.1`, `valhalla-models:0.5.2`, `valhalla-models-config:0.5.2`, `com.stadiamaps:osrm-openapi:0.0.10` (Gradle, ไม่ใช่ pub.dev) | routing engine — **ยืนยันแล้วว่าทำงานจริง (Phase 6a)** เชื่อมเข้า Dart ผ่าน MethodChannel ของเราเอง ไม่ใช้ `dart:ffi` ตรงๆ ตามที่ประเมินไว้ตอน Phase 0 — ต้องใช้ `minSdk = 26` |
| (dev-only, ไม่ใช่ runtime dep) `pyvalhalla` (Python, ต้อง Python 3.12+) + `osmium-tool` | สร้าง Valhalla routing tiles จาก `.osm.pbf` — รันบนเครื่อง dev เท่านั้น ไม่ใช่ dependency ของแอพ |

ต้องเช็ค maintenance status ของแต่ละ package ก่อนล็อกเวอร์ชันจริงตอน Phase 1 (ตาม spec section 47) — รายการนี้คือ candidate ที่ตรวจสอบเบื้องต้นแล้วว่ายัง active ณ ตอนเขียนเอกสารนี้ (กันยายน 2026)

---

## 11. Licensing

| Component | License | ข้อควรระวัง |
|---|---|---|
| OpenStreetMap data | ODbL | ต้องแสดง attribution "© OpenStreetMap contributors" บนแผนที่เสมอ — ใช้ในแอปของ user เองไม่มีปัญหา แต่ถ้าจะแจกจ่าย tile package ต่อสาธารณะต้องตรวจเงื่อนไข share-alike |
| Valhalla | MIT | permissive เต็มรูปแบบ ใช้เชิงพาณิชย์ได้ |
| GraphHopper (core) | Apache 2.0 | permissive — แยกจาก GraphHopper Directions API เชิงพาณิชย์ (ไม่ได้ใช้ตัวนั้น) |
| MapLibre (`maplibre_gl`) | BSD-2-Clause (ตัว plugin/engine) | permissive |
| Drift | MIT | permissive |
| BRouter (ตัดออกจาก candidate แล้ว) | LGPL/GPL family | ไม่เกี่ยวเพราะไม่ได้เลือกใช้ |

---

## 12. F. Development Roadmap

- [x] **Phase 0 — Architecture** (เอกสารนี้)
- [x] **Phase 1 — Flutter Foundation**: `flutter create` + Riverpod + go_router (dev-nav shell ไปทุก stub screen) + Material 3 theme (light/dark/system ผ่าน `ThemeModeController` เก็บลง `shared_preferences`) + `analysis_options.yaml` default lints — verify แล้ว: `flutter analyze` ผ่าน 0 issues, `flutter test` ผ่าน (2 widget test), **`flutter build apk --debug` build สำเร็จจริง + verify บนเครื่องจริง (Samsung A32, Android 13)**: build/install/launch ผ่าน `flutter run` จริง, navigation ไปทุก stub screen ได้, สลับ Light/Dark ใน Settings เปลี่ยนสีจริง, **ปิดแอพแล้วเปิดใหม่ theme ที่เลือกยังจำได้** (ยืนยัน `shared_preferences` persist ข้าม process จริง ไม่ใช่แค่ state ชั่วคราว) — deliverable "แอพเปิดได้ทั้ง 2 platform" ยืนยันฝั่ง Android ครบแล้ว — **iOS ยืนยันไม่ได้ในเครื่อง dev นี้เพราะ Xcode ยังติดตั้งไม่ครบ** ตัวโค้ดเป็น pure Dart/Flutter ไม่มี native code เฉพาะ platform เลยในเฟสนี้ ความเสี่ยงจึงต่ำ แต่ต้อง verify จริงบนเครื่องที่มี Xcode ก่อนเชื่อ 100%
- [x] **Phase 2 — GPS**: `LocationRepository`/`LocationPoint` (domain) + `GeolocatorLocationRepository` (data, wraps `geolocator`, ตรวจ service enabled + ขอ permission ก่อนเปิด stream เสมอ, โยน `LocationServiceDisabledException`/`LocationPermissionDeniedException` เป็น domain exception ของเราเอง ไม่ใช่ type ของ plugin) + `LocationStatusCard` แสดง lat/lng/accuracy/altitude/speed/heading real-time บน Home screen ชั่วคราว (ก่อนมี MapLibre ใน Phase 3) — เพิ่ม `ACCESS_FINE_LOCATION`/`ACCESS_COARSE_LOCATION` ใน AndroidManifest + `NSLocationWhenInUseUsageDescription` ใน Info.plist (foreground-only ตามที่ Phase 0 ตัดสินใจไว้ ยังไม่ขอ background) — **verify บนเครื่องจริง (Samsung A32) ครบ**: popup ขอสิทธิ์เด้งจริง, กด allow แล้ว GPS fix ขึ้นจริงพร้อมตัวเลข real-time — widget test ใช้ fake `LocationRepository` (ไม่พึ่ง platform channel จริงในเทสต์)
- [x] **Phase 3 — MapLibre**: `MapScreen` (MapLibreMap widget), current position marker (Circle annotation อัปเดตจาก `currentLocationStreamProvider`), tap-to-pin destination marker, recenter FAB, dev-menu เข้าถึงผ่านไอคอนบน AppBar — verify บนเครื่องจริงแล้ว: จุดสีน้ำเงิน/แดงขึ้นถูกต้อง, pinch-zoom/pan ทำงาน (ก่อนจะไปเจอปัญหา camera bounds ใน Phase 4)
- [x] **Phase 4 — Offline Maps**: `go-pmtiles extract` (dev-side pipeline จาก Protomaps daily build) → **2 พื้นที่ bundle เข้าแอพ**: "Bangkok Metro (detailed)" (zoom 0-16, ~42MB, เห็นซอยครบ) + "Thailand (overview)" (zoom 0-11, ~44MB, ไม่เห็นซอย) — `MapRepository`/`BundledAssetMapRepository` (download = คัดลอกจาก asset ไปที่ app support dir, ไม่ใช่ network จริง — เป็น MVP simplification ที่ตั้งใจ, สลับเป็น `HttpDownloadMapRepository` ทีหลังได้โดยไม่แก้ caller), Offline Maps screen แสดง list/download/delete/ขนาดไฟล์จริงต่อพื้นที่, `activeMapStyleProvider` เลือกพื้นที่ละเอียดที่สุดที่ดาวน์โหลดไว้อัตโนมัติ — **ทดลองแล้วถอด `cameraTargetBounds`/`minMaxZoomPreference` ออก** เพราะการล็อกกล้องให้อยู่ในขอบเขต region ทำให้ pinch-zoom/pan รู้สึกติดขัดบนเครื่องจริง (ยืนยันแล้วว่าไม่ใช่ปัญหาเรื่อง margin) เลือกยอมให้พื้นที่นอก region ที่ดาวน์โหลดไว้ขึ้นเป็นค่างเปล่าแทน (ตรงไปตรงมากว่า) — verify บนเครื่องจริงแล้ว: download/delete ทำงานจริง, ซูม/pan ลื่นหลังถอด bounds, recenter เตือนถูกต้องเมื่อตำแหน่งจริงอยู่นอก region ที่ดาวน์โหลด — **ยืนยัน airplane mode test แล้ว**: เปิด Airplane Mode → ปิดแอพ-เปิดใหม่ → แผนที่ (พื้นที่ที่ดาวน์โหลดไว้) ยังขึ้นได้ปกติ ไม่มีเน็ตเลย — deliverable "แผนที่ใช้ได้ใน airplane mode" ผ่านจริงบนเครื่อง Samsung A32
- [x] **Phase 5 — Ride Tracking**: `RideStatsCalculator`/`GpsFilter` (pure domain, unit-tested 12 เคส: distance/speed/elevation/moving-time/GPS-noise-rejection ตาม spec section 34) + Drift (`RideDatabase` ผ่าน `drift_flutter`'s `driftDatabase()` — ตัวช่วยที่แนะนำปัจจุบันปี 2026, ไม่ใช่ manual `NativeDatabase` แบบเก่า) + `DriftRideRepository` + `RideTrackingController` (Riverpod, ฟัง `LocationRepository` stream, กรอง noise ก่อนสะสม point, คำนวณ live stats) + Active Ride screen จริง (Start/Stop, ตัวเลข real-time พร้อม ticking timer ไม่ให้ค้างตอนหยุดนิ่ง) + Ride Summary screen จริง — **verify บนเครื่องจริงบางส่วน**: Start Ride ขึ้นตัวเลขจริง (ยืนยันแล้วว่า "0.0 km/h"/"+0 m" ตอนเคลื่อนที่แค่ 10 เมตรเป็นค่าที่ถูกต้อง ไม่ใช่บั๊ก เพราะ elevation threshold 1.5m และ moving-time threshold 0.5 m/s ยังไม่ถึง) — **ยังไม่ได้ยืนยันรอบทดสอบเดินไกลขึ้น (50-100m) + Stop→Summary flow แบบเต็ม** ต้องทำก่อนปิด Phase 5 100%
- [x] **Phase 6a — Routing Engine Feasibility Spike — Android ผ่านแล้ว, iOS ยังไม่ verify**:
  - **ค้นพบสำคัญที่เปลี่ยนแผนเดิม**: ไม่ต้อง cross-compile Valhalla C++ เอง — `io.github.rallista:valhalla-mobile` publish prebuilt AAR (Android) + Swift Package (iOS) ไว้บน Maven Central/SPM แล้ว (โปรเจกต์ MIT license, active, มี Kotlin/Swift API พร้อม `route`/`trace_route`/`height`/`sources_to_targets`)
  - **สร้าง routing tiles โดยไม่ต้อง Docker/compile C++**: `pyvalhalla` (Python, prebuilt wheel สำหรับ macOS arm64, ต้อง **Python 3.12+** ไม่ใช่ system python) มี CLI tools (`valhalla_build_tiles` เป็นต้น) ครบ — pipeline จริงที่ใช้: Geofabrik Thailand `.osm.pbf` → `osmium extract` (bbox เล็ก) → `pyvalhalla`'s `get_config()` + `valhalla_build_tiles` binary ตรงๆ (ไม่ผ่าน `python -m valhalla` wrapper เพราะมันมี bug หาตำแหน่ง executable ไม่เจอ — เรียก binary ใน `valhalla/bin/` ตรงๆแทน)
  - **Kotlin bridge**: `MainActivity.kt` เพิ่ม `MethodChannel("valhalla_spike")` เรียก `ValhallaConfigBuilder().withTileDir(path).build()` → `ValhallaFile` + `ValhallaConfigManager.writeConfig()` → `Valhalla(context, config, manager).route(RouteRequest(locations=[RoutingWaypoint...], costing=CostingModel.bicycle))` — **ต้องขึ้น `minSdk = 26`** (จาก 24 เดิม, library บังคับ) ไม่ใช่ dart:ffi ตรงๆอย่างที่ Phase 0 ประเมินไว้
  - **Verify บนเครื่องจริง (Samsung A32) สำเร็จ**: `distance=0.891 time=198.55s` — **ตรงกับผลจาก Python reference test เป๊ะ** (byte-for-byte เหมือนกัน) ยืนยันว่า native routing engine ทำงานถูกต้องจริงบน ARM64 Android ไม่ใช่แค่ compile ผ่าน
  - **ยังไม่ verify**: iOS (ไม่มี Xcode เต็มในเครื่อง dev นี้ — ตามที่ประเมินความเสี่ยงไว้ตั้งแต่ Phase 0), การจัดการ tile data จริงสำหรับ region (ตอนนี้ push ผ่าน `adb push` เพื่อทดสอบเท่านั้น ยังไม่ได้ผูกกับ `MapRepository`/region download flow ของ Phase 4)
  - **โค้ดที่เขียนไว้เป็น throwaway spike** (`MainActivity.kt`, ปุ่มใน Dev Menu) — Phase 6 ต้องออกแบบสถาปัตยกรรมจริง (`RoutingRepository` interface ตาม Phase 0, ไม่ใช่เรียก MethodChannel ตรงจาก UI)
- [x] **Phase 6 — Local Routing**: `RoutingRepository`/`CyclingRoute`/`GeoPoint` (domain, Clean Architecture ตาม Phase 0) + `ValhallaRoutingRepository` (data, เรียก MethodChannel → parse JSON → decode polyline) + `RouteToDestination` controller (Riverpod) + `MapScreen` วาดเส้นทางจริงเป็น Line annotation เมื่อแตะปักหมุด — รายละเอียด lib/algorithm เต็มอยู่ที่ [ROUTING-ENGINE-NOTES.md](ROUTING-ENGINE-NOTES.md) — **verify บนเครื่องจริงสำเร็จ**: คำนวณ route จริงจากตำแหน่งปัจจุบันไปหมุดปลายทางได้ พร้อม "stitch" เส้นให้แตะจุดที่ขอจริง (เพราะ Valhalla snap ไปถนนที่ใกล้ที่สุดเสมอ ไม่ตรงพิกัดดิบเป๊ะ) — **ขยาย routing tiles จากเขตทดสอบเล็กมากไปเป็นเขตที่ครอบคลุมตำแหน่งจริงของเจ้าของ repo ด้วย** (bbox `100.48,13.69,100.63,13.79`, ~20MB) หลังพบว่าเขตทดสอบเดิมไม่ครอบคลุมตำแหน่งจริงทำให้เส้นทาง "กระโดด" ไปมา — ไฟล์ tile เดิม 6 ไฟล์ที่ path เดิมถูกแทนที่ในเนื้อหา (ไม่ต้องแก้โค้ด/pubspec.yaml)
- [x] **Phase 7 — Multiple Routes (alternates + distance/duration + selection; elevation deferred, see below)**:
  - **`format` switched from Valhalla's native `json` to `osrm`** — found by inspecting the `io.github.rallista:valhalla-mobile:0.6.1` library's *compiled classes* (`javap`, since this isn't documented in its README): the native-`json`-format `RouteResponse` Kotlin model only carries a single `trip` (no field for the alternates Valhalla itself computes), but `RouteRequest.alternates = 2` + `format = osrm` gets them back as `osrmResponse.routes: List<Route>`. `shapeFormat = polyline6` keeps every route decodable with the existing `decodePolyline()` (Valhalla's own default precision) instead of OSRM's own default of polyline5.
  - **Unit footgun caught before it shipped**: OSRM-format routes report `distance` directly in meters / `duration` in seconds — Valhalla's native `json` format (Phase 6) reported `summary.length` in **kilometers**, which is why Phase 6's parser had a `* 1000`. Carrying that conversion over would have made every Phase 7 distance 1000x too large; caught by writing `parseValhallaRoutesResponse`'s unit test *before* trusting the parser, not by manual inspection.
  - `MainActivity.kt` — `runRoute()` now returns `{status, statusMessage, routes: [{distanceMeters, durationSeconds, shape}]}` (array, not a single route) — **verified on-device (Samsung A32)**: multiple real alternates come back and render.
  - **On-device verify (Samsung A32)**: alternates render as distinct lines, `RouteOptionsPanel` shows a card per route, tapping a card re-styles the selected/alternate lines correctly and highlights the tapped card — confirmed via a temporary debug tap log (`select($index) called, previous state=$state`, removed once confirmed) showing every tap firing the correct index with no double-fires or dropped taps.
  - **Process note for next session**: `flutter install` reused a stale APK from the previous day here instead of rebuilding — always `flutter build apk --debug` (or `flutter run`) explicitly before trusting an on-device test, don't rely on `flutter install` alone to pick up source changes.
  - **Bug found + fixed during on-device edge-case testing (Samsung A32) — Valhalla doesn't error on an out-of-coverage destination**: tapping a destination outside the downloaded bbox drew what looked like a real route (correct road-following geometry for the covered stretch), then silently jumped to one long straight line for the rest, with **no error shown at all**. Root cause: Valhalla always "succeeds" by snapping to the nearest road it can reach in the loaded tiles, however far away that is — it doesn't throw. Phase 6's "stitch" (meant to cosmetically bridge a few-meters gap from GPS drift or an off-road pin) then drew a straight line across that gap without ever checking how large it was. **Fixed** in `parseValhallaRoutesResponse`: compares the requested point to Valhalla's actual snapped point with `haversineMeters` (reused from `ride/domain/services/geo_math.dart`); anything past 500m is treated as "outside the routing data," filtered out, and the whole response throws `RoutingException` if nothing qualifies — 3 new unit tests lock this in. **Re-verified on-device**: same out-of-coverage tap now shows the "Routing failed: ..." SnackBar instead of a bogus straight-line route.
  - `RoutingRepository.calculateRoutes` already returned `List<CyclingRoute>` since Phase 6 (unused plurality, now used) — `ValhallaRoutingRepository` parsing extracted into a standalone `parseValhallaRoutesResponse()` function specifically so the parsing/unit-conversion logic could be unit tested without a real `MethodChannel` (this project's established split: platform-channel wiring itself is only verified manually on-device, e.g. `GeolocatorLocationRepository`) — 8 unit tests total in `test/features/routing/data/valhalla_routing_repository_test.dart` (5 from the initial parser + 3 for the snap-distance guard above).
  - `RouteToDestination` controller now holds `List<CyclingRoute>`; new `SelectedRouteIndex` notifier + `selectedRoute` derived provider track which alternate the user picked (resets to index 0 on every new `calculate()` call, so a stale selection from the previous destination can never silently apply to a new one).
  - `RouteOptionsPanel` (new widget) — horizontal scroll of route cards (distance + duration, "Suggested" for index 0 / "Alternate N" for the rest — Valhalla ranks by its own preference, not by strictly shortest/fastest, so cards are labeled by rank, not by a shortest/fastest claim not yet verified) overlaid on `MapScreen`, tap to select.
  - `MapScreen._redrawRoutes` draws every alternate as a `Line` (selected = bold blue `#1565C0` width 5, others = dimmed grey `#90A4AE` width 3), full remove+re-add on every redraw (not `updateLine` diffing) specifically so the selected route's `Line` is always added *last* and layers on top of the others — `updateLine` alone can't reorder layers.
  - **Elevation comparison — deferred, not implemented**: the library does expose an elevation query (`Valhalla.height(HeightRequest)`, confirmed via the same `javap` inspection), but Valhalla elevation lookups need their own DEM/raster tile dataset separate from the routing-graph tiles — and Phase 6's tile pipeline (`Geofabrik → osmium extract → pyvalhalla's valhalla_build_tiles`, see [ROUTING-ENGINE-NOTES.md](ROUTING-ENGINE-NOTES.md)) never built or bundled one. Calling `height()` against the current bundled tiles would very likely fail or return unusable data — writing that code now would be an unverified assumption shipped as done, which is exactly what this project's Phase 6a-style "spike before committing" methodology exists to avoid. Needs its own feasibility spike (does `pyvalhalla` support building/bundling elevation tiles at all? how much do they add to the bundle?) before it's worth writing.
  - **Routing coverage expanded to Bangkok + 5 adjacent provinces** (Nonthaburi, Pathum Thani, Samut Prakan, Samut Sakhon, Nakhon Pathom — bbox `99.75,13.30,100.95,14.40`), by request — same dev-machine pipeline as Phase 6 (Geofabrik → `osmium extract` → `pyvalhalla`'s `valhalla_build_tiles`, re-run with the bigger bbox), full details in [ROUTING-ENGINE-NOTES.md](ROUTING-ENGINE-NOTES.md) §9. Went from 6 tile files (~20MB) to 42 files (265MB raw, **87MB zipped**) — too many/too deeply nested to list individually in `pubspec.yaml` any more, so `BundledValhallaTilesProvisioner` was rewritten to bundle one zip asset (`assets/routing_regions/bangkok-metro.zip`) and extract it at runtime via the new `archive` dependency, instead of copying a hardcoded file list. Region renamed `bangkok-tiny` → `bangkok-metro` (replaced, not kept alongside). **Not yet verified on-device** that routing actually works correctly across the new area (e.g. a route crossing into Nonthaburi/Samut Prakan) — only that the build pipeline completed and file paths line up structurally.
- [ ] **Phase 8 — Navigation**: route progress, remaining distance, next maneuver, off-route detection
- [ ] **Phase 9 — Offline Rerouting**: off-route → local recalculation → continue navigation
- [ ] **Phase 10 — GPX**: import/export/preview
- [ ] **Phase 11 — Ride History**: list, detail, route map, stats, elevation profile
- [ ] **Phase 12 — Optimization**: battery, memory, map render, GPS processing, DB performance, large GPX

---

## 13. G. Technical Risks

| Risk | ผลกระทบ | Mitigation |
|---|---|---|
| ~~**Valhalla ไม่มี official mobile build artifact**~~ **[แก้ไขแล้วหลัง Phase 6a spike]** | ~~Phase 6 อาจล่าช้ามาก~~ — จริงๆมี `io.github.rallista:valhalla-mobile` prebuilt ให้แล้ว, verify บนเครื่องจริงผ่านแล้วสำหรับ Android | ยังเหลือ iOS ที่ไม่ verify (ไม่มี Xcode) — ถ้าจะ release iOS ต้องหาเครื่อง Mac ที่มี Xcode เต็มมา verify ก่อน |
| **Routing graph กับ map tiles เป็นข้อมูลคนละชุด** | ต้องสร้าง pipeline offline สองอันแยกกันก่อนใช้งานได้จริงในพื้นที่ไหนก็ตาม | เริ่มจากพื้นที่ทดสอบเล็กมาก (1 เขต) ก่อนขยาย |
| **iOS background location review** | Apple อาจ reject ถ้า justification ไม่ชัด | v1 เริ่มจาก foreground tracking ก่อน ขอ background ทีหลังเมื่อมี use case ชัดเจนพอ |
| **แบตเตอรี่จาก GPS + map rendering ต่อเนื่อง** | ผู้ใช้เลิกใช้แอพถ้าแบตหมดเร็วเกินไประหว่างปั่น | `LocationUpdatePolicy` ตั้งแต่ day 1 (High accuracy / Balanced / Battery saver) |
| **ขนาดไฟล์แอพ + ข้อมูล offline** | เก็บ native lib (Valhalla arm64 x2 platform) + PMTiles + routing graph รวมกันอาจใหญ่ | ดาวน์โหลด region แยกหลัง install (ไม่ bundle ใน binary) — มี Storage Management screen ตั้งแต่ spec เดิม |
| **OSM data license (ODbL) compliance** | ไม่ใช่ blocker แต่พลาดได้ถ้าลืม | Checklist: แสดง attribution เสมอ, เช็คเงื่อนไข share-alike ก่อนแจกจ่าย tile package ต่อสาธารณะ |
| **GPS noise/outlier ทำให้ distance/speed ผิด** | สถิติ ride ผิดเพี้ยน ผู้ใช้ไม่เชื่อแอพ | GPS filtering layer (accuracy validation → noise filter → outlier detection) ตาม spec section 11 — ต้องทำตั้งแต่ Phase 5 ไม่ใช่ทำทีหลัง |
| **[Phase 7] Elevation ต่อ route ต้องมี DEM tile แยกจาก routing graph tile** | ถ้าเขียน `height()` call ทับ tile ปัจจุบันไปตรงๆ มีโอกาสสูงที่จะ fail เงียบๆหรือคืนค่าใช้ไม่ได้ เพราะ pipeline ที่ bundle tile ไว้ (Phase 6a) ไม่เคยสร้าง elevation dataset เลย | เลื่อนออกจาก Phase 7 ตั้งใจ — ต้องทำ feasibility spike แยก (เช็คว่า `pyvalhalla` build elevation tile ได้ไหม, เพิ่มขนาด bundle เท่าไหร่) ก่อนเขียนโค้ดจริง แทนที่จะเขียนแบบเดาแล้วรอ verify ทีหลัง |

---

## 14. Open Questions

รอคำตอบจากเจ้าของ repo ก่อนเข้า Phase 1:

1. **Timebox ของ Phase 6a spike** — เสนอ 1-2 สัปดาห์ (เวลาว่าง/weekend) ตกลงไหม หรืออยากกำหนดเอง?
2. **พื้นที่ทดสอบแรก** — แนะนำเขต/ถนนที่ปั่นจริงในกรุงเทพฯ (region เล็กสุดที่มีความหมายกับการใช้งานจริง) — ขอบเขตพื้นที่ตรงไหน?
3. **ชื่อแอพ/branding** — `cycling-navigator` เป็นชื่อโฟลเดอร์ชั่วคราว ต้องการชื่อจริงไหม (ยังไม่กระทบโค้ดถ้าไม่เปลี่ยนตอนนี้)?
4. **ยืนยันการ time-box ต่อ BOP** — ตกลงตามที่เขียนใน [REQUIREMENTS.md](REQUIREMENTS.md) ว่าไม่แซง priority ของ Postgres persistence/Jenkins pipeline ของ BOP ใช่ไหม?

---

## 15. Reference

- [maplibre/flutter-maplibre-gl](https://github.com/maplibre/flutter-maplibre-gl) — official Flutter plugin, PMTiles + offline regions
- [PMTiles support doc](https://maplibre.org/flutter-maplibre-gl/advanced/pmtiles/)
- [Valhalla — Official Support for Mobile Build Artifacts (open discussion)](https://github.com/valhalla/valhalla/discussions/4509)
- [Valhalla GitHub](https://github.com/valhalla/valhalla)
- [GraphHopper](https://github.com/graphhopper/graphhopper) — v11.0 (Oct 2025), core Apache 2.0
- [BRouter](https://brouter.de/brouter/) — Android-only, AIDL service architecture
- [Flutter local database landscape 2026 (maintenance-first guide)](https://luci-studio.com/blog/the-flutter-local-database-landscape-in-2026-a-maintenance-first-guide-fe6d267c/)
