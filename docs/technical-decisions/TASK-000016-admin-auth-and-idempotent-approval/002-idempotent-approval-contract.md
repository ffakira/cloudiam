---
title: "Idempotent approval contract for TASK-000016"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000016
tags:
  - idempotency
  - approval
  - http
  - security
---

# Decision: Idempotent Approval Contract

## Decision

Console approve/deny mutations require an idempotency key.

Request header:

```http
Idempotency-Key: <uuid-v4>
```

The server stores an idempotency record for each console mutation with:

- key
- console operator subject
- HTTP method
- route/action
- target resource access request ID
- request fingerprint over stable mutation inputs
- response status
- response body or result reference
- created/expires timestamps

For approve/deny, the request fingerprint must include:

- action: `approve` or `deny`
- access request ID
- console operator subject
- idempotency key scope
- request body fields, if any

The fingerprint must not include live config snapshots, current policy text, exact reason strings, JWT values, or mutable target resolution from the latest hot-reload config.

Replays:

- Same key + same fingerprint returns the original result.
- Same key + different fingerprint returns `409 Conflict`.
- Missing key returns `400 Bad Request`.
- Malformed or non-v4 UUID key returns `400 Bad Request`.

Suggested retention for MVP idempotency records is at least 24 hours.

## Rationale

Resource approvals are privileged state transitions. Browser retries, double-clicks, network timeouts, or client refreshes must not create ambiguous approval behavior.

UUIDv4 keys give clients a simple, collision-resistant retry handle. Persisting the request fingerprint prevents accidental or malicious reuse of a key for a different approval.

The fingerprint is intentionally tied to the immutable access request ID. `TASK-000017` created the durable request object, `TASK-000023` pinned the approved permission boundary through `role_arn`, and `TASK-000018` made `approval_required` a target-level config flag that affects future requests. Approval idempotency should not depend on rebuilding the request from current config.

## Acceptance Criteria

- Approval and denial endpoints require UUIDv4 idempotency keys.
- Replay of the same request is stable.
- Reuse with different payload is rejected.
- Idempotency storage does not contain raw passwords, JWTs, CloudIAM session tokens, device codes, or AWS credentials.
