-- History ของ Step (ticket 08) อ่านตาม workflow ID + ลำดับ Step ครบทุก Attempt เรียงตามเวลา
-- (id ตัดสินลำดับเมื่อเวลาเท่ากัน) — คู่กับ log_lines_job_idx ของ log ของ Job
CREATE INDEX log_lines_step_idx ON log_lines (workflow_id, step_index, time, id) WHERE workflow_id IS NOT NULL;
