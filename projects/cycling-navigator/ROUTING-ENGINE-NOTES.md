# Routing Engine Notes — Library และ Algorithm ที่ใช้นำเส้นทาง (Phase 6)

> เอกสารสรุปเฉพาะเรื่อง routing สำหรับอ้างอิง/เล่าให้คนอื่นฟัง (เช่นตอนสัมภาษณ์งาน) — รายละเอียด phase/checklist เต็มอยู่ที่ [CYCLING-NAVIGATOR-TODO.md](CYCLING-NAVIGATOR-TODO.md)

## สรุปสั้นๆ

แอพนี้คำนวณเส้นทางจักรยาน**บนตัวมือถือเองทั้งหมด ไม่พึ่ง server ไม่พึ่งเน็ต** โดยใช้ **Valhalla** (open source routing engine) ผ่าน library ที่ห่อ Valhalla ไว้ให้ใช้บน Android/iOS ได้โดยตรง

## 1. ตัว Engine: Valhalla

[Valhalla](https://github.com/valhalla/valhalla) เป็น routing engine open source (MIT license) เขียนด้วย C++ พัฒนาต่อยอดมาจากทีม Mapzen เดิม (ปัจจุบัน Mapbox ดูแลต่อ) จุดเด่นที่เลือกใช้:

- มี **bicycle costing model** ของตัวเอง (ไม่ใช้สูตรเดียวกับรถยนต์) เข้าใจ tag OSM ที่เกี่ยวกับจักรยานตรงๆ เช่น `cycleway`, `bicycle=yes/no`, `surface`, `smoothness`
- รองรับ **route alternates** (หลายเส้นทาง) — ใช้จริงตั้งแต่ Phase 7 (ดูหัวข้อ 8 ด้านล่าง)
- ใช้ **hierarchical graph** 3 ระดับ (highway/arterial/local) ทำให้ค้นหาเส้นทางไกลๆได้เร็วโดยไม่ต้องไล่ทุกถนนเล็กๆตลอดทาง

## 2. Algorithm: Time-Dependent Forward A*

จาก log จริงตอน routing ทำงาน (`algorithm::time_dependent_forward_a*`) ยืนยันว่า Valhalla ใช้ **A\* search แบบ time-dependent** ในการหาเส้นทาง — หลักการคร่าวๆ:

- เป็น graph search ทั่วไป (เหมือน A* ที่เรียนกันทั่วไปในวิชา algorithm) แต่ **cost ของแต่ละ edge (ถนนแต่ละท่อน) ไม่ใช่ค่าคงที่** — เปลี่ยนไปตามเงื่อนไข เช่น ความชัน, ประเภทถนน, ความเร็วเฉลี่ยที่ time-of-day (สำหรับโหมด auto ที่มี traffic; โหมด bicycle ของเราไม่ได้ใช้ traffic แต่ยังใช้ framework เดียวกัน)
- ใช้ **heuristic แบบระยะทางตรง (straight-line distance)** ไปยังปลายทางเพื่อเร่งการค้นหา (มาตรฐานของ A*)
- เดินทาง**ข้าม hierarchy level** ได้ — เริ่มค้นจาก local roads ใกล้จุดเริ่ม/จบ แล้วขึ้นไปใช้ arterial/highway ตรงกลางเส้นทางเพื่อความเร็ว ก่อนลงมา local roads อีกทีใกล้ปลายทาง (เทคนิคมาตรฐานของ routing engine ระดับประเทศ/ทวีป)

## 3. Library ที่ใช้เชื่อม Valhalla เข้า Android: `io.github.rallista:valhalla-mobile`

**ไม่ได้ compile Valhalla C++ เองเลย** — ใช้โปรเจกต์ [Rallista/valhalla-mobile](https://github.com/Rallista/valhalla-mobile) (MIT license) ที่ build Valhalla เป็น native library สำเร็จรูปไว้แล้วสำหรับ Android (shared `.so`) และ iOS (static library) publish ผ่าน Maven Central:

```kotlin
implementation("io.github.rallista:valhalla-mobile:0.6.1")
implementation("io.github.rallista:valhalla-models:0.5.2")
implementation("io.github.rallista:valhalla-models-config:0.5.2")
```

Kotlin API ระดับสูงที่ใช้:

```kotlin
val config = ValhallaConfigBuilder().withTileDir(tilesDir.absolutePath).build()
val valhalla = Valhalla(context, config, configManager)
val response = valhalla.route(
    RouteRequest(
        locations = listOf(RoutingWaypoint(lat = ..., lon = ...), ...),
        costing = CostingModel.bicycle,
    )
)
```

**iOS ยังไม่ verify** (ไม่มี Xcode เต็มในเครื่อง dev) — โครงสร้าง library รองรับทั้งสอง platform ตาม README ของโปรเจกต์ แต่ยังไม่ได้ลองจริง

## 4. สถาปัตยกรรมการเชื่อม Flutter ↔ Native (MethodChannel)

Valhalla ให้ API แบบ "raw JSON in/out" ตัวจริง (`ValhallaActor.route(String): String`) แต่ class นั้นเป็น `internal` เข้าถึงจากนอก library ไม่ได้ — ต้องผ่าน class `Valhalla` (public) ที่รับ/คืนเป็น Kotlin object ที่ถูก type ไว้แทน

เพื่อให้ Dart ไม่ต้องรู้จัก type ของ Kotlin เลย ออกแบบให้ **Kotlin เป็นแค่ตัวส่งผ่านโง่ๆ (dumb pipe)**:

```text
Dart (ValhallaRoutingRepository)
   │ ส่ง: startLat/startLon/destLat/destLon/tilesDir (MethodChannel)
   ▼
Kotlin (MainActivity.kt)
   │ สร้าง RouteRequest ที่ type ถูกต้อง → เรียก valhalla.route()
   │ ดึงเฉพาะ status/distanceKm/timeSeconds/shape ออกมา
   │ ห่อเป็น JSON object เล็กๆด้วย org.json (SDK มีให้แล้ว ไม่ต้องเพิ่ม dependency)
   ▼
Dart รับ JSON string กลับมา → jsonDecode → decodePolyline(shape) → CyclingRoute
```

Business logic (แปลง request, validate error, decode geometry) **อยู่ฝั่ง Dart ทั้งหมด** — ฝั่ง native แก้แค่ตอนที่ Valhalla API เปลี่ยน ไม่ต้องแก้ตอน logic เปลี่ยน ตรงตามหลัก Clean Architecture ที่ยึดไว้ทั้งโปรเจกต์ (native = "framework/driver" layer)

## 5. รูปแบบเส้นทางที่ได้กลับมา: Encoded Polyline

Valhalla คืนเส้นทางเป็น**สตริงเดียว** (encoded polyline, algorithm เดียวกับที่ Google Maps ใช้) ไม่ใช่ array ของ lat/lng ตรงๆ — ต้อง decode เอง:

- **Precision 1e6** (ทศนิยม 6 ตำแหน่ง) — Valhalla ใช้ค่านี้เป็น default (Google เดิมใช้ 1e5 ทศนิยม 5 ตำแหน่ง คนละ default กัน ต้องระวังตรงนี้เวลาเอาโค้ด decode จากตัวอย่าง Google มาใช้ตรงๆ)
- Algorithm decode implement เองในไฟล์ `lib/features/routing/domain/services/polyline_codec.dart` (~30 บรรทัด, ไม่พึ่ง package ข้างนอกเพราะเป็น algorithm มาตรฐานสั้นๆ) มี unit test เทียบกับตัวอย่างทางการของ Google ยืนยันว่า decode ถูกต้อง

## 6. Data Pipeline: สร้าง Routing Tiles จาก OSM

Valhalla ต้องการ**ข้อมูลของตัวเอง** (routing graph, `.gph` files) **แยกจาก PMTiles ที่ใช้วาดแผนที่คนละชุดกันโดยสิ้นเชิง** — pipeline ที่ใช้จริง (รันบนเครื่อง Mac ของ dev เท่านั้น ไม่ใช่ dependency ของแอพ):

```text
Geofabrik Thailand .osm.pbf (ทั้งประเทศ ~312MB)
      ↓ osmium extract -b <bbox>
เขตเล็กที่ต้องการ (.osm.pbf ไม่กี่ MB)
      ↓ pyvalhalla's valhalla_build_tiles (ผ่าน prebuilt Python wheel, ไม่ต้อง compile C++ เอง)
Valhalla routing tiles (.gph files, แบ่งตาม hierarchy level 0/1/2)
      ↓ bundle เป็น Flutter asset → copy เข้า app support dir ตอนใช้งานจริง
```

**เครื่องมือสำคัญ**: [`pyvalhalla`](https://pypi.org/project/pyvalhalla/) — Python package ที่มี prebuilt wheel สำหรับ macOS arm64 (ต้อง **Python 3.12+**) มัน bundle Valhalla's CLI tools (`valhalla_build_tiles` ตัวจริง) มาให้ด้วย ทำให้สร้าง routing tiles ได้โดยไม่ต้อง Docker หรือ compile C++ toolchain เองเลย

## 7. ข้อจำกัดที่ตั้งใจไว้ (ยังไม่ทำ)

- **พื้นที่ที่มี routing data ตอนนี้ยังเป็นเขตเดียว** (ขยายจาก bbox ทดสอบเล็กมากไปเป็น bbox ครอบคลุมกรุงเทพฯ+5 จังหวัดปริมณฑลแล้ว — ดูหัวข้อ 9) — นอกเขตนี้คำนวณเส้นทางไม่ได้ ยังไม่ใช่ระบบดาวน์โหลดหลายภูมิภาคแบบ PMTiles (Phase 4) ที่ผู้ใช้เลือกเขตเองได้ ยังเป็น fixed region เดียว bundle มากับตัวแอพ
- **ไม่มี turn-by-turn instructions** — มีแค่เส้นทาง ยังไม่แปลงเป็นคำสั่งเลี้ยว (Phase 8)
- **iOS ไม่ verify**
- **ไม่มี elevation ต่อ route** (Phase 7 ตัดออกตั้งใจ — ดูหัวข้อ 8)
- **Route "snap-to-road" gap**: Valhalla จะ snap จุดเริ่ม/จบไปที่ถนนที่ใกล้ที่สุดเสมอ ไม่ตรงพิกัดดิบเป๊ะ — แก้ด้วยการ "stitch" พิกัดดิบเข้ากับหัว-ท้ายของเส้นทางที่วาด (ดูใน `MapScreen._redrawRoutes`) เพื่อให้เส้นแตะจุดสีน้ำเงิน/หมุดปลายทางเป๊ะทางสายตา แม้ Valhalla จะคำนวณจากจุด snap ที่ต่างออกไปเล็กน้อยก็ตาม — **มีเพดานไว้แล้วตั้งแต่ Phase 7 (ดูหัวข้อ 8) ว่า "เล็กน้อย" คือแค่ไหน**, เกินนั้นถือว่าไม่ใช่เขตที่มีถนนจริง ไม่ใช่แค่ gap ที่ stitch ได้

## 8. Phase 7: Multiple Routes (Alternates)

**`format` เปลี่ยนจาก `json` (native ของ Valhalla) เป็น `osrm`** — พบจากการ**เปิดดู compiled class ของ library ตรงๆด้วย `javap`** (ไม่มีเอกสารอย่างเป็นทางการของ `io.github.rallista:valhalla-mobile` พูดเรื่องนี้เลย): `RouteResponse` model ของ format `json` มีแค่ field `trip` เดียว — ไม่มีที่เก็บ alternate routes ที่ตัว Valhalla engine เองคำนวณไว้อยู่แล้ว ส่วน format `osrm` คืนเป็น `osrmResponse.routes: List<Route>` จริง จึงต้องส่ง `RouteRequest(alternates = 2, format = RouteRequest.Format.osrm, shapeFormat = RouteRequest.ShapeFormat.polyline6)` แทน (ตั้ง `shapeFormat` เป็น polyline6 ชัดเจน เพื่อให้ยังใช้ `decodePolyline()` ตัวเดิมได้โดยไม่ต้องแก้ precision — OSRM เองมี default เป็น polyline5 คนละค่ากับที่แอพนี้ใช้อยู่)

**หน่วยเปลี่ยนไปด้วย ต้องระวัง**: format `osrm` คืน `distance` เป็น**เมตร**และ `duration` เป็น**วินาที**ตรงๆ — ต่างจาก format `json` เดิม (Phase 6) ที่ `summary.length` เป็น**กิโลเมตร** (ต้องคูณ 1000 เอง) ถ้าย้าย logic คูณ 1000 เดิมมาใช้กับ format ใหม่โดยไม่แก้ ระยะทางทุกเส้นจะผิดพลาด 1000 เท่าแบบเงียบๆ — จุดนี้ป้องกันด้วย unit test ที่เขียนไว้ก่อนเชื่อว่า parser ถูก (`test/features/routing/data/valhalla_routing_repository_test.dart`)

**Elevation ต่อ route — ตัดออกจาก Phase 7 ตั้งใจ**: library มี endpoint elevation จริง (`Valhalla.height(HeightRequest)`, เจอผ่าน `javap` เหมือนกัน) แต่ elevation lookup ของ Valhalla ต้องการ DEM/raster tile ของตัวเอง แยกจาก routing graph tile โดยสิ้นเชิง — pipeline ที่ใช้ bundle tile ไว้ตอน Phase 6a (`Geofabrik → osmium extract → pyvalhalla`) ไม่เคยสร้าง/bundle elevation dataset เลย เรียก `height()` กับ tile ชุดปัจจุบันมีโอกาสสูงที่จะ fail หรือคืนค่าใช้ไม่ได้ — เขียนโค้ดส่วนนี้ตอนนี้จะเป็นการ implement บนสมมติฐานที่ยังไม่ผ่าน spike เลย (ขัดกับหลักการ "spike ก่อนเชื่อ" ที่ใช้มาตั้งแต่ Phase 6a) จึงเลื่อนไว้เป็น spike แยกที่ยังไม่ได้ทำ

**บั๊กที่เจอตอน verify บนเครื่องจริง (Samsung A32) — Valhalla ไม่ throw เมื่อจุดหมายอยู่นอกเขต tile**: ลองแตะจุดหมายไกลออกไปนอก bbox ที่มี routing data ตั้งใจเพื่อเช็คว่า error handling ทำงานถูก — สิ่งที่เกิดขึ้นจริงคือแอพวาดเส้นทางที่ดูเหมือนวิ่งตามถนนจริงช่วงต้น แล้วจู่ๆก็ "กระโดด" เป็นเส้นตรงยาวช่วงท้าย **ไม่ขึ้น error เลย** — สาเหตุคือ Valhalla เองไม่ fail กับกรณีนี้ มันแค่ snap จุดหมายไปถนนที่ใกล้ที่สุดเท่าที่ graph มันครอบคลุม ต่อให้ไกลกี่กิโลก็ตาม แล้ว logic "stitch" ของ Phase 6 (ที่ออกแบบมาแก้ gap แค่ไม่กี่เมตรจาก GPS drift) ก็ลากเส้นตรงเชื่อมช่องว่างนั้นไปโดยไม่เช็คระยะเลย ทำให้ดูเหมือนเป็นเส้นทางจริง — **แก้แล้วที่ `parseValhallaRoutesResponse`**: เทียบระยะห่างจริง (`haversineMeters` จาก `ride/domain/services/geo_math.dart`) ระหว่างจุดที่ขอกับจุดที่ Valhalla snap ไปจริง ถ้าเกิน 500 เมตรถือว่าอยู่นอกเขตข้อมูลจริง ไม่ใช่ gap ที่ stitch ได้ — filter เส้นทางนั้นทิ้ง ถ้าเหลือ 0 เส้นทาง throw `RoutingException` ให้ error message ขึ้นจริงแทนวาดเส้นหลอกๆ (มี unit test 3 เคสล็อกพฤติกรรมนี้ไว้)

## 9. Phase 7: ขยาย Routing Coverage เป็นกรุงเทพฯ + ปริมณฑล

ผู้ใช้ขอให้ขยายพื้นที่ routing data ให้ครอบคลุม "จังหวัดที่ติดกรุงเทพฯ" — ทำตาม pipeline เดียวกับหัวข้อ 6 ทุกขั้นตอน แค่เปลี่ยน bbox และวิธี bundle เข้าแอพ:

- **bbox ใหม่**: `99.75,13.30,100.95,14.40` ครอบคลุมกรุงเทพฯ + 5 จังหวัดปริมณฑล (นนทบุรี, ปทุมธานี, สมุทรปราการ, สมุทรสาคร, นครปฐม) — ขยายจาก bbox เดิม (`100.48,13.69,100.63,13.79`, ~16.6km × 11km) เป็นพื้นที่ประมาณ 120km × 120km
- **ขนาดข้อมูลจริงที่ได้**: `.osm.pbf` ที่ extract ได้ 57MB (จาก Thailand ทั้งประเทศ 312MB) → build tiles ได้ 42 ไฟล์ `.gph` รวม 265MB (จากเดิม 6 ไฟล์ ~20MB) → **zip แล้วเหลือ 87MB** (บีบอัดได้ดีเพราะ `.gph` มีโครงสร้างซ้ำเยอะ) — build tiles ใช้เวลาแค่ ~15 วินาทีบนเครื่อง dev (M-series Mac), ไม่ใช่ bottleneck
- **เปลี่ยนวิธี bundle**: จาก list ไฟล์ `.gph` ทีละไฟล์ใน `pubspec.yaml` (ใช้ได้ตอน 6 ไฟล์) เป็น **zip เดียว extract ตอน runtime** (`assets/routing_regions/bangkok-metro.zip`, ผ่าน package `archive`) — จำเป็นเพราะ 42 ไฟล์ที่ลึกถึง 4 ระดับ list มือไม่ไหว และจะยิ่งแย่กว่านี้ถ้าขยายพื้นที่อีกในอนาคต `BundledValhallaTilesProvisioner` เขียนใหม่ทั้งหมด: extract ทุก entry ในซิปไปที่ external files dir ครั้งเดียว แล้วเขียนไฟล์ marker `.provisioned` กันไม่ให้ extract ซ้ำ (และกันกรณี extract ค้างกลางทางจาก process ถูกฆ่า — เช็ค marker แทนที่จะเช็คแค่ว่าโฟลเดอร์มีอยู่)
- **region name เปลี่ยนจาก `bangkok-tiny` เป็น `bangkok-metro`** — แทนที่ของเดิมทั้งหมด ไม่ได้เก็บสองเขตคู่ขนาน (ยังเป็น fixed single region เหมือนเดิม ไม่ได้ทำระบบเลือกเขตแบบ Phase 4)
- **ขั้นตอนที่รันจริง** (บนเครื่อง dev, ไม่ใช่ dependency ของแอพ — เหมือนหัวข้อ 6): `curl` โหลด `thailand-latest.osm.pbf` จาก Geofabrik → `osmium extract -b 99.75,13.30,100.95,14.40` → `python3.12 -m venv` + `pip install pyvalhalla` → เรียก `valhalla/bin/valhalla_build_tiles` ตรงๆ (ไม่ผ่าน wrapper เหมือนหัวข้อ 6) พร้อม config จาก `valhalla.get_config()` → `zip -r` โฟลเดอร์ tiles ทั้งหมด (ต้อง `cd` เข้าไปใน tiles ก่อน zip ไม่งั้น path ข้างในซิปจะมี prefix `tiles/` เกินมา ทำให้ path ไม่ตรงกับที่ native ฝั่ง Kotlin คาดหวัง — `getExternalFilesDir(null)!!.resolve(tilesDirName)` ต้องเจอ `0/002/xxx.gph` ตรงๆ ไม่ใช่ `tiles/0/002/xxx.gph`)
- **ยังไม่ verify บนเครื่องจริง** ว่า route ในพื้นที่ใหม่ (เช่น ข้ามไปฝั่งนนทบุรี/สมุทรปราการ) คำนวณถูกต้อง — ทดสอบแล้วแค่ว่า build ผ่าน+ path ถูกต้องเชิงโครงสร้าง
