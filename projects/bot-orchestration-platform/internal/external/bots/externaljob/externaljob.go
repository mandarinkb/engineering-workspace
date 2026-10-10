// Package externaljob คือ bot ที่ "ห่อ" (wrap) Go binary ภายนอกที่คอมไพล์แยกต่างหากจาก
// BOP เอาไว้แล้ว (ไม่ใช่ source code ส่วนหนึ่งของ repo นี้) ให้สั่งรันผ่าน BOP ได้เหมือน bot
// ตัวอื่นๆ ทุกประการ (job เดี่ยว/workflow step/schedule) โดย "ไม่แก้โค้ดของ binary เดิมเลย
// แม้แต่บรรทัดเดียว" — BOP มองมันเป็น "opaque external program" ที่คุยกันผ่าน stdin/stdout/
// exit code เท่านั้น (เหมือนที่ shell ทั่วไปเรียกโปรแกรมอื่น ไม่รู้จักโครงสร้างภายในเลย)
//
// เหตุผลที่ทำเป็น bot เดียวทั่วไป (generic) แทนที่จะทำ 1 bot ต่อ 1 jobcode: binary เดิมมี
// jobcode หลายตัวรวมอยู่ในไฟล์เดียว เลือกด้วย flag --jobcode ตอนรัน (เช่น
// `./your-binary --jobcode=SEND_REPORT`) — ถ้าเพิ่ม jobcode ใหม่เข้าไปในไฟล์ binary เดิม
// ทีหลัง ผู้ใช้แค่กรอกชื่อ jobcode ใหม่ในฟอร์มของ dashboard ได้เลยทันที ไม่ต้องแก้/build/
// deploy โค้ดของ BOP ใหม่เลย (ดูรายละเอียดการตัดสินใจที่ EXTERNAL-JOB-BOT-TODO.md)
package externaljob

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"bop/internal/bot"
)

// defaultBinaryPath คือ path ของ binary ภายนอกบนเครื่อง/container ที่ process ของ BOP
// (cmd/bop สำหรับ job เดี่ยว, cmd/bop-worker สำหรับ workflow/schedule) กำลังรันอยู่ —
// ตอนนี้ยังไม่มี binary จริงให้ทดสอบ (ตัดสินใจไว้ตอนวางแผนแล้วว่าใช้ placeholder ไปก่อน)
// จึง hardcode เป็นค่าคงที่ตรงนี้ตาม pattern เดียวกับค่าคงที่อื่นๆ ในโปรเจกต์นี้ (เช่น
// addr ":8080" ใน cmd/bop/main.go) — ไม่ทำ config file/env var system ใหม่จนกว่าจะมี
// use case จริงที่ต้องเปลี่ยน path ได้โดยไม่ build ใหม่ (YAGNI) เปลี่ยนค่านี้ตรงๆ ตอนรู้
// path จริงของ binary แล้ว **สำคัญ**: binary ต้องอยู่ที่ path นี้บนทุกเครื่อง/container ที่
// รัน cmd/bop และ cmd/bop-worker เสมอ (ดู README.md หัวข้อ externaljob สำหรับรายละเอียด
// การ containerize)
const defaultBinaryPath = "/opt/bop/bin/external-job-runner"

// Bot คือ implementation ของ bot.Bot สำหรับ shell out ไปเรียก binary ภายนอก —
// เก็บ binaryPath เป็น field (ไม่ใช้ defaultBinaryPath ตรงๆ ใน Run) เพื่อให้ unit test
// สลับไปใช้ script ปลอมแทน binary จริงได้ (ดู externaljob_test.go ที่สร้าง *Bot ด้วย
// struct literal ตรงๆ เพราะอยู่ package เดียวกัน เข้าถึง field ที่ไม่ export ได้) โดยที่
// New() (ทางเข้าใช้งานจริงจาก cmd/bop, cmd/bop-worker) ยังคง hardcode ไปที่
// defaultBinaryPath เหมือนเดิมทุกประการ
type Bot struct {
	binaryPath string
}

// New สร้าง instance ของ bot นี้ ผูกกับ defaultBinaryPath เสมอ — เรียกจาก cmd/bop และ
// cmd/bop-worker ตอน register bot (ดู README.md)
func New() *Bot {
	return &Bot{binaryPath: defaultBinaryPath}
}

// Name คืนชื่อของ bot นี้ ใช้เป็น key ตอน register และเป็นค่าที่ client ต้องส่งมาตอน
// สร้าง job (เช่น POST /bots/external-job/jobs)
func (b *Bot) Name() string { return "external-job" }

