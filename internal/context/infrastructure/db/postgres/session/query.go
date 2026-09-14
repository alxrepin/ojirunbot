package session

const cancelActiveQuery = `
	UPDATE registration_sessions
	SET status='cancelled', updated_at=now()
	WHERE telegram_user_id=$1 AND chat_id=$2 AND status='active'`

const insertQuery = `
	INSERT INTO registration_sessions (kind, telegram_user_id, chat_id, step, payload_json, status, expires_at)
	VALUES ($1, $2, $3, $4, $5, 'active', now() + interval '30 minutes')
	RETURNING id::text`

const selectActiveQuery = `
	SELECT id::text, kind, coalesce(user_id::text, ''), telegram_user_id, chat_id, step, payload_json, coalesce(bot_message_id, 0), status, expires_at
	FROM registration_sessions
	WHERE kind=$1 AND telegram_user_id=$2 AND chat_id=$3 AND status='active' AND expires_at > now()
	ORDER BY created_at DESC
	LIMIT 1`

const setBotMessageQuery = `
	UPDATE registration_sessions
	SET bot_message_id=$4, updated_at=now()
	WHERE kind=$1 AND telegram_user_id=$2 AND chat_id=$3 AND status='active'`

const advanceQuery = `
	UPDATE registration_sessions
	SET step=$3, payload_json=$4, updated_at=now(), expires_at=now() + interval '30 minutes'
	WHERE id=$1 AND status='active' AND step=$2`

const completeQuery = `
	UPDATE registration_sessions
	SET status='completed', user_id=$2, updated_at=now()
	WHERE id=$1`

const closeQuery = `
	UPDATE registration_sessions
	SET status='completed', updated_at=now()
	WHERE id=$1 AND telegram_user_id=$2 AND status='active'`

const cancelQuery = `
	UPDATE registration_sessions
	SET status='cancelled', updated_at=now()
	WHERE id=$1 AND telegram_user_id=$2 AND status='active'`
