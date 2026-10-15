CREATE TABLE audit_events (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  event_type TEXT NOT NULL,
  principal TEXT NOT NULL,
  target TEXT NOT NULL,
  decision TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',

  provider TEXT NOT NULL DEFAULT '',
  account TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT '',
  credential_expires_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT audit_events_event_type_check
    CHECK (event_type IN ('access_requested', 'access_denied', 'access_issued', 'access_failed')),
  CONSTRAINT audit_events_decision_check
    CHECK (decision IN ('allowed', 'denied', 'failed')),
  CONSTRAINT audit_events_principal_check
    CHECK (length(trim(principal)) > 0),
  CONSTRAINT audit_events_target_check
    CHECK (length(trim(target)) > 0)
);

CREATE INDEX audit_events_occurred_at_idx
  ON audit_events (occurred_at DESC);

CREATE INDEX audit_events_principal_occurred_at_idx
  ON audit_events (principal, occurred_at DESC);

CREATE INDEX audit_events_target_occurred_at_idx
  ON audit_events (target, occurred_at DESC);
