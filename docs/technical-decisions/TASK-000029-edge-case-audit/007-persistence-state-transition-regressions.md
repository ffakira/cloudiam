---
title: "Persistence state-transition regressions"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - persistence
  - concurrency
  - postgres
  - testing
---

# Decision: Persistence State-Transition Regressions

## Context

E4 covers persisted approval, access-request, device-authorization, and session state transitions. The production-safety question is whether duplicate, stale, expired, or concurrent transitions can create two terminal outcomes, issue credentials twice, consume an expired authorization, or keep a revoked/expired session usable.

The existing Postgres integration suite already covered the highest-risk approval and device/session race conditions:

- approved access requests are single-claim under concurrent issuance attempts
- expired approved access requests are not claimable
- approval mutations are idempotent under concurrent same-key retries
- mismatched same-key approval mutations conflict
- approve/deny and approve/expire races settle into one terminal state
- claim-at-expiry does not double issue
- device-code exchange is single-use under concurrency
- device approve/deny and approve/expire races settle into one terminal state
- denied expired device authorizations fail closed
- revoked sessions are not returned by token lookup

## Decision

Accept the current transaction design as the E4 persistence contract:

- Approval mutations use a transaction, lock idempotency rows with `FOR UPDATE`, lock the target access request with `FOR UPDATE`, and re-check idempotency after acquiring the access-request lock.
- Approved access-request claims use a single conditional `UPDATE ... RETURNING` so only one caller can move an approved request to `issued`.
- Device-code exchange locks the authorization row with `FOR UPDATE`, validates approved/not-expired/principal-present state, creates the session, and marks the authorization consumed in one transaction.
- Session lookup excludes revoked and expired sessions.
- Repeated terminal operations fail closed with domain errors instead of silently mutating state again.

## Implementation

Added focused Postgres integration regressions in `server/internal/adapters/postgres/auth_store_integration_test.go`:

- repeated session revocation returns `auth.ErrNotFound`
- exchanging an expired but previously approved device authorization returns `auth.ErrExpired`, creates no session, and does not mark the authorization consumed

No production persistence code change was required for E4.

## Verification

Run:

```bash
cd server
GOCACHE=/tmp/cloudiam-go-cache go test ./...
GOCACHE=/tmp/cloudiam-go-cache go test -tags=integration ./internal/adapters/postgres
```

The integration command requires local Postgres access through `CLOUDIAM_DATABASE_URL`.

## Consequences

- `E1-DB-001` is covered by existing and new approval/access-request persistence regressions.
- `E1-DB-002` is covered by existing and new device/session persistence regressions.
- Task 29 does not add new distributed locking, workflow orchestration, or serializable transaction infrastructure. The MVP uses single-row/conditional SQL transitions and row locks where the current state machine needs them.
