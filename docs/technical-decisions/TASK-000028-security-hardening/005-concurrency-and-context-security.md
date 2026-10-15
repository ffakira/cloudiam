---
title: "Concurrency and context security invariants for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - concurrency
  - context
  - race-detection
  - denial-of-service
---

# Decision: Concurrency And Context Security Invariants

## Context

CloudIAM is a Go service whose security boundaries depend on single-use device authorization, idempotent approval mutations, claim-before-STS issuance, config hot reload, Redis-backed rate limiting, and request-scoped database work.

In Go, race and context bugs can become security bugs even when no cryptographic or authorization primitive is directly broken. A data race can corrupt authorization state, a missing context can let attacker-controlled requests pin resources, and a non-atomic retry path can double-issue credentials.

`TASK-000028` should therefore treat concurrency and context handling as part of security hardening when the behavior affects authentication, authorization, replay resistance, credential issuance, audit integrity, or denial-of-service resistance.

## Related Work

- `TASK-000003`: device authorization exchange is single-use.
- `TASK-000010`: config hot reload uses snapshots and must not expose partial generations.
- `TASK-000016`: approval mutations use idempotency keys.
- `TASK-000021`: idempotency records have a bounded replay window.
- `TASK-000023`: approved access issuance uses claim-before-STS semantics.
- `TASK-000028`: Redis-backed rate limiting adds shared and fallback counters for unauthenticated HTTP routes.

## Decision

Use the following invariants for concurrency and context security review.

### Context Invariants

- HTTP and gRPC handlers must pass request contexts into core services, stores, Redis operations, and AWS calls.
- A canceled request context must stop new privileged work before state mutation when the code can observe cancellation.
- Database transactions and Redis calls must use bounded request or shutdown contexts rather than background contexts.
- Server shutdown must use bounded contexts for graceful HTTP shutdown and must not leave request-owned privileged work running indefinitely.
- Background work that intentionally outlives a request must be explicit, documented, and must not carry raw request credentials or mutable request state.

### Race And Atomicity Invariants

- Device authorization exchange must be single-use under concurrent polls or retries.
- Approved access requests must be claimed before STS issuance and must not issue credentials twice under concurrent retries.
- Approval idempotency records must reject same-key different-fingerprint reuse under concurrent requests.
- Config hot reload must expose complete immutable snapshots; readers must not observe partial config generations.
- In-memory fallback rate limiting must be safe under concurrent requests.
- Shared Redis rate limiting must fall back locally on Redis errors without disabling abuse protection.

### Verification Invariants

- `go test -race ./...` is the preferred broad race screen for Task 28 when local runtime cost is acceptable.
- Focused tests should cover cancellation, replay, and concurrent retry behavior where a broad race run cannot prove the security property.
- Integration tests should cover database-enforced atomicity for device exchange, idempotency, and approved-request claiming when Postgres is available.
- Any race detector finding in auth, approval, access issuance, audit recording, config snapshots, or rate limiting is security-relevant until proven otherwise.

## Task Boundary

Fix focused gaps inside `TASK-000028` when the issue affects:

- authentication or session creation
- authorization decisions
- replay resistance
- credential issuance
- audit integrity
- unauthenticated route abuse resistance
- denial-of-service exposure from request-controlled work

Split larger work into production-hardening tasks when the finding requires:

- retry/outbox redesign for STS issuance
- distributed locks beyond existing database claims or Redis counters
- broad worker-pool lifecycle redesign
- production SSO/OIDC or long-lived revocation infrastructure
- load testing or operational capacity planning

## Consequences

- Task 28 includes concurrency and context audit work, not only API-level authentication checks.
- Race detector and cancellation tests are valid security verification for this task.
- Production-hardening follow-ups remain appropriate for broad operational resilience work that exceeds narrow security fixes.
