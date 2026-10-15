---
title: "Device authorization schema for TASK-000003"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000003
tags:
  - database
  - schema
  - auth
  - sessions
---

# Decision: Device Authorization Schema For TASK-000003

## Related Work

- `TASK-000003`: CLI device-code login.
- [003-database-persistence.md](003-database-persistence.md)
- [004-local-docker-compose-runtime.md](004-local-docker-compose-runtime.md)

## Decision

Create a minimal Postgres schema for auth MVP.

Use `BIGINT GENERATED ALWAYS AS IDENTITY` instead of UUIDs for now. Public APIs should not expose these IDs, so there is no product value in adding UUID machinery yet.

Store hashes as `BYTEA` and enforce SHA-256 length with `octet_length(...) = 32`.

## SQL Schema

```sql
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
```

## Status Meaning

| Status | Meaning |
|---|---|
| `pending` | CLI has started login and browser approval has not happened yet. |
| `approved` | Browser approval succeeded and CLI may exchange the device code for one session. |
| `denied` | Browser approval denied the login. |
| `expired` | Authorization expired before approval or exchange. |
| `consumed` | Authorization was exchanged for a session and cannot be reused. |

## Required Repository Operations

The DB adapter should expose operations equivalent to:

```text
CreateDeviceAuthorization
FindDeviceAuthorizationByDeviceCodeHash
FindDeviceAuthorizationByUserCodeHash
ApproveDeviceAuthorization
DenyDeviceAuthorization
ExpireDeviceAuthorization
ConsumeDeviceAuthorization
CreateSession
FindSessionByTokenHash
RevokeSession
```

The exchange from approved device authorization to session must happen in one transaction:

```text
SELECT approved authorization FOR UPDATE
-> verify status = approved
-> verify expires_at > now()
-> insert session
-> mark authorization consumed
-> commit
```

## Rules

- Device codes are short-lived.
- Sessions expire after one hour.
- Consumed device authorizations cannot issue a second session.
- Expired or denied device authorizations cannot issue sessions.
- Raw device codes and raw session tokens are not stored.
- AWS credentials are not stored.
- `user_code_display` is stored because it is shown to the user in the browser approval flow; lookup still uses `user_code_hash`.
- `updated_at` is application-managed in this MVP. No trigger is required for the first slice.

## Acceptance Criteria

- Schema can create pending device authorization.
- Schema can approve device authorization.
- Schema can exchange approved device authorization for exactly one session.
- Schema rejects expired device authorization.
- Schema can find a session by token hash.
- Schema enforces 32-byte token hashes.
- Schema prevents more than one session per device authorization.
- Schema does not contain AWS credential columns.
