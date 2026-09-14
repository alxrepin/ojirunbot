ALTER TABLE registration_sessions
    ADD COLUMN IF NOT EXISTS bot_message_id bigint;
