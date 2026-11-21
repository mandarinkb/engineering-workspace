package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"bop/internal/domain/apperr"
	"bop/internal/domain/job"
)

// JobRepository เก็บ Job ในตาราง jobs ทำให้ประวัติ Job อยู่รอดข้าม restart/deploy ของ bop-api
// — พฤติกรรมเหมือน memory.JobRepository ทุกอย่าง (ยืนยันด้วย contract test ชุดเดียวกันใน
// internal/repository) ค่าที่คืนเป็น struct ใหม่จากแถวใน database เสมอ จึงไม่มีปัญหา pointer
// ที่ใช้ร่วมกันข้าม goroutine แบบที่ memory ต้อง Clone
type JobRepository struct {
	db *pgxpool.Pool
}

var _ job.Repository = (*JobRepository)(nil)

// jobColumns คือ column ที่ Get/List อ่าน เรียงตรงกับ scanJob
const jobColumns = `id, bot_name, config, timeout_seconds, status, summary, error,
	created_at, started_at, ended_at`

// NewJobRepository ผูก repository เข้ากับ pool ที่ Connect เปิดไว้ (schema ต้อง Migrate แล้ว)
func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

// Create บันทึก Job ใหม่ — ID ซ้ำชน primary key แล้วคืน error เหมือน memory
func (r *JobRepository) Create(ctx context.Context, j *job.Job) error {
	cfg, err := marshalConfig(j.Config)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `INSERT INTO jobs (`+jobColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		j.ID, j.BotName, cfg, j.TimeoutSeconds, j.Status, j.Summary, j.Error,
		j.CreatedAt, j.StartedAt, j.EndedAt)
	if err != nil {
		return fmt.Errorf("postgres: create job %s: %w", j.ID, err)
	}
	return nil
}

// Update บันทึกทับสถานะและผลลัพธ์ล่าสุดของ Job — ไม่พบแถว = error เพราะ Job ต้องถูก Create
// ก่อนเสมอ (worker pool เรียก Update หลัง usecase Create แล้วเท่านั้น)
func (r *JobRepository) Update(ctx context.Context, j *job.Job) error {
	cfg, err := marshalConfig(j.Config)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE jobs SET bot_name = $2, config = $3, timeout_seconds = $4,
		status = $5, summary = $6, error = $7, created_at = $8, started_at = $9, ended_at = $10
		WHERE id = $1`,
		j.ID, j.BotName, cfg, j.TimeoutSeconds, j.Status, j.Summary, j.Error,
		j.CreatedAt, j.StartedAt, j.EndedAt)
	if err != nil {
		return fmt.Errorf("postgres: update job %s: %w", j.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.WrapNotFound(fmt.Errorf("postgres: job %s not found", j.ID))
	}
	return nil
}

// Get ค้นหา Job ด้วย ID — ไม่พบคืน error ประเภท apperr.ErrNotFound
func (r *JobRepository) Get(ctx context.Context, id string) (*job.Job, error) {
	row := r.db.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id)
	j, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.WrapNotFound(fmt.Errorf("postgres: job %s not found", id))
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: get job %s: %w", id, err)
	}
	return j, nil
}

// List คืน Job ทั้งหมดเรียงจากสร้างล่าสุดไปเก่าสุด (ลำดับเดียวกับ memory ที่หน้า Jobs ใช้)
func (r *JobRepository) List(ctx context.Context) ([]*job.Job, error) {
	rows, err := r.db.Query(ctx, `SELECT `+jobColumns+` FROM jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("postgres: list jobs: %w", err)
	}
	defer rows.Close()

	// เริ่มจาก slice ว่างแทน nil เพราะ GET /jobs ต้องส่ง [] ไม่ใช่ null (เหมือน memory)
	out := []*job.Job{}
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: list jobs: %w", err)
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list jobs: %w", err)
	}
	return out, nil
}

// FailInterrupted ตั้ง Job ที่ pending/running ทุกแถวเป็น failed ใน statement เดียว (atomic) —
// เรียกตอน bop-api start ก่อนเปิดรับงาน จึงไม่ชนกับ Job ใหม่ของ process นี้ (ถือว่ารัน bop-api
// ตัวเดียว ดู ARCHITECTURE.md) แถวที่จบแล้วไม่ถูกแตะ started_at คงเดิม (pending ยังเป็น NULL)
func (r *JobRepository) FailInterrupted(ctx context.Context, reason string, now time.Time) (int, error) {
	tag, err := r.db.Exec(ctx, `UPDATE jobs SET status = $1, error = $2, ended_at = $3
		WHERE status IN ($4, $5)`,
		job.StatusFailed, reason, now, job.StatusPending, job.StatusRunning)
	if err != nil {
		return 0, fmt.Errorf("postgres: fail interrupted jobs: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// scanJob อ่านหนึ่งแถวตามลำดับ jobColumns
func scanJob(row pgx.Row) (*job.Job, error) {
	var (
		j   job.Job
		cfg []byte
	)
	if err := row.Scan(&j.ID, &j.BotName, &cfg, &j.TimeoutSeconds, &j.Status, &j.Summary, &j.Error,
		&j.CreatedAt, &j.StartedAt, &j.EndedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cfg, &j.Config); err != nil {
		return nil, fmt.Errorf("decode config of job %s: %w", j.ID, err)
	}
	return &j, nil
}

// marshalConfig แปลง Config เป็น JSON สำหรับ column jsonb — nil map เก็บเป็น {} เพื่อไม่ให้
// ชน NOT NULL
func marshalConfig(cfg map[string]string) ([]byte, error) {
	if cfg == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: encode job config: %w", err)
	}
	return b, nil
}
