// ไฟล์นี้ implement K8sJobActivities.RunKubernetesJob — Temporal Activity ที่สั่งสร้าง
// Kubernetes Job (ephemeral container) ต่อ step หนึ่งครั้ง แล้วรอดูผล แทนที่จะรันผ่าน
// bot.Bot registry ในกระบวนการเดียวกัน (Activities.RunStep, activity.go) หรือ route ไปยัง
// remote Temporal task queue (Step.TaskQueue) — ดู design doc เต็มๆ ที่
// K8S-JOB-HYBRID-EXECUTION-TODO.md และตารางเทียบ 3 ทางเลือกในนั้น
//
// สถานะตอนเขียน: ยัง**ไม่เคยทดสอบกับ K8s cluster จริง** เลย (รอ Phase 2.1 containerize BOP
// เข้า K8s ก่อน — ไม่มี cluster ให้ต่อตอนนี้) unit test ในไฟล์นี้ใช้
// k8s.io/client-go/kubernetes/fake แทน ซึ่งเป็นแค่ CRUD store เปล่าๆ ไม่มี controller
// logic จริง (ไม่มีใคร transition Job.Status ให้อัตโนมัติเหมือนของจริง) — test จึงต้อง
// จำลอง controller เอง (ดู k8sjob_test.go)
package workflow

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"bop/internal/logger"
)

// defaultPollInterval คือความถี่ที่ awaitJobCompletion เช็คสถานะ Job + ส่ง heartbeat —
// ตั้งเป็นค่าเริ่มต้นตอน NewK8sJobActivities แต่ test เปลี่ยนได้ตรงๆ ผ่าน field ที่ไม่
// export (white-box, เหมือน binaryPath ของ internal/bots/externaljob และ
// total/heartbeatEvery ของ projects/remote-job-worker) เพื่อไม่ต้องรอจริงหลายวินาทีทุกครั้ง
// ที่รัน test
const defaultPollInterval = 2 * time.Second

// jobTTLSecondsAfterFinished คือเวลาที่ปล่อยให้ Job object ที่จบแล้ว (สำเร็จ/ล้มเหลว) ค้าง
// อยู่ใน cluster ก่อนถูกลบอัตโนมัติ (Kubernetes TTL controller เป็นคนลบให้เอง ไม่ใช่ BOP)
// — ตั้งสั้นพอสมควรเพราะ log ถูก stream เข้า BOP logger ไปแล้วตอน Job จบ (ดู
// streamPodLogs) ไม่ต้องพึ่ง Job/Pod object ที่ค้างอยู่ใน cluster เพื่อดู log ย้อนหลัง
const jobTTLSecondsAfterFinished = int32(300)

// K8sJobActivities คือตัวห่อ (adapter) ที่ทำให้ Step.K8sJob (workflow.go) ถูกเรียกจาก
// Temporal Activity ได้ — คนละ struct กับ Activities (activity.go) ตั้งใจ เพราะไม่มี
// bot.Bot registry เกี่ยวข้องเลย มีแค่ Kubernetes clientset + logger เท่านั้น
type K8sJobActivities struct {
	clientset        kubernetes.Interface
	defaultNamespace string
	log              logger.Logger
	pollInterval     time.Duration
}

// NewK8sJobActivities สร้าง K8sJobActivities พร้อมใช้งาน — clientset ควรเป็น
// in-cluster config ตอนรันจริงใน K8s (ดู cmd/bop-worker/main.go เรื่อง best-effort
// construction เพราะ local dev ยังไม่มี cluster) defaultNamespace คือ namespace ที่ Job
// จะถูกสร้างถ้า Step.K8sJob.Namespace ไม่ได้ระบุมา (แนะนำใช้ namespace แยกจาก BOP เอง
// เช่น "bop-jobs" ดู K8S-JOB-HYBRID-EXECUTION-TODO.md Open Question ข้อ 3)
func NewK8sJobActivities(clientset kubernetes.Interface, defaultNamespace string, log logger.Logger) *K8sJobActivities {
	return &K8sJobActivities{
		clientset:        clientset,
		defaultNamespace: defaultNamespace,
		log:              log,
		pollInterval:     defaultPollInterval,
	}
}

