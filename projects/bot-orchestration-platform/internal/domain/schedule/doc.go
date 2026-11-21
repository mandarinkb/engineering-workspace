// Package schedule คือ domain ของ Schedule: Spec (เมื่อไหร่ควรรัน), Policy, Detail และ
// port Scheduler ที่ usecase ใช้
//
// Temporal Schedule เป็นแหล่งข้อมูลจริงเพียงที่เดียวของ Schedule (ADR 0001) — ทั้งตาราง
// เวลา สถานะ pause และ steps ของ Workflow (อยู่ใน action args) BOP จึงไม่มี repository
// ของ Schedule เอง ถ้าวันหนึ่งต้องเก็บข้อมูลที่ Temporal เก็บไม่ได้ ให้ทำเป็นตารางเสริมที่
// อ้างด้วย schedule ID ไม่ใช่ย้าย steps กลับมาเก็บซ้ำ
package schedule
