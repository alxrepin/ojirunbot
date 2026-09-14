ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS chat_id bigint;

CREATE INDEX IF NOT EXISTS jobs_running_chat_idx
    ON jobs (chat_id) WHERE status = 'running';

CREATE INDEX IF NOT EXISTS meal_entries_chat_created_idx
    ON meal_entries (chat_id, created_at);
