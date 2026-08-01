# บทที่ 2: Vault, Secrets, และ Network Policy

บทที่แล้วพูดถึงช่องโหว่ระดับโค้ด บทนี้ขยับขึ้นมาที่ระดับ**การจัดการความลับและการควบคุมการสื่อสาร** — สอง pillar สำคัญของการทำให้ระบบปลอดภัยในระดับ infrastructure: **Secret management** (Vault) และ **Network segmentation** (Kubernetes NetworkPolicy)

## 2.1 ปัญหาของ Static Secret

ก่อนพูดถึง Vault ต้องเข้าใจปัญหาของวิธีจัดการ secret แบบดั้งเดิมก่อน: **static credential** — username/password หรือ API key ที่ตั้งค่าไว้ครั้งเดียวแล้วใช้ตลอดไปจนกว่าจะมีคนตั้งใจไปเปลี่ยน

ปัญหาของ static secret ไม่ใช่แค่ "อาจหลุด" แต่คือ**ผลกระทบเมื่อหลุดแล้ว**:

- ถ้า credential นี้หลุด (commit ลง git โดยไม่ตั้งใจ, หลุดจาก log, หลุดจาก config ที่ backup ไม่ปลอดภัย) มันจะ**ใช้งานได้ต่อไปเรื่อยๆ จนกว่าจะมีคนรู้ตัวและไปหมุน (rotate) มันด้วยมือ** — ในทางปฏิบัติ บ่อยครั้งไม่มีใครรู้ตัวเลยด้วยซ้ำว่าหลุดไปแล้ว
- Credential เดียวมักถูกใช้ร่วมกันหลาย service/หลายคน ทำให้ตรวจสอบ (audit) ไม่ได้ว่า "ใครเป็นคนใช้ credential นี้ทำอะไรเมื่อไหร่" เพราะทุกคน/ทุก service log เข้าด้วย identity เดียวกัน

## 2.2 HashiCorp Vault — Secret Engine

**Vault** คือระบบจัดการ secret แบบรวมศูนย์ ที่ออกแบบมาแก้ปัญหาข้างต้นโดยตรง ผ่านแนวคิด **secret engine** หลายแบบ

### KV Secret Engine — Static Secret ที่จัดการดีขึ้น

**KV (Key-Value) engine** เก็บ secret แบบ static เหมือนเดิม (เช่น API key ของบริการภายนอกที่ควบคุมรูปแบบ credential เองไม่ได้) แต่ได้ประโยชน์เพิ่มจาก Vault: **versioning** (ย้อนดู secret เวอร์ชันก่อนหน้าได้), **access control แบบละเอียด** (policy กำหนดได้ว่า service ไหนอ่าน path ไหนได้บ้าง), และ **audit log** (ทุกครั้งที่มีการอ่าน secret ถูกบันทึกไว้ว่าใคร/เมื่อไหร่)

```bash
vault kv put secret/order-service/db password="s3cr3t"
vault kv get secret/order-service/db
```

### Database Secret Engine — Dynamic Secret

นี่คือจุดที่ Vault ต่างจากการเก็บ secret แบบเดิมอย่างมีนัยสำคัญ **Database secret engine** ไม่ได้เก็บ password ที่ตั้งไว้ล่วงหน้า แต่**สร้าง credential ใหม่ที่มีอายุสั้น (short-lived) ทุกครั้งที่มีคนขอ**:

```bash
vault read database/creds/order-service-readonly
# Key                Value
# ---                -----
# lease_id            database/creds/order-service-readonly/abc123
# lease_duration      1h
# password            A1b2C3-generated-random-password
# username            v-approle-order-svc-x7z9
```

Vault เชื่อมต่อไปสร้าง user ใหม่ในฐานข้อมูลจริงๆ (ผ่าน admin credential ที่ Vault เก็บไว้เองและไม่มีใครอื่นเห็น) ให้สิทธิ์ตามที่กำหนดไว้ล่วงหน้า (`order-service-readonly` = สิทธิ์อ่านอย่างเดียว) แล้วส่ง username/password ที่สร้างใหม่กลับมา พร้อม **lease** (อายุการใช้งาน — ในตัวอย่างนี้ 1 ชั่วโมง) เมื่อครบเวลา Vault จะไปลบ user นั้นออกจากฐานข้อมูลให้อัตโนมัติ

