-- สร้าง database และ user `bop` บน PostgreSQL ตัวเดียวกับ Temporal (แยก database/user กัน)
--
-- volume ใหม่: compose ที่ root mount โฟลเดอร์นี้เป็น /docker-entrypoint-initdb.d ให้ image
-- postgres รันไฟล์นี้เองครั้งเดียวตอนสร้าง volume
-- volume เดิมที่มีข้อมูล Temporal อยู่แล้ว: init script ไม่รันซ้ำ ให้รันไฟล์นี้เองครั้งเดียว
-- (คำสั่งอยู่ใน README.md หัวข้อ "PostgreSQL ของ BOP")
--
-- **รหัสผ่าน `bop` เป็นค่า dev เท่านั้น ห้ามใช้ใน production** (เหมือน BOP_JWT_SECRET ใน compose)
-- schema ของตารางไม่ได้สร้างที่นี่ — bop-api รัน migration เองตอน start
CREATE USER bop WITH PASSWORD 'bop';
CREATE DATABASE bop OWNER bop;
