ALTER TABLE audit_events
  ADD COLUMN access_request_id BIGINT,
  ADD COLUMN operator_subject TEXT NOT NULL DEFAULT '',
  ADD COLUMN idempotency_key_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE audit_events
  DROP CONSTRAINT audit_events_event_type_check,
  ADD CONSTRAINT audit_events_event_type_check
    CHECK (event_type IN (
      'access_requested',
      'access_denied',
      'access_issued',
      'access_failed',
      'approval_requested',
      'approval_approved',
      'approval_denied',
      'approval_expired'
    ));

ALTER TABLE audit_events
  DROP CONSTRAINT audit_events_decision_check,
  ADD CONSTRAINT audit_events_decision_check
    CHECK (decision IN ('allowed', 'denied', 'failed', 'pending'));

ALTER TABLE audit_events
  ADD CONSTRAINT audit_events_idempotency_key_hash_check
    CHECK (idempotency_key_hash = '' OR idempotency_key_hash ~ '^[0-9a-f]{64}$');

CREATE INDEX audit_events_access_request_id_idx
  ON audit_events (access_request_id)
  WHERE access_request_id IS NOT NULL;

CREATE INDEX audit_events_operator_subject_occurred_at_idx
  ON audit_events (operator_subject, occurred_at DESC)
  WHERE operator_subject <> '';
