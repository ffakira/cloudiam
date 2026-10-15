---
title: "Production hardening acceptance invariants for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - invariants
  - reliability
  - scope
---

# Decision: Production Hardening Acceptance Invariants

## Context

Production hardening must be finite. Every system has more possible reliability and operations work than one task can complete.

The acceptance invariants below define what belongs in `TASK-000027` and what becomes explicit follow-up work.

## Decision

Use these invariants to decide whether a finding belongs in Task 27.

### Invariant 1: Runtime Safety First

Task 27 work must improve one of these existing runtime boundaries:

- startup validation
- production-mode defaults
- dependency readiness
- HTTP/gRPC lifecycle behavior
- database and Redis dependency behavior
- config reload behavior
- background cleanup behavior
- AWS STS dependency behavior
- operator-facing diagnostics
- runbook coverage

If a finding requires a new product workflow, it becomes backlog.

### Invariant 2: Fail Closed For Unsafe Production State

In production mode, the server must reject unsafe state rather than silently falling back to demo behavior.

Examples:

- weak JWT or session secrets
- insecure console cookies
- missing Redis when shared production rate limits are required
- demo console user seeding without explicit production override
- missing required AWS region or invalid AWS runtime config
- missing database migrations or incomplete schema
- production verification URLs that use local/insecure HTTP assumptions

### Invariant 3: Fail Predictably For Dependency Problems

Dependency failures must become stable, diagnosable states.

Accepted outcomes:

- startup failure with non-secret error text
- readiness failure while liveness remains healthy
- bounded shutdown fallback
- previous known-good config remains active
- cleanup failure is surfaced as an operator signal

Rejected outcomes:

- accepting privileged requests with missing critical dependencies
- exposing raw DSNs, Redis URLs, AWS credentials, JWTs, device codes, session tokens, or password hashes
- hanging indefinitely during startup or shutdown

### Invariant 4: Operability Without Secret Leakage

Operators need enough context to triage, but diagnostics must not expose:

- AWS credentials
- CloudIAM session tokens
- console JWTs
- raw device/user codes
- raw idempotency keys
- database, Redis, password, or JWT signing secret material

If diagnostic detail conflicts with no-leakage, no-leakage wins.

### Invariant 5: Liveness Is Not Readiness

A running process is not necessarily ready to serve privileged traffic.

Task 27 must distinguish:

- liveness: process is up and event loop/server can respond
- readiness: required dependencies and runtime state are healthy enough to accept traffic

Readiness should cover the dependencies and state needed by the current MVP, not every possible future integration.

### Invariant 6: Documentation Is Part Of The Control

For production hardening, docs are not optional polish. If an operator cannot determine what to do during startup failure, reload failure, readiness failure, audit failure, or STS failure, the control is incomplete.

Task 27 must leave a runbook or equivalent operational guide.

### Invariant 7: Explicit Residual Risk

When a production concern is real but larger than the AWS-only MVP boundary, record:

- the residual risk
- why it is accepted now
- what trigger would make it mandatory
- how it should be verified when implemented

## Stop Rule

Task 27 stops when the accepted slice register is complete, verification passes, and residual risks are documented.

Do not keep adding production slices unless a current slice reveals a concrete unsafe production behavior that must be fixed before closure.
