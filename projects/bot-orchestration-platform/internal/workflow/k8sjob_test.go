// package workflow (ไม่ใช่ workflow_test) เพราะ test บางตัวในไฟล์นี้เรียก unexported
// method/function ตรงๆ (awaitJobCompletion, deleteJob, sanitizeK8sName) — pattern เดียวกับ
// projects/remote-job-worker/internal/jobs/jobs_test.go ที่ทดสอบ runLongTask ตรงๆ โดยไม่
// ผ่าน Temporal test environment เพื่อเลี่ยงปัญหาเดียวกัน (activity.RecordHeartbeat/
// activity.GetInfo ต้องการ activity execution context จริงถึงจะเรียกได้โดยไม่ panic)
//
// หมายเหตุสำคัญ: test ทั้งหมดในไฟล์นี้ใช้ k8s.io/client-go/kubernetes/fake ซึ่งเป็นแค่
// CRUD store เปล่าๆ ไม่มี controller logic ของจริง (ไม่มีใคร transition Job.Status ให้
// อัตโนมัติเหมือน K8s cluster จริง) — test ที่ต้องการให้ Job "สำเร็จ/ล้มเหลว" ต้องจำลอง
// controller เอง (goroutine แยกที่ไปอัปเดต Status ให้ตรงๆ) ไม่ได้พิสูจน์ behavior ของ K8s
// Job controller จริง แค่พิสูจน์ว่าโค้ดฝั่ง RunKubernetesJob ตอบสนองสถานะที่เปลี่ยนไปถูกต้อง
package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	memlogger "bop/internal/logger/memory"
)

// newK8sTestEnv สร้าง TestActivityEnvironment พร้อม register activity function ที่จะ
// ทดสอบไว้แล้ว ภายใต้ชื่อ K8sJobActivityName — จำเป็นสำหรับ test ที่เรียก RunKubernetesJob
// เต็มรูปแบบ เพราะข้างในเรียก activity.GetInfo/activity.RecordHeartbeat ซึ่งต้องการ
// activity execution context จริง (สร้างให้โดย TestActivityEnvironment เท่านั้น)
func newK8sTestEnv(t *testing.T, fn interface{}) *testsuite.TestActivityEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: K8sJobActivityName})
	return env
}

