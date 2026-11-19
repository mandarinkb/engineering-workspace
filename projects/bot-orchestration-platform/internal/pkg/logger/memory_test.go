package logger_test

import (
	"context"
	"sync"
	"testing"

	"bop/internal/domain/logger"
	memlogger "bop/internal/pkg/logger"
)

// TestLogger_UnsubscribeWhileLogging ทดสอบว่า Log กับ unsubscribe ที่เกิดพร้อมกัน (เช่น
// bot กำลังเขียน log อยู่ตอนที่ผู้ใช้ปิดหน้า live log) ต้องไม่ทำให้ส่งข้อมูลเข้า channel ที่
// ปิดไปแล้ว ซึ่งจะ panic และทำให้ process ล่มทั้งตัว
func TestLogger_UnsubscribeWhileLogging(_ *testing.T) {
	l := memlogger.New()
	ctx := context.Background()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					l.Log(ctx, "job-1", logger.LevelInfo, "tick")
				}
			}
		}()
	}

	for range 2000 {
		_, unsubscribe := l.Subscribe("job-1")
		unsubscribe()
	}
	close(stop)
	wg.Wait()
}

// TestLogger_SubscriberReceivesLines ทดสอบพฤติกรรมปกติ: subscriber ได้รับบรรทัดที่ log
// หลังจาก subscribe และ channel ถูกปิดหลัง unsubscribe
func TestLogger_SubscriberReceivesLines(t *testing.T) {
	l := memlogger.New()
	ch, unsubscribe := l.Subscribe("job-1")

	l.Log(context.Background(), "job-1", logger.LevelInfo, "hello")
	line := <-ch
	if line.Message != "hello" || line.JobID != "job-1" {
		t.Fatalf("got %+v, want message hello for job-1", line)
	}

	unsubscribe()
	if _, ok := <-ch; ok {
		t.Fatal("channel still open after unsubscribe")
	}
	unsubscribe() // เรียกซ้ำต้องไม่ panic
}
