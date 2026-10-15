CREATE TABLE device_authorizations (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

  device_code_hash BYTEA NOT NULL,
  user_code_hash BYTEA NOT NULL,
  user_code_display TEXT NOT NULL,

  workspace TEXT NOT NULL,
  principal TEXT,
  endpoint TEXT NOT NULL,

  status TEXT NOT NULL DEFAULT 'pending',
  interval_seconds INTEGER NOT NULL DEFAULT 5,

  expires_at TIMESTAMPTZ NOT NULL,
  approved_at TIMESTAMPTZ,
  denied_at TIMESTAMPTZ,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT device_authorizations_status_check
    CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'consumed')),
  CONSTRAINT device_authorizations_device_code_hash_len_check
    CHECK (octet_length(device_code_hash) = 32),
  CONSTRAINT device_authorizations_user_code_hash_len_check
    CHECK (octet_length(user_code_hash) = 32),
  CONSTRAINT device_authorizations_user_code_display_check
    CHECK (length(trim(user_code_display)) > 0),
  CONSTRAINT device_authorizations_workspace_check
    CHECK (length(trim(workspace)) > 0),
  CONSTRAINT device_authorizations_endpoint_check
    CHECK (length(trim(endpoint)) > 0),
  CONSTRAINT device_authorizations_interval_check
    CHECK (interval_seconds > 0),
  CONSTRAINT device_authorizations_expiry_check
    CHECK (expires_at > created_at),
  CONSTRAINT device_authorizations_approved_principal_check
    CHECK (status NOT IN ('approved', 'consumed') OR principal IS NOT NULL),
  CONSTRAINT device_authorizations_approved_at_check
    CHECK (status NOT IN ('approved', 'consumed') OR approved_at IS NOT NULL),
  CONSTRAINT device_authorizations_denied_at_check
    CHECK (status <> 'denied' OR denied_at IS NOT NULL),
  CONSTRAINT device_authorizations_consumed_at_check
    CHECK (status <> 'consumed' OR consumed_at IS NOT NULL)
);

CREATE UNIQUE INDEX device_authorizations_device_code_hash_idx
  ON device_authorizations (device_code_hash);

CREATE UNIQUE INDEX device_authorizations_user_code_hash_idx
  ON device_authorizations (user_code_hash);

CREATE INDEX device_authorizations_pending_expires_at_idx
  ON device_authorizations (expires_at)
  WHERE status = 'pending';

CREATE INDEX device_authorizations_workspace_principal_idx
  ON device_authorizations (workspace, principal);

CREATE TABLE sessions (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

  session_token_hash BYTEA NOT NULL,
  device_authorization_id BIGINT NOT NULL REFERENCES device_authorizations(id),

  workspace TEXT NOT NULL,
  principal TEXT NOT NULL,
  endpoint TEXT NOT NULL,

  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  revoked_at TIMESTAMPTZ,

  CONSTRAINT sessions_token_hash_len_check
    CHECK (octet_length(session_token_hash) = 32),
  CONSTRAINT sessions_workspace_check
    CHECK (length(trim(workspace)) > 0),
  CONSTRAINT sessions_principal_check
    CHECK (length(trim(principal)) > 0),
  CONSTRAINT sessions_endpoint_check
    CHECK (length(trim(endpoint)) > 0),
  CONSTRAINT sessions_expiry_check
    CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX sessions_session_token_hash_idx
  ON sessions (session_token_hash);

CREATE UNIQUE INDEX sessions_device_authorization_id_idx
  ON sessions (device_authorization_id);

CREATE INDEX sessions_active_token_hash_idx
  ON sessions (session_token_hash)
  WHERE revoked_at IS NULL;

CREATE INDEX sessions_workspace_principal_idx
  ON sessions (workspace, principal);