// markFirstJobAs ค้นหา Job แรกใน namespace ที่ระบุ (poll จนกว่า RunKubernetesJob จะสร้าง
// เสร็จ) แล้วอัปเดต Status ให้เป็นสำเร็จ/ล้มเหลวตามที่สั่ง — จำลองพฤติกรรมของ K8s Job
// controller จริงที่ fake clientset ไม่มีให้ ต้อง run เป็น goroutine แยกคู่ขนานกับ
// env.ExecuteActivity ที่กำลัง block รออยู่ (poll ผ่าน awaitJobCompletion)
func markFirstJobAs(t *testing.T, clientset *fake.Clientset, namespace string, succeeded bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		jobs, err := clientset.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
		if err == nil && len(jobs.Items) > 0 {
			j := jobs.Items[0]
			if succeeded {
				j.Status.Succeeded = 1
			} else {
				j.Status.Failed = 1
			}
			if _, err := clientset.BatchV1().Jobs(namespace).UpdateStatus(context.Background(), &j, metav1.UpdateOptions{}); err != nil {
				t.Errorf("update job status: %v", err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("timed out waiting for RunKubernetesJob to create the job")
}

// TestRunKubernetesJob_MissingImage ทดสอบว่าไม่ระบุ Image มาเลย ต้องได้ error ที่ห่อเป็น
// temporal.ApplicationError ชนิด "ConfigError" (non-retryable) — เหตุผลเดียวกับ
// bot.ConfigError ของ httpstatus/externaljob: config ผิด/ขาด retry กี่ครั้งก็ผลลัพธ์เดิม
func TestRunKubernetesJob_MissingImage(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	activities := NewK8sJobActivities(clientset, "default", memlogger.New())
	env := newK8sTestEnv(t, activities.RunKubernetesJob)

	_, err := env.ExecuteActivity(activities.RunKubernetesJob, K8sJobSpec{})
	if err == nil {
		t.Fatal("expected error for missing image")
	}
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected error chain to contain *temporal.ApplicationError, got %T: %v", err, err)
	}
	if appErr.Type() != "ConfigError" {
		t.Errorf("expected ApplicationError Type %q, got %q", "ConfigError", appErr.Type())
	}
}

// TestRunKubernetesJob_Success ทดสอบเส้นทางปกติ: Job ที่ Status.Succeeded>0 ต้องคืนผล
// สำเร็จ พร้อมยืนยันด้วยว่า Job ที่สร้างจริงมี BackoffLimit=0 (ห้ามให้ K8s retry เอง —
// ดูเหตุผลที่ buildJobObject) และ RestartPolicy=Never
func TestRunKubernetesJob_Success(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	activities := NewK8sJobActivities(clientset, "default", memlogger.New())
	activities.pollInterval = 20 * time.Millisecond
	env := newK8sTestEnv(t, activities.RunKubernetesJob)

	go markFirstJobAs(t, clientset, "default", true)

	val, err := env.ExecuteActivity(activities.RunKubernetesJob, K8sJobSpec{Image: "example.com/report-job:v1"})
	if err != nil {
		t.Fatalf("ExecuteActivity: %v", err)
	}
	var summary string
	if err := val.Get(&summary); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}

	jobs, err := clientset.BatchV1().Jobs("default").List(context.Background(), metav1.ListOptions{})
	if err != nil || len(jobs.Items) != 1 {
		t.Fatalf("expected exactly 1 job created, got %d (err=%v)", len(jobs.Items), err)
	}
	job := jobs.Items[0]
	if job.Spec.BackoffLimit == nil || *job.Spec.BackoffLimit != 0 {
		t.Errorf("expected BackoffLimit=0, got %v", job.Spec.BackoffLimit)
	}
	if job.Spec.Template.Spec.RestartPolicy != "Never" {
		t.Errorf("expected RestartPolicy=Never, got %v", job.Spec.Template.Spec.RestartPolicy)
	}
	if job.Spec.Template.Spec.Containers[0].Image != "example.com/report-job:v1" {
		t.Errorf("expected container image to match spec, got %q", job.Spec.Template.Spec.Containers[0].Image)
	}
}

// TestRunKubernetesJob_Failure ทดสอบว่า Job ที่ Status.Failed>0 คืน error ธรรมดา (ไม่ใช่
// ApplicationError ชนิด ConfigError) — K8s ไม่มีแนวคิด "config error" ของตัวเอง แยกแยะ
// ไม่ได้จากภายนอกว่า container fail เพราะอะไร จึงถือเป็น error ธรรมดา/retryable เป็น
// default (เหมือนที่ตัดสินใจไว้กับ externaljob ตอน exit code ไม่ใช่ 0)
func TestRunKubernetesJob_Failure(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	activities := NewK8sJobActivities(clientset, "default", memlogger.New())
	activities.pollInterval = 20 * time.Millisecond
	env := newK8sTestEnv(t, activities.RunKubernetesJob)

	go markFirstJobAs(t, clientset, "default", false)

	_, err := env.ExecuteActivity(activities.RunKubernetesJob, K8sJobSpec{Image: "example.com/report-job:v1"})
	if err == nil {
		t.Fatal("expected error for failed job")
	}
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Type() == "ConfigError" {
		t.Fatalf("expected an ordinary (retryable) error, got non-retryable ConfigError: %v", err)
	}
}

// TestRunKubernetesJob_DefaultNamespace ทดสอบว่า K8sJobSpec.Namespace ว่างเปล่า ทำให้ Job
// ถูกสร้างใน defaultNamespace ที่ตั้งไว้ตอน NewK8sJobActivities (ไม่ใช่ namespace ว่างเปล่า
// หรือ "default" ของ K8s เอง)
func TestRunKubernetesJob_DefaultNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	activities := NewK8sJobActivities(clientset, "bop-jobs", memlogger.New())
	activities.pollInterval = 20 * time.Millisecond
	env := newK8sTestEnv(t, activities.RunKubernetesJob)

	go markFirstJobAs(t, clientset, "bop-jobs", true)

	if _, err := env.ExecuteActivity(activities.RunKubernetesJob, K8sJobSpec{Image: "example.com/report-job:v1"}); err != nil {
		t.Fatalf("ExecuteActivity: %v", err)
	}

	jobs, err := clientset.BatchV1().Jobs("bop-jobs").List(context.Background(), metav1.ListOptions{})
	if err != nil || len(jobs.Items) != 1 {
		t.Fatalf("expected job to be created in default namespace %q, got %d jobs (err=%v)", "bop-jobs", len(jobs.Items), err)
	}
}

