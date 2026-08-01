# บทที่ 1: OWASP Top 10

## 1.0 OWASP Top 10 คืออะไร

**OWASP (Open Worldwide Application Security Project) Top 10** คือรายการช่องโหว่ระดับ application ที่พบบ่อยและมีผลกระทบสูงที่สุด จัดทำและปรับปรุงเป็นระยะโดยชุมชนความปลอดภัยระดับโลก ใช้เป็น checklist มาตรฐานที่แทบทุกทีม security review และ pentest อ้างอิงถึง — ไม่ใช่กฎที่ครอบคลุมทุกช่องโหว่ที่เป็นไปได้ แต่เป็น "10 อันดับที่ถ้า engineer ไม่รู้จักเลย ระบบมีความเสี่ยงสูงมาก"

บทนี้ไล่ทีละหมวด: อธิบาย failure mode จริงว่าเกิดจากอะไร ตัวอย่างโค้ดที่มีช่องโหว่เทียบกับโค้ดที่แก้แล้ว (เท่าที่ทำได้อย่างเป็นรูปธรรมด้วย Go) และเชื่อมโยงกับสิ่งที่เรียนมาแล้วในเล่มก่อนหน้า

## 1.1 Broken Access Control

**Failure mode**: ระบบตรวจสอบ "ยืนยันตัวตนได้ว่าใคร" (authentication) ถูกต้อง แต่ลืมตรวจสอบ "คนนี้มีสิทธิ์ทำสิ่งนี้จริงไหม" (authorization) — เป็นช่องโหว่ที่ครองอันดับ 1 ในผลสำรวจ OWASP หลายรอบติดต่อกัน เพราะเกิดง่ายมากจากความประมาทเดียว: ลืมเช็ค ownership ของ resource ก่อนให้เข้าถึง

ตัวอย่างคลาสสิกคือ **IDOR (Insecure Direct Object Reference)** — endpoint รับ ID ของ resource มาจาก client ตรงๆ แล้วดึงข้อมูลโดยไม่เช็คว่า user ที่ login อยู่เป็นเจ้าของ resource นั้นจริงหรือไม่

```go
// ช่องโหว่: ไม่เช็คว่า order นี้เป็นของ user ที่ login อยู่จริงไหม
func GetOrder(w http.ResponseWriter, r *http.Request) {
    orderID := r.URL.Query().Get("id")
    order, _ := db.Query("SELECT * FROM orders WHERE id = $1", orderID)
    json.NewEncoder(w).Encode(order)
    // user A ที่ login อยู่ เปลี่ยน id=1001 เป็น id=1002 ก็ดู order ของ user B ได้ทันที
}
```

```go
// แก้แล้ว: เช็ค ownership เสมอ ไม่เชื่อ ID ที่ client ส่งมาเฉยๆ
func GetOrder(w http.ResponseWriter, r *http.Request) {
    orderID := r.URL.Query().Get("id")
    userID := auth.UserIDFromContext(r.Context()) // มาจาก token ที่ verify แล้ว ไม่ใช่จาก client

    var order Order
    err := db.QueryRow(
        "SELECT * FROM orders WHERE id = $1 AND user_id = $2",
        orderID, userID,
    ).Scan(&order)
    if err == sql.ErrNoRows {
        http.Error(w, "not found", http.StatusNotFound) // ไม่บอกว่า "403 forbidden" เพราะจะยืนยันว่า ID มีอยู่จริง
        return
    }
    json.NewEncoder(w).Encode(order)
}
```

หลักการสำคัญ: **ห้ามเชื่อค่าที่มาจาก client เพื่อกำหนดสิทธิ์เด็ดขาด** สิ่งที่ใช้กำหนดสิทธิ์ต้องมาจากค่าที่ server ยืนยันแล้วเท่านั้น (เช่น user ID ที่ decode มาจาก JWT ที่ verify ลายเซ็นแล้ว) ไม่ใช่ field ใน request body/query string ที่ client ส่งมาเอง

## 1.2 Cryptographic Failures

**Failure mode**: ข้อมูลที่ควรถูกเข้ารหัส (sensitive data) ถูกเก็บหรือส่งแบบ plaintext, ใช้ algorithm การเข้ารหัสที่อ่อนแอ/ล้าสมัย (MD5, SHA-1 สำหรับ password), หรือจัดการ key ผิดพลาด (hardcode key ในโค้ด, ใช้ key เดียวกันทุก environment)

