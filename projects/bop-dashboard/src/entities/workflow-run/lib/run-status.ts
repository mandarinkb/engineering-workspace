// RunStatus คือสถานะ workflow run แบบเดียวที่ dashboard ใช้ภายใน
//
// backend ส่ง info.GetStatus().String() ของ Temporal SDK มาตรงๆ ซึ่ง go.temporal.io/api
// เวอร์ชันปัจจุบันคืนชื่อสั้น ("Running", "Completed", "Failed", ...) ไม่ใช่ชื่อ enum ยาว
// ("WORKFLOW_EXECUTION_STATUS_RUNNING") ตามที่โค้ดเดิมเข้าใจ — เจอจากข้อมูลจริงบน stack ที่
// Mini PC ทำให้ badge เป็นสีเทาหมดและ pipeline ไม่เห็นว่า run ยังรันอยู่ normalizeRunStatus
// รับได้ทั้งสองแบบ เผื่อ SDK เปลี่ยนกลับหรือมีข้อมูลเก่า
export type RunStatus =
  | "running"
  | "completed"
  | "failed"
  | "canceled"
  | "terminated"
  | "timed_out"
  | "continued_as_new"
  | "unknown";

const BY_KEY: Record<string, RunStatus> = {
  running: "running",
  completed: "completed",
  failed: "failed",
  canceled: "canceled",
  terminated: "terminated",
  timedout: "timed_out",
  continuedasnew: "continued_as_new",
};

export function normalizeRunStatus(raw: string): RunStatus {
  // ตัด prefix enum ยาวออก แล้วเทียบแบบไม่สนตัวพิมพ์และ "_" — "WORKFLOW_EXECUTION_STATUS_TIMED_OUT"
  // กับ "TimedOut" จะได้ key เดียวกันคือ "timedout"
  const key = raw.replace(/^WORKFLOW_EXECUTION_STATUS_/, "").replace(/_/g, "").toLowerCase();
  return BY_KEY[key] ?? "unknown";
}

// สถานะที่นับว่า "ล้มเหลว" (ใช้ทั้ง badge สีแดงและรายการ fail ในหน้า Overview)
export function isFailedRunStatus(status: RunStatus): boolean {
  return status === "failed" || status === "terminated" || status === "timed_out";
}

// isRunInProgress บอกว่า run ยังไม่จบ ต้องดึงข้อมูลใหม่ต่อไหม — เฉพาะ running เท่านั้น ส่วน
// ไม่มีข้อมูล (ยังโหลดอยู่หรือ error) หรือสถานะที่ไม่รู้จัก ไม่ดึงซ้ำ เพราะไม่รู้ว่าจะจบเมื่อไหร่
// และไม่อยากยิง API วนไม่หยุดเมื่อมีปัญหา
export function isRunInProgress(status: string | undefined): boolean {
  return status !== undefined && normalizeRunStatus(status) === "running";
}
