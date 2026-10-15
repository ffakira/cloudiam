---
title: "Concurrency hardening implementation slices for TASK-000028"
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
  - implementation-plan
---

# Decision: Concurrency Hardening Implementation Slices

## Context

`TASK-000028` now treats Go context propagation, cancellation, race-sensitive state, and concurrent retry behavior as security hardening concerns.

The invariants are accepted in `005-concurrency-and-context-security.md`. This decision turns those invariants into an ordered implementation queue so future Task 28 work can continue without asking which concurrency section comes next.

## Decision

Use the slices below as the Task 28 concurrency hardening backlog. Keep each slice small enough to review independently. Prefer tests that prove a security property over broad refactors.

### Slice C1: Request Context Gates

Status: completed.

Scope:

- Stop auth and access services before privileged mutations when `ctx.Err()` is already observable.
- Keep HTTP, gRPC, store, Redis, and AWS calls on caller-provided contexts.
- Reject canceled rate-limit checks instead of falling open.

Verification:

- Unit tests prove canceled contexts do not create device authorizations, approve device login, exchange sessions, create approval requests, claim approved requests, issue STS credentials, or record request-owned audit events.
- Focused race test coverage includes the local fallback rate limiter.

### Slice C2: Database Single-Use Concurrency

Status: completed.

Scope:

- Prove device authorization exchange remains single-use under concurrent polls or retries.
- Prove approved access request claiming remains single-use under concurrent issuance attempts.
- Prove approval idempotency is stable under concurrent same-key requests and rejects same-key different-fingerprint requests.

Verification:

- Postgres integration tests run concurrent goroutines against the real store methods.
- Exactly one concurrent device exchange creates a CloudIAM session; other exchanges fail closed.
- Exactly one concurrent approved access request claim succeeds before STS issuance.
- Concurrent idempotency replays return one mutation result for matching fingerprints and conflict for mismatched fingerprints.
- `go test -race -tags=integration ./internal/adapters/postgres` passes against the local Postgres test database.

### Slice C3: Expiration And State Transition Races

Status: completed.

Scope:

- Audit and test races between pending, approved, denied, expired, consumed, and issued transitions.
- Prioritize device authorization approve/expire/exchange and access request approve/deny/expire/claim paths.
- Keep existing product semantics unless the tests expose an impossible or unsafe state.

Verification:

- Tests cover concurrent approve versus exchange, approve versus expire, deny versus approve, and claim versus expiration where applicable.
- No test permits a second session or second AWS credential issuance from the same single-use object.
- `go test -race -tags=integration ./internal/adapters/postgres` passes against the local Postgres test database.

### Slice C4: Config Snapshot Concurrency

Status: completed.

Scope:

- Prove config hot reload exposes complete immutable snapshots to readers.
- Prove in-flight access decisions use one resolved target snapshot and do not observe partial reload state.
- Avoid broad config architecture changes unless a race or partial-generation exposure is found.

Verification:

- Focused tests run concurrent reloads and access requests.
- Race detector passes for config store tests.
- Tests assert old requests keep their resolved role ARN while new requests can use the reloaded role ARN.
- Tests prove initial snapshots are cloned and concurrent readers do not observe mixed role ARN and approval-required generations.
- `go test -race ./internal/app/server` passes.

### Slice C5: Shutdown And Background Work Audit

Status: completed.

Scope:

- Audit goroutine ownership in server startup and shutdown.
- Confirm request-owned work does not detach from request context.
- Confirm shutdown uses bounded contexts and does not leave privileged request work running indefinitely.

Verification:

- Tests or documented audit checks cover HTTP server goroutines, reload loops, cleanup-on-startup work, and graceful shutdown.
- Any intentional request-detached background work is documented and uses a context that does not carry raw request credentials.
- Server exit paths stop sibling HTTP and gRPC servers with bounded shutdown behavior.
- `go test -race ./internal/app/server` passes.

### Slice C6: Broad Race And Security Scans

Status: completed.

Scope:

- Run `go test -race ./...` when feasible.
- Run dependency and security scans available in the local environment.
- Treat race findings in auth, approval, access issuance, audit, config, and rate limiting as security findings until reviewed.

Verification:

- `go test -race ./...` passes for the Go server module.
- `go vet ./...` passes for the Go server module.
- `go mod verify` passes with all modules verified.
- Narrowed secret and token text scans found expected test fixtures, documentation references, CLI credential export output, cookie/session-token handling code, and the STS session-name constant; no real committed secret was identified.
- `govulncheck`, `gosec`, and `golangci-lint` are pinned in the isolated `server/tools` module and run through `make server-security-check`.
- `make server-security-check` passes: module verification, vet, unit tests, race tests, vulnerability scan, SAST, and lint.
- `govulncheck` found standard-library and `golang.org/x/net` vulnerabilities on the previous Go/tool dependency baseline; the slice fixed them by moving the server module to Go `1.26.9` and `golang.org/x/net v0.60.0`.
- Split any future broad operational, architectural, or scanner-tooling finding into a separate production-hardening task.

Tooling note:

- `govulncheck` needs network access to refresh the Go vulnerability database.

## Ordering

Use this order unless a test failure changes priority:

1. C2: Database single-use concurrency.
2. C3: Expiration and state transition races.
3. C4: Config snapshot concurrency.
4. C5: Shutdown and background work audit.
5. C6: Broad race and security scans.

## Task Boundary

Inside `TASK-000028`:

- Add focused tests for security-sensitive concurrency behavior.
- Add narrow code fixes that preserve accepted product semantics.
- Document any unimplemented production-hardening follow-up.

Outside `TASK-000028`:

- Retry/outbox redesign around STS issuance.
- Distributed locking beyond existing Postgres atomic updates or Redis counters.
- Broad worker-pool lifecycle redesign.
- Load testing, capacity planning, and production SLO work.

## Consequences

- The next Task 28 concurrency section is always the first planned slice in this ADR.
- Future work can report progress by slice ID rather than rediscovering the concurrency roadmap.
- Completion of C2 through C6 satisfies the concurrency portion of `REQ-000028-CONCURRENCY-001`.
