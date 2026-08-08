// Package http คือ control-plane API ของทั้งระบบ — หน้า dashboard (roadmap Phase 1)
// จะคุยกับ backend ผ่าน endpoint ที่นี่เท่านั้น ไม่มีทางอื่น package นี้พึ่งพาแค่
// interface ของ job.Repository, logger.Logger, และ *worker.Pool เท่านั้น ไม่เคยรู้จัก
// bot ชนิดใดชนิดหนึ่งโดยเฉพาะเลย
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	"bop/internal/auth"
	"bop/internal/bot"
	"bop/internal/job"
	"bop/internal/logger"
	"bop/internal/schedule"
	"bop/internal/worker"
	"bop/internal/workflow"
)

// Server คือ HTTP API ของ BOP รวม handler ทุกตัวไว้ในที่เดียว
type Server struct {
	mux          *http.ServeMux // ตัว router มาตรฐานของ Go (ใช้ pattern matching แบบ method+path ของ Go 1.22+)
	repo         job.Repository
	log          logger.Logger
	pool         *worker.Pool
	temporal     client.Client       // ใช้ query ผลลัพธ์ workflow run + จัดการ Temporal Schedule
	scheduleRepo schedule.Repository // เก็บ workflow definition ของแต่ละ schedule (ดู internal/schedule)
}

// NewServer ผูก handler ทุกตัวเข้ากับ dependency ที่ส่งเข้ามา แล้วลงทะเบียน route ให้เรียบร้อย
func NewServer(repo job.Repository, log logger.Logger, pool *worker.Pool, temporalClient client.Client, scheduleRepo schedule.Repository) *Server {
	s := &Server{mux: http.NewServeMux(), repo: repo, log: log, pool: pool, temporal: temporalClient, scheduleRepo: scheduleRepo}
	s.routes()
	return s
}

// dashboardOrigin คือ origin ของ Next.js dev server (projects/bop-dashboard/) — ต้องเปิด
// CORS ให้ origin นี้เรียก API ได้ เพราะ dashboard แยก repo/deploy ออกจาก backend ไปเลย
// (ไม่ใช่ same-origin) ดู decision log ที่ projects/bop-dashboard/PHASE1-DASHBOARD-TODO.md
// — hardcode ไปก่อนเหมือนค่าคงที่อื่นๆ ในไฟล์นี้ (เช่น addr ":8080" ใน cmd/bop/main.go)
// พอ deploy จริงค่อยเปลี่ยนเป็นอ่านจาก environment variable
const dashboardOrigin = "http://localhost:3000"

