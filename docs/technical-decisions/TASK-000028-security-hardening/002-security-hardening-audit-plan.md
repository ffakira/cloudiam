---
title: "Security hardening audit plan for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - audit
  - testing
---

# Decision: Security Hardening Audit Plan

## Context

The completed MVP now has multiple security-sensitive flows: CLI device login, CloudIAM sessions, gRPC access requests, YAML governance config, optional console resource approval, idempotency, audit history, cookie-backed console sessions, persisted console users, and AWS STS issuance.

A production push needs a focused audit that turns earlier ADRs into executable checks without broadening the product to multi-account, multi-cloud, or production SSO.

## Decision

Run `TASK-000028` as a staged audit and hardening pass.

### Stage 1: Contract Extraction

Extract security invariants from accepted ADRs and completed task files.

Primary outputs:

- invariant register
- code-area checklist
- list of stale or superseded assumptions

### Stage 2: Code Audit

Review these areas against the invariant register:

- `server/internal/app/server`
- `server/internal/core/auth`
- `server/internal/core/consoleauth`
- `server/internal/core/approval`
- `server/internal/core/access`
- `server/internal/core/audit`
- `server/internal/adapters/http`
- `server/internal/adapters/grpc`
- `server/internal/adapters/postgres`
- `server/internal/adapters/aws`
- `server/internal/adapters/cli`
- `client/src/api`
- `client/src/routes`

Audit questions:

- Can an unauthenticated caller reach a privileged mutation?
- Can a caller reuse a token, idempotency key, device code, or approved request outside its intended window?
- Can caller-controlled input influence provider, account, role ARN, credential values, approval state, or audit metadata beyond the accepted contract?
- Can any response, log, audit row, idempotency row, persisted session, or client state expose secret material?
- Can hot reload or stale config change the permission boundary of an already-created approval object?
- Can concurrent retries double-issue CloudIAM sessions or AWS credentials?
- Can request cancellation or timeout still allow privileged state mutation after the boundary should have stopped?
- Can concurrent retries double-consume device authorizations, approval idempotency keys, approved access requests, rate-limit buckets, or config snapshots?
- Can goroutines, transactions, Redis calls, or AWS calls outlive their intended context and create denial-of-service risk?

### Stage 3: Verification

Prefer focused tests before production polish:

- handler tests for route/auth boundaries
- core service tests for session, replay, state-transition, and smuggling cases
- Postgres integration tests for persistence invariants where the database enforces state
- focused cancellation tests for privileged mutations
- concurrency tests for database-enforced single-use, idempotency, and claim-before-issuance contracts
- source scans for forbidden client-side token persistence and forbidden response fields
- `go test ./...` with an isolated build cache
- `go test -race ./...` when feasible
- dependency/security scanning when tools are available in the local environment

### Stage 4: Fix Or Split

Fix narrow hardening gaps inside `TASK-000028`.

Create separate backlog tasks when a finding requires:

- production SSO/OIDC
- multi-account support
- multi-cloud support
- long-lived token revocation infrastructure
- new dashboard product workflows
- substantial retry/outbox redesign for STS issuance

## Consequences

- Task 28 can include both documentation and code/test hardening.
- The first implementation step after this decision should be a code-area checklist that maps each invariant to existing tests and missing tests.
- Findings should be ranked by security impact and fixed in small, reviewable slices.