// RunKubernetesJob คือ Temporal Activity function ที่ cmd/bop-worker register ไว้ภายใต้
// ชื่อ K8sJobActivityName ("RunKubernetesJob") — ขั้นตอนการทำงาน:
//  1. validate ว่ามี Image (required — ไม่มี image ทั่วไปที่ใช้ default แทนได้) ถ้าไม่มี
//     ถือเป็น config ผิดฝั่ง BOP เอง คืน bot.ConfigError-style non-retryable error (เหมือน
//     pattern ของ activity.go/externaljob ทุกประการ — ผิด config ยังไง retry ก็ผลเดิม)
//  2. สร้าง batchv1.Job ด้วย BackoffLimit=0 เสมอ (**ห้ามเปลี่ยน** — เหตุผลอยู่ที่คอมเมนต์
//     ของ field นี้ด้านล่าง) แล้ว Create เข้า cluster
//  3. รอจนกว่า Job จะจบ (awaitJobCompletion) พร้อมส่ง heartbeat เป็นระยะระหว่างรอ
//  4. stream log ของ Pod ที่ Job สร้างขึ้นเข้า BOP logger ทันทีที่รู้ว่า Job จบแล้ว (ไม่รอ
//     TTL controller ลบ Job ทิ้งก่อน — ไม่งั้น log หายถาวร)
//  5. แปลผล: Job.Status.Succeeded>0 = สำเร็จ, Failed>0 = ล้มเหลว (error ธรรมดา/retryable
//     — K8s ไม่มีแนวคิด "config error" ของตัวเอง แยกแยะไม่ได้จากภายนอกว่า container fail
//     เพราะ config ผิดหรือปัญหาชั่วคราว จึงใช้ error ธรรมดาเป็น default เหมือนที่ตัดสินใจ
//     ไว้กับ externaljob ตอน exit code ไม่ใช่ 0)
//
// ถ้า ctx ถูกยกเลิก/timeout ระหว่างรออยู่ (ขั้นตอน 3) จะลบ Job ทิ้งทันที (deleteJob) ก่อน
// return — หลักการเดียวกับที่ externaljob ต้อง kill ทั้ง process group ตอน cancel ไม่ใช่
// ปล่อย orphan ทิ้งไว้ (ที่นี่ "orphan" คือ Job/Pod ที่ยังรันต่อใน cluster ทั้งที่ Temporal
// ถือว่า activity นี้จบไปแล้ว)
func (a *K8sJobActivities) RunKubernetesJob(ctx context.Context, spec K8sJobSpec) (string, error) {
	if spec.Image == "" {
		err := fmt.Errorf("k8sjob: missing required field %q", "image")
		return "", temporal.NewApplicationErrorWithCause(err.Error(), "ConfigError", err)
	}

	ns := spec.Namespace
	if ns == "" {
		ns = a.defaultNamespace
	}

	info := activity.GetInfo(ctx)
	name := k8sJobName(info)
	logID := fmt.Sprintf("%s-%s", info.WorkflowExecution.ID, info.ActivityID)
	log := func(msg string) { a.log.Log(ctx, logID, logger.LevelInfo, msg) }

	job := buildJobObject(name, ns, spec)
	if _, err := a.clientset.BatchV1().Jobs(ns).Create(ctx, job, metav1.CreateOptions{}); err != nil {
		return "", fmt.Errorf("k8sjob: create job %s/%s: %w", ns, name, err)
	}
	log(fmt.Sprintf("created kubernetes job %s/%s (image=%s)", ns, name, spec.Image))

	succeeded, err := a.awaitJobCompletion(ctx, ns, name)
	if err != nil {
		a.deleteJob(ns, name) // ctx หมดเวลา/ถูกยกเลิก — ต้อง cleanup ไม่ปล่อย Job ทำงานต่อเป็น orphan
		return "", err
	}

	a.streamPodLogs(context.Background(), ns, name, log) // context ใหม่ กัน ctx เดิมหมดเวลาไปพอดีตอนกำลังดึง log ของ Job ที่เพิ่งจบ

	if !succeeded {
		return "", fmt.Errorf("k8sjob: job %s/%s failed", ns, name)
	}
	return fmt.Sprintf("kubernetes job %s/%s completed", ns, name), nil
}

// buildJobObject สร้าง batchv1.Job object จาก K8sJobSpec — RestartPolicyNever คู่กับ
// BackoffLimit=0 (ไม่ใช่ OnFailure) เพราะไม่ต้องการให้ K8s พยายามรัน container ซ้ำเองเลย
// แม้แต่ครั้งเดียวถ้า fail — retry decision ทั้งหมดเป็นหน้าที่ของ Temporal RetryPolicy
// (nonRetryableErrorTypes ด้านบนไฟล์ temporal.go) แต่ผู้เดียว ถ้าปล่อยให้ K8s Job มี
// BackoffLimit>0 ด้วย จะเกิด retry ซ้อนกัน 2 ชั้น (K8s retry ข้างในหนึ่ง Activity attempt,
// Temporal retry ทั้ง Activity อีกชั้น) ซึ่ง debug ยากมากว่า "ทำไม job นี้รันไปแล้วกี่รอบ"
func buildJobObject(name, namespace string, spec K8sJobSpec) *batchv1.Job {
	envVars := make([]corev1.EnvVar, 0, len(spec.Env))
	for k, v := range spec.Env {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	backoffLimit := int32(0)
	ttl := jobTTLSecondsAfterFinished
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:    "job",
						Image:   spec.Image,
						Command: spec.Command,
						Args:    spec.Args,
						Env:     envVars,
					}},
				},
			},
		},
	}
}

