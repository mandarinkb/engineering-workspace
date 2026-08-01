# บทที่ 1: REST และ gRPC

REST กับ gRPC คือสองวิธีหลักที่ service คุยกัน ไม่มีตัวไหน "ดีกว่า" ในทุกกรณี — คำถามที่ต้องตอบให้ได้คือ ใครคือผู้บริโภค API นี้ (browser/third-party หรือ service ภายในทีมเดียวกัน) และ trade-off ระหว่าง human-readability กับ performance คุ้มค่าตรงไหน

## 1.1 REST

### Resource-based Design

หัวใจของ REST (Representational State Transfer) คือมอง API เป็น **resource** (คำนาม) ไม่ใช่ action (คำกริยา) — URL ควรบอกว่า "อะไร" ไม่ใช่ "ทำอะไร":

```
ถูก (resource-based):
  GET    /users/42
  POST   /users
  PUT    /users/42
  DELETE /users/42

ผิด (RPC-style ปนใน REST):
  GET  /getUser?id=42
  POST /createUser
  POST /deleteUser?id=42
```

เมื่อ resource มีความสัมพันธ์แบบลำดับชั้น ให้สะท้อนใน path: `GET /users/42/orders` หมายถึง "order ทั้งหมดของ user 42" — แต่ไม่ควรซ้อนลึกเกิน 2-3 ระดับ (`/users/42/orders/7/items/3/reviews/...` อ่านยากและ couple โครงสร้าง URL เข้ากับ data model แน่นเกินไป) ถ้าลึกเกินไปให้พิจารณาทำ resource นั้นเป็น top-level ที่ filter ผ่าน query parameter แทน เช่น `GET /reviews?item_id=3`

### ใช้ HTTP Verb ให้ตรงความหมาย

| Verb | ความหมาย | Idempotent? | Safe (ไม่เปลี่ยน state)? |
|---|---|---|---|
| `GET` | อ่านข้อมูล | ใช่ | ใช่ |
| `POST` | สร้างใหม่ / action ที่ไม่ idempotent | ไม่ใช่ | ไม่ใช่ |
| `PUT` | สร้าง/แทนที่ทั้ง resource | ใช่ | ไม่ใช่ |
| `PATCH` | แก้ไขบางส่วนของ resource | โดยทั่วไปไม่ใช่ (ขึ้นกับการออกแบบ) | ไม่ใช่ |
| `DELETE` | ลบ resource | ใช่ | ไม่ใช่ |

"Idempotent" หมายถึงเรียกซ้ำกี่ครั้งก็ได้ผลลัพธ์สุดท้ายเหมือนเดิม (เช่น `DELETE /users/42` เรียกครั้งที่สองก็ยัง "ไม่มี user 42" เหมือนเดิม ต่างจาก `POST /users` ที่เรียกซ้ำจะสร้าง user ใหม่ซ้ำอีกตัว) — คุณสมบัตินี้สำคัญมากตอนออกแบบ retry logic (เชื่อมกับ [บทที่ 5 — Idempotency](05-idempotency-and-tracing.md)) เพราะ client สามารถ retry `PUT`/`DELETE` ได้อย่างปลอดภัยเมื่อไม่แน่ใจว่า request ก่อนหน้าไปถึงปลายทางหรือไม่ แต่ retry `POST` ตรงๆ โดยไม่มีกลไกป้องกันจะเสี่ยงสร้างข้อมูลซ้ำ

ข้อผิดพลาดที่พบบ่อย: ใช้ `GET` แต่มี side effect (เช่น `GET /users/42/send-email` ที่ยิงอีเมลทุกครั้งที่เรียก) ผิดหลัก "safe" ของ GET — browser, proxy, หรือ crawler อาจเรียก GET ซ้ำโดยไม่ตั้งใจ (prefetch, retry) แล้วทำให้เกิด side effect ที่ไม่คาดคิด

### Status Code Discipline

