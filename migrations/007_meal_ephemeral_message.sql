ALTER TABLE meal_entries
    ADD COLUMN IF NOT EXISTS ephemeral_message_id bigint;