// awaitJobCompletion poll สถานะของ Job เป็นระยะ (ทุก pollInterval) จนกว่าจะเห็น
// Succeeded>0 หรือ Failed>0 — ใช้ polling ธรรมดาแทน K8s Watch API เพื่อความง่าย (ยังไม่ต้อง
// การประสิทธิภาพระดับ watch สำหรับ use case นี้ — จำนวน Job ที่รันพร้อมกันไม่เยอะขนาดที่
// polling จะเป็นปัญหา) เช็ค ctx.Done() ทุกรอบเพื่อหยุดทันทีถ้าถูกยกเลิก/timeout (เหมือน
// runLongTask ของ projects/remote-job-worker) และเรียก activity.RecordHeartbeat ทุกรอบ
// เพื่อบอก Temporal ว่า activity นี้ยังทำงานอยู่จริง ไม่ได้ค้าง (สำคัญเพราะ activity นี้
// ส่วนใหญ่ใช้เวลาแค่ "รอ" ไม่ได้ทำงานหนักเอง)
func (a *K8sJobActivities) awaitJobCompletion(ctx context.Context, namespace, name string) (succeeded bool, err error) {
	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-ticker.C:
			activity.RecordHeartbeat(ctx, "waiting for kubernetes job "+namespace+"/"+name)
			j, getErr := a.clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
			if getErr != nil {
				continue // K8s API error ชั่วคราว (network blip) — ลองใหม่รอบหน้า ไม่ fail activity ทันที
			}
			switch {
			case j.Status.Succeeded > 0:
				return true, nil
			case j.Status.Failed > 0:
				return false, nil
			}
		}
	}
}

// streamPodLogs หา Pod ที่ Job สร้างขึ้น (Kubernetes ติด label "job-name=<ชื่อ Job>" ให้
// Pod ลูกของ Job ทุกตัวโดยอัตโนมัติ ไม่ต้องตั้งเอง) แล้วอ่าน log ทีละบรรทัดส่งเข้า log()
// ที่ส่งเข้ามา — ความล้มเหลวระหว่างขั้นตอนนี้ (หา Pod ไม่เจอ/ดึง log ไม่ได้) ไม่ถือเป็น
// fatal error ของ activity ทั้งหมด (Job ทำงานสำเร็จ/ล้มเหลวไปแล้วจริงๆ ไม่เกี่ยวกับดึง log
// ได้หรือไม่ได้ — pattern เดียวกับ handleGetWorkflowRun ใน internal/api/http/handler.go
// ที่ปล่อย steps เป็น slice ว่างถ้า QueryWorkflow error แทนที่จะ fail response ทั้งก้อน)
func (a *K8sJobActivities) streamPodLogs(ctx context.Context, namespace, jobName string, log func(string)) {
	pods, err := a.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + jobName})
	if err != nil || len(pods.Items) == 0 {
		return
	}

	stream, err := a.clientset.CoreV1().Pods(namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{}).Stream(ctx)
	if err != nil {
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		log(scanner.Text())
	}
}

// deleteJob ลบ Job (พร้อม Pod ลูกทั้งหมดผ่าน Foreground propagation) — เรียกเฉพาะตอน
// ctx ถูกยกเลิก/timeout ระหว่างรอ (ปกติปล่อยให้ TTL controller ของ K8s ลบเองหลังจบงาน
// ตาม jobTTLSecondsAfterFinished) ใช้ context.Background() แทน ctx เดิม เพราะ ctx เดิม
// น่าจะ done ไปแล้วตอนถูกเรียก (เหตุผลเดียวกับ pattern "context ใหม่สำหรับบันทึกครั้ง
// สุดท้าย" ใน internal/worker/pool.go)
func (a *K8sJobActivities) deleteJob(namespace, name string) {
	propagation := metav1.DeletePropagationForeground
	_ = a.clientset.BatchV1().Jobs(namespace).Delete(context.Background(), name, metav1.DeleteOptions{PropagationPolicy: &propagation})
}

// k8sJobName สร้างชื่อ Job จาก Temporal WorkflowID+ActivityID (unique เสมอต่อการรัน
// activity หนึ่งครั้ง) แล้ว sanitize ให้ผ่านกฎการตั้งชื่อ object ของ Kubernetes (RFC 1123
// label: ตัวพิมพ์เล็ก, ตัวเลข, "-" เท่านั้น, ห้ามขึ้นต้น/ลงท้ายด้วย "-", ยาวไม่เกิน 63
// ตัวอักษร) — ต้องยาวไม่เกิน 63 เพราะชื่อนี้ถูกใช้เป็นค่าของ label "job-name" บน Pod ลูก
// ด้วย (label value มีข้อจำกัดยาวสุด 63 ตัวอักษรเหมือนกัน)
func k8sJobName(info activity.Info) string {
	raw := fmt.Sprintf("bop-%s-%s", info.WorkflowExecution.ID, info.ActivityID)
	return sanitizeK8sName(raw)
}

func sanitizeK8sName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 63 {
		out = strings.TrimRight(out[:63], "-")
	}
	if out == "" {
		out = "bop-job"
	}
	return out
}
