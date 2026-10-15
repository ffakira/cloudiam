---
title: "Idempotency record retention"
id: TASK-000021
type: chore
status: completed
created_at: 2026-10-07
updated_at: 2026-10-10
completed_at: 2026-10-10
owner: codex
phase: "operations-1"
priority: P2
sprint_candidate: false
outcome: "Expired console approval idempotency records are cleaned up at startup using the existing expires_at retention boundary."
tags:
  - idempotency
  - operations
  - cleanup
---

# Completed Task: Idempotency Record Retention

## Outcome

Console approval idempotency records now use `idempotency_records.expires_at` as the single retention boundary. The MVP keeps the 24-hour replay window from `TASK-000016` and removes expired rows during database-backed server startup.

Cleanup deletes only idempotency records where `expires_at <= cleanup_time`. It returns a deleted-row count for startup/tests and does not expose idempotency keys, fingerprints, response bodies, JWTs, CloudIAM tokens, device/user codes, or AWS credentials.

## Edge Cases Covered

- Active records survive cleanup.
- Records expiring exactly at the cleanup timestamp are deleted.
- Cleanup does not mutate access requests, approval state, or audit events.
- A key remains blocked before expiry and can be accepted as a fresh mutation after expired cleanup.
- Startup cleanup errors fail startup with contextual, non-secret errors.

## Verification

- `cd server && GOCACHE=/tmp/segura-go-build go test ./...`
- `GOCACHE=/tmp/segura-go-build make server-test-integration`