### ทำไม Dynamic Secret ถึงเป็นสถานะความปลอดภัยที่แข็งแรงกว่าจริงๆ

เทียบกันตรงๆ:

| | Static Secret | Dynamic Secret |
|---|---|---|
| ถ้าหลุด | ใช้งานได้ตลอดไปจนกว่าคนจะไปหมุนเอง (อาจเป็นเดือน/ปี) | หมดอายุเองภายในเวลาสั้นๆ (นาที/ชั่วโมง) ตาม lease |
| ขอบเขตสิทธิ์ | มักกว้างเกินจำเป็น (สร้างไว้ครั้งเดียวให้ครอบคลุมทุกกรณีใช้งาน) | จำกัดตาม role ที่ขอ (เช่น read-only แยกจาก read-write) |
| ตรวจสอบว่าใครใช้ | ยากมาก (credential เดียวใช้ร่วมกันหลายที่) | ทุก credential ผูกกับ lease และ identity ที่ขอ ตรวจสอบย้อนหลังได้ |
| การ rotate | ต้องทำเอง มักถูกลืม | เกิดขึ้นเองทุกครั้งที่ credential หมดอายุ ไม่ต้องมีคนจำ |

พูดสั้นๆ: **static secret ที่หลุดคือความเสียหายระยะยาวที่ควบคุมไม่ได้ ส่วน dynamic secret ที่หลุดคือความเสียหายที่ถูกจำกัดขอบเขตและเวลาไว้แล้วตั้งแต่ต้น** — นี่คือเหตุผลที่ dynamic secret ถือเป็นแนวทางที่แข็งแรงกว่าในเชิงหลักการความปลอดภัย ไม่ใช่แค่ "สะดวกกว่า"

## 2.3 Vault Authentication Method

ก่อน service จะขอ secret จาก Vault ได้ ต้องพิสูจน์ตัวตนกับ Vault ก่อนเสมอ (Vault เองก็ต้องรู้ว่า "ใครกำลังขอ" เพื่อกำหนดว่าให้เข้าถึง path ไหนได้บ้าง)

### AppRole — สำหรับ Service

**AppRole** ออกแบบมาสำหรับ machine-to-machine authentication โดยเฉพาะ ประกอบด้วยสองส่วน: **Role ID** (เหมือน username เปิดเผยได้ ฝังใน config ได้) และ **Secret ID** (เหมือน password ต้องเก็บลับ) — service ที่ต้องการ login ส่งทั้งสองค่านี้ไปแลก **Vault token** ที่มีอายุจำกัดกลับมา แล้วใช้ token นั้นขอ secret ต่อไป

### Kubernetes Auth Method

สำหรับ workload ที่รันใน Kubernetes อยู่แล้ว Vault มี **Kubernetes auth method** ที่สะดวกกว่า AppRole มาก — ใช้ **ServiceAccount token** ที่ Kubernetes ออกให้ pod โดยอัตโนมัติอยู่แล้ว (เชื่อมกับ [Volume 8 — Kubernetes บทที่ 2](../08-kubernetes/02-configuration.md) ที่อธิบายเรื่อง ConfigMap/Secret) เป็นหลักฐานยืนยันตัวตน — pod ไม่ต้องรู้จัก Role ID/Secret ID ใดๆ ล่วงหน้าเลย เพียงแค่ Vault ถูกตั้งค่าให้เชื่อถือ Kubernetes API server ของคลัสเตอร์นั้น แล้วกำหนด policy ว่า ServiceAccount ไหน (ผูกกับ namespace ไหน) เข้าถึง path secret อะไรได้บ้าง

```
Pod (ServiceAccount: order-service)
   │  1. ส่ง ServiceAccount token ไปให้ Vault
   ▼
Vault  ──► 2. ถาม Kubernetes API server ว่า token นี้ valid จริงไหม, เป็นของ SA ไหน
   │
   ▼  3. ถ้าถูกต้อง ออก Vault token ตาม policy ของ SA นั้น
Pod ใช้ Vault token ขอ secret (KV หรือ dynamic database credential) ต่อไป
```

### Vault-managed Secret vs Kubernetes Secret แบบดิบ

