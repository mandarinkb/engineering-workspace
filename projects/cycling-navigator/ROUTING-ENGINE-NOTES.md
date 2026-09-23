# Routing Engine Notes — Library และ Algorithm ที่ใช้นำเส้นทาง (Phase 6)

> เอกสารสรุปเฉพาะเรื่อง routing สำหรับอ้างอิง/เล่าให้คนอื่นฟัง (เช่นตอนสัมภาษณ์งาน) — รายละเอียด phase/checklist เต็มอยู่ที่ [CYCLING-NAVIGATOR-TODO.md](CYCLING-NAVIGATOR-TODO.md)

## สรุปสั้นๆ

แอพนี้คำนวณเส้นทางจักรยาน**บนตัวมือถือเองทั้งหมด ไม่พึ่ง server ไม่พึ่งเน็ต** โดยใช้ **Valhalla** (open source routing engine) ผ่าน library ที่ห่อ Valhalla ไว้ให้ใช้บน Android/iOS ได้โดยตรง

## 1. ตัว Engine: Valhalla

[Valhalla](https://github.com/valhalla/valhalla) เป็น routing engine open source (MIT license) เขียนด้วย C++ พัฒนาต่อยอดมาจากทีม Mapzen เดิม (ปัจจุบัน Mapbox ดูแลต่อ) จุดเด่นที่เลือกใช้:

- มี **bicycle costing model** ของตัวเอง (ไม่ใช้สูตรเดียวกับรถยนต์) เข้าใจ tag OSM ที่เกี่ยวกับจักรยานตรงๆ เช่น `cycleway`, `bicycle=yes/no`, `surface`, `smoothness`
- รองรับ **route alternates** (หลายเส้นทาง) — ยังไม่ได้ใช้ใน Phase 6 (จะใช้ตอน Phase 7)
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

- **พื้นที่ที่มี routing data ตอนนี้จำกัดแค่เขตเดียว** (bbox `100.48,13.69,100.63,13.79` ครอบคลุมกรุงเทพฯบางส่วน) — นอกเขตนี้คำนวณเส้นทางไม่ได้ ต้องขยายเป็นระบบดาวน์โหลดตามภูมิภาคเหมือน PMTiles (Phase 4) ในอนาคต
- **Single route เท่านั้น** — ยังไม่มี alternates (Phase 7)
- **ไม่มี turn-by-turn instructions** — มีแค่เส้นทาง ยังไม่แปลงเป็นคำสั่งเลี้ยว (Phase 8)
- **iOS ไม่ verify**
- **Route "snap-to-road" gap**: Valhalla จะ snap จุดเริ่ม/จบไปที่ถนนที่ใกล้ที่สุดเสมอ ไม่ตรงพิกัดดิบเป๊ะ — แก้ด้วยการ "stitch" พิกัดดิบเข้ากับหัว-ท้ายของเส้นทางที่วาด (ดูใน `MapScreen._drawRoute`) เพื่อให้เส้นแตะจุดสีน้ำเงิน/หมุดปลายทางเป๊ะทางสายตา แม้ Valhalla จะคำนวณจากจุด snap ที่ต่างออกไปเล็กน้อยก็ตาม
