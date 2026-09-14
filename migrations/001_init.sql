CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_user_id bigint NOT NULL UNIQUE,
    telegram_username text,
    display_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    sex text NOT NULL CHECK (sex IN ('male', 'female')),
    age integer NOT NULL CHECK (age BETWEEN 10 AND 120),
    height_cm numeric(5,2) NOT NULL CHECK (height_cm BETWEEN 100 AND 250),
    weight_kg numeric(5,2) NOT NULL CHECK (weight_kg BETWEEN 30 AND 300),
    activity_level text NOT NULL CHECK (activity_level IN ('low', 'light', 'moderate', 'high', 'athlete')),
    goal text NOT NULL CHECK (goal IN ('lose', 'maintain', 'gain')),
    daily_calories_kcal integer NOT NULL,
    daily_protein_g integer NOT NULL,
    daily_fat_g integer NOT NULL,
    daily_carbs_g integer NOT NULL,
    formula_version text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS group_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id bigint NOT NULL,
    status text NOT NULL,
    checked_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, chat_id)
);

CREATE TABLE IF NOT EXISTS registration_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    telegram_user_id bigint NOT NULL,
    chat_id bigint NOT NULL,
    step text NOT NULL,
    payload_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS registration_sessions_active_idx
    ON registration_sessions (telegram_user_id, chat_id, status);

CREATE TABLE IF NOT EXISTS meal_entries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id bigint NOT NULL,
    source_message_id bigint NOT NULL,
    bot_message_id bigint,
    status text NOT NULL,
    meal_date date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    accepted_at timestamptz,
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS meal_entries_user_date_idx
    ON meal_entries (user_id, meal_date, status);

CREATE INDEX IF NOT EXISTS meal_entries_chat_message_idx
    ON meal_entries (chat_id, bot_message_id);

CREATE TABLE IF NOT EXISTS meal_entry_revisions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_entry_id uuid NOT NULL REFERENCES meal_entries(id) ON DELETE CASCADE,
    revision integer NOT NULL,
    user_text text NOT NULL DEFAULT '',
    correction_text text NOT NULL DEFAULT '',
    ai_request_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    ai_response_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    summary text NOT NULL,
    total_calories_kcal numeric(10,2) NOT NULL,
    total_protein_g numeric(10,2) NOT NULL,
    total_fat_g numeric(10,2) NOT NULL,
    total_carbs_g numeric(10,2) NOT NULL,
    confidence numeric(4,3) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (meal_entry_id, revision)
);

CREATE TABLE IF NOT EXISTS meal_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_entry_revision_id uuid NOT NULL REFERENCES meal_entry_revisions(id) ON DELETE CASCADE,
    name text NOT NULL,
    estimated_weight_g numeric(10,2) NOT NULL,
    calories_kcal numeric(10,2) NOT NULL,
    protein_g numeric(10,2) NOT NULL,
    fat_g numeric(10,2) NOT NULL,
    carbs_g numeric(10,2) NOT NULL,
    confidence numeric(4,3) NOT NULL
);

CREATE TABLE IF NOT EXISTS meal_audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_entry_id uuid NOT NULL REFERENCES meal_entries(id) ON DELETE CASCADE,
    event text NOT NULL,
    previous_status text NOT NULL,
    telegram_user_id bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS meal_audit_events_meal_idx
    ON meal_audit_events (meal_entry_id);

CREATE TABLE IF NOT EXISTS photos (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    meal_entry_id uuid NOT NULL REFERENCES meal_entries(id) ON DELETE CASCADE,
    telegram_file_id text NOT NULL,
    telegram_file_unique_id text NOT NULL,
    local_path text NOT NULL,
    mime_type text NOT NULL,
    file_size bigint NOT NULL,
    sha256 text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS daily_reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id bigint NOT NULL,
    report_date date NOT NULL,
    bot_message_id bigint,
    status text NOT NULL,
    recommendation_request_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    recommendation_response_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    sent_at timestamptz,
    UNIQUE (user_id, chat_id, report_date)
);