Status code ไม่ใช่แค่ "200 คือสำเร็จ, อื่นๆ คือ error" — การเลือก code ที่แม่นยำช่วยให้ client (และ monitoring/alerting) ตัดสินใจถูกโดยไม่ต้อง parse response body:

| กลุ่ม | ตัวอย่าง | ความหมาย |
|---|---|---|
| 2xx | `200 OK`, `201 Created`, `204 No Content` | สำเร็จ — `201` ใช้ตอนสร้าง resource ใหม่ (ควรมี header `Location` ชี้ไปที่ resource ที่สร้าง), `204` ใช้ตอนสำเร็จแต่ไม่มี body กลับ (เช่น `DELETE`) |
| 3xx | `301`, `304 Not Modified` | Redirect / cache validation |
| 4xx | `400`, `401`, `403`, `404`, `409`, `422`, `429` | **ความผิดของ client** — `401` ไม่ได้ authenticate, `403` authenticate แล้วแต่ไม่มีสิทธิ์, `404` ไม่มี resource นี้, `409 Conflict` ขัดแย้งกับ state ปัจจุบัน (เช่น optimistic lock ล้มเหลว), `422` รูปแบบข้อมูลถูกต้องแต่ business validation ไม่ผ่าน, `429 Too Many Requests` โดน rate limit |
| 5xx | `500`, `502`, `503` | **ความผิดของ server** — client ทำถูกทุกอย่างแล้วแต่ server มีปัญหา (เชื่อมกับ [บทที่ 4 — Circuit Breaker](04-saga-and-resilience.md) ที่ client ควรตอบสนองต่อ 5xx ต่างจาก 4xx) |

แยก 4xx กับ 5xx ให้ถูกต้องสำคัญมากในทางปฏิบัติ เพราะ retry logic ฝั่ง client มักตั้งกฎว่า "retry เฉพาะ 5xx หรือ network error เท่านั้น" — ถ้า server ตอบ `500` ทั้งที่จริงเป็นความผิด client (เช่น validation error) client จะ retry ซ้ำๆ โดยไม่มีทางสำเร็จ สิ้นเปลืองทั้ง resource ทั้งสองฝั่งโดยเปล่าประโยชน์

### API Versioning

เมื่อ API ต้อง breaking change (เปลี่ยน field, เปลี่ยนความหมาย) แต่ client เก่ายังใช้งานอยู่ ต้องมีกลยุทธ์ versioning:

- **URI versioning** — `/v1/users`, `/v2/users` — ชัดเจน เห็นตรงๆ ใน log/URL แต่ทำให้ resource เดียวกันมีหลาย URL
- **Header versioning** — `Accept: application/vnd.myapi.v2+json` — URL สะอาด แต่ debug ยากกว่า (ต้องดู header ไม่ใช่แค่ดู URL)
- **Query parameter versioning** — `/users?version=2` — พบน้อยกว่าสองแบบแรก

ในทางปฏิบัติ **URI versioning** เป็นที่นิยมที่สุดเพราะ debug ง่ายและ cache ตาม URL ได้ตรงไปตรงมา ไม่ว่าจะเลือกแบบไหน หลักสำคัญคือต้องมี**นโยบาย deprecation ที่ชัดเจน** (เช่น ประกาศล่วงหน้า 6 เดือน, ส่ง header `Deprecation`/`Sunset` เตือน client) ไม่ใช่ตัด v1 ทิ้งกะทันหัน

### HATEOAS (แนวคิดที่รู้ไว้ แต่ไม่ค่อยได้ใช้จริง)

**HATEOAS** (Hypermedia As The Engine Of Application State) คือแนวคิดใน REST ระดับสูงสุด (Richardson Maturity Model ระดับ 3) ที่ response ควรมี **link ไปยัง action ที่ทำได้ต่อไป** ฝังมาด้วย แทนที่ client จะต้อง hardcode URL เอง:

```json
{
  "id": 42,
  "status": "pending",
  "_links": {
    "self": { "href": "/orders/42" },
    "cancel": { "href": "/orders/42/cancel" },
    "pay": { "href": "/orders/42/pay" }
  }
}
```

