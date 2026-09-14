package report

const enqueueDailyReportsQuery = `
	INSERT INTO jobs (queue, kind, dedupe_key, payload_json)
	SELECT $2::text, $3::text,
	       $3::text || ':' || active.user_id::text || ':' || to_char($1::date, 'YYYY-MM-DD'),
	       jsonb_build_object('user_id', active.user_id::text, 'report_date', to_char($1::date, 'YYYY-MM-DD'))
	FROM (
		SELECT DISTINCT e.user_id
		FROM meal_entries e
		JOIN user_profiles p ON p.user_id=e.user_id
		WHERE e.meal_date=$1::date AND e.status='accepted'
	) active
	ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING`

const reportUserQuery = `
	SELECT u.id::text, u.telegram_user_id, coalesce(u.telegram_username, ''), u.display_name,
	       p.user_id::text, p.sex, p.age, p.height_cm, p.weight_kg, p.activity_level, p.goal,
	       p.daily_calories_kcal, p.daily_protein_g, p.daily_fat_g, p.daily_carbs_g, p.formula_version
	FROM users u
	JOIN user_profiles p ON p.user_id=u.id
	WHERE u.id=$1`

const mealsForReportQuery = `
	SELECT e.created_at, r.summary, r.total_calories_kcal, r.total_protein_g, r.total_fat_g, r.total_carbs_g
	FROM meal_entries e
	JOIN LATERAL (
		SELECT *
		FROM meal_entry_revisions
		WHERE meal_entry_id=e.id
		ORDER BY revision DESC
		LIMIT 1
	) r ON true
	WHERE e.user_id=$1 AND e.meal_date=$2::date AND e.status='accepted'
	ORDER BY e.created_at`

const hasMealsInChatQuery = `
	SELECT EXISTS (
		SELECT 1
		FROM meal_entries
		WHERE user_id=$1 AND chat_id=$2 AND status <> 'deleted'
	)`

const tryCreateQuery = `
	INSERT INTO daily_reports (user_id, chat_id, report_date, status)
	VALUES ($1, $2, $3::date, 'creating')
	ON CONFLICT (user_id, chat_id, report_date) DO UPDATE
	SET status='creating', created_at=now()
	WHERE daily_reports.status='failed'
	   OR (daily_reports.status='creating' AND daily_reports.created_at < now() - interval '10 minutes')
	RETURNING id::text`

const setStatusQuery = `UPDATE daily_reports SET status=$2 WHERE id=$1`

const markSentQuery = `
	UPDATE daily_reports
	SET status='sent',
	    bot_message_id=$2,
	    recommendation_request_json=$3,
	    recommendation_response_json=$4,
	    sent_at=now()
	WHERE id=$1`
