package user

const upsertQuery = `
	INSERT INTO users (telegram_user_id, telegram_username, display_name)
	VALUES ($1, $2, $3)
	ON CONFLICT (telegram_user_id) DO UPDATE
	SET telegram_username = EXCLUDED.telegram_username,
	    display_name = CASE WHEN users.display_name = '' THEN EXCLUDED.display_name ELSE users.display_name END,
	    updated_at = now()
	RETURNING id::text, telegram_user_id, coalesce(telegram_username, ''), display_name`

const selectByTelegramIDQuery = `
	SELECT id::text, telegram_user_id, coalesce(telegram_username, ''), display_name
	FROM users
	WHERE telegram_user_id=$1`

const selectByUsernameQuery = `
	SELECT id::text, telegram_user_id, coalesce(telegram_username, ''), display_name
	FROM users
	WHERE lower(telegram_username)=lower($1)
	LIMIT 1`