แนวคิดสวยงามในทางทฤษฎี (client "ค้นพบ" ว่าทำอะไรได้บ้างจาก response โดยไม่ต้องรู้ล่วงหน้า คล้ายเว็บที่คนกด link ตามที่เห็น) แต่ในทางปฏิบัติ API ส่วนใหญ่ในโลกจริง**ไม่ทำ HATEOAS เต็มรูปแบบ** เพราะ client (โดยเฉพาะ mobile app, SPA) มักถูก hardcode logic ไว้อยู่แล้วว่าต้องเรียก endpoint ไหนต่อ การมี hypermedia link เพิ่มความซับซ้อนโดยไม่ได้ประโยชน์ตามสัดส่วน ควรรู้จักไว้เป็นความรู้ว่า REST มีระดับความสมบูรณ์ไปได้ไกลแค่ไหน แต่ไม่ต้องคาดหวังว่าจะเจอ/ต้องทำในงานส่วนใหญ่

## 1.2 gRPC

### Protocol Buffers และ Code Generation

**gRPC** ใช้ **Protocol Buffers (protobuf)** เป็นทั้ง interface definition language (IDL) และ binary serialization format แทน JSON ที่ REST ใช้ทั่วไป — เขียน contract ในไฟล์ `.proto` ครั้งเดียว แล้ว generate โค้ด client/server สำหรับหลายภาษาอัตโนมัติ:

```protobuf
syntax = "proto3";

package order.v1;

service OrderService {
  rpc GetOrder (GetOrderRequest) returns (Order);
  rpc CreateOrder (CreateOrderRequest) returns (Order);
}

message GetOrderRequest {
  string order_id = 1;
}

message Order {
  string order_id = 1;
  string status = 2;
  repeated OrderItem items = 3;
}

message OrderItem {
  string sku = 1;
  int32 quantity = 2;
}
```

รันคำสั่ง `protoc` (หรือ `buf generate`) เพื่อสร้างโค้ด Go struct + gRPC client/server stub อัตโนมัติ (เชื่อมกับ [Volume 4 — Golang](../04-golang/README.md)) — ทีมฝั่ง client กับ server ทำงานจาก **contract เดียวกัน** ที่ compiler บังคับให้ตรงกัน ต่างจาก REST ที่ contract มักอยู่ในเอกสาร (OpenAPI/Swagger) ซึ่งไม่มีอะไรบังคับว่าโค้ดจริงจะตรงกับเอกสารเสมอไป — ความคลาดเคลื่อนระหว่างเอกสารกับ implementation จริงคือปัญหาที่ REST เจอบ่อย แต่ gRPC ป้องกันได้ตั้งแต่ compile time เพราะ field ที่หายไปหรือ type ผิดจะทำให้ build ไม่ผ่านทันที

field number (`= 1`, `= 2`, ...) ใน protobuf ไม่ใช่แค่ label แต่คือ**ตำแหน่งจริงในรูปแบบ binary** — ห้ามเปลี่ยนเลขนี้ของ field ที่มีอยู่แล้วเมื่อ deploy ไปแล้ว (จะทำให้อ่าน message เก่าผิด) การเพิ่ม field ใหม่ให้ backward compatible ต้องใช้เลขใหม่ที่ไม่ซ้ำเสมอ

### ทำไม Binary + HTTP/2 ถึงเร็วกว่า

gRPC ได้เปรียบด้าน performance จากสองมิติที่เสริมกัน:

1. **Protobuf เป็น binary format** — เทียบกับ JSON ที่เป็น text (ต้อง parse string, แปลง type เอง, มี overhead ของ key name ซ้ำๆ ทุก object) protobuf เข้ารหัสเป็น binary ที่กระชับกว่ามาก (ไม่ต้องส่งชื่อ field ซ้ำ ใช้แค่เลข field number) ทั้ง serialize/deserialize เร็วกว่า JSON parsing หลายเท่า
2. **HTTP/2 เป็น transport บังคับ** — REST ทั่วไปมักวิ่งบน HTTP/1.1 ส่วน gRPC ออกแบบมาบน **HTTP/2** ตั้งแต่ต้น (เชื่อมกับ [Volume 2 — HTTP](../02-network/03-http.md)) ได้ประโยชน์จาก multiplexing (หลาย RPC call พร้อมกันบน TCP connection เดียว ไม่ต้องเปิด connection ใหม่ทุกครั้ง) และ HPACK header compression โดยอัตโนมัติ

