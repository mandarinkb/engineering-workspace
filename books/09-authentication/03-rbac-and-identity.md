# บทที่ 3: RBAC และ User Identity vs Service Identity

## 3.1 RBAC (Role-Based Access Control)

ทุกระบบที่ผ่าน Authentication มาแล้ว (รู้ว่า "คุณคือใคร" จากบทที่ 1-2) ยังต้องตอบคำถามถัดไปเสมอ: "**คุณทำอะไรได้บ้าง**" — นี่คือ Authorization ซึ่งโมเดลที่ใช้แพร่หลายที่สุดคือ **RBAC**

แนวคิดหลัก: แทนที่จะผูกสิทธิ์ (permission) เข้ากับ user แต่ละคนโดยตรง (ซึ่งจัดการยากมากเมื่อ user มีเป็นพัน) RBAC แทรกชั้นกลางที่เรียกว่า **role** เข้ามา:

```
User ──belongs to──► Role ──grants──► Permission

Somchai   ─────►   editor   ─────►   create_post, edit_post
Malee     ─────►   viewer   ─────►   read_post
Admin_1   ─────►   admin    ─────►   create_post, edit_post, delete_post, manage_users
```

### ตัวอย่างรูปธรรม: ระบบจัดการบทความ

| Role | create_post | edit_post | delete_post | manage_users |
|---|---|---|---|---|
| `viewer` | ✗ | ✗ | ✗ | ✗ |
| `editor` | ✓ | ✓ | ✗ | ✗ |
| `admin` | ✓ | ✓ | ✓ | ✓ |

การเช็คสิทธิ์ในโค้ดกลายเป็นคำถามสองชั้น: "user นี้มี role อะไร" → "role นั้นมี permission ที่ action นี้ต้องการหรือไม่" ข้อดีคือเวลาต้องเปลี่ยนสิทธิ์ของ "editor ทุกคนในระบบ" แก้ที่ role เดียว ไม่ต้องไล่แก้ทีละ user — และ user คนหนึ่งสามารถมีได้หลาย role พร้อมกัน (permission รวมกัน)

## 3.2 RBAC vs ABAC

RBAC เหมาะกับกรณีที่สิทธิ์แบ่งตาม "ประเภทของคน" ได้ชัดเจน แต่มีข้อจำกัดเมื่อกฎซับซ้อนขึ้น เช่น "editor แก้ไขโพสต์ของตัวเองได้เสมอ แต่แก้โพสต์ของคนอื่นได้เฉพาะถ้าอยู่ทีมเดียวกันและอยู่ในเวลาทำการ" — เงื่อนไขแบบนี้ผูกกับ **attribute** หลายตัว (เจ้าของ resource, เวลา, ทีม) ไม่ใช่แค่ role เดียว

**ABAC (Attribute-Based Access Control)** ตัดสินใจ authorization จาก attribute ของ user + resource + environment ประกอบกัน (เช่น policy engine อย่าง OPA/Rego) ยืดหยุ่นกว่ามากแต่ก็ซับซ้อนกว่าในการออกแบบ debug และคาดเดาผลลัพธ์ ในทางปฏิบัติหลายระบบใช้ RBAC เป็นชั้นหลัก (หยาบ กว้าง เข้าใจง่าย) แล้วเสริม ABAC เฉพาะจุดที่กฎซับซ้อนจริงๆ แทนที่จะเลือกสุดทางใดทางหนึ่ง

## 3.3 RBAC ตัวอย่างจริง: Kubernetes RBAC

Kubernetes ใช้ RBAC เป็นกลไก authorization หลักในการควบคุมว่าใคร (user หรือ service account) เรียก Kubernetes API ทำอะไรกับ object ไหนได้บ้าง (เชื่อมกับ [Volume 8 — Kubernetes](../08-kubernetes/README.md)) โครงสร้างตรงกับโมเดลในหัวข้อ 3.1 เป๊ะๆ เพียงแค่เปลี่ยนชื่อ:

- **Role** — ก้อน permission (เรียกว่า `rules` ระบุ `verbs` เช่น get/list/create/delete บน `resources` เช่น pods/deployments) ขอบเขตแค่ 1 namespace (หรือ `ClusterRole` ถ้าครอบคลุมทั้งคลัสเตอร์)
- **RoleBinding** — คือความสัมพันธ์ "User/Role ↔ ตัวตน" (subject) ตรงกับลูกศร "belongs to" ในหัวข้อ 3.1

