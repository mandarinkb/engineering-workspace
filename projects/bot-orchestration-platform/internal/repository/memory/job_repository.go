// Package memory คือ implementation ของ job.Repository แบบเก็บข้อมูลไว้ใน memory
// ล้วนๆ (ไม่มี database จริงข้างหลัง) — มีไว้เพื่อให้พัฒนาและทดสอบระบบได้ทันทีโดยไม่ต้อง
// ติดตั้งอะไรเพิ่ม ข้อเสียคือข้อมูลจะหายหมดทุกครั้งที่ปิดโปรแกรม (เหมาะกับ dev เท่านั้น
// ไม่ใช่ production)
package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"bop/internal/job"
)

// JobRepository เก็บ job ทั้งหมดไว้ใน map ธรรมดาในหน่วยความจำ ป้องกันการเข้าถึงพร้อมกัน
// จากหลาย goroutine ด้วย sync.RWMutex (อ่านพร้อมกันได้หลาย goroutine แต่เขียนได้ทีละคนเท่านั้น)
type JobRepository struct {
	mu   sync.RWMutex        // ป้องกันไม่ให้หลาย goroutine เข้าถึง map ด้านล่างพร้อมกันจนข้อมูลเสีย
	jobs map[string]*job.Job // เก็บ job ทั้งหมด โดยใช้ job ID เป็น key
}

// NewJobRepository สร้าง repository เปล่าๆ พร้อมใช้งาน
func NewJobRepository() *JobRepository {
	return &JobRepository{jobs: make(map[string]*job.Job)}
}

// หมายเหตุสำคัญเกี่ยวกับทุก method ด้านล่าง: ทุกตัวจะ "copy" ค่า Job ก่อนเก็บ/คืนค่าเสมอ
// ไม่เคยส่ง pointer ตัวเดิมที่ผู้เรียกส่งเข้ามา (ตอน Create/Update) หรือ pointer ที่เก็บไว้
// ใน map ออกไปให้ผู้เรียก (ตอน Get/List) โดยตรง — เหตุผลคือ job.Job ถูกแก้ไข field ต่างๆ
// ระหว่างที่ worker pool กำลังรัน job นั้นอยู่ (ดู internal/worker) ถ้าปล่อยให้ pointer
// เดียวกันถูกถือโดยหลาย goroutine พร้อมกัน (เช่น worker กำลังเขียน ในขณะที่ HTTP handler
// กำลังอ่านเพื่อส่ง response) จะเกิด data race ทันที — การ copy ทุกครั้งที่ผ่านเข้า-ออก
// repository ทำให้ mutex ตัวนี้เป็นจุดเดียวที่ต้องป้องกัน ผู้เรียกที่ถือ Job ที่ได้จาก Get/List
// ไปแล้วจะไม่มีทางไปชนกับสิ่งที่ worker pool กำลังทำอยู่เลย เพราะเป็นคนละ struct กันแล้ว

// Create บันทึก job ใหม่ จะ error ถ้ามี ID นี้อยู่แล้วในระบบ
func (r *JobRepository) Create(_ context.Context, j *job.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.jobs[j.ID]; exists {
		return fmt.Errorf("memory: job %s already exists", j.ID)
	}
	cp := *j // copy ค่าทั้งหมดของ struct (ไม่ใช่แค่ copy pointer)
	r.jobs[j.ID] = &cp
	return nil
}

// Update บันทึกทับข้อมูล job ที่มีอยู่แล้วด้วยค่าล่าสุด (ใช้ตอนสถานะของ job เปลี่ยน
// เช่นจาก pending เป็น running หรือ running เป็น succeeded)
func (r *JobRepository) Update(_ context.Context, j *job.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *j
	r.jobs[j.ID] = &cp
	return nil
}

// Get ค้นหา job ด้วย ID คืน error ถ้าไม่พบ
func (r *JobRepository) Get(_ context.Context, id string) (*job.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	j, ok := r.jobs[id]
	if !ok {
		return nil, fmt.Errorf("memory: job %s not found", id)
	}
	cp := *j
	return &cp, nil
}

// List คืนรายการ job ทั้งหมด เรียงจากล่าสุด (สร้างล่าสุด) ไปเก่าสุด — เหมาะกับการเอาไป
// แสดงในหน้า job history ของ dashboard ที่มักอยากเห็น job ล่าสุดอยู่บนสุด
func (r *JobRepository) List(_ context.Context) ([]*job.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*job.Job, 0, len(r.jobs))
	for _, j := range r.jobs {
		cp := *j
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].CreatedAt.After(out[k].CreatedAt) })
	return out, nil
}