ตัวอย่างที่พบบ่อยคือการเก็บ password แบบ hash ด้วย MD5/SHA-1 ตรงๆ ซึ่งเร็วเกินไป (ออกแบบมาเพื่อความเร็ว ไม่ใช่ความปลอดภัย) ทำให้ brute-force/rainbow table โจมตีได้ง่าย ควรใช้ algorithm ที่ออกแบบมาสำหรับ password โดยเฉพาะ (ช้าโดยตั้งใจ, มี salt ในตัว) เช่น `bcrypt`, `argon2`

```go
// ผิด: SHA-256 ตรงๆ ไม่มี salt, คำนวณเร็วมาก brute-force ได้ง่าย
hash := sha256.Sum256([]byte(password))

// ถูก: bcrypt มี salt ในตัว และตั้งใจให้คำนวณช้า (cost factor ปรับได้)
hashedBytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// ตรวจสอบตอน login
err := bcrypt.CompareHashAndPassword(hashedBytes, []byte(inputPassword))
```

เชื่อมกับ [Volume 2 — Network บทที่ 4](../02-network/04-tls-security.md): ข้อมูลระหว่างทาง (in-transit) ต้องเข้ารหัสด้วย TLS เสมอ — ระบบที่ยัง serve API ผ่าน HTTP ธรรมดา (ไม่ใช่ HTTPS) ในโลกจริงถือเป็น Cryptographic Failure โดยตรง เพราะข้อมูล sensitive (token, password, PII) เดินทางเป็น plaintext ให้ผู้ดักฟังอ่านได้

## 1.3 Injection

**Failure mode**: input จาก user ถูกนำไปต่อ (concatenate) เข้ากับ command/query โดยตรง ทำให้ผู้โจมตีแทรก syntax ของตัวเองเข้าไปเปลี่ยนความหมายของ command/query เดิมได้ — **SQL Injection** เป็นตัวอย่างที่รู้จักกันมากที่สุด แต่หลักการเดียวกันเกิดได้กับ command injection, LDAP injection, NoSQL injection ก็ได้เช่นกัน

```go
// ช่องโหว่: string concatenation ตรงๆ
query := fmt.Sprintf("SELECT * FROM users WHERE username = '%s'", username)
rows, _ := db.Query(query)
// ถ้า username = "admin' --" กลายเป็น
// SELECT * FROM users WHERE username = 'admin' --'  → ส่วนที่เหลือถูก comment ทิ้ง, bypass การเช็ค password ไปเลย
```

```go
// แก้แล้ว: parameterized query — driver แยก data ออกจาก SQL syntax ให้เสมอ
rows, _ := db.Query("SELECT * FROM users WHERE username = $1", username)
// ไม่ว่า username จะมี ' หรือ -- อยู่ในนั้นแค่ไหน ก็ถูกปฏิบัติเป็น "ค่าข้อมูล" เท่านั้น ไม่มีทางกลายเป็น SQL syntax ได้
```

[Volume 5 — PostgreSQL](../05-postgresql/README.md) สอนการเขียน SQL ที่ถูกต้องอยู่แล้ว — หลักการ parameterized query (ใช้ placeholder อย่าง `$1`, `?` แทนการต่อ string) คือ**การป้องกัน SQL Injection ที่มีประสิทธิภาพที่สุดและควรใช้เป็น default เสมอ** ไม่มีเหตุผลทางธุรกิจใดที่คุ้มค่าพอจะ concatenate string เข้ากับ SQL โดยตรง

Command injection ก็ใช้หลักการเดียวกัน: ห้ามส่ง input ของ user ไปต่อกับ shell command ตรงๆ (เช่น `exec.Command("sh", "-c", "convert " + userFilename)`) ให้ส่ง argument แยกเป็น list แทน (`exec.Command("convert", userFilename)`) เพื่อไม่ให้ shell ตีความ input เป็น syntax พิเศษ (`;`, `|`, `&&`)

## 1.4 Insecure Design

**Failure mode**: ต่างจากข้อ 1.1-1.3 ที่เป็นข้อผิดพลาดระดับ implementation, **Insecure Design** คือปัญหาที่ฝังอยู่ในสถาปัตยกรรม/business logic ตั้งแต่ตอนออกแบบ — ต่อให้ implement ไม่มีบั๊กเลย ระบบก็ยังไม่ปลอดภัยอยู่ดี เพราะ "flow" เองมีช่องโหว่

