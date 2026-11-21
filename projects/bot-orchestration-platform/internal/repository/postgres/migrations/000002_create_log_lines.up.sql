-- log line ของ BOP — เจ้าของมีได้ 2 แบบ (ดู spec backend-reliability หัวข้อ Persistence):
--   Job:  job_id
--   Step: workflow_id + step_index (ลำดับ Step) + attempt (หมายเลข Attempt) — ticket 08 เริ่มเขียน
-- CHECK บังคับให้มีเจ้าของแบบใดแบบหนึ่งพอดี ไม่มี foreign key ไป jobs เพราะ log ของ Step ไม่มี Job
-- และ logger แบบ memory ก็รับ ID ใดก็ได้
CREATE TABLE log_lines (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_id      text,
    workflow_id text,
    step_index  integer,
    attempt     integer,
    time        timestamptz NOT NULL,
    level       text        NOT NULL,
    message     text        NOT NULL,
    CONSTRAINT log_lines_one_owner CHECK (
        (job_id IS NOT NULL AND workflow_id IS NULL AND step_index IS NULL AND attempt IS NULL)
        OR (job_id IS NULL AND workflow_id IS NOT NULL AND step_index IS NOT NULL AND attempt IS NOT NULL)
    )
);

-- History ของ Job อ่านตามเวลา (id ตัดสินลำดับเมื่อเวลาเท่ากัน)
CREATE INDEX log_lines_job_idx ON log_lines (job_id, time, id) WHERE job_id IS NOT NULL;

-- งานลบ log ที่เก่ากว่า 30 วันลบตามเวลาอย่างเดียว ไม่สนเจ้าของ
CREATE INDEX log_lines_time_idx ON log_lines (time);
