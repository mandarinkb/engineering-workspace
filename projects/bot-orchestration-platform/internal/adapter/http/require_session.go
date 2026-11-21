package http

import (
	"context"
	"net/http"
)

// publicRoutes คือ route เดียวที่เรียกได้โดยไม่ต้องมี session — key คือ pattern ตรงตามที่
// ลงทะเบียนใน routes() (ServeMux.Handler คืน pattern นี้กลับมา) route อื่นทั้งหมด รวมถึง
// route ที่เพิ่มทีหลัง ถูกบังคับ login โดยอัตโนมัติ: จะเปิด route ใหม่ให้ไม่ต้อง login
// ต้องตั้งใจเพิ่มชื่อมันที่นี่เท่านั้น
//   - /healthz, /readyz: K8s probe ไม่มี cookie
//   - /login: ทางเดียวที่จะได้ session มา
//   - /logout: ล้าง cookie ได้เสมอแม้ token หมดอายุแล้ว (dashboard เรียกตอนโดน 401 เพื่อล้าง
//     cookie เสียก่อนพาไปหน้า login)
var publicRoutes = map[string]bool{
	"GET /healthz": true,
	"GET /readyz":  true,
	"POST /login":  true,
	"POST /logout": true,
}

// sessionUserKey คือ key ของ context ที่เก็บชื่อ user ของ session ที่ผ่านการตรวจแล้ว
type sessionUserKey struct{}

// requireSession ตรวจ session ก่อนส่ง request ต่อให้ handler — คืน false (และตอบ 401 ไป
// แล้ว) ถ้า route ไม่อยู่ใน publicRoutes และไม่มี cookie bop_session ที่ valid
//
// ตรวจที่ ServeHTTP จุดเดียวแทนการห่อ handler ทีละตัว เพื่อให้ route ใหม่ที่ลงทะเบียนด้วย
// s.mux.HandleFunc ตามปกติถูกบังคับ login ทันทีโดยไม่ต้องมีใครจำว่าต้องห่อ route ที่ mux
// หาไม่เจอ (404/405) ก็ได้ 401 เหมือนกัน คนที่ยังไม่ login จึงสำรวจไม่ได้ว่ามี route อะไรบ้าง
//
// ไม่ห่อ ResponseWriter เลย http.Flusher ของ SSE (stream_job_logs.go) จึงยังใช้ได้เหมือนเดิม
func (s *Server) requireSession(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	if _, pattern := s.mux.Handler(r); publicRoutes[pattern] {
		return r, true
	}
	username, ok := s.sessionUser(r)
	if !ok {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return r, false
	}
	return r.WithContext(context.WithValue(r.Context(), sessionUserKey{}, username)), true
}

// sessionUser คืนชื่อ user จาก cookie bop_session ถ้า token ถูกต้องและยังไม่หมดอายุ — ไม่
// บอก client ว่าพังเพราะอะไร (ไม่มี cookie, signature ผิด, หมดอายุ) ตอบ 401 เหมือนกันหมด
func (s *Server) sessionUser(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", false
	}
	username, err := s.auth.Verify(cookie.Value)
	if err != nil {
		return "", false
	}
	return username, true
}

// usernameFrom คืนชื่อ user ที่ requireSession ตรวจแล้วใส่ไว้ใน context
func usernameFrom(ctx context.Context) string {
	username, _ := ctx.Value(sessionUserKey{}).(string)
	return username
}