ตัวอย่าง: ระบบ "ลืมรหัสผ่าน" ที่ยอมให้ผู้ใช้ตั้งรหัสผ่านใหม่ได้ทันทีหลังตอบคำถามความปลอดภัย (security question) เช่น "นามสกุลเดิมของแม่คืออะไร" — ต่อให้ implement สมบูรณ์แบบ (ไม่มี SQL injection, ไม่มี bug) แต่ตัว design เองอ่อนแอ เพราะคำตอบของคำถามพวกนี้มักหาได้จาก social media การป้องกันคือต้อง**คิดเรื่อง threat model ตั้งแต่ตอนออกแบบ** (threat modeling) ไม่ใช่ไปแก้ทีหลังตอน code review เช่น ใช้ multi-factor แทน security question, จำกัดจำนวนครั้งที่ลองผิดได้ (rate limit), ส่ง link ที่หมดอายุเร็วไปยัง email/phone ที่ verify แล้วแทน

## 1.5 Security Misconfiguration

**Failure mode**: ระบบถูก deploy ด้วยค่า default ที่ไม่ปลอดภัย, เปิด feature/endpoint ที่ไม่จำเป็นทิ้งไว้, error message เปิดเผยรายละเอียดภายในมากเกินไป, หรือลืมปิด debug mode ใน production

ตัวอย่างที่พบบ่อย: framework บางตัวเปิด debug/stack trace endpoint ให้เห็น environment variable, database connection string ทั้งหมดเมื่อเกิด error — ถ้า config นี้หลุดไปอยู่ใน production ผู้โจมตีแค่ทำให้ endpoint error ก็เห็นข้อมูลลับได้ทันที การป้องกันคือแยก config ตาม environment ให้ชัดเจน (`DEBUG=false` ใน production เสมอ), ปิด endpoint ที่ใช้แค่ตอน dev (Swagger UI แบบเปิดสาธารณะ, `/actuator` endpoint ของ Spring Boot ที่ไม่ได้ป้องกัน), และตรวจสอบ default credential ของทุก component ที่ deploy (database, message queue, admin panel) ว่าถูกเปลี่ยนจาก default แล้ว

## 1.6 Vulnerable and Outdated Components

**Failure mode**: ระบบใช้ library/framework/base image เวอร์ชันที่มี CVE (Common Vulnerabilities and Exposures) ที่รู้จักแล้วและมี patch ออกมาแล้ว แต่ทีมไม่ได้ update — ปัญหานี้อันตรายเพราะ**ผู้โจมตีไม่ต้องหาช่องโหว่เอง** แค่ scan หา component เวอร์ชันเก่าที่มี CVE สาธารณะแล้วใช้ exploit ที่มีคนเผยแพร่ไว้แล้วโจมตีตรงๆ

การป้องกันเชิงระบบ (ไม่ใช่แค่ "อย่าลืม update"): ตั้ง dependency scanning เป็นส่วนหนึ่งของ pipeline (เชื่อมกับ [Volume 16 — CI/CD](../16-cicd/02-harbor-and-pipelines.md) ที่พูดเรื่อง image vulnerability scanning) ให้ build fail อัตโนมัติถ้าเจอ dependency ที่มี CVE severity สูง, ใช้ base image ที่เล็กและอัปเดตสม่ำเสมอ (distroless/alpine พร้อม rebuild เป็นระยะ แม้โค้ดแอปจะไม่เปลี่ยนก็ตาม เพราะ base image เองก็มี patch ใหม่ออกเรื่อยๆ)

## 1.7 Identification and Authentication Failures

**Failure mode**: กลไกยืนยันตัวตนอ่อนแอ — ยอมให้ตั้งรหัสผ่านง่ายเกินไป, ไม่มี rate limit ต่อการพยายาม login (brute force ได้ไม่จำกัด), session ID เดาง่ายหรือไม่หมดอายุ, หรือไม่มี multi-factor authentication สำหรับบัญชีที่สำคัญ

หัวข้อนี้เชื่อมโยงโดยตรงกับ [Volume 9 — Authentication](../09-authentication/README.md) ทั้งเล่ม — สิ่งที่เรียนมาเรื่อง session-based auth (cookie ต้องมี `HttpOnly`, `Secure`, `SameSite`), JWT (`exp` ต้องสั้นพอสมควร, ต้องมี refresh token pattern ที่ revoke ได้), และ OAuth2/OIDC (ใช้ Authorization Code + PKCE ไม่ใช่ Implicit flow ที่ deprecated แล้ว) ล้วนคือมาตรการป้องกันหมวดนี้โดยตรง ตัวอย่างที่จับต้องได้: endpoint login ที่ไม่มี rate limit เปิดช่องให้ผู้โจมตี brute-force รหัสผ่านได้ไม่จำกัดจำนวนครั้ง ควรจำกัดจำนวนครั้งที่ผิดต่อ IP/account ภายในช่วงเวลาหนึ่ง (เช่น ผิด 5 ครั้งใน 15 นาที ให้ lock ชั่วคราว)

