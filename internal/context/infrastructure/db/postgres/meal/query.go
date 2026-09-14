package meal

const createEntryQuery = `
	INSERT INTO meal_entries (user_id, chat_id, source_message_id, status, meal_date)
	VALUES ($1, $2, $3, 'received', $4::date)
	RETURNING id::text, user_id::text, chat_id, source_message_id, coalesce(bot_message_id, 0), coalesce(ephemeral_message_id, 0), status, meal_date, created_at`

const setBotMessageQuery = `UPDATE meal_entries SET bot_message_id=$2 WHERE id=$1`

const setEphemeralMessageQuery = `UPDATE meal_entries SET ephemeral_message_id=$2 WHERE id=$1`

const lockUserMealsQuery = `SELECT pg_advisory_xact_lock(hashtext($1::text))`

const countCreatedSinceQuery = `SELECT count(*) FROM meal_entries WHERE user_id=$1 AND created_at >= $2`

const countInFlightQuery = `
	SELECT count(*)
	FROM meal_entries
	WHERE user_id=$1
	  AND status IN ('received', 'downloading_photo', 'analyzing', 'reanalyzing')
	  AND created_at > now() - interval '15 minutes'`

const lockChatMealsQuery = `SELECT pg_advisory_xact_lock(hashtext('chat:' || $1::bigint::text))`

const countChatCreatedSinceQuery = `SELECT count(*) FROM meal_entries WHERE chat_id=$1 AND created_at >= $2`

const setStatusQuery = `UPDATE meal_entries SET status=$2 WHERE id=$1`

const insertRevisionQuery = `
	INSERT INTO meal_entry_revisions (
		meal_entry_id, revision, user_text, correction_text, ai_request_json, ai_response_json,
		summary, total_calories_kcal, total_protein_g, total_fat_g, total_carbs_g, confidence
	)
	VALUES (
		$1,
		coalesce((SELECT max(revision) + 1 FROM meal_entry_revisions WHERE meal_entry_id=$1), 1),
		$2, $3, $4, $5, $6, $7, $8, $9, $10, $11
	)
	RETURNING id::text, revision, created_at`

const insertItemQuery = `
	INSERT INTO meal_items (
		meal_entry_revision_id, name, estimated_weight_g, calories_kcal,
		protein_g, fat_g, carbs_g, confidence
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

const markPendingQuery = `UPDATE meal_entries SET status='pending_confirmation' WHERE id=$1`

const selectForActionQuery = `
	SELECT e.id::text, e.user_id::text, u.telegram_user_id, e.chat_id, e.source_message_id,
	       coalesce(e.bot_message_id, 0), coalesce(e.ephemeral_message_id, 0), e.status, e.meal_date, e.created_at,
	       r.id::text, r.revision, r.user_text, r.correction_text, r.ai_response_json,
	       r.summary, r.total_calories_kcal, r.total_protein_g, r.total_fat_g, r.total_carbs_g, r.confidence, r.created_at
	FROM meal_entries e
	JOIN users u ON u.id=e.user_id
	LEFT JOIN LATERAL (
		SELECT *
		FROM meal_entry_revisions
		WHERE meal_entry_id=e.id
		ORDER BY revision DESC
		LIMIT 1
	) r ON true
	WHERE e.id=$1`

const selectItemsQuery = `
	SELECT name, estimated_weight_g, calories_kcal, protein_g, fat_g, carbs_g, confidence
	FROM meal_items
	WHERE meal_entry_revision_id=$1
	ORDER BY id`

const acceptQuery = `
	UPDATE meal_entries e
	SET status='accepted', accepted_at=coalesce(accepted_at, now())
	FROM users u
	WHERE e.id=$1
	  AND e.user_id=u.id
	  AND u.telegram_user_id=$2
	  AND e.status IN ('pending_confirmation', 'accepted')`

const awaitCorrectionQuery = `
	UPDATE meal_entries e
	SET status='awaiting_correction'
	FROM users u
	WHERE e.id=$1
	  AND e.user_id=u.id
	  AND u.telegram_user_id=$2
	  AND e.status='pending_confirmation'`

const cancelCorrectionQuery = `
	UPDATE meal_entries e
	SET status='pending_confirmation'
	FROM users u
	WHERE e.id=$1
	  AND e.user_id=u.id
	  AND u.telegram_user_id=$2
	  AND e.status='awaiting_correction'`

const lockForDeleteQuery = `
	SELECT e.status
	FROM meal_entries e
	JOIN users u ON u.id=e.user_id
	WHERE e.id=$1 AND u.telegram_user_id=$2
	FOR UPDATE OF e`

const insertDeleteAuditQuery = `
	INSERT INTO meal_audit_events (meal_entry_id, event, previous_status, telegram_user_id)
	VALUES ($1, 'deleted_after_accept', $2, $3)`

const markDeletedQuery = `
	UPDATE meal_entries SET status='deleted', deleted_at=coalesce(deleted_at, now()) WHERE id=$1`

const findByMessageQuery = `
	SELECT e.id::text, e.user_id::text, u.telegram_user_id, e.chat_id, e.source_message_id,
	       coalesce(e.bot_message_id, 0), coalesce(e.ephemeral_message_id, 0), e.status, e.meal_date, e.created_at
	FROM meal_entries e
	JOIN users u ON u.id=e.user_id
	WHERE e.chat_id=$1 AND (e.source_message_id=$2 OR e.bot_message_id=$2)
	ORDER BY e.created_at DESC
	LIMIT 1`

const setMealDateQuery = `
	UPDATE meal_entries e
	SET meal_date=$3::date
	FROM users u
	WHERE e.id=$1
	  AND e.user_id=u.id
	  AND u.telegram_user_id=$2
	  AND e.status IN ('pending_confirmation', 'accepted')`

const findAwaitingCorrectionQuery = `
	SELECT e.id::text
	FROM meal_entries e
	JOIN users u ON u.id=e.user_id
	WHERE u.telegram_user_id=$1 AND e.chat_id=$2 AND e.status='awaiting_correction'
	ORDER BY e.created_at DESC
	LIMIT 1`

const selectEntryQuery = `
	SELECT e.id::text, e.user_id::text, u.telegram_user_id, e.chat_id, e.source_message_id,
	       coalesce(e.bot_message_id, 0), coalesce(e.ephemeral_message_id, 0), e.status, e.meal_date, e.created_at
	FROM meal_entries e
	JOIN users u ON u.id=e.user_id
	WHERE e.id=$1`
