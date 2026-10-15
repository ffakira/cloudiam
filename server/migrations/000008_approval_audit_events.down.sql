DROP INDEX IF EXISTS audit_events_operator_subject_occurred_at_idx;
DROP INDEX IF EXISTS audit_events_access_request_id_idx;

ALTER TABLE audit_events
  DROP CONSTRAINT audit_events_idempotency_key_hash_check;

ALTER TABLE audit_events
  DROP CONSTRAINT audit_events_decision_check,
  ADD CONSTRAINT audit_events_decision_check
    CHECK (decision IN ('allowed', 'denied', 'failed'));

ALTER TABLE audit_events
  DROP CONSTRAINT audit_events_event_type_check,
  ADD CONSTRAINT audit_events_event_type_check
    CHECK (event_type IN ('access_requested', 'access_denied', 'access_issued', 'access_failed'));

ALTER TABLE audit_events
  DROP COLUMN operator_subject,
  DROP COLUMN idempotency_key_hash,
  DROP COLUMN access_request_id;