## 1.8 Software and Data Integrity Failures

**Failure mode**: ระบบ trust โค้ด/data/update ที่มาจากแหล่งที่ยืนยันความถูกต้อง (integrity) ไม่ได้ — เช่น ดึง dependency จาก package registry สาธารณะโดยไม่ตรวจสอบ checksum/signature, ใช้ CI/CD pipeline ที่ยอมรับ plugin จากแหล่งไม่น่าเชื่อถือ, หรือ deserialize ข้อมูลจากแหล่งที่ควบคุมไม่ได้โดยไม่ตรวจสอบก่อน (insecure deserialization นำไปสู่ remote code execution ได้ในบาง library)

ตัวอย่างที่โด่งดังระดับอุตสาหกรรมคือกรณี supply-chain attack ที่ผู้โจมตีแฮ็ค package ที่นิยมใช้กันมากบน public registry แล้วแทรกโค้ดอันตรายเข้าไปในเวอร์ชันใหม่ — โปรเจกต์ที่ pin เวอร์ชันแบบ auto-update (`^1.2.0` แทนที่จะ pin เป๊ะ) จะดึงโค้ดอันตรายเข้ามาโดยอัตโนมัติในรอบ build ถัดไป การป้องกัน: pin เวอร์ชัน dependency ให้แน่นอน (lock file), ตรวจสอบ signature ของ image ก่อน deploy (เชื่อมกับแนวคิด image signing ใน [Volume 16 — CI/CD บทที่ 2](../16-cicd/02-harbor-and-pipelines.md))

## 1.9 Security Logging and Monitoring Failures

**Failure mode**: ระบบถูกโจมตีสำเร็จ แต่ไม่มีใครรู้ตัวเป็นเวลานาน (บางเคสในโลกจริงพบหลังผ่านไปหลายเดือน) เพราะไม่มี log ของ security event ที่สำคัญ (login ผิดพลาดซ้ำๆ, การเปลี่ยนสิทธิ์ของ user, การเข้าถึง resource ที่ผิดปกติ) หรือมี log แต่ไม่มีใคร monitor/alert เมื่อ pattern ผิดปกติเกิดขึ้น

หัวข้อนี้คือเหตุผลที่ [Volume 14 — Observability](../14-observability/README.md) ทั้งเล่มสำคัญต่อ security ไม่ใช่แค่ต่อ reliability — log ที่ structured (บทที่ 2 ของเล่มนั้น) พร้อม field อย่าง `user_id`, `ip`, `action` ทำให้ query หา pattern ผิดปกติได้ (เช่น login ผิด 50 ครั้งจาก IP เดียวใน 1 นาที) และ Alertmanager/Grafana Alerting (บทที่ 1) ตั้งเงื่อนไขแจ้งเตือนทีม security ได้ทันทีที่ pattern แบบนี้เกิดขึ้น แทนที่จะรู้ตัวหลังความเสียหายเกิดไปแล้ว

## 1.10 Server-Side Request Forgery (SSRF)

**Failure mode**: แอปพลิเคชันมี feature ที่ทำ HTTP request ไปยัง URL ที่ user เป็นคนกำหนด (เช่น "ดึงรูปจาก URL", "webhook callback", "ตรวจสอบ URL ของเว็บไซต์") โดยไม่ตรวจสอบว่า URL นั้นชี้ไปที่ resource ภายใน (internal) ที่ไม่ควรให้ user เข้าถึงได้ผ่านทาง server — ผู้โจมตีใช้ช่องนี้ให้ server เป็น "ตัวแทน" ยิง request เข้าไปใน network ภายในที่ตัวเองเข้าถึงตรงๆ ไม่ได้ (เช่น cloud metadata endpoint ที่มักอยู่ที่ `169.254.169.254` ซึ่งเก็บ credential ของ instance, หรือ admin panel ภายในที่ไม่ได้เปิดสู่ภายนอก)

การเข้าใจ SSRF ต้องอาศัยความเข้าใจพื้นฐานเรื่อง HTTP request จาก [Volume 2 — Network](../02-network/README.md) — SSRF คือการที่ **server เองกลายเป็นผู้ส่ง request แทนผู้โจมตี** ดังนั้น request นั้นจึงมาจาก IP ภายในเครือข่ายที่ server อยู่ (ผ่าน firewall ที่บล็อกเฉพาะ IP ภายนอกไปแล้ว) ตัวอย่างช่องโหว่:

