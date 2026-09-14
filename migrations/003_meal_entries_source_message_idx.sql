CREATE INDEX IF NOT EXISTS meal_entries_chat_source_message_idx
    ON meal_entries (chat_id, source_message_id);
