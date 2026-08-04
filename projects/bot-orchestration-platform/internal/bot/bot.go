// Package bot กำหนด interface หลักที่ automation task (bot) ทุกตัวในระบบต้อง implement
// ตัว package นี้จงใจไม่รู้จัก HTTP, การเก็บข้อมูล (storage), หรือ logging backend ใดๆ เลย —
// สิ่งเหล่านั้นจะถูกส่งเข้ามาตอนเรียกใช้งาน (ผ่าน LogFunc) หรืออยู่แยกต่างหากไปเลยนอก package นี้
// (เช่น job.Repository, logger.Logger) เพื่อให้เวลาสร้าง bot ชนิดใหม่ ไม่ต้องแก้โค้ดของ
// worker pool หรือ API layer เลยแม้แต่บรรทัดเดียว — นี่คือหัวใจของการออกแบบแบบ "pluggable"
package bot

import "context"

// LogFunc คือ function type ที่ bot เรียกใช้เพื่อส่งข้อความ progress ออกมาระหว่างทำงาน
// (เช่น "กำลังโหลดหน้าเว็บ...", "พบข้อมูล 5 รายการ") โดยที่ตัว bot เองไม่ต้องรู้เลยว่า
// ข้อความนี้จะถูกเก็บที่ไหน (in-memory, PostgreSQL, หรือ OpenSearch) หรือถูกส่งไปแสดง
// บนหน้า dashboard แบบ real-time ได้อย่างไร — รายละเอียดพวกนั้นอยู่ที่ package internal/logger
// ทั้งหมด ทำให้ bot แต่ละตัวโฟกัสแค่ "ทำงานของตัวเองและรายงานความคืบหน้า" เท่านั้น
type LogFunc func(message string)

// Bot คือ interface ที่ automation task ทุกชนิด (web scraper, price checker, ตัวส่งแจ้งเตือน ฯลฯ)
// ต้อง implement ให้ครบทั้ง 3 method นี้ — worker pool และ API layer จะรู้จักแค่ interface นี้
// เท่านั้น ไม่เคยรู้จัก concrete type ของ bot จริงๆ เลย (นี่คือหลักการ "dependency inversion"
// ที่อธิบายไว้ใน books/04-golang/04-architecture-patterns.md)
type Bot interface {
	// Name คืนชื่อที่ใช้ระบุชนิดของ bot นี้ เช่น "http-status-checker" — ชื่อนี้ถูกใช้เป็น
	// key ตอน register bot เข้า worker pool (ดู worker.Pool.Register) และเป็นค่าที่ client
	// ต้องส่งมาตอนสร้าง job ใหม่ผ่าน API (เช่น POST /bots/http-status-checker/jobs)
	Name() string

	// ConfigSchema ประกาศว่า bot ตัวนี้ต้องการ config field อะไรบ้าง (key, label, type,
	// จำเป็นไหม) — เพิ่มเข้ามาเพื่อให้ dashboard (roadmap Phase 1.3) สร้าง dynamic form
	// ที่ field เปลี่ยนไปตามชนิดของ bot ที่เลือกได้ โดยที่ frontend ไม่ต้อง hard-code
	// รู้จัก bot แต่ละตัวเลยแม้แต่ตัวเดียว (เรียกผ่าน GET /bots) — ตรงกับหลักการ "pluggable"
	// ของทั้งไฟล์นี้: เพิ่ม bot ชนิดใหม่ ไม่ต้องแก้โค้ด backend/frontend ส่วนกลางเลย
	ConfigSchema() []ConfigField

	// Run คือ method หลักที่ประมวลผลงานจริงของ bot หนึ่ง job
	//   - ctx: ใช้เช็คว่าถูกสั่งยกเลิก (cancel) หรือหมดเวลา (timeout) หรือยัง — bot ที่ดีต้อง
	//     เช็ค ctx.Done() เป็นระยะๆ ระหว่างทำงาน (โดยเฉพาะงานที่ใช้เวลานาน) แล้วหยุดทำงาน
	//     ทันทีถ้า context ถูกยกเลิก ไม่ใช่ทำงานต่อไปเรื่อยๆ โดยไม่สนใจ
	//   - log: เรียกใช้เพื่อรายงานความคืบหน้าระหว่างทำงาน (ดูคำอธิบาย LogFunc ด้านบน)
	//   - cfg: ค่า config ที่ user กรอกมาตอนสร้าง job (เช่น URL ที่จะเช็ค)
	// คืนค่า Result ถ้าสำเร็จ หรือ error ถ้าล้มเหลว (worker pool จะบันทึกผลลัพธ์นี้ลง
	// job.Repository ให้อัตโนมัติ ตัว bot ไม่ต้องจัดการเรื่องการบันทึกเอง)
	Run(ctx context.Context, log LogFunc, cfg Config) (Result, error)
}

