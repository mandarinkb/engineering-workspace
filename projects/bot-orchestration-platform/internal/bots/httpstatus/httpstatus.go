// Package httpstatus คือ bot ตัวอย่างที่ง่ายที่สุดในระบบ: ยิง HTTP GET ไปที่ URL ที่กำหนด
// แล้วรายงานว่า response กลับมาเป็น status code อะไร ใช้เวลานานแค่ไหน — จุดประสงค์คือ
// พิสูจน์ว่า bot.Bot interface ใช้งานได้จริงโดยไม่ต้องพึ่ง library ภายนอกเลย (ใช้แค่
// standard library ของ Go) เป็น bot ประเภท "monitor" (เช็คว่าเว็บยังตอบสนองอยู่ไหม)
// ไม่ใช่ scraper — bot สำหรับ scrape ข้อมูลจริงจัง (ที่ใช้ colly/chromedp) จะเสียบเข้า
// ระบบด้วยวิธีเดียวกันนี้เป๊ะๆ ดู FULLSTACK-INFRA-ROADMAP.md Phase 0.1
package httpstatus

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"bop/internal/bot"
)

// Bot คือ implementation ของ bot.Bot สำหรับเช็คสถานะ URL
type Bot struct{}

// New สร้าง instance ของ bot นี้พร้อม register เข้า worker pool ได้เลย (ไม่มี
// dependency ภายในต้อง config อะไรเลย เพราะ bot นี้ไม่เก็บ state ใดๆ)
func New() *Bot { return &Bot{} }

// Name คืนชื่อของ bot นี้ ใช้เป็น key ตอน register และเป็นค่าที่ client ต้องส่งมา
// ตอนสร้าง job (เช่น POST /bots/http-status-checker/jobs)
func (b *Bot) Name() string { return "http-status-checker" }

// Run คือ logic การทำงานจริงของ bot นี้ ขั้นตอน:
//  1. อ่านค่า "url" จาก config ที่ user ส่งมา ถ้าไม่มีหรือว่างเปล่าให้ error ทันที
//  2. สร้าง HTTP request แบบผูกกับ ctx (สำคัญมาก: ถ้า ctx ถูกยกเลิกหรือหมดเวลาระหว่าง
//     กำลังรอ response อยู่ http.DefaultClient.Do จะ return error ออกมาทันที ไม่ค้างรอ
//     จนกว่า network timeout ปกติ)
//  3. จับเวลาก่อน-หลังยิง request เพื่อคำนวณ latency
//  4. รายงานผลผ่าน log ทั้งก่อนยิงและหลังได้ response กลับมา (ให้ user เห็นความคืบหน้า
//     แบบ real-time ถ้าดูผ่าน live log)
//  5. คืนค่า Result ที่มีทั้งข้อความสรุปอ่านง่าย (Summary) และข้อมูลดิบ (Data) สำหรับ
//     เอาไปใช้ต่อ (เช่น เก็บสถิติ status code ย้อนหลัง)
func (b *Bot) Run(ctx context.Context, log bot.LogFunc, cfg bot.Config) (bot.Result, error) {
	url, ok := cfg["url"]
	if !ok || url == "" {
		return bot.Result{}, bot.NewConfigError(fmt.Errorf("http-status-checker: missing required config key %q", "url"))
	}

	log(fmt.Sprintf("requesting %s", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return bot.Result{}, fmt.Errorf("http-status-checker: build request: %w", err)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return bot.Result{}, fmt.Errorf("http-status-checker: request failed: %w", err)
	}
	defer resp.Body.Close() // ต้องปิด response body เสมอ ไม่งั้น connection จะรั่วไหลสะสม (connection leak)
	latency := time.Since(start)

	log(fmt.Sprintf("received %d after %s", resp.StatusCode, latency.Round(time.Millisecond)))

	return bot.Result{
		Summary: fmt.Sprintf("%s → %d in %s", url, resp.StatusCode, latency.Round(time.Millisecond)),
		Data: map[string]any{
			"status_code": resp.StatusCode,
			"latency_ms":  latency.Milliseconds(),
		},
	}, nil
}
