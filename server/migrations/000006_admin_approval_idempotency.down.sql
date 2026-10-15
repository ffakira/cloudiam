DROP TABLE IF EXISTS idempotency_records;

ALTER TABLE access_requests
	DROP COLUMN IF EXISTS denied_by,
	DROP COLUMN IF EXISTS approved_by;
