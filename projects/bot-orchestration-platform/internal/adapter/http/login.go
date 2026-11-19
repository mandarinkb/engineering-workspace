package http

import (
	"encoding/json"
	"errors"
	"net/http"

	authuc "bop/internal/usecase/auth"
)

// cookieName คือชื่อ cookie ที่เก็บ JWT — ใช้ชื่อเดียวกันทั้งตอน set (handleLogin),
// ตอนอ่าน (handleMe), และตอนล้าง (handleLogout)
const cookieName = "bop_session"

// loginRequest คือ JSON body ที่ POST /login ต้องรับ
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleLogin ตรวจ username/password ผ่าน usecase auth ถ้าถูกต้องได้ JWT กลับมา แล้วฝังใน
// httpOnly cookie ส่งกลับไป
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

	token, err := s.auth.Login(req.Username, req.Password)
	if errors.Is(err, authuc.ErrInvalidCredentials) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
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
		MaxAge:   int(authuc.TokenTTL.Seconds()),
	})
	w.WriteHeader(http.StatusNoContent)
}
