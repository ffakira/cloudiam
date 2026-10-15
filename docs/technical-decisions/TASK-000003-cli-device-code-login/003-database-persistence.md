---
title: "Database persistence for TASK-000003"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000003
tags:
  - database
  - postgres
  - auth
  - device-code
---

# Decision: Database Persistence For TASK-000003

## Related Work

- `TASK-000003`: CLI device-code login.
- [001-contract.md](001-contract.md)
- [002-cli-session-storage.md](002-cli-session-storage.md)

## Context

Device login has browser and CLI participants. Even for the MVP, keeping pending device approvals only in memory would create friction when the frontend/browser side starts.

The MVP should introduce persistence narrowly, without pulling in dashboard read models, audit history, or broad product schema.

## Decision

Use Postgres for TASK-000003 auth state.

Persist only:

- pending device authorizations
- approved/denied/expired device authorization state
- CloudIAM server-side sessions

Do not persist:

- AWS credentials
- audit history
- dashboard read models
- full user management
- billing/product data

## Token Storage

Store hashes of sensitive tokens, not raw tokens:

- `device_code_hash`
- `session_token_hash`

The CLI can hold the raw device code while polling and the raw CloudIAM session token in its local session file. The database stores only hashes for lookup/verification.

## Acceptance Criteria

- Database schema includes device authorization state.
- Database schema includes CloudIAM sessions.
- Raw AWS credentials are not stored.
- Raw session tokens are not stored server-side.
- Expiry is stored and enforced.
- Implementation remains small enough for the one-week sprint.
