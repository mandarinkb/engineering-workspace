package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

// TestServe_ShutsDownPromptlyWithOpenStream ทดสอบว่า serve ปิดตัวได้ทันทีเมื่อ ctx ถูก
// ยกเลิก แม้จะมี request แบบ stream (เหมือน SSE live log) ที่รอ r.Context() ค้างอยู่ —
// ถ้าไม่ผูก BaseContext ไว้ serve จะค้างรอจนหมด shutdownTimeout (30 วินาที)
func TestServe_ShutsDownPromptlyWithOpenStream(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()
	t.Setenv("BOP_HTTP_ADDR", addr)

	streaming := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(streaming)
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- serve(ctx, handler) }()

	var resp *http.Response
	for range 50 { // รอให้ server เปิด port ก่อน
		if resp, err = http.Get("http://" + addr + "/stream"); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer resp.Body.Close()
	<-streaming

	cancel()
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not shut down while a stream was open")
	}
}
