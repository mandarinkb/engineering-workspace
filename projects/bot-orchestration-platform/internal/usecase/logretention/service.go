// Package logretention คืองานลบ log ที่เก่ากว่า 30 วันใน bop-api — Schedule รายนาทีสร้าง log
// วันละหลายพันบรรทัด ถ้าไม่ลบดิสก์ของ PostgreSQL จะเต็มในที่สุด ลบเฉพาะ log line ส่วน Job ยังอยู่
// ให้เห็นสถานะและ summary ต่อไป (ดู spec backend-reliability หัวข้อ Persistence)
package logretention

import (
	"context"
	"fmt"
	"log"
	"time"

	"bop/internal/domain/logger"
)

const (
	// MaxAge คืออายุสูงสุดของ log line ก่อนถูกลบ
	MaxAge = 30 * 24 * time.Hour
	// Interval คือความถี่ของงานลบ (วันละครั้ง)
	Interval = 24 * time.Hour
)

// Service ลบ log ที่เก่ากว่า MaxAge ผ่าน logger.Pruner
type Service struct {
	pruner   logger.Pruner
	now      func() time.Time
	interval time.Duration
}

// Option ปรับค่าของ Service ตอนสร้าง
type Option interface {
	apply(*Service)
}

type clockOption func() time.Time

func (c clockOption) apply(s *Service) { s.now = c }

// WithClock เปลี่ยนนาฬิกาที่ใช้คำนวณ cutoff (ค่าเริ่มต้น time.Now)
func WithClock(now func() time.Time) Option {
	return clockOption(now)
}

type intervalOption time.Duration

func (i intervalOption) apply(s *Service) { s.interval = time.Duration(i) }

// WithInterval เปลี่ยนความถี่ของ Run (ค่าเริ่มต้น Interval) — ใช้ใน test
func WithInterval(d time.Duration) Option {
	return intervalOption(d)
}

// NewService สร้างงานลบ log ที่ลบผ่าน pruner
func NewService(pruner logger.Pruner, opts ...Option) *Service {
	s := &Service{pruner: pruner, now: time.Now, interval: Interval}
	for _, opt := range opts {
		opt.apply(s)
	}
	return s
}

// PruneOnce ลบ log line ที่เก่ากว่า MaxAge นับจากตอนนี้หนึ่งรอบ แล้วคืนจำนวนบรรทัดที่ลบ
func (s *Service) PruneOnce(ctx context.Context) (int64, error) {
	cutoff := s.now().Add(-MaxAge)
	n, err := s.pruner.DeleteBefore(ctx, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune logs older than %s: %w", cutoff.Format(time.RFC3339), err)
	}
	return n, nil
}

// Run ลบทันทีหนึ่งรอบ (bop-api restart บ่อยตอน deploy ถ้ารอครบวันก่อนอาจไม่ได้ลบเลย) แล้วลบซ้ำ
// ทุก interval จนกว่า ctx จะถูกยกเลิก — รอบที่พลาดแค่ log ไว้แล้วรอรอบถัดไป เพราะงานนี้ไม่ใช่
// งานหลักของ API ผู้เรียกต้องรอให้ Run คืนก่อนปิด connection pool
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		if n, err := s.PruneOnce(ctx); err != nil {
			log.Printf("log retention: %v", err)
		} else if n > 0 {
			log.Printf("log retention: deleted %d log lines older than %s", n, MaxAge)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