หัวข้อนี้สำคัญมากเพราะเป็นความเข้าใจผิดที่พบบ่อย: **Kubernetes Secret ไม่ได้ "เข้ารหัส" ข้อมูลโดย default** — ค่าที่เก็บใน Kubernetes Secret ถูกเก็บเป็น **base64** เท่านั้น ซึ่ง **base64 ไม่ใช่การเข้ารหัส** เป็นแค่การ encode ที่ decode กลับได้ทันทีโดยไม่ต้องมี key ใดๆ (`echo <base64> | base64 -d`) ใครก็ตามที่มีสิทธิ์อ่าน object `Secret` ผ่าน Kubernetes API (RBAC อนุญาต) หรือเข้าถึง etcd โดยตรงก็เห็นค่าจริงได้ทันที

| | Kubernetes Secret (ดิบ) | Vault |
|---|---|---|
| การเข้ารหัส at-rest | ต้องเปิด encryption at rest เองต่างหาก (ไม่ใช่ default) ค่าเริ่มต้นแค่ base64 | เข้ารหัสจริงเสมอ (มี key management ในตัว) |
| Dynamic secret | ไม่รองรับ — เป็น static ตายตัว | รองรับผ่าน secret engine |
| Audit | ไม่มีในตัว ต้องพึ่ง Kubernetes audit log ทั่วไป | มี audit log เฉพาะทางของทุกการเข้าถึง secret |
| Rotation | ต้องทำเอง | dynamic secret rotate อัตโนมัติตาม lease |
| Access control | ผ่าน Kubernetes RBAC เท่านั้น | policy ละเอียดกว่า แยกตาม path/secret engine ได้ |

แนวทางที่นิยมในทางปฏิบัติ: ใช้ **Vault เป็นแหล่งความจริง (source of truth) ของ secret** แล้วใช้เครื่องมืออย่าง Vault Agent Injector หรือ External Secrets Operator ดึงค่าจาก Vault มา sync เข้า Kubernetes Secret หรือ inject ตรงเข้า pod ผ่าน sidecar ในขณะรันไทม์ แทนที่จะเก็บค่า secret จริงไว้ใน Kubernetes Secret object ตรงๆ

## 2.4 หลักการจัดการ Secret ทั่วไป

ไม่ว่าจะใช้ Vault หรือไม่ หลักการพื้นฐานเหล่านี้ต้องยึดถือเสมอ:

1. **ห้าม commit secret ลง git เด็ดขาด** — แม้แต่ใน private repository เพราะ git history เก็บทุกอย่างไว้ถาวร ต่อให้ลบไฟล์ทิ้งใน commit ถัดไป secret ก็ยังอยู่ใน history เดิม ต้องใช้เครื่องมืออย่าง `git-secrets` หรือ pre-commit hook สแกนหา pattern ที่ดูเหมือน credential ก่อน commit เสมอ
2. **Rotate สม่ำเสมอ** — แม้ไม่มีหลักฐานว่าหลุด การหมุน secret เป็นระยะลดหน้าต่างความเสี่ยง (window of exposure) ถ้ามันหลุดไปแล้วโดยที่ไม่มีใครรู้ตัว
3. **Least privilege** — แต่ละ service ควรมีสิทธิ์เข้าถึง secret เท่าที่จำเป็นต่อการทำงานเท่านั้น ไม่ใช้ credential เดียวกันข้าม service เพื่อความสะดวก เพราะถ้า service หนึ่งถูกเจาะ ผลกระทบไม่ควรลามไปถึง secret ของ service อื่นที่ไม่เกี่ยวข้อง

## 2.5 Kubernetes NetworkPolicy

Secret management ป้องกันไม่ให้ credential หลุด แต่ถ้า pod หนึ่งถูกเจาะเข้ามาได้ (เช่นผ่านช่องโหว่ในโค้ดแอปเอง) คำถามถัดมาคือ **"pod ที่ถูกเจาะแล้วเข้าไปคุยกับ pod อื่นต่อได้ไกลแค่ไหน"** — นี่คือสิ่งที่ **NetworkPolicy** ควบคุม

โดย default ใน Kubernetes **pod ทุกตัวคุยกับ pod อื่นได้ทั้งหมดในคลัสเตอร์** (flat network) ไม่มีการจำกัดใดๆ — นี่คือปัญหาด้าน security เพราะถ้า pod หนึ่งถูกเจาะ ผู้โจมตีสามารถ scan และเข้าถึง service อื่นทั้งหมดในคลัสเตอร์ได้ทันที (lateral movement)