ผลรวมคือ payload เล็กลง, CPU ใช้ parse น้อยลง, และ connection overhead ต่ำกว่า — สำคัญมากเมื่อ service เรียกกันเองเป็นพันๆ ครั้งต่อวินาทีภายใน cluster เดียวกัน ที่ latency ระดับ millisecond สะสมกันมีผลจริง

ข้อแลกเปลี่ยน: payload แบบ binary **อ่านด้วยตาเปล่าไม่ได้** ต้องมี tool เฉพาะ (`grpcurl`, `grpcui`) หรือ `.proto` file ประกอบถึงจะ debug ได้ ต่างจาก REST/JSON ที่เปิด browser dev tools หรือรัน `curl` ธรรมดาก็อ่านออกทันที

### Streaming สี่แบบของ gRPC

gRPC รองรับรูปแบบการสื่อสารที่ REST/HTTP ทั่วไปทำได้ยากหรือทำไม่ได้เลย เพราะอาศัย stream ของ HTTP/2 ที่เปิดค้างไว้สื่อสารสองทางได้:

```
1) Unary                    2) Server Streaming
   Client ──req──► Server      Client ──req──►  Server
   Client ◄─res─── Server      Client ◄─res1─── Server
                                Client ◄─res2─── Server
                                Client ◄─res3─── Server

3) Client Streaming          4) Bidirectional Streaming
   Client ──req1──► Server      Client ──req1──► Server
   Client ──req2──► Server      Client ◄─res1─── Server
   Client ──req3──► Server      Client ──req2──► Server
   Client ◄──res─── Server      Client ◄─res2─── Server
                                   (สลับกันได้อิสระทั้งสองทาง)
```

| แบบ | Use Case ตัวอย่าง |
|---|---|
| **Unary** | Request/response ปกติแบบเดียวกับ REST เช่น `GetOrder(order_id)` คืนค่า order เดียว — ใช้บ่อยที่สุด |
| **Server Streaming** | Client ขอครั้งเดียว server ทยอยส่งผลลัพธ์กลับหลายชุด เช่น subscribe ราคาหุ้นที่อัปเดตต่อเนื่อง, stream log แบบ real-time, ส่งผลลัพธ์ query ขนาดใหญ่ทีละหน้าโดยไม่ต้องรอครบก่อนส่ง |
| **Client Streaming** | Client ทยอยส่งข้อมูลหลายชุด server รวบรวมแล้วตอบครั้งเดียวตอนจบ เช่น upload ไฟล์เป็น chunk, ส่ง sensor data สะสมแล้วให้ server สรุปผลตอนท้าย |
| **Bidirectional Streaming** | ทั้งสองฝั่งส่งข้อมูลอิสระต่อกันบน connection เดียว เช่น chat แบบ real-time, ระบบแปลภาษาสด (ส่งเสียงเข้าไปเรื่อยๆ รับคำแปลกลับมาเรื่อยๆ), การเจรจาต่อรอง protocol ที่ต้องมีการโต้ตอบหลายรอบ |

REST ทำ streaming แบบ server-push ได้บ้างผ่าน Server-Sent Events หรือ WebSocket แต่ต้องเพิ่ม protocol แยกต่างหาก ไม่ได้เป็นส่วนหนึ่งของ REST/HTTP semantics ปกติ ในขณะที่ gRPC มี streaming เป็น first-class citizen ของตัว protocol เอง

## 1.3 REST vs gRPC — เลือกใช้เมื่อไหร่

