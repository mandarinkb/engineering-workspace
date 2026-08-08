// ไฟล์นี้อยู่ package externaljob เดียวกับ externaljob.go (ไม่ใช่ externaljob_test แบบ
// black-box) ตั้งใจเพื่อให้ test เข้าถึง field ที่ไม่ export (binaryPath) ได้ตรงๆ ผ่าน
// struct literal — เป็นวิธีเดียวที่สลับไปใช้ script ปลอมแทน binary จริงได้โดยไม่ต้องเพิ่ม
// public API (constructor ที่รับ path) ที่ผู้ใช้จริง (cmd/bop, cmd/bop-worker) ไม่ต้องการเลย
// (New() ยังคง hardcode ไปที่ defaultBinaryPath เหมือนเดิมทุกประการ ดู externaljob.go)
package externaljob

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"bop/internal/bot"
)

// fakeScript สร้าง shell script ปลอมไว้ใน t.TempDir() (ลบตัวเองอัตโนมัติหลัง test จบ)
// แทนที่ binary จริงของเจ้าของ repo (ยังไม่มีให้ใช้ตอนเขียน test นี้) — script รับ
// --jobcode=<code> แล้วเลือกพฤติกรรมตาม code:
//
//	OK      -> echo ลง stdout 1 บรรทัด แล้ว exit 0 (จำลอง job สำเร็จ)
//	FAIL    -> echo ลง stderr 1 บรรทัด แล้ว exit 3 (จำลอง job ล้มเหลว)
//	SLEEP   -> sleep 5 วินาที (จำลอง job ที่ทำงานนาน ใช้ทดสอบการยกเลิก)
//	LOGGY   -> echo สลับ stdout/stderr หลายบรรทัด (ใช้ทดสอบ log streaming)
//	ARGS    -> echo argument ทั้งหมดที่ได้รับ (ใช้ทดสอบว่า extra_args ถูกส่งต่อจริง)
//
// เป็น POSIX sh ธรรมดา ไม่พึ่ง bash เพื่อให้รันได้บน CI ทั่วไป
func fakeScript(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-binary.sh")
	script := `#!/bin/sh
jobcode=""
for arg in "$@"; do
  case "$arg" in
    --jobcode=*) jobcode="${arg#--jobcode=}" ;;
  esac
done

case "$jobcode" in
  OK)
    echo "ok output"
    exit 0
    ;;
  FAIL)
    echo "boom" >&2
    exit 3
    ;;
  SLEEP)
    sleep 5
    exit 0
    ;;
  LOGGY)
    echo "out-1"
    echo "err-1" >&2
    echo "out-2"
    echo "err-2" >&2
    exit 0
    ;;
  ARGS)
    echo "args: $*"
    exit 0
    ;;
  *)
    echo "unknown jobcode: $jobcode" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}
	return path
}

// collectLog คืน bot.LogFunc ที่เก็บทุกบรรทัดที่ถูก log เข้า slice (thread-safe พอสำหรับ
// test นี้เพราะ externaljob.Run เรียก log จากหลาย goroutine พร้อมกัน — ต้อง sync)
func collectLog(t *testing.T) (bot.LogFunc, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var lines []string
	fn := func(msg string) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, msg)
	}
	get := func() []string {
		mu.Lock()
		defer mu.Unlock()
		out := make([]string, len(lines))
		copy(out, lines)
		return out
	}
	return fn, get
}

// TestRun_Success ทดสอบเส้นทางปกติ: jobcode ที่ exit 0 ต้องคืนผลสำเร็จ ไม่มี error
func TestRun_Success(t *testing.T) {
	b := &Bot{binaryPath: fakeScript(t)}
	log, getLines := collectLog(t)

	result, err := b.Run(context.Background(), log, bot.Config{"jobcode": "OK"})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result.Data["jobcode"] != "OK" {
		t.Errorf("expected result data jobcode=OK, got %+v", result.Data)
	}

	found := false
	for _, l := range getLines() {
		if l == "ok output" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stdout line %q to be logged, got %v", "ok output", getLines())
	}
}

// TestRun_MissingJobcode ทดสอบว่าไม่ส่ง jobcode มาเลย ต้องได้ bot.ConfigError (non-retryable)
// ไม่ใช่ error ธรรมดา — ตรงกับ Open Question ข้อ 4 ที่ตัดสินใจไว้ว่า config ผิด/ขาดต้องแยก
// ประเภทจาก error ที่เกิดตอนรันจริง
func TestRun_MissingJobcode(t *testing.T) {
	b := &Bot{binaryPath: fakeScript(t)}
	log, _ := collectLog(t)

	_, err := b.Run(context.Background(), log, bot.Config{})
	if err == nil {
		t.Fatal("expected error for missing jobcode")
	}
	var cfgErr *bot.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected *bot.ConfigError, got %T: %v", err, err)
	}
}

// TestRun_NonZeroExitIsOrdinaryError ทดสอบว่า jobcode ที่ exit code ไม่ใช่ 0 คืน error
// ธรรมดา (ไม่ใช่ bot.ConfigError) — ตรงกับ Open Question ข้อ 4: retryable เพราะอาจเป็น
// ปัญหาชั่วคราวของ binary เอง ไม่ใช่ config ที่ผู้ใช้กรอกผิด
func TestRun_NonZeroExitIsOrdinaryError(t *testing.T) {
	b := &Bot{binaryPath: fakeScript(t)}
	log, getLines := collectLog(t)

	_, err := b.Run(context.Background(), log, bot.Config{"jobcode": "FAIL"})
	if err == nil {
		t.Fatal("expected error for non-zero exit code")
	}
	var cfgErr *bot.ConfigError
	if errors.As(err, &cfgErr) {
		t.Fatalf("expected ordinary (retryable) error, got bot.ConfigError: %v", err)
	}

	found := false
	for _, l := range getLines() {
		if l == "[stderr] boom" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stderr line to be logged with [stderr] prefix, got %v", getLines())
	}
}

// TestRun_ContextCancelKillsProcess ทดสอบว่า ctx ถูกยกเลิกระหว่างที่ process ลูกกำลังทำงาน
// อยู่ (jobcode SLEEP ใช้เวลา 5 วินาที) ทำให้ Run คืนกลับเร็วกว่านั้นมาก (พิสูจน์ว่า
// exec.CommandContext kill process ลูกจริง ไม่ปล่อยให้ทำงานจนจบ) — pattern เดียวกับ
// TestPool_CancelStopsAJobInFlight ใน internal/worker/pool_test.go
func TestRun_ContextCancelKillsProcess(t *testing.T) {
	b := &Bot{binaryPath: fakeScript(t)}
	log, _ := collectLog(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := b.Run(ctx, log, bot.Config{"jobcode": "SLEEP"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error after context cancellation")
	}
	if elapsed >= 5*time.Second {
		t.Fatalf("expected Run to return promptly after cancel (process should be killed), took %s", elapsed)
	}
}

// TestRun_StreamsBothStdoutAndStderr ทดสอบว่าทุกบรรทัดจากทั้ง stdout และ stderr ถูกส่งเข้า
// log() ครบ ไม่มีบรรทัดหาย — ไม่เช็ค ordering ข้าม stream กัน (stdout/stderr เป็นคนละ
// goroutine อ่านพร้อมกัน ลำดับสลับกันได้ตามธรรมชาติ) เช็คแค่ว่าครบทุกบรรทัดที่ควรมี
func TestRun_StreamsBothStdoutAndStderr(t *testing.T) {
	b := &Bot{binaryPath: fakeScript(t)}
	log, getLines := collectLog(t)

	_, err := b.Run(context.Background(), log, bot.Config{"jobcode": "LOGGY"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]bool{
		"out-1":         false,
		"[stderr] err-1": false,
		"out-2":         false,
		"[stderr] err-2": false,
	}
	for _, l := range getLines() {
		if _, ok := want[l]; ok {
			want[l] = true
		}
	}
	for line, seen := range want {
		if !seen {
			t.Errorf("expected line %q to be logged, got %v", line, getLines())
		}
	}
}

// TestRun_ExtraArgsPassedThrough ทดสอบว่า extra_args ถูก split ด้วยช่องว่างแล้วส่งต่อเป็น
// argument เพิ่มให้ process ลูกจริง (นอกจาก --jobcode)
func TestRun_ExtraArgsPassedThrough(t *testing.T) {
	b := &Bot{binaryPath: fakeScript(t)}
	log, getLines := collectLog(t)

	_, err := b.Run(context.Background(), log, bot.Config{"jobcode": "ARGS", "extra_args": "--foo=1 --bar=2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, l := range getLines() {
		if l == "args: --jobcode=ARGS --foo=1 --bar=2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected extra_args to be passed through to the process, got %v", getLines())
	}
}