```go
// ช่องโหว่: ยอมให้ user ระบุ URL อะไรก็ได้ให้ server ไปดึงมา
func FetchImage(w http.ResponseWriter, r *http.Request) {
    url := r.URL.Query().Get("url")
    resp, _ := http.Get(url) // ผู้โจมตีส่ง url=http://169.254.169.254/latest/meta-data/iam/security-credentials/
    io.Copy(w, resp.Body)     // server ไปดึง credential ภายในมาให้ผู้โจมตีอ่านโดยไม่รู้ตัว
}
```

การป้องกัน: ทำ **allowlist** ของ domain/IP ที่อนุญาตให้ดึงได้อย่างชัดเจน (ไม่ใช่ blocklist ซึ่งหลีกเลี่ยงได้ง่ายด้วย redirect หรือ DNS trick), ปฏิเสธ IP ที่อยู่ในช่วง private/loopback/link-local (`10.0.0.0/8`, `127.0.0.0/8`, `169.254.0.0/16`) เสมอไม่ว่า URL จะมาจาก user คนไหน, และแยก network segment ที่ service ที่ต้องรับ URL จาก user อยู่ ออกจาก network ภายในที่ sensitive (เชื่อมกับแนวคิด Network Policy ใน [บทที่ 2](02-vault-secrets-network-policy.md) ของเล่มนี้)

## สรุปย่อ

| หมวด | Failure mode หลัก | แนวทางป้องกันหลัก |
|---|---|---|
| Broken Access Control | ตรวจ authentication แต่ลืม authorization/ownership | เช็ค ownership จากค่าที่ server verify แล้วเสมอ ไม่เชื่อ ID จาก client |
| Cryptographic Failures | เก็บ/ส่งข้อมูล sensitive แบบ plaintext หรือ algorithm อ่อนแอ | TLS ทุก endpoint, bcrypt/argon2 สำหรับ password |
| Injection | ต่อ input ของ user เข้ากับ command/query ตรงๆ | parameterized query เสมอ, แยก argument เมื่อเรียก shell |
| Insecure Design | ช่องโหว่ฝังอยู่ใน business logic/flow | threat modeling ตั้งแต่ตอนออกแบบ |
| Security Misconfiguration | ค่า default ไม่ปลอดภัย, debug mode เปิดใน production | แยก config ตาม environment, ปิด endpoint ที่ไม่จำเป็น |
| Vulnerable Components | ใช้ dependency เก่าที่มี CVE | dependency/image scanning อัตโนมัติใน pipeline |
| Auth Failures | brute-force ได้ไม่จำกัด, session/token จัดการไม่ดี | rate limit, MFA, short-lived token (Volume 9) |
| Integrity Failures | trust โค้ด/data จากแหล่งที่ยืนยันไม่ได้ | pin เวอร์ชัน, ตรวจ signature ก่อน deploy |
| Logging/Monitoring Failures | ไม่มี log/alert ของ security event | structured log + alert (Volume 14) |
| SSRF | server ยิง request ตาม URL ที่ user กำหนดโดยไม่กรอง | allowlist domain, บล็อก IP range ภายในเสมอ |

## คำถามทบทวน

1. อธิบายว่าทำไม IDOR ถึงเป็นตัวอย่างของ Broken Access Control และวิธีแก้ที่ถูกต้องต้องอาศัยค่าจากไหน ไม่ใช่จากไหน
2. ทำไม SHA-256 ตรงๆ ถึงไม่เหมาะกับการ hash password ทั้งที่เป็น algorithm ที่ปลอดภัยสำหรับงานอื่น?
3. parameterized query ป้องกัน SQL Injection ได้อย่างไรในทางเทคนิค (driver ทำอะไรต่างจากการ concatenate string)?
4. อธิบายความแตกต่างระหว่าง Insecure Design กับช่องโหว่ที่เป็นข้อผิดพลาดระดับ implementation (เช่น Injection) ด้วยตัวอย่างของตัวเอง
5. ทำไม SSRF ถึงอันตรายกว่าที่คิด แม้ server จะไม่ได้ "รัน" โค้ดอันตรายใดๆ เลยก็ตาม?

---

ถัดไป: [บทที่ 2 — Vault, Secrets, และ Network Policy](02-vault-secrets-network-policy.md)
