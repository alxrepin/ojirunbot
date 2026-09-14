CREATE INDEX IF NOT EXISTS meal_entries_user_created_idx
    ON meal_entries (user_id, created_at);