// ConfigSchema ประกาศว่า bot นี้ต้องการ config field 2 ตัว:
//   - jobcode (บังคับกรอก) — ตรงกับ flag --jobcode ที่ binary เดิมรับอยู่แล้ว ปล่อยเป็น
//     free text ธรรมดา ไม่ทำ dropdown/enum เพราะยังไม่รู้ว่า jobcode มีลิสต์ตายตัวชัดเจน
//     แค่ไหน (YAGNI — ตัดสินใจไว้ตอนวางแผนแล้ว เพิ่ม dropdown ทีหลังได้ถ้าจำเป็นจริง)
//   - extra_args (ไม่บังคับ) — jobcode บางตัวต้องการ flag/parameter เพิ่มนอกจาก --jobcode
//     (เช่น --output=/tmp/report.csv) เก็บเป็น free text string เดียว คั่นด้วยช่องว่าง
//     แล้ว split เองตอน Run (ดูฟังก์ชัน strings.Fields ด้านล่าง) — เลือกทางนี้แทนที่จะ
//     ออกแบบ "schema ของ schema" ที่เปลี่ยน field ตาม jobcode ที่เลือก เพราะซับซ้อนกว่า
//     มากและยังไม่มี use case จริงที่ต้องการ (ตัดสินใจไว้ตอนวางแผนแล้วเช่นกัน)
func (b *Bot) ConfigSchema() []bot.ConfigField {
	return []bot.ConfigField{
		{Key: "jobcode", Label: "Job Code", Type: bot.ConfigFieldText, Required: true},
		{Key: "extra_args", Label: "Extra Arguments", Type: bot.ConfigFieldText, Required: false},
	}
}

