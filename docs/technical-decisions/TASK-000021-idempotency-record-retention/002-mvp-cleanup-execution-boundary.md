---
title: "MVP cleanup execution boundary for TASK-000021"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000021
tags:
  - idempotency
  - cleanup
  - operations
  - startup
---

# Decision: MVP Cleanup Execution Boundary

## Context

Task 21 needs expired idempotency records removed, but the MVP should not grow a background scheduler or retention framework. The local server already opens Postgres during startup and verifies schema readiness before serving the dashboard/API path.

`TASK-000016` also deletes an expired idempotency record for the same key/operator before processing a new mutation. That narrow path helps with one key reuse case, but it does not clean unrelated expired records.

## Options

1. Add a manual CLI cleanup command.
2. Run opportunistic cleanup during server startup.
3. Add a recurring background job.

## Decision

Use startup opportunistic cleanup for the MVP.

When the database-backed server starts, it should call a store cleanup method after schema verification and before accepting requests. The cleanup should:

- delete rows from `idempotency_records` where `expires_at <= now`
- return the number of deleted rows for logs/tests
- accept or derive one cleanup timestamp and use it consistently for the whole delete operation
- fail startup if the cleanup query fails
- avoid logging idempotency keys, request fingerprints, response bodies, or console JWTs

Keep the existing per-mutation expired-key deletion as a harmless local optimization, but do not rely on it as the task 21 cleanup mechanism.

## Edge Cases And Failure Handling

Startup cleanup must run before the HTTP and gRPC listeners begin accepting requests. If cleanup fails, startup should return an error that names idempotency cleanup without including row contents.

The cleanup operation must be idempotent:

- running it twice with the same cleanup timestamp should delete rows only on the first run
- running it when no rows are expired should succeed with a zero count
- cleanup must not fail because an expired row references an access request that still exists

Concurrent approval mutation remains protected by the mutation transaction and unique key scope. If cleanup and mutation ever overlap in a future runtime shape, the database may serialize row-level effects, but mutation code must still validate/reserve the idempotency key atomically. Cleanup must not be treated as the concurrency control mechanism.

Tests should cover:

- successful startup cleanup calls the store boundary
- cleanup failure prevents serving
- deleted-row count is available for assertions/logging
- no secret or request-body material is returned from cleanup

## Rationale

Startup cleanup gives the MVP predictable behavior without adding a scheduler, distributed locking, or operational knobs. Failing startup on cleanup failure is acceptable because idempotency storage is part of the approval safety contract; silently serving with broken cleanup would hide a persistence problem.

## Consequences

- No long-running background worker is needed.
- Tests should cover the store cleanup operation and startup wiring.
- Cleanup may not run while a long-lived local server stays up, which is acceptable for the MVP.
- A production design can replace startup cleanup with scheduled maintenance later.