// TestAwaitJobCompletion_StopsWhenContextCancelled ทดสอบว่า awaitJobCompletion เช็ค
// ctx.Done() จริงและหยุดทันทีถ้า context ถูกยกเลิกไปแล้วก่อนเริ่มด้วยซ้ำ — เรียก method
// ตรงๆ (ไม่ผ่าน TestActivityEnvironment) เพราะ ctx ที่ยกเลิกไปแล้วทำให้ select เลือก
// ctx.Done() case ทันทีตั้งแต่รอบแรก ไม่มีทางไปถึงบรรทัดที่เรียก activity.RecordHeartbeat
// เลย (ซึ่งต้องพึ่ง activity execution context ของจริงถึงจะเรียกได้โดยไม่ panic) — pattern
// เดียวกับ TestRunLongTask_StopsWhenContextCancelled ของ projects/remote-job-worker
func TestAwaitJobCompletion_StopsWhenContextCancelled(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	activities := NewK8sJobActivities(clientset, "default", memlogger.New())
	activities.pollInterval = 2 * time.Second // ยาวตั้งใจ — ถ้า cancellation ไม่ทำงานจริง test จะค้างจนกว่าจะ timeout

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := activities.awaitJobCompletion(ctx, "default", "some-job")
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("expected awaitJobCompletion to return promptly after cancellation, took %s", elapsed)
	}
}

// TestDeleteJob ทดสอบว่า deleteJob ลบ Job ออกจาก cluster จริง (ใช้ตอน RunKubernetesJob
// ถูกยกเลิกกลางทาง กัน Job กลายเป็น orphan ทำงานต่อทั้งที่ Temporal ถือว่า activity จบแล้ว)
func TestDeleteJob(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	activities := NewK8sJobActivities(clientset, "default", memlogger.New())

	job := buildJobObject("test-job", "default", K8sJobSpec{Image: "example.com/x:1"})
	if _, err := clientset.BatchV1().Jobs("default").Create(context.Background(), job, metav1.CreateOptions{}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	activities.deleteJob("default", "test-job")

	if _, err := clientset.BatchV1().Jobs("default").Get(context.Background(), "test-job", metav1.GetOptions{}); err == nil {
		t.Fatal("expected job to be deleted, but Get succeeded")
	}
}

// TestSanitizeK8sName ทดสอบว่า sanitizeK8sName แปลง input ใดๆ ให้ผ่านกฎการตั้งชื่อ object
// ของ Kubernetes เสมอ (RFC 1123 label: ตัวพิมพ์เล็ก/ตัวเลข/"-" เท่านั้น, ไม่ขึ้น/ลงท้ายด้วย
// "-", ยาวไม่เกิน 63 ตัวอักษร) ไม่ว่า input จะมีตัวอักษรที่ผิดกฎแค่ไหนก็ตาม
func TestSanitizeK8sName(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"already valid", "bop-job-abc123"},
		{"uppercase and special chars", "BOP_Job.ID:With/Weird:Chars!!"},
		{"very long input gets truncated to 63 chars", "bop-" + string(make([]byte, 200))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeK8sName(tc.input)
			if len(got) == 0 {
				t.Fatal("expected non-empty result")
			}
			if len(got) > 63 {
				t.Errorf("expected result length <= 63, got %d (%q)", len(got), got)
			}
			if got[0] == '-' || got[len(got)-1] == '-' {
				t.Errorf("expected result to not start/end with '-', got %q", got)
			}
			for _, r := range got {
				valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-'
				if !valid {
					t.Errorf("unexpected character %q in sanitized result %q", r, got)
				}
			}
		})
	}
}
