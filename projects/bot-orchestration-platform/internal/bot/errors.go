package bot

// ConfigError คือ error ชนิดพิเศษที่ bot ควรคืนกลับเมื่อปัญหาเกิดจาก config ที่ user
// กรอกมาผิด/ขาด (เช่น ลืมใส่ "url") — ต่างจาก error อื่นๆ (network timeout, 5xx จาก
// ปลายทาง) ตรงที่ "ลองใหม่กี่ครั้งก็ได้ผลลัพธ์เดิม" เพราะ config เดิมไม่เปลี่ยน จึงต้องมี
// type แยกต่างหากให้ชั้นที่เรียก bot (Temporal Activity ใน internal/workflow/activity.go)
// แยกแยะได้ว่า error นี้ "ไม่ควร retry" ต่างจาก error ทั่วไปที่ยัง retry ได้ตามปกติ
type ConfigError struct {
	err error
}

// NewConfigError ห่อ error ธรรมดาให้กลายเป็น ConfigError — bot เรียกใช้ตอนตรวจพบว่า
// config ที่ user ส่งมาไม่ครบ/ไม่ถูกต้อง (ดูตัวอย่างใน internal/bots/httpstatus)
func NewConfigError(err error) error {
	return &ConfigError{err: err}
}

// Error ทำให้ ConfigError implement interface error มาตรฐานของ Go — คืนข้อความเดียว
// กับ error ที่ห่อไว้ข้างในทุกประการ ผู้เรียกที่ไม่สนใจแยกประเภท error จะเห็นข้อความ
// เหมือนเดิมไม่มีผิดสังเกต
func (e *ConfigError) Error() string {
	return e.err.Error()
}

// Unwrap ทำให้ errors.Is/errors.As มองทะลุผ่าน ConfigError ไปถึง error ตัวจริงข้างในได้
// (มาตรฐาน error wrapping ของ Go ตั้งแต่ 1.13) — และทำให้ errors.As(err, &configErr)
// จากภายนอกใช้แยกแยะว่า error นี้เป็น ConfigError หรือไม่ได้ด้วย
func (e *ConfigError) Unwrap() error {
	return e.err
}