```yaml
# Role — จำกัดว่า "ทำอะไรได้บ้าง" ใน namespace "team-a"
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: team-a
  name: pod-reader
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
---
# RoleBinding — ผูก Role เข้ากับตัวตนจริง (user หรือ service account)
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: team-a
  name: read-pods-binding
subjects:
  - kind: User
    name: somchai@example.com
    apiGroup: rbac.authorization.k8s.io
  - kind: ServiceAccount
    name: ci-deployer
    namespace: team-a
roleRef:
  kind: Role
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

สังเกตว่า `subjects` รับได้ทั้ง `User` (คน) และ `ServiceAccount` (service) ผูกเข้ากับ Role เดียวกันได้ — นี่คือจุดเชื่อมโยงตรงไปยังหัวข้อถัดไป: ระบบต้องรองรับทั้งสองประเภทตัวตนพร้อมกัน แต่กลไกที่ใช้ "พิสูจน์ว่าเป็นใคร" ก่อนจะมาถึงขั้น RBAC นี้ ต่างกันโดยสิ้นเชิงระหว่างคนกับ service

## 3.4 User Identity vs Service Identity: ทำไมต้องแยก

จนถึงตอนนี้บททั้งหมดพูดถึง "การพิสูจน์ตัวตน" แบบรวมๆ แต่ในระบบจริง มีตัวตนสองประเภทที่ต้องออกแบบกลไกยืนยันแยกกันอย่างชัดเจน เพราะบริบทการใช้งานต่างกันโดยพื้นฐาน:

### User Identity — ยืนยันตัวตนคน

- มี **UX** เข้ามาเกี่ยวข้องเสมอ (หน้า login, ปุ่มกด, การพิมพ์ password)
- ต้องรองรับ **MFA** (multi-factor authentication — OTP, authenticator app) เพราะมนุษย์ทำ password รั่ว/ใช้ซ้ำได้
- Session มีอายุจำกัดตาม "การใช้งานจริง" ของคน (เช่น auto-logout หลังไม่มี activity 30 นาที)
- ต้องมีกระบวนการ "ลืม password", "กู้บัญชี" ที่ involve การยืนยันตัวตนแบบมนุษย์ (อีเมล, เบอร์โทร)

### Service Identity — ยืนยันตัวตน service ต่อ service

- **ไม่มี UX** เลย — ไม่มีใครนั่งกรอกฟอร์ม login ให้ cron job หรือ microservice ที่รันอัตโนมัติ
- ต้องการความเร็วสูง ทำงานอัตโนมัติเต็มรูปแบบ เช่น **API key**, **service account** (แบบที่เห็นใน RoleBinding ข้างต้น), หรือ **mTLS** (ดู[บทที่ 4](04-mtls.md)) — ไม่มี MFA เพราะไม่มีมนุษย์อยู่ตรงกลาง flow
- อายุ credential มักยาวกว่า (หรือ rotate อัตโนมัติผ่านระบบ ไม่ใช่ user จำ/พิมพ์เอง) แต่ revoke ต้องทำได้เร็วเช่นกันถ้า service ถูกโจมตี
- ผูกกับ **Client Credentials grant** ใน OAuth2 ([บทที่ 2 หัวข้อ 2.4](02-oauth2-and-oidc.md#24-grant-type-client-credentials)) ที่ไม่มี resource owner (คน) อยู่ในสมการเลย

### ทำไมออกแบบปนกันไม่ได้

ถ้าเอากลไกของฝั่งหนึ่งไปบังคับใช้กับอีกฝั่ง ปัญหาจะเกิดทันที: บังคับให้ cron job ต้อง "login ผ่านหน้าเว็บ + กรอก OTP" ทุกครั้งที่รัน เป็นไปไม่ได้ในทางปฏิบัติ (ไม่มีคนกดยืนยัน) — กลับกัน ถ้าให้มนุษย์ถือ static API key แบบเดียวกับที่ service ใช้ (ไม่มี MFA, ไม่มี session timeout) ความเสี่ยงจะสูงมากถ้า key หลุด เพราะไม่มีกลไกอย่าง "ตรวจจับพฤติกรรมผิดปกติของมนุษย์คนนี้" มาช่วยจำกัดความเสียหาย

## 3.5 ตารางเปรียบเทียบ

| ประเด็น | User Identity | Service Identity |
|---|---|---|
| ใครคือเจ้าของ | คนจริง | service/process/job |
| กลไกยืนยัน | password + MFA, OAuth2/OIDC login | API key, service account, mTLS ([บทที่ 4](04-mtls.md)) |
| UX | จำเป็น (หน้า login, consent) | ไม่มี ต้อง automate เต็มรูปแบบ |
| อายุ credential | สั้น/กลาง ผูกกับ session การใช้งาน | ยาวกว่า แต่ควร rotate อัตโนมัติ |
| Grant type ที่ใช้ (ถ้าเป็น OAuth2) | Authorization Code (+ PKCE) | Client Credentials |
| Authorization ที่ตามมา | RBAC ผ่าน `User` subject | RBAC ผ่าน `ServiceAccount` subject |

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| RBAC | user → role → permission, แก้สิทธิ์ที่ role เดียว ไม่ต้องไล่ทีละ user |
| RBAC vs ABAC | RBAC ง่าย/หยาบ, ABAC ยืดหยุ่นกว่าด้วย attribute แต่ซับซ้อนกว่า |
| Kubernetes RBAC | Role (permission) + RoleBinding (ผูกกับ User หรือ ServiceAccount) |
| User vs Service Identity | ต่างกันที่มี/ไม่มี UX, MFA เข้ามาเกี่ยวข้อง — ออกแบบปนกันไม่ได้ |

## คำถามทบทวน

1. อธิบายว่าทำไม RBAC ถึงลดงานดูแลสิทธิ์เมื่อเทียบกับการผูก permission เข้ากับ user โดยตรง
2. ยกตัวอย่างกฎ authorization ที่ RBAC อย่างเดียวทำไม่ได้ ต้องใช้ ABAC ช่วย
3. ใน Kubernetes RoleBinding หนึ่งตัวสามารถผูก Role เข้ากับทั้ง `User` และ `ServiceAccount` พร้อมกันได้ สะท้อนอะไรเกี่ยวกับความสัมพันธ์ระหว่าง RBAC กับ identity สองประเภท?
4. ทำไม MFA ถึงเหมาะกับ User Identity แต่ไม่เหมาะ (หรือเป็นไปไม่ได้เลย) กับ Service Identity?
5. ถ้าระบบให้มนุษย์ใช้ static API key แบบเดียวกับที่ service ใช้กัน มีความเสี่ยงอะไรเพิ่มขึ้นเมื่อเทียบกับ login แบบมี MFA?

---

ก่อนหน้า: [บทที่ 2 — OAuth2 และ OIDC](02-oauth2-and-oidc.md) | ถัดไป: [บทที่ 4 — mTLS](04-mtls.md)
