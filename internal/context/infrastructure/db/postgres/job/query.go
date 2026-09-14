package job

const enqueueQuery = `
	INSERT INTO jobs (queue, kind, dedupe_key, payload_json)
	VALUES ($1, $2, nullif($3, ''), $4)
	ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING`

const claimQuery = `
	UPDATE jobs
	SET status='running', attempts=attempts+1, locked_until=now() + make_interval(secs => $2), updated_at=now()
	WHERE id = (
		SELECT id
		FROM jobs
		WHERE queue=$1
		  AND ((status='queued' AND run_at <= now()) OR (status='running' AND locked_until < now()))
		ORDER BY run_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED
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
	WHERE queue=$1 AND status='queued' AND run_at <= now()`
