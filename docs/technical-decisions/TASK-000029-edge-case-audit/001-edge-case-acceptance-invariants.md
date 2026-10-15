---
title: "Edge-case acceptance invariants for TASK-000029"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - reliability
  - production-safety
  - scope
---

# Decision: Edge-Case Acceptance Invariants

## Context

`TASK-000029` can become unbounded because every production system has more edge cases than one task can responsibly close.

The task must prove that CloudIAM understands production safety without turning into speculative platform work. The target is the MVP we have: AWS-only access brokerage, CLI login/access, gRPC access issuance, HTTP dashboard/device approval, persistence, config reload, audit, and local/demo runtime wiring.

Task 29 is not "find every edge case." It is "prove the MVP fails closed and predictably at the boundaries we already own."

## Decision

Use these invariants to decide what belongs in Task 29.

### Invariant 1: Boundary-First

Every Task 29 edge case must sit at an existing boundary:

- CLI input and local session file behavior
- gRPC request/response behavior
- HTTP request/response behavior
- dashboard form/API behavior
- persistence state transitions
- config validation and hot reload
- startup and shutdown
- AWS STS adapter behavior
- audit/event recording behavior

If the edge case requires a new workflow, new subsystem, new product feature, or broader operating model, it becomes backlog.

### Invariant 2: Fail-Closed

Malformed, ambiguous, stale, expired, duplicate, or dependency-failed states must resolve to one of:

- reject the request
- no-op safely
- return a retry-safe response
- preserve the previous known-good state
- emit an explicit non-secret operator error
- create a durable non-secret audit event when the decision is security-relevant

Task 29 must not introduce "best effort" behavior that grants access when state is unclear.

### Invariant 3: No Secret Expansion

Edge-case handling must not add new exposure for:

- CloudIAM session tokens
- console JWTs
- raw device codes
- user codes outside transient request/form use
- raw idempotency keys
- AWS credentials
- database, Redis, password, or JWT signing secret material

If diagnostic value conflicts with no-leakage, no-leakage wins.

### Invariant 4: No Duplicate Authority

Retries, duplicate clicks, duplicate polls, stale pages, concurrent requests, clock-boundary cases, or partial failures must not create:

- duplicate approval authority
- duplicate credential issuance from one approved request
- privilege escalation from spoofed principal/target/provider/account/role fields
- replay outside accepted idempotency or token lifetimes

### Invariant 5: Operator Predictability

Errors should be stable and actionable enough for an operator to identify the failing subsystem without exposing internals.

Accepted examples:

- "config reload failed"
- "console session cookie is invalid"
- "access request expired"
- "resolve cloudiam session failed"

Rejected examples:

- raw SQL rows
- raw DSNs
- raw Redis URLs with credentials
- AWS credential values
- raw JWTs, session tokens, device codes, or idempotency keys

### Invariant 6: Existing Architecture Only

Task 29 must work through existing seams:

- validation
- error mapping
- adapters
- domain services
- stores
- focused tests
- ADR-backed deferrals

Do not add a workflow engine, policy engine, queue, external observability platform, or new browser E2E stack during Task 29 unless a concrete high-risk bug cannot be closed without it.

### Invariant 7: One Fix Per Real Gap

When Task 29 finds a real bug, fix it narrowly and add a focused regression.

When Task 29 finds a production concern that is bigger than the MVP boundary, record it as backlog with:

- scope
- trigger for implementation
- risk rationale
- verification expectations

## What Task 29 Will Not Implement Now

These are out of scope unless a current MVP bug makes them mandatory:

- production SSO/OIDC edge cases
- multi-account authorization edge cases
- multi-cloud provider edge cases
- distributed workflow orchestration
- full chaos testing
- full load testing
- broad browser E2E automation
- SIEM/export/alerting behavior
- external secret-manager rotation behavior
- long-lived refresh-token family or revocation-list semantics

## Consequences

- Task 29 can stop after bounded boundary coverage is complete.
- The task is allowed to say "not now" when the answer would be product or production-platform expansion.
- Follow-up work must be explicit backlog, not hidden TODOs.
