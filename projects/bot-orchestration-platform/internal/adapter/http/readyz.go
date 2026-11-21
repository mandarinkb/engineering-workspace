package http

import (
	"context"
	"net/http"
	"time"
)

// databasePingTimeout จำกัดเวลา ping database ใน /readyz — probe ของ K8s/Docker มี timeout ของ
// ตัวเอง database ที่ค้างต้องตอบ "ไม่พร้อม" ให้ทันแทนที่จะลาก probe ให้หมดเวลา
const databasePingTimeout = 2 * time.Second

// DatabasePinger คือสิ่งเดียวที่ /readyz ต้องการจาก database (pgxpool.Pool มี Ping นี้อยู่แล้ว)
// ประกาศที่นี่เพราะ adapter ชั้นนอกสุดห้าม import ตัว driver ตรงๆ
type DatabasePinger interface {
	Ping(ctx context.Context) error
}

// handleReadyz ตอบ GET /readyz — readiness: process พร้อมรับ traffic จริงไหม (เช็คว่าต่อ
// Temporal ได้อยู่ และถ้าตั้ง database ไว้ ก็ต้อง ping ได้ด้วย) K8s หยุดส่ง traffic มาที่ Pod นี้
// ชั่วคราว (ไม่ restart) ถ้า readiness fail — ต่างจาก liveness ตรงนี้แหละ: Temporal หรือ
// database ล่มชั่วคราวไม่ควรทำให้ Pod ถูก restart วนไปเรื่อยๆ (CrashLoopBackOff) แค่หยุดรับ
// traffic จนกว่าจะกลับมา body บอกสาเหตุ (Temporal เช็คก่อน) ส่วนโหมด memory ไม่มี database
// ให้เช็ค
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.runs.Ready(r.Context()); err != nil {
		http.Error(w, "temporal unreachable", http.StatusServiceUnavailable)
		return
	}
	if s.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), databasePingTimeout)
		defer cancel()
		if err := s.db.Ping(ctx); err != nil {
			http.Error(w, "database unreachable", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
