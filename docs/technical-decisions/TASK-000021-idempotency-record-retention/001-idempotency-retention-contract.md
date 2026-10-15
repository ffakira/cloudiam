---
title: "Idempotency retention contract for TASK-000021"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000021
tags:
  - idempotency
  - retention
  - persistence
  - approval
---

# Decision: Idempotency Retention Contract

## Context

`TASK-000016` added idempotent console approve/deny mutations. Each successful mutation stores a replay record in `idempotency_records` with the raw key, console operator subject, request fingerprint, stable response body, `created_at`, and `expires_at`.

The existing contract suggests at least 24 hours of replay safety. The current schema already has `expires_at` and an index on it, so task 21 should not introduce a second retention timestamp.

## Decision

Use `idempotency_records.expires_at` as the sole retention boundary.

For the MVP, approval idempotency records expire 24 hours after creation. Cleanup may delete only rows where:

```sql
expires_at <= now
```

Cleanup must not inspect or mutate `access_requests`. It only removes replay records whose replay window has ended.

The implementation should preserve the existing unique key scope:

```text
(key, admin_subject)
```

After an expired record is cleaned up, the same idempotency key may be accepted again for that console operator as a new mutation. That is acceptable because the replay safety window has ended.

## Edge Cases And Test Matrix

Cleanup must use a single cleanup timestamp supplied through the store boundary. Tests should not depend on real wall-clock timing.

The cleanup boundary is inclusive:

| Record `expires_at` | Cleanup time | Expected |
|---|---|---|
| before cleanup time | fixed time | deleted |
| equal to cleanup time | fixed time | deleted |
| after cleanup time | fixed time | preserved |

Retention tests should also cover:

- active records remain replayable or conflict-producing before expiry
- expired records from multiple operators and access requests are eligible for cleanup
- cleanup deletes idempotency rows only
- access request status and approval metadata remain unchanged
- audit events remain unchanged
- raw keys, fingerprints, response bodies, JWTs, CloudIAM tokens, and AWS credentials are not exposed by cleanup return values
- a key reused after its expired row is deleted is treated as a fresh mutation, not as a replay

## Rationale

The idempotency row protects against browser retries, double-clicks, and short-term network uncertainty. A 24-hour replay window is enough for the local/demo console path and matches the earlier `TASK-000016` contract.

Keeping one retention timestamp avoids ambiguous cleanup rules. Keeping cleanup independent from `access_requests` avoids deleting domain history: approved/denied/issued request state and approval audit events remain durable after replay records expire.

## Consequences

- No new migration is required for an expiry column.
- Store tests should assert that active records are preserved and expired records are deleted.
- Boundary tests should assert `expires_at <= cleanup_time` deletion and `expires_at > cleanup_time` preservation.
- Documentation should state the 24-hour MVP retention window.
- A future production system may add configurable retention or scheduled jobs, but that is outside this task.
