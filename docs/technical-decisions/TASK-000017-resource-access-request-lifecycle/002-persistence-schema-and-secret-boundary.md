---
title: "Persistence schema and secret boundary for TASK-000017"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000017
tags:
  - persistence
  - schema
  - secrets
---

# Decision: Persistence Schema And Secret Boundary

## Decision

Persist resource access requests in Postgres.

Store:

- requester principal
- target
- requested duration
- resolved provider/account/role metadata
- resolved AWS role ARN as non-secret permission-boundary metadata
- policy reason/context
- status
- expiration
- approval/denial/issued timestamps

Do not store:

- AWS access key IDs
- AWS secret access keys
- AWS session tokens
- CloudIAM session tokens
- raw console JWTs
- raw device/user codes

## Rationale

The access request record needs enough context for admin review and later issuance, but issued AWS credentials remain transient output only.

The AWS role ARN is intentionally stored because approved access must pin the permission boundary reviewed at request time. `TASK-000023` owns the implementation hardening for this post-review correction.

## Task 16 Watch

Task 16 may add admin subject fields and idempotency records. Those should reference the access request ID rather than duplicating request metadata.
