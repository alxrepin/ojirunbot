package job

const enqueueQuery = `
	INSERT INTO jobs (queue, kind, dedupe_key, payload_json, chat_id)
	VALUES ($1, $2, nullif($3, ''), $4, nullif($5::bigint, 0))
	ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING`

const lockQueueQuery = `SELECT pg_advisory_xact_lock(hashtext('jobs:' || $1::text))`

const claimQuery = `
	UPDATE jobs
	SET status='running', attempts=attempts+1, locked_until=now() + make_interval(secs => $2), updated_at=now()
	WHERE id = (
		SELECT j.id
		FROM jobs j
		WHERE j.queue=$1
		  AND ((j.status='queued' AND j.run_at <= now()) OR (j.status='running' AND j.locked_until < now()))
		  AND (j.chat_id IS NULL OR NOT EXISTS (
			SELECT 1
			FROM jobs r
			WHERE r.queue=j.queue AND r.chat_id=j.chat_id AND r.id<>j.id
			  AND r.status='running' AND r.locked_until >= now()
		  ))
		ORDER BY coalesce(j.chat_id < 0, false), j.run_at
		LIMIT 1
		FOR UPDATE OF j SKIP LOCKED
	)
	RETURNING id::text, queue, kind, payload_json, attempts, max_attempts`

const completeQuery = `DELETE FROM jobs WHERE id=$1`

const retryQuery = `
	UPDATE jobs
	SET status='queued', run_at=now() + make_interval(secs => $2), locked_until=NULL, last_error=$3, updated_at=now()
	WHERE id=$1`

const failQuery = `
	UPDATE jobs
	SET status='failed', locked_until=NULL, last_error=$2, updated_at=now()
	WHERE id=$1`

const backlogQuery = `
	SELECT count(*)
	FROM jobs
	WHERE queue=$1 AND status='queued' AND run_at <= now()
	  AND ($2::bigint < 0 OR chat_id IS NULL OR chat_id > 0)`
