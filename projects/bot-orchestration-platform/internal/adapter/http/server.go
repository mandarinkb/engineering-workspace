// Package http คือ control-plane API ของทั้งระบบ — หน้า dashboard (roadmap Phase 1)
// จะคุยกับ backend ผ่าน endpoint ที่นี่เท่านั้น ไม่มีทางอื่น package นี้เป็น adapter ชั้นนอก
// สุดของ clean architecture: แปลง HTTP request เป็นการเรียก usecase (internal/usecase/...)
// แล้วแปลงผลลัพธ์กลับเป็น JSON — ไม่มี business logic และไม่รู้จัก Temporal SDK,
// worker pool, หรือ storage ตัวจริงเลย
package http

import (
	"net/http"
	"os"

	authuc "bop/internal/usecase/auth"
	jobuc "bop/internal/usecase/job"
	scheduleuc "bop/internal/usecase/schedule"
	workflowrunuc "bop/internal/usecase/workflowrun"
)

// Server คือ HTTP API ของ BOP รวม handler ทุกตัวไว้ในที่เดียว
type Server struct {
	mux       *http.ServeMux // ตัว router มาตรฐานของ Go (ใช้ pattern matching แบบ method+path ของ Go 1.22+)
	jobs      *jobuc.Service
	runs      *workflowrunuc.Service
	schedules *scheduleuc.Service
	auth      *authuc.Service
}

// NewServer ผูก handler ทุกตัวเข้ากับ usecase ที่ส่งเข้ามา แล้วลงทะเบียน route ให้เรียบร้อย
func NewServer(jobs *jobuc.Service, runs *workflowrunuc.Service, schedules *scheduleuc.Service, auth *authuc.Service) *Server {
	s := &Server{mux: http.NewServeMux(), jobs: jobs, runs: runs, schedules: schedules, auth: auth}
	s.routes()
	return s
}

// dashboardOrigin คือ origin ของ Next.js dev server (projects/bop-dashboard/) — ต้องเปิด
// CORS ให้ origin นี้เรียก API ได้ เพราะ dashboard แยก repo/deploy ออกจาก backend ไปเลย
// (ไม่ใช่ same-origin) ดู decision log ที่ projects/bop-dashboard/PHASE1-DASHBOARD-TODO.md
// — อ่านจาก env var BOP_DASHBOARD_ORIGIN ก่อนเสมอ (ตอน deploy จริงค่านี้จะเป็น origin
// จริงของ dashboard เช่น https://bop.example.com) fallback เป็น localhost:3000 เดิมถ้า
// ไม่ได้ตั้ง (local dev)
var dashboardOrigin = getenv("BOP_DASHBOARD_ORIGIN", "http://localhost:3000")

// getenv คืนค่า environment variable ชื่อ key ถ้าตั้งไว้ (และไม่ใช่ค่าว่าง) ไม่งั้นคืนค่า
// fallback แทน — ใช้กับ dashboardOrigin (ค่า config ของชั้น HTTP เอง ส่วน secret/credential
// ของ auth ย้ายไปอ่านใน cmd/bop/main.go แล้ว)
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ServeHTTP ทำให้ Server เป็น http.Handler ได้ — ก่อนส่งต่อให้ mux ภายในจัดการ จะใส่
// CORS header ให้ทุก response ก่อนเสมอ (Access-Control-Allow-Credentials ต้องเป็น "true"
// เพราะ dashboard ส่ง JWT ผ่าน httpOnly cookie มาด้วยทุก request — ตาม spec ของ CORS
// การอนุญาต credentials แบบนี้ "ห้าม" ใช้ Access-Control-Allow-Origin: "*" เด็ดขาด ต้องระบุ
// origin ที่อนุญาตตรงๆ เท่านั้น) request แบบ OPTIONS (preflight ที่เบราว์เซอร์ส่งมาก่อน
// request จริงเองอัตโนมัติเวลามี custom header อย่าง Authorization) ตอบ 204 ทันทีโดยไม่ต้อง
// ส่งต่อให้ mux เลย เพราะไม่มี route ไหนใน routes() รองรับ method OPTIONS ตรงๆ (และ
// เบราว์เซอร์ไม่แนบ cookie ไปกับ preflight จึงต้องตอบก่อนบังคับ login)
//
// request อื่นทั้งหมดผ่าน requireSession ก่อนถึง mux — ทุก route ต้องมี session เป็นค่า
// เริ่มต้น ยกเว้นที่อยู่ใน publicRoutes (ดู require_session.go)
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", dashboardOrigin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.mux.ServeHTTP(w, r)
}

