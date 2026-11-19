package http

import (
	"net/http"

	"bop/internal/domain/bot"
)

// botSummary คือรูปแบบ JSON ที่คืนจาก GET /bots — ห่อ bot.Bot (interface, encode
// ตรงๆ ไม่ได้เพราะไม่รู้จัก concrete type) ให้เหลือแค่สิ่งที่ dashboard ต้องใช้จริง:
// ชื่อ (ไปใช้เป็น {name} ใน POST /bots/{name}/jobs) กับ config schema (ไปสร้าง
// dynamic form ตาม roadmap ข้อ 1.3)
type botSummary struct {
	Name         string            `json:"name"`
	ConfigSchema []bot.ConfigField `json:"config_schema"`
}

// handleListBots คืนรายชื่อ bot ที่ register ไว้ทั้งหมดพร้อม config schema ของแต่ละตัว —
// เรียงตามชื่อ (a-z) ตาม job.Executor.List
func (s *Server) handleListBots(w http.ResponseWriter, r *http.Request) {
	bots := s.jobs.ListBots()
	out := make([]botSummary, 0, len(bots))
	for _, b := range bots {
		out = append(out, botSummary{Name: b.Name(), ConfigSchema: b.ConfigSchema()})
	}
	writeJSON(w, http.StatusOK, out)
}