// ConfigFieldType คือชนิดของ input ที่ dashboard ควรใช้เรนเดอร์ field นี้ — เก็บเป็น string
// enum แบบเปิด (ไม่ใช่ closed type) เพราะ frontend เป็นคนตัดสินใจสุดท้ายว่าจะ map ค่าไหน
// ไปเป็น input component แบบไหน (เช่น "url" อาจ render เป็น <input type="url"> ที่มี
// validation ของเบราว์เซอร์เพิ่มเข้ามาให้ฟรีๆ)
type ConfigFieldType string

const (
	ConfigFieldText ConfigFieldType = "text" // ข้อความทั่วไป
	ConfigFieldURL  ConfigFieldType = "url"  // URL — dashboard ควร validate รูปแบบ URL ด้วย
)

// ConfigField คือ field เดียวใน config schema ของ bot หนึ่งตัว — ตรงกับ 1 key ใน
// bot.Config (map[string]string) ที่ bot ตัวนั้นคาดหวังว่าจะได้รับตอนถูกสั่งรัน
type ConfigField struct {
	Key      string          `json:"key"`      // ตรงกับ key ใน bot.Config ที่ bot ตัวนี้อ่าน (เช่น "url")
	Label    string          `json:"label"`    // ข้อความอ่านง่ายให้ dashboard แสดงเป็น label ของ input
	Type     ConfigFieldType `json:"type"`     // ชนิด input ที่ dashboard ควรใช้ (ดู ConfigFieldType)
	Required bool            `json:"required"` // true ถ้า user ต้องกรอกก่อนสั่งรัน bot นี้ได้
}

// Config คือค่า config ที่ user กำหนดตอนสร้าง job หนึ่งครั้ง เก็บเป็น map ธรรมดา (key-value
// เป็น string ทั้งคู่) แทนที่จะเป็น struct ที่มี field ตายตัว เพราะ bot แต่ละชนิดต้องการ
// config คนละแบบกัน (เช่น http-status-checker ต้องการแค่ "url" แต่ bot ตัวอื่นอาจต้องการ
// "selector", "webhook_url" ฯลฯ) — การใช้ map ทำให้เพิ่ม bot ชนิดใหม่ที่มี config รูปแบบ
// ต่างไปได้เลยโดยไม่ต้องแก้ type นี้หรือโค้ดส่วนกลางใดๆ (แลกกับ type safety ที่หายไปบ้าง
// ซึ่ง bot แต่ละตัวต้อง validate ค่าที่ตัวเองต้องการเอง เช่นดูตัวอย่างใน internal/bots/httpstatus)
type Config map[string]string

// Result คือผลลัพธ์ตอน bot ทำงานสำเร็จ
type Result struct {
	Summary string         // ข้อความสรุปสั้นๆ อ่านง่าย ใช้แสดงในหน้า job list ของ dashboard
	Data    map[string]any // ข้อมูลผลลัพธ์แบบละเอียด รูปแบบขึ้นอยู่กับแต่ละ bot (เช่น status code, ราคาที่เจอ)
}