// ServeHTTP ทำให้ Server เป็น http.Handler ได้ — ก่อนส่งต่อให้ mux ภายในจัดการ จะใส่
// CORS header ให้ทุก response ก่อนเสมอ (Access-Control-Allow-Credentials ต้องเป็น "true"
// เพราะ dashboard ส่ง JWT ผ่าน httpOnly cookie มาด้วยทุก request — ตาม spec ของ CORS
// การอนุญาต credentials แบบนี้ "ห้าม" ใช้ Access-Control-Allow-Origin: "*" เด็ดขาด ต้องระบุ
// origin ที่อนุญาตตรงๆ เท่านั้น) request แบบ OPTIONS (preflight ที่เบราว์เซอร์ส่งมาก่อน
// request จริงเองอัตโนมัติเวลามี custom header อย่าง Authorization) ตอบ 204 ทันทีโดยไม่ต้อง
// ส่งต่อให้ mux เลย เพราะไม่มี route ไหนใน routes() รองรับ method OPTIONS ตรงๆ
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", dashboardOrigin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// routes ลงทะเบียน endpoint ทั้งหมดของ API — ใช้ syntax "METHOD /path" ของ Go 1.22+
// ที่รองรับ wildcard แบบ {name} ในตัว ServeMux เองโดยไม่ต้องพึ่ง library ภายนอก
func (s *Server) routes() {
	s.mux.HandleFunc("GET /bots", s.handleListBots)                       // ดูรายชื่อ bot ที่ register ไว้ทั้งหมด + config schema ของแต่ละตัว
	s.mux.HandleFunc("POST /bots/{name}/jobs", s.handleCreateJob)         // สร้าง job ใหม่และสั่งรันทันที
	s.mux.HandleFunc("GET /jobs", s.handleListJobs)                       // ดูรายการ job ทั้งหมด
	s.mux.HandleFunc("GET /jobs/{id}", s.handleGetJob)                    // ดูรายละเอียด job เดียว (รวมสถานะล่าสุด)
	s.mux.HandleFunc("GET /jobs/{id}/logs", s.handleGetJobLogs)           // ดู log ทั้งหมดของ job เดียว (REST snapshot ไว้ backfill)
	s.mux.HandleFunc("GET /jobs/{id}/logs/stream", s.handleStreamJobLogs) // ดู log แบบ real-time ผ่าน SSE (roadmap ข้อ 1.5)
	s.mux.HandleFunc("DELETE /jobs/{id}", s.handleCancelJob)              // สั่งยกเลิก job ที่กำลังรันอยู่

	s.mux.HandleFunc("GET /workflow-runs", s.handleListWorkflowRuns)     // ดูรายการ workflow run ทั้งหมด (ทั้งที่ scheduler สั่งอัตโนมัติและที่ trigger เอง)
	s.mux.HandleFunc("GET /workflow-runs/{id}", s.handleGetWorkflowRun)  // ดูรายละเอียด workflow run เดียว รวมผลลัพธ์ทีละ step
	s.mux.HandleFunc("POST /workflow-runs", s.handleCreateWorkflowRun)   // รัน step (1 หรือหลาย step) ทันทีแบบ ad-hoc โดยไม่ต้องสร้าง schedule ก่อน

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

// botSummary คือรูปแบบ JSON ที่คืนจาก GET /bots — ห่อ bot.Bot (interface, encode
// ตรงๆ ไม่ได้เพราะไม่รู้จัก concrete type) ให้เหลือแค่สิ่งที่ dashboard ต้องใช้จริง:
// ชื่อ (ไปใช้เป็น {name} ใน POST /bots/{name}/jobs) กับ config schema (ไปสร้าง
// dynamic form ตาม roadmap ข้อ 1.3)
type botSummary struct {
	Name         string            `json:"name"`
	ConfigSchema []bot.ConfigField `json:"config_schema"`
}

// handleListBots คืนรายชื่อ bot ที่ register ไว้ทั้งหมดพร้อม config schema ของแต่ละตัว —
// เรียงตามชื่อ (a-z) ตาม worker.Pool.List
func (s *Server) handleListBots(w http.ResponseWriter, r *http.Request) {
	bots := s.pool.List()
	out := make([]botSummary, 0, len(bots))
	for _, b := range bots {
		out = append(out, botSummary{Name: b.Name(), ConfigSchema: b.ConfigSchema()})
	}
	writeJSON(w, http.StatusOK, out)
}

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

	j := &job.Job{
		ID:        newID(),
		BotName:   botName,
		Config:    cfg,
		Status:    job.StatusPending,
		CreatedAt: time.Now(),
	}

	if raw := r.URL.Query().Get("timeout_seconds"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			http.Error(w, "timeout_seconds must be a positive integer", http.StatusBadRequest)
			return
		}
		j.TimeoutSeconds = seconds
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

// handleStreamJobLogs สตรีม log ของ job ที่ระบุแบบ real-time ผ่าน Server-Sent Events
// (SSE) — เลือกใช้ SSE แทน WebSocket เพราะ log stream เป็นทิศทางเดียว (server→client
// เท่านั้น dashboard ไม่เคยต้องส่งอะไรกลับผ่าน channel นี้เลย) SSE เป็น HTTP ธรรมดา
// (`Content-Type: text/event-stream`) เบราว์เซอร์มี auto-reconnect ในตัวผ่าน
// `EventSource` โดยไม่ต้องเขียน reconnect logic เอง และฝั่ง Go เขียนง่ายกว่า WebSocket
// มาก (ไม่ต้องพึ่ง library ภายนอกอย่าง gorilla/websocket เพื่อรักษาหลักการ dependency
// น้อยที่สุดของโปรเจกต์นี้) — ดู decision log เต็มๆ ที่
// projects/bop-dashboard/PHASE1-DASHBOARD-TODO.md
//
// ขั้นตอนการทำงาน (ลำดับสำคัญมาก อย่าสลับ):
//  1. Subscribe เข้า logger.Logger ก่อนเสมอ — เปิด channel รับ log บรรทัดใหม่ที่จะเกิดขึ้น
//     "นับจากตอนนี้เป็นต้นไป" ก่อน
//  2. ค่อยเรียก History เอา log ทั้งหมดที่เคยบันทึกไว้ก่อนหน้านี้มาส่งเป็น backfill —
//     ลำดับนี้ (Subscribe ก่อน History) ป้องกันไม่ให้พลาด log บรรทัดที่เกิดขึ้นระหว่าง
//     สอง call นี้ (ถ้าสลับเป็น History ก่อน Subscribe จะมีช่วงเวลาสั้นๆ ที่ log ใหม่
//     เกิดขึ้นแล้วไม่มีใครจับไว้เลย ข้อมูลหายจริง) — ข้อเสียที่ยอมรับได้ของลำดับนี้คือ
//     "อาจ" เห็น log บรรทัดเดียวกันซ้ำ 2 ครั้งได้ในกรณีที่ Log() ถูกเรียกพอดีในช่วงเวลา
//     สั้นๆ ระหว่าง Subscribe กับ History (ซ้ำแต่ไม่หาย ยอมรับได้มากกว่าหายไปเงียบๆ)
//  3. ส่ง history ทั้งหมดออกไปก่อน (backfill) แล้ว flush ทันทีให้ client เห็นของเก่าครบ
//  4. เข้าลูป: รอ log บรรทัดใหม่จาก channel แล้วส่งต่อทันที หรือรอ ctx ของ request ถูก
//     ยกเลิก (client ปิดแท็บ/network หลุด) ก็เลิกลูปแล้ว unsubscribe ทิ้ง (ผ่าน defer)
func (s *Server) handleStreamJobLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch, unsubscribe := s.log.Subscribe(id)
	defer unsubscribe()

	history, err := s.log.History(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for _, line := range history {
		writeSSELine(w, line)
	}
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done(): // client ปิด connection แล้ว ไม่ต้องส่งต่ออีก
			return
		case line, ok := <-ch:
			if !ok { // channel ถูกปิด (ปกติเกิดจาก unsubscribe ของเราเอง ไม่ควรเข้า case นี้ตอนอื่น)
				return
			}
			writeSSELine(w, line)
			flusher.Flush()
		}
	}
}