**NetworkPolicy** เป็น Kubernetes resource ที่กำหนดกฎการสื่อสารระหว่าง pod โดยใช้ **label selector** เป็นตัวระบุ pod ต้นทาง/ปลายทาง (ไม่ใช่ IP ตายตัว เพราะ pod ใน Kubernetes ถูกสร้าง/ทำลายและเปลี่ยน IP ตลอดเวลา)

### Default-deny แล้วค่อย Allow เฉพาะที่จำเป็น

แนวทางที่ปลอดภัยที่สุดคือเริ่มจาก **default-deny** (ปฏิเสธทุกการเชื่อมต่อ) แล้วค่อยเปิดเฉพาะ path ที่จำเป็นจริงๆ ทีละเส้น:

```yaml
# ขั้นที่ 1: ปฏิเสธ ingress ทั้งหมดใน namespace นี้เป็นค่าเริ่มต้น
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-ingress
  namespace: production
spec:
  podSelector: {}        # เลือกทุก pod ใน namespace
  policyTypes:
    - Ingress
---
# ขั้นที่ 2: เปิดเฉพาะทางที่จำเป็น — order-service รับ traffic ได้เฉพาะจาก api-gateway เท่านั้น
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-gateway-to-order-service
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: order-service
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: api-gateway
      ports:
        - protocol: TCP
          port: 8080
```

ผลลัพธ์: pod ที่มี label `app: order-service` จะรับ connection ได้เฉพาะจาก pod ที่มี label `app: api-gateway` บน port 8080 เท่านั้น — pod อื่นใดในคลัสเตอร์ (แม้จะอยู่ namespace เดียวกัน) เชื่อมต่อเข้ามาไม่ได้เลย ต่อให้ pod นั้นถูกเจาะแล้วก็ตาม

### NetworkPolicy กับ mTLS — คนละครึ่งของ Zero-Trust

[Volume 2 — Network บทที่ 4](../02-network/04-tls-security.md) อธิบายเรื่อง mTLS ไว้ว่าใช้เป็น service identity ยืนยันตัวตนระหว่าง service กับ service ในสถาปัตยกรรม zero-trust — NetworkPolicy กับ mTLS **แก้ปัญหาคนละครึ่งของ zero-trust** ที่ต้องใช้คู่กันถึงจะสมบูรณ์:

```
┌─────────────────────────────────────────────────────────────┐
│  Layer ที่ควบคุม "ใครเข้าถึงใครได้" (NetworkPolicy)             │
│  ─────────────────────────────────────────────────────────    │
│  Pod A (label: app=frontend)  ──✗ ปฏิเสธที่ network layer──►  │
│                                    Pod C (label: app=database)│
│  Pod B (label: app=order-svc) ──✓ อนุญาต────────────────────►│
│                                    Pod C (label: app=database)│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼ ถ้าผ่าน NetworkPolicy มาได้แล้ว
┌─────────────────────────────────────────────────────────────┐
│  Layer ที่ควบคุม "คุณคือใครจริงๆ" (mTLS)                        │
│  ─────────────────────────────────────────────────────────    │
│  Pod B เชื่อมต่อมาถึง Pod C ได้ (ผ่าน NetworkPolicy แล้ว)         │
│  แต่ Pod C ยังต้องตรวจ certificate ของ Pod B ก่อนคุยต่อ          │
│  ป้องกันกรณี pod แปลกปลอมสวมรอย label เดียวกันเข้ามา             │
└─────────────────────────────────────────────────────────────┘
```

- **NetworkPolicy ควบคุม "WHO CAN REACH WHOM"** — ที่ระดับ network เท่านั้น จำกัดว่า packet จาก pod ไหนถึงจะถูกส่งต่อไปยัง pod ปลายทางได้เลย แต่ **ไม่รู้และไม่สนใจว่าฝั่งที่เชื่อมต่อเข้ามาเป็นใครจริงๆ** รู้แค่ว่า source IP/label ตรงกับกฎที่อนุญาตไว้เท่านั้น — ถ้าผู้โจมตีหาทางปลอมตัวให้ pod ของตัวเองมี label ตรงกับที่อนุญาตได้ (เช่นเจาะช่องโหว่ที่ให้สร้าง pod ใน namespace นั้นได้) NetworkPolicy เพียงอย่างเดียวก็ป้องกันไม่ได้
- **mTLS พิสูจน์ "WHO YOU ARE" เมื่อเชื่อมต่อถึงกันแล้ว** — ด้วย certificate เฉพาะตัวที่ออกให้แต่ละ service ต่อให้ NetworkPolicy อนุญาตให้ connection ผ่านมาได้ ปลายทางยังต้องตรวจสอบ certificate ก่อนว่าฝั่งตรงข้ามเป็น service ที่ถูกต้องจริงหรือเป็นตัวปลอมที่แค่มี label ตรงกัน