// Run คือ logic การทำงานจริง — shell out ไปเรียก binary ภายนอกด้วย exec.CommandContext
// แล้ว stream stdout/stderr กลับมาเป็น log แบบ real-time ผ่าน log ที่ส่งเข้ามา
//
// ขั้นตอนการทำงาน:
//  1. อ่าน "jobcode" จาก config (บังคับ) และ "extra_args" (ไม่บังคับ) — ถ้าไม่มี jobcode
//     ถือเป็น config ผิด (ผู้ใช้กรอกไม่ครบ) จึงคืน bot.ConfigError (non-retryable — ต่อให้
//     retry กี่ครั้งก็ยังไม่มี jobcode เหมือนเดิม ไม่มีทางสำเร็จ)
//  2. สร้าง exec.Cmd ผ่าน exec.CommandContext(ctx, ...) — จุดสำคัญ: การผูก ctx ตรงนี้ทำให้
//     ถ้า ctx ถูกยกเลิก (ผู้ใช้กด cancel job หรือ timeout) Go จะสั่ง kill process ลูกให้เอง
//     อัตโนมัติทันที ไม่ต้องเขียน logic ดักจับ/kill เองเลย ตรงกับ contract ของ bot.Bot.Run
//     ที่ต้อง "หยุดทำงานทันทีถ้า context ถูกยกเลิก"
//  3. เปิด stdout/stderr เป็น pipe แยกกัน แล้วอ่านทั้งสองพร้อมกันคนละ goroutine ผ่าน
//     bufio.Scanner (อ่านทีละบรรทัด) ส่งแต่ละบรรทัดเข้า log() ทันทีที่อ่านได้ — เหตุผลที่
//     ต้องแยก goroutine กันระหว่าง stdout กับ stderr: ถ้าอ่านสอง pipe นี้แบบเรียงลำดับ
//     (stdout ให้จบก่อนค่อยอ่าน stderr) แล้ว process ลูกเขียนลง stderr เต็ม buffer ของ OS
//     ระหว่างที่เรายังไม่ได้อ่าน (เพราะติดรออ่าน stdout อยู่) process ลูกจะ "บล็อค" ตอน
//     เขียน stderr ต่อไม่ได้ ทำให้ทั้งคู่ค้างรอกันไปเรื่อยๆ (deadlock) — การอ่านคู่ขนานกัน
//     แก้ปัญหานี้เพราะไม่มี pipe ไหนที่ไม่มีคนอ่านเลย
//  4. ใช้ sync.WaitGroup รอให้ทั้งสอง goroutine อ่านจนจบ (EOF เมื่อ process ปิด pipe ตอน
//     จบการทำงาน) ก่อนค่อยเรียก cmd.Wait() เพื่อรอ process จบจริงๆ และได้ exit code —
//     ต้องรอให้อ่าน pipe จบก่อน Wait() เสมอ (เอกสารของ os/exec ระบุไว้ชัดว่าห้ามเรียก Wait
//     ก่อนอ่าน pipe จบ ไม่งั้นข้อมูลที่เหลือค้างใน pipe จะหายไป)
//  5. แปล exit code เป็นผลลัพธ์: exit code 0 = สำเร็จ, อื่นๆ = ล้มเหลว — คืน error ธรรมดา
//     (ไม่ใช่ bot.ConfigError) เพราะตัดสินใจไว้แล้วว่า exit code ที่ไม่ใช่ 0 ถือเป็นปัญหา
//     ที่ "อาจ" เกิดจากเหตุชั่วคราวได้ (เช่น dependency ภายนอกของ binary เองล่มชั่วคราว)
//     จึงยอมให้ Temporal/worker pool retry ตามปกติ ต่างจาก config ผิดที่ retry ไปก็ผล
//     เดิมแน่นอน
func (b *Bot) Run(ctx context.Context, log bot.LogFunc, cfg bot.Config) (bot.Result, error) {
	jobcode, ok := cfg["jobcode"]
	if !ok || jobcode == "" {
		return bot.Result{}, bot.NewConfigError(fmt.Errorf("external-job: missing required config key %q", "jobcode"))
	}

	args := []string{"--jobcode=" + jobcode}
	if extra := cfg["extra_args"]; extra != "" {
		args = append(args, strings.Fields(extra)...) // strings.Fields ตัดด้วยช่องว่าง (รองรับหลาย space/tab ติดกันด้วย)
	}

	cmd := exec.CommandContext(ctx, b.binaryPath, args...)

	// ค่า default ของ exec.CommandContext ตอน ctx ถูกยกเลิก คือ kill แค่ "process ลูกโดยตรง"
	// (ตัว binary/script ที่เรียกเอง) เพียงตัวเดียว — ถ้า process นั้นเปิด process ลูกของ
	// ตัวเองต่ออีกที (เช่น shell script ที่เรียก sleep/curl/subprocess อื่นแบบไม่ได้ exec
	// แทนที่ตัวเอง) การ kill แค่ตัวแรกจะทิ้ง grandchild เหล่านั้นไว้ให้ทำงานต่อไปเรื่อยๆ
	// เป็น orphan (reparent ไปอยู่ใต้ init) — ที่ร้ายกว่านั้นคือ grandchild ยังถือ file
	// descriptor ของ stdout/stderr pipe ที่สืบทอดมาจาก parent อยู่ ทำให้ wg.Wait() ด้านล่าง
	// ค้างรอ EOF ไม่มีวันมาถึงจนกว่า grandchild จะจบทำงานเอง (พิสูจน์จริงจาก test
	// TestRun_ContextCancelKillsProcess: sleep 5 ที่ยิงจาก script ไม่ตายตาม script แม่เลย)
	//
	// วิธีแก้: ตั้งให้ process ลูกเป็นผู้นำ process group ใหม่ของตัวเอง (Setpgid: true) แล้ว
	// ตอน cancel ยิง SIGKILL ไปที่ "process group" ทั้งกลุ่ม (pid ติดลบ = สัญญาณไปทั้งกลุ่ม
	// ตาม convention ของ Unix) แทนที่จะ kill แค่ process เดียว — ทำให้ grandchild ทุกตัวที่
	// เกิดจาก process นี้ตายไปพร้อมกันด้วย ไม่เหลือ orphan process ค้างเลย
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return bot.Result{}, fmt.Errorf("external-job: open stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return bot.Result{}, fmt.Errorf("external-job: open stderr pipe: %w", err)
	}

	log(fmt.Sprintf("starting: %s %s", b.binaryPath, strings.Join(args, " ")))

	if err := cmd.Start(); err != nil {
		return bot.Result{}, fmt.Errorf("external-job: start process: %w", err)
	}

	// สตรีม stdout และ stderr เข้า log() พร้อมกันคนละ goroutine (ดูเหตุผลเรื่อง deadlock
	// ในคอมเมนต์ด้านบน) — ใส่ prefix "[stderr]" ให้บรรทัดจาก stderr เพื่อให้ผู้ดู log
	// แยกออกว่าบรรทัดไหนมาจาก stream ไหน (log() ตัวเดียวไม่มีช่องทางบอก stream แยกกัน)
	var wg sync.WaitGroup
	streamToLog := func(r io.Reader, prefix string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if prefix != "" {
				log(prefix + scanner.Text())
			} else {
				log(scanner.Text())
			}
		}
	}
	wg.Add(2)
	go streamToLog(stdout, "")
	go streamToLog(stderr, "[stderr] ")
	wg.Wait() // ต้องรอ pipe อ่านจบก่อนเรียก cmd.Wait() เสมอ (ดูคอมเมนต์ข้อ 4 ด้านบน)

	if err := cmd.Wait(); err != nil {
		return bot.Result{}, fmt.Errorf("external-job: jobcode %q exited with error: %w", jobcode, err)
	}

	log(fmt.Sprintf("jobcode %q finished successfully", jobcode))

	return bot.Result{
		Summary: fmt.Sprintf("jobcode %s completed", jobcode),
		Data: map[string]any{
			"jobcode":   jobcode,
			"exit_code": 0,
		},
	}, nil
}