// writeSSELine เข้ารหัส log บรรทัดหนึ่งเป็น JSON แล้วเขียนตาม format ของ Server-Sent
// Events ("data: <payload>\n\n" — ต้องมี \n\n คั่นท้ายเสมอ ไม่งั้นเบราว์เซอร์จะยังไม่ถือว่า
// event นี้จบ) EventSource ฝั่ง client จะได้ event แต่ละอันเป็น log.Line ในรูป JSON string
// พร้อม JSON.parse ต่อได้ทันที
func writeSSELine(w http.ResponseWriter, line logger.Line) {
	b, err := json.Marshal(line)
	if err != nil {
		return // ข้าม line นี้ถ้า encode ไม่ได้ (ไม่ควรเกิดจริง เพราะ logger.Line เป็น plain struct ล้วนๆ)
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
}

// cookieName คือชื่อ cookie ที่เก็บ JWT — ใช้ชื่อเดียวกันทั้งตอน set (handleLogin),
// ตอนอ่าน (handleMe), และตอนล้าง (handleLogout)
const cookieName = "bop_session"

// authSecret คือ key ที่ใช้เซ็น/ตรวจ JWT (ดู internal/auth) — hardcode ไปก่อนสำหรับ
// local dev เหมือนค่าคงที่อื่นๆ ในไฟล์นี้ **ห้ามใช้ค่านี้ใน production เด็ดขาด** ต้อง
// เปลี่ยนไปอ่านจาก environment variable ก่อน deploy จริง (บันทึกไว้เป็น Open Question
// ใน projects/bop-dashboard/PHASE1-DASHBOARD-TODO.md แล้ว)
var authSecret = []byte("bop-dev-secret-change-me-before-deploying")

// adminUsername/adminPassword คือ credential เดียวที่ระบบรู้จักตอนนี้ — hardcode ไปก่อน
// เพราะยังไม่มี user database เลย (BOP เป็น single-operator tool ควบคุม bot ของตัวเอง
// ไม่ใช่ multi-tenant SaaS ที่ต้องมี user table จริงจังตั้งแต่ต้น) เปลี่ยนเป็นอ่านจาก
// environment variable ก่อน deploy จริงเหมือน authSecret ด้านบน
const (
	adminUsername = "admin"
	adminPassword = "changeme"
)

// loginRequest คือ JSON body ที่ POST /login ต้องรับ
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleLogin ตรวจ username/password กับค่าที่ hardcode ไว้ (ดูหมายเหตุด้านบน) ถ้าถูกต้อง
// ออก JWT (ผ่าน internal/auth) แล้วฝังใน httpOnly cookie ส่งกลับไป
//
// ทำไมเก็บ token ใน httpOnly cookie แทนที่จะให้ frontend เก็บเองใน localStorage:
// httpOnly cookie เข้าถึงจาก JavaScript ฝั่ง client ไม่ได้เลย (แม้แต่โค้ดของ dashboard
// เองก็อ่านไม่ได้) ป้องกัน token ถูกขโมยผ่าน XSS (ถ้ามี script อันตรายหลุดเข้ามารันบน
// หน้าเว็บ) เบราว์เซอร์แนบ cookie ให้เองอัตโนมัติทุก request ถัดไปที่ยิงไป backend
// origin เดียวกันตาม cookie policy โดย frontend ไม่ต้องจัดการเรื่องแนบ header เอง
//
// SameSite=Lax + Secure=false: dashboard (localhost:3000) กับ backend (localhost:8080)
// เป็นคนละ origin (คนละ port) แต่ยังนับเป็น "same site" ตาม spec ของเบราว์เซอร์ (site
// กำหนดจาก registrable domain ไม่นับ port) SameSite=Lax จึงยังใช้งานได้ปกติตอน local
// dev โดยไม่ต้องเปิด HTTPS — **ต้องกลับมาทบทวนค่านี้ก่อน deploy จริง** ถ้า dashboard กับ
// backend อยู่คนละ domain จริงๆ (ไม่ใช่แค่คนละ port บน localhost) ต้องเปลี่ยนเป็น
// SameSite=None + Secure=true (บังคับต้องผ่าน HTTPS เท่านั้น ไม่งั้นเบราว์เซอร์จะทิ้ง
// cookie นี้ไปเงียบๆ)
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Username != adminUsername || req.Password != adminPassword {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	const ttl = 24 * time.Hour
	token, err := auth.Issue(authSecret, req.Username, ttl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // TODO: true ตอน deploy จริงผ่าน HTTPS (ดูหมายเหตุด้านบน)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
	w.WriteHeader(http.StatusNoContent)
}

// handleLogout ล้าง cookie ทิ้งด้วยการตั้ง MaxAge เป็นค่าติดลบ (สั่งเบราว์เซอร์ให้ลบ
// cookie นี้ทันที) — ไม่ต้องเช็ค token เดิมเลยก่อนล้าง เพราะการล้าง cookie ปลอดภัยเสมอ
// ไม่ว่า token เดิมจะยังใช้ได้อยู่หรือหมดอายุไปแล้วก็ตาม
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

// handleMe ตรวจ cookie ปัจจุบันว่ายัง valid อยู่ไหม (signature ถูกต้อง + ยังไม่หมดอายุ)
// คืนชื่อ user ถ้าผ่าน — dashboard เรียก endpoint นี้ตอนโหลดหน้าแรกเพื่อรู้ว่า user
// ยัง login ค้างอยู่ไหม (แทนที่จะพยายามอ่าน/ถอดรหัส JWT เองฝั่ง client ซึ่งทำไม่ได้อยู่
// แล้วเพราะ cookie เป็น httpOnly)
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	claims, err := auth.Verify(authSecret, cookie.Value)
	if err != nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": claims.Subject})
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

// workflowRunSummary คือรูปแบบ JSON ที่คืนให้ dashboard ดู — แปลงมาจาก
// *workflowpb.WorkflowExecutionInfo ของ Temporal อีกที (ตัว protobuf type เดิมมี field
// เยอะเกินจำเป็นและใช้ *timestamppb.Timestamp ที่ไม่ใช่ JSON-friendly ตรงๆ) เก็บไว้แค่
// เท่าที่ dashboard ต้องใช้จริง ใกล้เคียงกับ workflow.Run เดิมก่อนย้ายมาใช้ Temporal
type workflowRunSummary struct {
	WorkflowID   string     `json:"workflow_id"`
	RunID        string     `json:"run_id"`
	WorkflowName string     `json:"workflow_name"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
}

// toRunSummary แปลง WorkflowExecutionInfo ที่ได้จาก Temporal (ทั้งจาก ListWorkflow และ
// DescribeWorkflowExecution) ให้เป็น workflowRunSummary — StartTime ของ Temporal เป็น
// *timestamppb.Timestamp เสมอ (ไม่มีทางเป็น nil สำหรับ execution ที่มีอยู่จริง) ส่วน
// CloseTime เป็น nil ตราบใดที่ workflow ยังไม่จบ จึงต้องเช็คก่อนแปลง
func toRunSummary(info *workflowpb.WorkflowExecutionInfo) workflowRunSummary {
	summary := workflowRunSummary{
		WorkflowID:   info.GetExecution().GetWorkflowId(),
		RunID:        info.GetExecution().GetRunId(),
		WorkflowName: info.GetType().GetName(),
		Status:       info.GetStatus().String(),
		StartedAt:    info.GetStartTime().AsTime(),
	}
	if info.GetCloseTime() != nil {
		ended := info.GetCloseTime().AsTime()
		summary.EndedAt = &ended
	}
	return summary
}

// handleListWorkflowRuns คืนรายการ workflow run ทั้งหมด (ทั้งที่กำลังรันอยู่และจบไปแล้ว)
// เรียงจากล่าสุดไปเก่าสุดตาม default ordering ของ Temporal Visibility API — ใช้ดูว่า
// scheduled workflow ทำงานไปแล้วกี่รอบ สำเร็จ/ล้มเหลวรอบไหนบ้าง แทนที่การอ่านจาก
// workflow.RunRepository (in-memory) เดิม ตอนนี้ query ตรงไปที่ Temporal server เอง
// (Temporal เป็นคนเก็บ event history + visibility record ให้ทั้งหมดอยู่แล้ว)
func (s *Server) handleListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	resp, err := s.temporal.ListWorkflow(r.Context(), &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: workflow.Namespace,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	runs := make([]workflowRunSummary, 0, len(resp.Executions))
	for _, e := range resp.Executions {
		runs = append(runs, toRunSummary(e))
	}
	writeJSON(w, http.StatusOK, runs)
}

// workflowRunDetail ต่อยอดจาก workflowRunSummary เพิ่มผลลัพธ์ทีละ step เข้ามา
type workflowRunDetail struct {
	workflowRunSummary
	Steps []workflow.StepResult `json:"steps"`
}

// handleGetWorkflowRun คืนรายละเอียด workflow run เดียว รวมผลลัพธ์ทีละ step ที่รันไปแล้ว
// — id ในที่นี้คือ Temporal Workflow ID (ไม่ใช่ Run.ID แบบสุ่มของเวอร์ชันเดิมอีกต่อไป)
// ดึงสถานะโดยรวมผ่าน DescribeWorkflowExecution และดึงผลลัพธ์ทีละ step ผ่าน
// QueryWorkflow เรียก query handler ชื่อ "steps" ที่ประกาศไว้ใน workflow.Execute
// (internal/workflow/temporal.go) — ใช้ได้ทั้งตอน workflow ยังรันอยู่และจบไปแล้ว
func (s *Server) handleGetWorkflowRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	desc, err := s.temporal.DescribeWorkflowExecution(r.Context(), id, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var steps []workflow.StepResult
	// QueryWorkflow อาจ error ได้หลายกรณี (เช่น เพิ่งเริ่ม workflow ยังไม่ทัน register
	// query handler, หรือ history ถูกลบไปตาม retention แล้ว) — ไม่ถือเป็น fatal error
	// ของ endpoint นี้ทั้งหมด ปล่อยให้ steps เป็น slice ว่างแทนที่จะ error ทั้ง response
	if val, qerr := s.temporal.QueryWorkflow(r.Context(), id, "", "steps"); qerr == nil {
		_ = val.Get(&steps)
	}

	writeJSON(w, http.StatusOK, workflowRunDetail{
		workflowRunSummary: toRunSummary(desc.WorkflowExecutionInfo),
		Steps:              steps,
	})
}

// createWorkflowRunRequest คือ JSON body ของ POST /workflow-runs — เหมือน steps/workflow_name
// ของ createScheduleRequest แต่ไม่มี spec/overlap เพราะ endpoint นี้ไม่ได้สร้าง schedule
// (การรันซ้ำตามตารางเวลา) แค่รัน step ที่ระบุ "ทันทีครั้งเดียว" (ad-hoc) เท่านั้น
type createWorkflowRunRequest struct {
	WorkflowName string          `json:"workflow_name"`
	Steps        []workflow.Step `json:"steps"`
}

// createWorkflowRunResponse คือ JSON response — ให้แค่ id ที่พอเอาไป poll สถานะต่อผ่าน
// GET /workflow-runs/{id} ที่มีอยู่แล้ว (ไม่ต้องเพิ่ม endpoint หรือ repository ใหม่เลย)
type createWorkflowRunResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// handleCreateWorkflowRun สั่งรัน workflow (1 หรือหลาย step เรียงกัน) ทันทีแบบ ad-hoc ผ่าน
// client.ExecuteWorkflow ตรงๆ — ต่างจาก POST /schedules ตรงที่ไม่สร้างอะไรที่ "รันซ้ำตาม
// ตารางเวลา" เลย แค่เริ่ม 1 run แล้วจบ เหมาะกับทั้ง (1) อยากรัน step ชุดหนึ่งครั้งเดียวโดย
// ไม่ต้องผ่านการสร้าง/ลบ schedule ให้ยุ่งยาก และ (2) "job เดี่ยว" ที่ต้อง route ไปยัง Temporal
// Task Queue อื่น (ผ่าน Step.TaskQueue — ดู internal/workflow/workflow.go) ซึ่งเป็นสิ่งที่
// worker.Pool เดิม (ใช้กับ POST /bots/{name}/jobs) ทำไม่ได้เลย เพราะไม่มี Temporal เกี่ยวข้อง
// อยู่ในเส้นทางนั้น
//
// เหมือนกับ handleCreateJob เดิม endpoint นี้เป็น fire-and-forget (ExecuteWorkflow แค่เริ่ม
// workflow แล้ว return ทันที ไม่รอผลลัพธ์) — client ต้อง poll สถานะทีหลังเองผ่าน
// GET /workflow-runs/{workflow_id} ที่มีอยู่แล้ว (ไม่ต้องเพิ่ม repository/endpoint ใหม่)
func (s *Server) handleCreateWorkflowRun(w http.ResponseWriter, r *http.Request) {
	var req createWorkflowRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.WorkflowName == "" {
		http.Error(w, "workflow_name is required", http.StatusBadRequest)
		return
	}
	if len(req.Steps) == 0 {
		http.Error(w, "steps must not be empty", http.StatusBadRequest)
		return
	}

	// ผูก workflow ID กับ workflow_name + newID() (helper เดิมท้ายไฟล์นี้) เพื่อกัน ID ชน
	// กันเวลาเรียก endpoint นี้ซ้ำหลายครั้งด้วย workflow_name เดียวกัน — ต่างจาก
	// ensureExampleSchedule/handleCreateSchedule ที่ใช้ ID ตายตัวจาก client เพราะที่นั่น
	// ID ต้อง stable ข้าม request (เอาไว้ pause/trigger/delete schedule เดิมซ้ำได้) แต่ที่นี่
	// แต่ละ request คือ run ใหม่ที่ไม่เกี่ยวข้องกับ run ก่อนหน้าเลย
	wf := workflow.Workflow{Name: req.WorkflowName, Steps: req.Steps}
	run, err := s.temporal.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s-%s", req.WorkflowName, newID()),
		TaskQueue: workflow.TaskQueueName, // task queue ของตัว "workflow" เอง (cmd/bop-worker) — คนละเรื่องกับ Step.TaskQueue ของแต่ละ activity ภายใน
	}, workflow.Execute, wf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, createWorkflowRunResponse{
		WorkflowID: run.GetID(),
		RunID:      run.GetRunID(),
	})
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
