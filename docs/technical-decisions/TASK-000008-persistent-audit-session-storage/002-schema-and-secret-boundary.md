---
title: "Schema and secret storage boundary for TASK-000008"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000008
tags:
  - persistence
  - schema
  - secrets
  - security
---

# Decision: Schema And Secret Storage Boundary

## Decision

Persist only governance state and non-secret metadata.

Persistent tables include:

- `device_authorizations`
- `sessions`
- `audit_events`

Store hashes for sensitive CloudIAM auth tokens:

- `device_code_hash`
- `user_code_hash`
- `session_token_hash`

Do not store:

- raw device codes
- raw user codes
- raw CloudIAM session tokens
- AWS access key IDs
- AWS secret access keys
- AWS session tokens

Audit events store scalar governance metadata: event type, principal, target, decision, reason, provider, account, role, and credential expiration time. They do not store raw request bodies or issued credentials.

## Rationale

The server needs enough durable state to resume CloudIAM sessions across process restarts and to reconstruct access decisions during demos or investigations. It does not need raw bearer tokens or cloud credentials after issuance.

Hash-only lookup limits database blast radius: a leaked database should not immediately become a bearer-token or AWS-credential leak.

## Acceptance Criteria

- Session lookup uses `session_token_hash`.
- Device-code polling uses hashed device/user codes.
- Audit schema has no credential/token/code columns.
- Hash columns enforce expected digest length.
- Integration tests cover secret-column absence.
