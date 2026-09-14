UPDATE meal_entry_revisions r
SET ai_request_json = regexp_replace(
        r.ai_request_json::text,
        '"data:image/[A-Za-z0-9.+-]+;base64,[A-Za-z0-9+/=]+"',
        '"photo:sha256:' || p.sha256 || '"',
        'g'
    )::jsonb
FROM (
    SELECT DISTINCT ON (meal_entry_id) meal_entry_id, sha256
    FROM photos
    ORDER BY meal_entry_id, created_at DESC
) p
WHERE p.meal_entry_id = r.meal_entry_id
  AND r.ai_request_json::text LIKE '%;base64,%';

UPDATE meal_entry_revisions
SET ai_request_json = regexp_replace(
        ai_request_json::text,
        '"data:image/[A-Za-z0-9.+-]+;base64,[A-Za-z0-9+/=]+"',
        '"photo:stripped"',
        'g'
    )::jsonb
WHERE ai_request_json::text LIKE '%;base64,%';

CREATE INDEX IF NOT EXISTS meal_items_revision_idx
    ON meal_items (meal_entry_revision_id);

CREATE INDEX IF NOT EXISTS photos_meal_entry_idx
    ON photos (meal_entry_id);
