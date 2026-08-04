// Package http คือ control-plane API ของทั้งระบบ — หน้า dashboard (roadmap Phase 1)
// จะคุยกับ backend ผ่าน endpoint ที่นี่เท่านั้น ไม่มีทางอื่น package นี้พึ่งพาแค่
// interface ของ job.Repository, logger.Logger, และ *worker.Pool เท่านั้น ไม่เคยรู้จัก
// bot ชนิดใดชนิดหนึ่งโดยเฉพาะเลย
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"bop/internal/job"
	"bop/internal/logger"
	"bop/internal/worker"
	"bop/internal/workflow"
)

// Server คือ HTTP API ของ BOP รวม handler ทุกตัวไว้ในที่เดียว
type Server struct {
	mux     *http.ServeMux // ตัว router มาตรฐานของ Go (ใช้ pattern matching แบบ method+path ของ Go 1.22+)
	repo    job.Repository
	log     logger.Logger
	pool    *worker.Pool
	runRepo workflow.RunRepository
}

// NewServer ผูก handler ทุกตัวเข้ากับ dependency ที่ส่งเข้ามา แล้วลงทะเบียน route ให้เรียบร้อย
func NewServer(repo job.Repository, log logger.Logger, pool *worker.Pool, runRepo workflow.RunRepository) *Server {
	s := &Server{mux: http.NewServeMux(), repo: repo, log: log, pool: pool, runRepo: runRepo}
	s.routes()
	return s
}

// ServeHTTP ทำให้ Server เป็น http.Handler ได้ (ส่งต่อทุก request ให้ mux ภายในจัดการ)
// เพื่อให้ใช้กับ http.ListenAndServe ได้ตรงๆ
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// routes ลงทะเบียน endpoint ทั้งหมดของ API — ใช้ syntax "METHOD /path" ของ Go 1.22+
// ที่รองรับ wildcard แบบ {name} ในตัว ServeMux เองโดยไม่ต้องพึ่ง library ภายนอก
func (s *Server) routes() {
	s.mux.HandleFunc("POST /bots/{name}/jobs", s.handleCreateJob) // สร้าง job ใหม่และสั่งรันทันที
	s.mux.HandleFunc("GET /jobs", s.handleListJobs)               // ดูรายการ job ทั้งหมด
	s.mux.HandleFunc("GET /jobs/{id}", s.handleGetJob)            // ดูรายละเอียด job เดียว (รวมสถานะล่าสุด)
	s.mux.HandleFunc("GET /jobs/{id}/logs", s.handleGetJobLogs)   // ดู log ทั้งหมดของ job เดียว
	s.mux.HandleFunc("DELETE /jobs/{id}", s.handleCancelJob)      // สั่งยกเลิก job ที่กำลังรันอยู่

	s.mux.HandleFunc("GET /workflow-runs", s.handleListWorkflowRuns)    // ดูรายการ workflow run ทั้งหมด (ทั้งที่ scheduler สั่งอัตโนมัติและที่ trigger เอง)
	s.mux.HandleFunc("GET /workflow-runs/{id}", s.handleGetWorkflowRun) // ดูรายละเอียด workflow run เดียว รวมผลลัพธ์ทีละ step
}

// handleCreateJob รับ config เป็น JSON object (key-value เป็น string ล้วน ตรงกับ bot.Config)
// จาก URL path จะได้ชื่อ bot ที่ต้องการรัน แล้วสร้าง job ใหม่พร้อมสถานะ pending
// บันทึกลง repository ก่อน แล้วค่อยส่งเข้า worker pool ให้เริ่มทำงาน สุดท้ายตอบกลับ
// client ทันทีด้วยสถานะ 202 Accepted พร้อมข้อมูล job (ที่ยังเป็น pending อยู่ เพราะ
// job เพิ่งถูกส่งเข้า queue ยังไม่ได้เริ่มทำงานจริง — client ต้องไปเช็คสถานะทีหลังเองผ่าน
// GET /jobs/{id})
//
// TODO(Phase 1.5): พอมี WebSocket handler แล้ว ให้ stream ผ่าน s.log.Subscribe(job.ID)
// เพื่อให้ dashboard เห็นข้อมูลแบบ real-time แทนที่จะต้อง poll ซ้ำๆ ด้วย GET /jobs/{id}
// และ GET /jobs/{id}/logs
func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	botName := r.PathValue("name") // ดึงค่าจาก {name} ใน path pattern

	var cfg map[string]string
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	j := &job.Job{
		ID:        newID(),
		BotName:   botName,
		Config:    cfg,
		Status:    job.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.repo.Create(r.Context(), j); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.pool.Submit(j); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// หมายเหตุ: อ่านค่า j ตรงนี้ปลอดภัย เพราะ pool.Submit ได้ copy ค่าออกไปเป็นของตัวเอง
	// แล้วตั้งแต่ตอนเรียก (ดูคำอธิบายใน internal/worker/pool.go) จึงไม่มีทางชนกับ
	// goroutine ที่กำลังรัน job อยู่เบื้องหลัง
	writeJSON(w, http.StatusAccepted, j)
}

// handleListJobs คืนรายการ job ทั้งหมดในระบบ เรียงจากล่าสุดไปเก่าสุด
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

// handleGetJob คืนข้อมูล job เดียวตาม ID ที่ระบุใน path (สถานะล่าสุด ณ ตอนที่เรียก)
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	j, err := s.repo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, j)
}

// handleGetJobLogs คืน log ทั้งหมดที่บันทึกไว้ของ job นั้น (ยังไม่ใช่ real-time
// ตอนนี้ — เป็นแค่ snapshot ของ log ณ ตอนที่เรียก endpoint นี้)
func (s *Server) handleGetJobLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	lines, err := s.log.History(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, lines)
}

// handleCancelJob สั่งยกเลิก job ที่กำลังรันอยู่ คืน 409 Conflict ถ้า job นั้นไม่ได้
// กำลังรันอยู่แล้ว (จบไปแล้ว หรือ ID ไม่มีอยู่จริง)
func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.pool.Cancel(id) {
		http.Error(w, "job not running", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleListWorkflowRuns คืนรายการ workflow run ทั้งหมด (ทั้งที่กำลังรันอยู่และจบไปแล้ว)
// เรียงจากล่าสุดไปเก่าสุด — ใช้ดูว่า scheduled workflow ทำงานไปแล้วกี่รอบ สำเร็จ/ล้มเหลว
// รอบไหนบ้าง
func (s *Server) handleListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.runRepo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

// handleGetWorkflowRun คืนรายละเอียด workflow run เดียว รวมผลลัพธ์ทีละ step ที่รันไปแล้ว
// (แต่ละ step มี JobID กำกับ ไปดู log เต็มๆ ของ step นั้นต่อได้ที่ GET /jobs/{id}/logs)
func (s *Server) handleGetWorkflowRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := s.runRepo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// writeJSON คือ helper เขียน response กลับเป็น JSON พร้อมตั้ง header/status code ให้ครบ
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// newID สุ่มสร้างรหัสเฉพาะสำหรับ job ใหม่ โดยใช้ crypto/rand (สุ่มแบบปลอดภัยทาง cryptography)
// สร้าง byte สุ่ม 16 ตัวแล้วแปลงเป็น string hex — ไม่ใช้ library ภายนอกอย่าง uuid
// เพื่อให้โปรเจกต์นี้ยังคง zero external dependency อยู่
func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
