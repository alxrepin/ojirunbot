ALTER TABLE registration_sessions
    ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'registration';
