-- Job เดี่ยวของ bop-api (ดู domain/job.Job) — config เก็บเป็น jsonb เพราะเป็น map ที่ key ต่างกันไปตาม Bot
CREATE TABLE jobs (
    id              text PRIMARY KEY,
    bot_name        text        NOT NULL,
    config          jsonb       NOT NULL DEFAULT '{}'::jsonb,
    timeout_seconds integer     NOT NULL DEFAULT 0,
    status          text        NOT NULL,
    summary         text        NOT NULL DEFAULT '',
    error           text        NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL,
    started_at      timestamptz,
    ended_at        timestamptz
);

-- List เรียงจาก Job ล่าสุดก่อนเสมอ
CREATE INDEX jobs_created_at_idx ON jobs (created_at DESC);