| ปัจจัย | REST | gRPC |
|---|---|---|
| ผู้บริโภคหลัก | Public API, third-party, browser โดยตรง | Internal service-to-service |
| รองรับ browser | รองรับเนทีฟ (`fetch`, `XMLHttpRequest`) | ต้องผ่าน gRPC-Web + proxy แปลง ไม่รองรับเนทีฟ |
| Human-debuggability | สูง — อ่าน JSON, `curl` ทดสอบได้ทันที | ต่ำ — ต้องมี `.proto` + tool เฉพาะ |
| Performance | พอใช้ — text-based, overhead สูงกว่า | สูง — binary + HTTP/2 multiplexing |
| Contract enforcement | อ่อน — เอกสาร (OpenAPI) แยกจากโค้ด อาจไม่ตรงกัน | เข้ม — `.proto` บังคับที่ compile time |
| Streaming | จำกัด (ต้องพึ่ง SSE/WebSocket แยก) | native 4 แบบ |
| Tooling/ecosystem | เก่าแก่ ครบเครื่องที่สุด (Postman, Swagger UI, ฯลฯ) | เติบโตเร็วแต่ยังน้อยกว่า โดยเฉพาะฝั่ง frontend web |

กฎง่ายๆ ที่ใช้ได้จริง: **public-facing API ที่ third-party หรือ browser ต้องเรียกตรง → REST** (ต้องการ interoperability และ debuggability สูงสุด) ส่วน **internal service-to-service ภายใน cluster เดียวกันที่ต้องการ performance และ contract ที่เข้มงวด → gRPC** ระบบขนาดใหญ่จำนวนมากใช้ทั้งคู่พร้อมกัน: gRPC ระหว่าง internal service, และมี API Gateway (เชื่อมกับ [Volume 10 — APISIX](../10-apisix/README.md)) แปลงเป็น REST/JSON ให้ client ภายนอกอีกที

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| REST resource design | มอง API เป็น resource (คำนาม) ไม่ใช่ action, ใช้ HTTP verb ให้ตรงความหมาย |
| Idempotency ของ verb | `GET`/`PUT`/`DELETE` idempotent, `POST` ไม่ใช่ — สำคัญต่อ retry safety |
| Status code discipline | แยก 4xx (client ผิด) กับ 5xx (server ผิด) ให้ถูก มีผลต่อ retry logic |
| API versioning | URI versioning นิยมสุด ต้องมี deprecation policy ชัดเจน |
| HATEOAS | แนวคิดสวยงามแต่ไม่ค่อยใช้จริง เพราะ client ส่วนใหญ่ hardcode logic อยู่แล้ว |
| gRPC + Protobuf | binary format + `.proto` เป็น contract ที่บังคับ compile-time |
| gRPC เร็วกว่าเพราะ | binary serialization + HTTP/2 multiplexing |
| Streaming 4 แบบ | unary, server-streaming, client-streaming, bidirectional — เลือกตาม use case |
| REST vs gRPC | REST สำหรับ public/browser-facing, gRPC สำหรับ internal service-to-service |

## คำถามทบทวน

1. ทำไม `POST` ถึงไม่ idempotent แต่ `PUT` idempotent ยกตัวอย่างสถานการณ์ retry ที่แสดงความต่างนี้ให้ชัดเจน
2. ทำไม client ควรตั้งกฎ retry เฉพาะ 5xx ไม่ใช่ 4xx? จะเกิดอะไรขึ้นถ้า server ส่ง `500` ทั้งที่จริงเป็น validation error ของ client?
3. field number ใน protobuf message สำคัญอย่างไร ทำไมถึงห้ามเปลี่ยนเลขของ field ที่ deploy ไปแล้ว?
4. อธิบายว่า HTTP/2 multiplexing ช่วยให้ gRPC เร็วกว่า REST บน HTTP/1.1 อย่างไร
5. ยกตัวอย่างสถานการณ์จริงที่เหมาะกับ server streaming และอธิบายว่าทำไม unary RPC ธรรมดาถึงทำแบบนั้นไม่ได้

---

ถัดไป: [บทที่ 2 — Service Discovery](02-service-discovery.md)
