package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bop/internal/domain/logger"
)

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

	ch, unsubscribe := s.jobs.SubscribeLogs(id)
	defer unsubscribe()

	history, err := s.jobs.Logs(r.Context(), id)
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
