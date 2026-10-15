ALTER TABLE access_requests
	ADD COLUMN approved_by TEXT NOT NULL DEFAULT '',
	ADD COLUMN denied_by TEXT NOT NULL DEFAULT '';

CREATE TABLE idempotency_records (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	key TEXT NOT NULL,
	admin_subject TEXT NOT NULL,
	method TEXT NOT NULL,
	route TEXT NOT NULL,
	action TEXT NOT NULL,
	access_request_id BIGINT NOT NULL REFERENCES access_requests(id) ON DELETE RESTRICT,
	request_fingerprint TEXT NOT NULL,
	response_status INTEGER NOT NULL,
	response_body JSONB NOT NULL DEFAULT '{}'::jsonb,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (key, admin_subject)
);

CREATE INDEX idempotency_records_expires_at_idx ON idempotency_records (expires_at);
CREATE INDEX idempotency_records_access_request_id_idx ON idempotency_records (access_request_id);
