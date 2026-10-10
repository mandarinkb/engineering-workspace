// Package job คือ usecase ของ "job เดี่ยว" — สั่งรัน bot หนึ่งตัวทันทีผ่าน job.Executor
// (ไม่ผ่าน Temporal) แล้วติดตามสถานะ/log ผ่าน job.Repository กับ logger.Logger
// usecase นี้รู้จักแค่ port ใน internal/domain ไม่รู้จัก HTTP, worker pool, หรือ storage
// ตัวจริงเลย
package job

import (
	"context"
	"time"

	"bop/internal/domain/apperr"
	"bop/internal/domain/bot"
	"bop/internal/domain/job"
	"bop/internal/domain/logger"
	"bop/internal/pkg/idgen"
)

// Service รวม use case ทั้งหมดของ job เดี่ยว
type Service struct {
	repo     job.Repository
	executor job.Executor
	log      logger.Logger
}

// NewService ผูก use case เข้ากับ port ที่ส่งเข้ามา (ประกอบใน cmd/bop/main.go)
func NewService(repo job.Repository, executor job.Executor, log logger.Logger) *Service {
	return &Service{repo: repo, executor: executor, log: log}
}

// ListBots คืน bot ทั้งหมดที่ register ไว้ เรียงตามชื่อ (a-z)
func (s *Service) ListBots() []bot.Bot {
	return s.executor.List()
}

// Create สร้าง job ใหม่สถานะ pending บันทึกลง repository ก่อน แล้วค่อยส่งเข้า executor ให้
// เริ่มทำงาน — คืน job ที่ยังเป็น pending ทันทีโดยไม่รอผล (ผู้เรียกต้องไปเช็คสถานะทีหลังเอง
// ผ่าน Get) timeoutSeconds เป็น 0 ได้ (ใช้ timeout กลางของ executor แทน)
//
// error จาก executor (เช่น ไม่รู้จัก bot) ห่อเป็น apperr.ErrInvalid เพราะเป็นความผิดของ
// input จาก client ส่วน error จาก repository ปล่อยเป็น error ธรรมดา
//
// อ่านค่า job ที่คืนไปได้อย่างปลอดภัย เพราะ executor copy ค่าออกไปเป็นของตัวเองแล้วตั้งแต่
// ตอน Submit (ดู internal/pkg/worker/pool.go) จึงไม่มีทางชนกับ goroutine ที่กำลังรัน
// job อยู่เบื้องหลัง
func (s *Service) Create(ctx context.Context, botName string, cfg map[string]string, timeoutSeconds int) (*job.Job, error) {
	j := &job.Job{
		ID:             idgen.New(),
		BotName:        botName,
		Config:         cfg,
		TimeoutSeconds: timeoutSeconds,
		Status:         job.StatusPending,
		CreatedAt:      time.Now(),
	}

	if err := s.repo.Create(ctx, j); err != nil {
		return nil, err
	}
	if err := s.executor.Submit(j); err != nil {
		return nil, apperr.WrapInvalid(err)
	}
	return j, nil
}

// List คืนรายการ job ทั้งหมดในระบบ
func (s *Service) List(ctx context.Context) ([]*job.Job, error) {
	return s.repo.List(ctx)
}

// Get คืน job เดียวตาม ID (สถานะล่าสุด ณ ตอนที่เรียก)
func (s *Service) Get(ctx context.Context, id string) (*job.Job, error) {
	return s.repo.Get(ctx, id)
}

// Logs คืน log ทั้งหมดที่บันทึกไว้ของ job นั้น (snapshot ณ ตอนที่เรียก)
func (s *Service) Logs(ctx context.Context, id string) ([]logger.Line, error) {
	return s.log.History(ctx, id)
}

// SubscribeLogs เปิดช่องทางรับ log บรรทัดใหม่ของ job แบบ real-time — ผู้เรียกต้องเรียก
// unsubscribe เมื่อเลิกฟัง ดูลำดับการใช้คู่กับ Logs ที่ handler SSE ใน
// internal/adapter/http/handler.go
func (s *Service) SubscribeLogs(id string) (<-chan logger.Line, func()) {
	return s.log.Subscribe(id)
}

// Cancel สั่งยกเลิก job ที่กำลังรันอยู่ — false ถ้า job นั้นไม่ได้กำลังรัน (จบไปแล้ว หรือ
// ID ไม่มีอยู่จริง)
func (s *Service) Cancel(id string) bool {
	return s.executor.Cancel(id)
}
