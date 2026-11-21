package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"
)

// TestAuthFlow ทดสอบเส้นทางเต็มของ login: username/password ผิด -> 401, ถูก -> ได้
// cookie กลับมาแล้ว GET /me ต้องเห็น username เดียวกัน, logout แล้ว /me ต้อง 401 อีกครั้ง
func TestAuthFlow(t *testing.T) {
	ts := newAnonymousTestServer(t)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}

	// 1) credential ผิดต้องโดน 401 และไม่ได้ cookie ติดมือกลับมา
	badResp, err := client.Post(ts.URL+"/login", "application/json", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
	if err != nil {
		t.Fatalf("POST /login (bad): %v", err)
	}
	badResp.Body.Close()
	if badResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad credentials, got %d", badResp.StatusCode)
	}

	meResp, err := client.Get(ts.URL + "/me")
	if err != nil {
		t.Fatalf("GET /me (before login): %v", err)
	}
	meResp.Body.Close()
	if meResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 before login, got %d", meResp.StatusCode)
	}

	// 2) credential ถูกต้องต้องได้ cookie แล้ว /me ต้องเห็น username ตรงกัน
	goodResp, err := client.Post(ts.URL+"/login", "application/json", bytes.NewBufferString(`{"username":"admin","password":"changeme"}`))
	if err != nil {
		t.Fatalf("POST /login (good): %v", err)
	}
	goodResp.Body.Close()
	if goodResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for good credentials, got %d", goodResp.StatusCode)
	}

	meResp2, err := client.Get(ts.URL + "/me")
	if err != nil {
		t.Fatalf("GET /me (after login): %v", err)
	}
	defer meResp2.Body.Close()
	if meResp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after login, got %d", meResp2.StatusCode)
	}
	var me struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(meResp2.Body).Decode(&me); err != nil {
		t.Fatalf("decode /me: %v", err)
	}
	if me.Username != "admin" {
		t.Errorf("expected username %q, got %q", "admin", me.Username)
	}

	// 3) logout แล้ว /me ต้อง 401 อีกครั้ง
	logoutResp, err := client.Post(ts.URL+"/logout", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /logout: %v", err)
	}
	logoutResp.Body.Close()

	meResp3, err := client.Get(ts.URL + "/me")
	if err != nil {
		t.Fatalf("GET /me (after logout): %v", err)
	}
	meResp3.Body.Close()
	if meResp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", meResp3.StatusCode)
	}
}
