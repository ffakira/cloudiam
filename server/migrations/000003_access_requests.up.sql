CREATE TABLE access_requests (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

  principal TEXT NOT NULL,
  target TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  account TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT '',

  requested_duration_seconds INTEGER NOT NULL,
  policy_reason TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending_approval',

  expires_at TIMESTAMPTZ NOT NULL,
  approved_at TIMESTAMPTZ,
  denied_at TIMESTAMPTZ,
  issued_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT access_requests_principal_check
    CHECK (length(trim(principal)) > 0),
  CONSTRAINT access_requests_target_check
    CHECK (length(trim(target)) > 0),
  CONSTRAINT access_requests_duration_check
    CHECK (requested_duration_seconds > 0),
  CONSTRAINT access_requests_status_check
    CHECK (status IN ('pending_approval', 'approved', 'denied', 'expired', 'issued')),
  CONSTRAINT access_requests_expiry_check
    CHECK (expires_at > created_at),
  CONSTRAINT access_requests_approved_at_check
    CHECK (status <> 'approved' OR approved_at IS NOT NULL),
  CONSTRAINT access_requests_denied_at_check
    CHECK (status <> 'denied' OR denied_at IS NOT NULL),
  CONSTRAINT access_requests_issued_at_check
    CHECK (status <> 'issued' OR issued_at IS NOT NULL)
);

CREATE INDEX access_requests_status_expires_at_idx
  ON access_requests (status, expires_at);

CREATE INDEX access_requests_principal_created_at_idx
  ON access_requests (principal, created_at DESC);

CREATE INDEX access_requests_target_created_at_idx
  ON access_requests (target, created_at DESC);
