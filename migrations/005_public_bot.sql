CREATE TABLE IF NOT EXISTS jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    queue text NOT NULL,
    kind text NOT NULL,
    dedupe_key text,
    payload_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'failed')),
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 3,
    run_at timestamptz NOT NULL DEFAULT now(),
    locked_until timestamptz,
    last_error text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS jobs_dedupe_key_idx
    ON jobs (dedupe_key) WHERE dedupe_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS jobs_claim_idx
    ON jobs (queue, run_at) WHERE status IN ('queued', 'running');

CREATE INDEX IF NOT EXISTS meal_entries_date_status_idx
    ON meal_entries (meal_date, status, user_id);
