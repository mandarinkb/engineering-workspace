# Volume 9 — Authentication

การยืนยันตัวตนและสิทธิ์การเข้าถึง เป็นหัวข้อที่ผิดพลาดง่ายและมีผลกระทบสูงถ้าออกแบบไม่ดี ครอบคลุมทั้งฝั่ง user (คนใช้แอป) และฝั่ง service (service คุยกับ service)

## เป้าหมายการเรียนรู้

- อธิบายความแตกต่างระหว่าง Authentication กับ Authorization
- เลือกกลยุทธ์ session-based หรือ token-based (JWT) ให้เหมาะกับสถานการณ์
- อธิบาย OAuth2/OIDC flow ต่างๆ และใช้ให้ถูก flow (Authorization Code, Client Credentials ฯลฯ)
- ออกแบบ RBAC และแยกแนวคิด User Identity กับ Service Identity ได้

## สารบัญ

### [บทที่ 1 — Session และ JWT](01-session-and-jwt.md)

- [ ] Session-based auth ทำงานอย่างไร (session ID + server-side store)
- [ ] Session store: in-memory, Redis (เชื่อมกับ [Volume 6 — Redis](../06-redis/README.md))
- [ ] Cookie: `HttpOnly`, `Secure`, `SameSite`
- [ ] โครงสร้าง JWT (header.payload.signature), claim มาตรฐาน (`exp`, `iat`, `sub`)
- [ ] Signing algorithm: HMAC (symmetric) vs RSA/ECDSA (asymmetric)
- [ ] ข้อดี/ข้อเสียเทียบกับ session (stateless แต่ revoke ยาก), refresh token pattern

### [บทที่ 2 — OAuth2 และ OIDC](02-oauth2-and-oidc.md)

- [ ] Role: Resource Owner, Client, Authorization Server, Resource Server
- [ ] Grant type: Authorization Code (+ PKCE), Client Credentials, Refresh Token
- [ ] เมื่อไหร่ควรใช้ grant type ไหน
- [ ] OpenID Connect คือ layer เพิ่มบน OAuth2 สำหรับ authentication (ไม่ใช่แค่ authorization)
- [ ] ID Token vs Access Token
- [ ] UserInfo endpoint, discovery document

### [บทที่ 3 — RBAC และ User Identity vs Service Identity](03-rbac-and-identity.md)

- [ ] Role-Based Access Control — role, permission, การ map user → role → permission
- [ ] เทียบกับ ABAC (Attribute-Based Access Control) แบบคร่าวๆ
- [ ] ใช้ RBAC ใน Kubernetes เป็นตัวอย่างจริง (เชื่อมกับ [Volume 8](../08-kubernetes/README.md))
- [ ] **User Identity** — ยืนยันตัวตนคน (login, MFA)
- [ ] **Service Identity** — ยืนยันตัวตน service ต่อ service (API key, service account, mTLS)
- [ ] ทำไมสอง concept นี้ต้องแยกกันออกแบบ

### [บทที่ 4 — mTLS](04-mtls.md)

- [ ] Mutual TLS สำหรับยืนยันตัวตนทั้ง client และ server (ต่อจาก [Volume 2 — Network](../02-network/README.md))
- [ ] ใช้เป็น service identity ใน service mesh / zero-trust network

## Lab / Source Code ที่เกี่ยวข้อง

- [source-code/golang/](../../source-code/golang/README.md) — ตัวอย่าง JWT middleware

## แหล่งอ้างอิงแนะนำ

ดู [resources/](../../resources/README.md)