ทั้งสองต้องใช้**ร่วมกัน**: NetworkPolicy ลดพื้นที่ผิวการโจมตี (attack surface) โดยจำกัด path ที่เป็นไปได้ตั้งแต่ระดับ network ก่อนที่จะไปถึงชั้น application เลย ส่วน mTLS รับประกันว่าแม้ connection จะผ่าน NetworkPolicy มาได้ ก็ยังมั่นใจได้ว่ากำลังคุยกับ service ตัวจริง ไม่ใช่ตัวปลอมที่แค่ label ตรง — นี่คือความหมายที่แท้จริงของ "zero trust ภายในคลัสเตอร์": ไม่เชื่อ traffic ใดๆ โดย default แม้จะมาจากภายใน network เดียวกัน ต้องพิสูจน์ทั้งสิทธิ์การเชื่อมต่อ (NetworkPolicy) และตัวตน (mTLS) เสมอ

## สรุปย่อ

| แนวคิด | ประเด็นสำคัญที่สุด |
|---|---|
| Static vs Dynamic Secret | Static ที่หลุดใช้ได้ตลอดไปจนถูก rotate เอง, Dynamic หมดอายุเองตาม lease และตรวจสอบได้ว่าใครขอ |
| Vault KV Engine | เก็บ static secret แต่ได้ versioning/audit/access control เพิ่มจาก Vault |
| Vault Database Engine | สร้าง credential ใหม่ทุกครั้งที่ขอ พร้อมอายุจำกัด (lease) |
| AppRole vs Kubernetes Auth | AppRole ใช้ Role ID + Secret ID, Kubernetes auth ใช้ ServiceAccount token ที่มีอยู่แล้ว |
| K8s Secret ดิบ vs Vault | K8s Secret แค่ base64 (ไม่ใช่การเข้ารหัส) ไม่มี dynamic/audit, Vault มีครบ |
| หลักจัดการ secret | ห้าม commit ลง git, rotate สม่ำเสมอ, least privilege |
| NetworkPolicy | default-deny แล้วเปิดเฉพาะ path ที่จำเป็น ผ่าน label selector |
| NetworkPolicy vs mTLS | NetworkPolicy คุม "ใครเข้าถึงใครได้" (network layer), mTLS พิสูจน์ "คุณคือใคร" (identity layer) — ต้องใช้คู่กัน |

## คำถามทบทวน

1. อธิบายว่าทำไม dynamic secret ถึงถือเป็นสถานะความปลอดภัยที่แข็งแรงกว่า static secret แม้จะยัง "หลุด" ได้เหมือนกัน
2. Vault Kubernetes auth method ใช้อะไรเป็นหลักฐานยืนยันตัวตนของ pod โดยไม่ต้องตั้งค่า credential ล่วงหน้า?
3. ทำไม base64 ใน Kubernetes Secret ถึงไม่ถือว่าเป็นการเข้ารหัส ต่างจาก encryption อย่างไร?
4. เขียนอธิบาย NetworkPolicy แบบ default-deny-then-allow ว่าทำไมถึงปลอดภัยกว่าการเริ่มจาก allow-all แล้วค่อยปิดทีหลัง
5. อธิบายว่าทำไม NetworkPolicy เพียงอย่างเดียว (ไม่มี mTLS) ยังไม่เพียงพอสำหรับ zero-trust ที่สมบูรณ์ ยกตัวอย่างสถานการณ์ที่ NetworkPolicy ป้องกันไม่ได้แต่ mTLS ป้องกันได้

---

ก่อนหน้า: [บทที่ 1 — OWASP Top 10](01-owasp-top-10.md) | กลับสู่ [สารบัญ Volume 15](README.md)
