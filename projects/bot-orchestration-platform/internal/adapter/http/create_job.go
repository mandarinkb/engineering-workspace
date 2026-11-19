package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"bop/internal/domain/apperr"
)

// handleCreateJob รับ config เป็น JSON object (key-value เป็น string ล้วน ตรงกับ bot.Config)
// จาก URL path จะได้ชื่อ bot ที่ต้องการรัน แล้วสร้าง job ใหม่พร้อมสถานะ pending
// บันทึกลง repository ก่อน แล้วค่อยส่งเข้า worker pool ให้เริ่มทำงาน สุดท้ายตอบกลับ
// client ทันทีด้วยสถานะ 202 Accepted พร้อมข้อมูล job (ที่ยังเป็น pending อยู่ เพราะ
// job เพิ่งถูกส่งเข้า queue ยังไม่ได้เริ่มทำงานจริง — client ต้องไปเช็คสถานะทีหลังเองผ่าน
// GET /jobs/{id})
//
// timeout_seconds เป็น query parameter (ไม่ใช่ field ใน JSON body) เพราะ body ตรงนี้คือ
// bot.Config ล้วนๆ (flat map ที่ bot แต่ละตัวอ่านเองตาม ConfigSchema) มาตั้งแต่ต้น — การ
// ห่อ body ใหม่เป็น {"config": {...}, "timeout_seconds": N} จะทำลาย wire format เดิมที่
// dashboard (projects/bop-dashboard/) เรียกอยู่แล้ว ใช้ query param แทนเพื่อเพิ่มความ
// สามารถนี้แบบ "เสริม" (additive) โดยไม่กระทบ client เดิมที่ไม่ได้ส่งค่านี้มาเลย (ปล่อยเป็น
// 0 แล้ว worker.Pool จะ fallback ไปใช้ timeout กลางของระบบตามเดิมทุกประการ)
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	botName := r.PathValue("name") // ดึงค่าจาก {name} ใน path pattern

	var cfg map[string]string
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	timeoutSeconds := 0
	if raw := r.URL.Query().Get("timeout_seconds"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			http.Error(w, "timeout_seconds must be a positive integer", http.StatusBadRequest)
			return
		}
		timeoutSeconds = seconds
	}

	// usecase ห่อ error จาก executor (เช่น ไม่รู้จัก bot) เป็น apperr.ErrInvalid -> 400 ส่วน
	// error จาก repository เป็น error ธรรมดา -> 500
	j, err := s.jobs.Create(r.Context(), botName, cfg, timeoutSeconds)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, apperr.ErrInvalid) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, http.StatusAccepted, j)
}
