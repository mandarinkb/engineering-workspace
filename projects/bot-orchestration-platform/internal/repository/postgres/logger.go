package postgres

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bop/internal/domain/logger"
)

// writeTimeout จำกัดเวลา insert log หนึ่งบรรทัด — Log คืน error ไม่ได้และถูกเรียกระหว่างที่ Bot
// กำลังรัน ถ้า database ค้างต้องไม่ลาก Bot ค้างตามไปด้วย
const writeTimeout = 5 * time.Second

// Logger เก็บ log ของ Job ในตาราง log_lines ทำให้ GET /jobs/{id}/logs และ backfill ของ SSE stream
// อ่าน log เดิมได้หลัง restart/deploy ของ bop-api — พฤติกรรมเหมือน pkg/logger แบบ memory ทุกอย่าง
// (ยืนยันด้วย contract test ชุดเดียวกันใน internal/repository)
//
// ส่วน real-time (Subscribe) ยังกระจายใน process เดียวกัน เพราะ log ของ Job มาจาก worker pool
// ใน bop-api เองเท่านั้น ไม่ต้องใช้ LISTEN/NOTIFY ข้าม process — โค้ด fan-out ซ้ำกับของ memory
// โดยตั้งใจ เพราะ depguard ห้าม repository import pkg/logger
type Logger struct {
	db  *pgxpool.Pool
	now func() time.Time

	mu   sync.Mutex                    // ป้องกัน subs
	subs map[string][]chan logger.Line // job ID -> channel ของคนที่กำลังฟัง log สด
}

var (
	_ logger.Logger = (*Logger)(nil)
	_ logger.Pruner = (*Logger)(nil)
)

// LoggerOption ปรับค่าของ Logger ตอนสร้าง
type LoggerOption interface {
	apply(*Logger)
}

type clockOption func() time.Time

func (c clockOption) apply(l *Logger) { l.now = c }

// WithClock เปลี่ยนนาฬิกาที่ใช้ประทับเวลา log line (ค่าเริ่มต้น time.Now) — ใช้ใน test ที่ต้องการ
// log ย้อนหลังหลายวันเพื่อพิสูจน์งานลบตามอายุ
func WithClock(now func() time.Time) LoggerOption {
	return clockOption(now)
}

// NewLogger ผูก logger เข้ากับ pool ที่ Connect เปิดไว้ (schema ต้อง Migrate แล้ว)
func NewLogger(db *pgxpool.Pool, opts ...LoggerOption) *Logger {
	l := &Logger{
		db:   db,
		now:  time.Now,
		subs: make(map[string][]chan logger.Line),
	}
	for _, opt := range opts {
		opt.apply(l)
	}
	return l
}

// Log บันทึก log หนึ่งบรรทัดลง database ก่อน แล้วค่อยส่งต่อให้ subscriber ของ Job นั้น
//
//   - ตัดเวลาเหลือระดับ microsecond (ความละเอียดของ timestamptz) ก่อนทั้งเก็บและส่ง เพื่อให้
//     บรรทัดที่ stream ส่งไปกับที่ History อ่านกลับมามีเวลาเท่ากันทุกประการ
//   - insert ด้วย context ที่ไม่ถูกยกเลิกตาม ctx ของผู้เรียก เพราะ worker pool log "job cancelled"
//     ด้วย ctx ของ Job ที่เพิ่งถูกยกเลิก บรรทัดสำคัญแบบนี้ต้องไม่หาย
//   - insert พลาด = log เตือนใน stdout ของ process แล้วยังส่งให้ subscriber ต่อ (Log คืน error
//     ไม่ได้ตาม interface และ Bot ต้องรันต่อได้แม้ database มีปัญหาชั่วคราว)
func (l *Logger) Log(ctx context.Context, jobID string, level logger.Level, message string) {
	line := logger.Line{
		JobID:   jobID,
		Time:    l.now().Truncate(time.Microsecond),
		Level:   level,
		Message: message,
	}

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), writeTimeout)
	defer cancel()
	if _, err := l.db.Exec(writeCtx,
		`INSERT INTO log_lines (job_id, time, level, message) VALUES ($1, $2, $3, $4)`,
		line.JobID, line.Time, line.Level, line.Message); err != nil {
		log.Printf("postgres: store log line of job %s: %v", jobID, err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, ch := range l.subs[jobID] {
		select {
		case ch <- line:
		default: // subscriber อ่านไม่ทัน: ทิ้งบรรทัดนี้แทนการบล็อค (เหมือน memory)
		}
	}
}

// History คืน log ทั้งหมดของ Job เรียงตามเวลา (บรรทัดที่เวลาเท่ากันเรียงตามลำดับที่ insert)
func (l *Logger) History(ctx context.Context, jobID string) ([]logger.Line, error) {
	rows, err := l.db.Query(ctx,
		`SELECT job_id, time, level, message FROM log_lines WHERE job_id = $1 ORDER BY time, id`,
		jobID)
	if err != nil {
		return nil, fmt.Errorf("postgres: history of job %s: %w", jobID, err)
	}
	defer rows.Close()

	// เริ่มจาก slice ว่างแทน nil เพราะ GET /jobs/{id}/logs ต้องส่ง [] ไม่ใช่ null (เหมือน memory)
	out := []logger.Line{}
	for rows.Next() {
		var line logger.Line
		if err := rows.Scan(&line.JobID, &line.Time, &line.Level, &line.Message); err != nil {
			return nil, fmt.Errorf("postgres: history of job %s: %w", jobID, err)
		}
		out = append(out, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: history of job %s: %w", jobID, err)
	}
	return out, nil
}

// Subscribe เปิด channel รับ log สดของ Job — buffer 16 บรรทัดเท่ากับของ memory (ส่งแบบไม่บล็อค
// บรรทัดที่เกิน buffer ถูกทิ้ง จึงไม่โตไม่จำกัด) unsubscribe ลบออกจากรายการแล้วปิด channel
// เรียกซ้ำได้
func (l *Logger) Subscribe(jobID string) (<-chan logger.Line, func()) {
	ch := make(chan logger.Line, 16)

	l.mu.Lock()
	l.subs[jobID] = append(l.subs[jobID], ch)
	l.mu.Unlock()

	unsubscribe := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		subs := l.subs[jobID]
		for i, c := range subs {
			if c == ch {
				l.subs[jobID] = append(subs[:i], subs[i+1:]...)
				if len(l.subs[jobID]) == 0 {
					delete(l.subs, jobID)
				}
				close(ch)
				break
			}
		}
	}
	return ch, unsubscribe
}

// DeleteBefore ลบ log line ทุกเจ้าของ (Job และ Step) ที่เก่ากว่า cutoff — ไม่แตะตาราง jobs
// Job เก่าจึงยังเห็นสถานะและ summary แม้ log หายไปแล้ว
func (l *Logger) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := l.db.Exec(ctx, `DELETE FROM log_lines WHERE time < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("postgres: delete log lines before %s: %w", cutoff.Format(time.RFC3339), err)
	}
	return tag.RowsAffected(), nil
}