// routes ลงทะเบียน endpoint ทั้งหมดของ API — ใช้ syntax "METHOD /path" ของ Go 1.22+
// ที่รองรับ wildcard แบบ {name} ในตัว ServeMux เองโดยไม่ต้องพึ่ง library ภายนอก
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz) // liveness: process ยังตอบสนองอยู่ไหม
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)   // readiness: พร้อมรับ traffic ไหม (ต่อ Temporal ได้)

	s.mux.HandleFunc("GET /bots", s.handleListBots)                       // ดูรายชื่อ bot ที่ register ไว้ทั้งหมด + config schema ของแต่ละตัว
	s.mux.HandleFunc("POST /bots/{name}/jobs", s.handleCreateJob)         // สร้าง job ใหม่และสั่งรันทันที
	s.mux.HandleFunc("GET /jobs", s.handleListJobs)                       // ดูรายการ job ทั้งหมด
	s.mux.HandleFunc("GET /jobs/{id}", s.handleGetJob)                    // ดูรายละเอียด job เดียว (รวมสถานะล่าสุด)
	s.mux.HandleFunc("GET /jobs/{id}/logs", s.handleGetJobLogs)           // ดู log ทั้งหมดของ job เดียว (REST snapshot ไว้ backfill)
	s.mux.HandleFunc("GET /jobs/{id}/logs/stream", s.handleStreamJobLogs) // ดู log แบบ real-time ผ่าน SSE (roadmap ข้อ 1.5)
	s.mux.HandleFunc("DELETE /jobs/{id}", s.handleCancelJob)              // สั่งยกเลิก job ที่กำลังรันอยู่

	s.mux.HandleFunc("GET /workflow-runs", s.handleListWorkflowRuns)        // ดูรายการ workflow run ทั้งหมด (ทั้งที่ scheduler สั่งอัตโนมัติและที่ trigger เอง)
	s.mux.HandleFunc("GET /workflow-runs/{id}", s.handleGetWorkflowRun)     // ดูรายละเอียด workflow run เดียว รวมผลลัพธ์ทีละ step
	s.mux.HandleFunc("GET /workflow-runs/count", s.handleCountWorkflowRuns) // นับ workflow run ตามกลุ่มสถานะ (path ตายตัวชนะ {id} ตามกฎของ ServeMux)
	s.mux.HandleFunc("POST /workflow-runs", s.handleCreateWorkflowRun)      // รัน step (1 หรือหลาย step) ทันทีแบบ ad-hoc โดยไม่ต้องสร้าง schedule ก่อน

	s.mux.HandleFunc("POST /login", s.handleLogin)   // ตรวจ credential แล้วออก JWT ใส่ httpOnly cookie (roadmap ข้อ 1.8)
	s.mux.HandleFunc("POST /logout", s.handleLogout) // ล้าง cookie ทิ้ง
	s.mux.HandleFunc("GET /me", s.handleMe)          // เช็คว่า cookie ปัจจุบัน valid อยู่ไหม คืนชื่อ user ถ้าใช่

	s.mux.HandleFunc("POST /schedules", s.handleCreateSchedule)                 // สร้าง schedule ใหม่ (interval หรือ cron)
	s.mux.HandleFunc("GET /schedules", s.handleListSchedules)                   // list schedule ทั้งหมด
	s.mux.HandleFunc("GET /schedules/{id}", s.handleGetSchedule)                // ดูรายละเอียด schedule เดียว
	s.mux.HandleFunc("PUT /schedules/{id}", s.handleUpdateSchedule)             // แทนที่ schedule ทั้งก้อน (full replace)
	s.mux.HandleFunc("DELETE /schedules/{id}", s.handleDeleteSchedule)          // ลบ schedule ถาวร
	s.mux.HandleFunc("POST /schedules/{id}/pause", s.handlePauseSchedule)       // หยุดชั่วคราว (เทียบ Control-M "Hold")
	s.mux.HandleFunc("POST /schedules/{id}/unpause", s.handleUnpauseSchedule)   // เริ่มใหม่หลัง pause (เทียบ "Free")
	s.mux.HandleFunc("POST /schedules/{id}/trigger", s.handleTriggerSchedule)   // สั่งรันทันที นอกตารางเวลา (เทียบ "Order Now")
	s.mux.HandleFunc("POST /schedules/{id}/backfill", s.handleBackfillSchedule) // รันย้อนหลังตามช่วงเวลา (เทียบ "Rerun")
}
