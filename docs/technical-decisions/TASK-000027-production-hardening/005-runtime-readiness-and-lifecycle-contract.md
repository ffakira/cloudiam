---
title: "Runtime readiness and lifecycle contract for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - readiness
  - lifecycle
  - shutdown
---

# Decision: Runtime Readiness And Lifecycle Contract

## Context

Production operation needs more than successful startup. Operators and orchestrators need to know whether the process is alive, whether it is ready to accept privileged traffic, and whether shutdown/reload paths behave predictably.

## Decision

Task 27 must define and implement a runtime lifecycle contract for the current MVP.

## Liveness

Liveness means the process is up and the HTTP server can respond to a basic health request.

Liveness must not require:

- Postgres
- Redis
- AWS STS
- current config reload success
- background cleanup success

A dependency outage should not automatically make the process look dead.

## Readiness

Readiness means the process can safely accept normal privileged traffic for the current MVP.

Readiness should account for:

- active config snapshot exists and is valid
- Postgres is reachable when persistence is required
- required schema has been verified
- Redis is reachable when production shared rate limiting is required
- AWS runtime configuration needed for STS issuance is present
- startup cleanup is not in a known failed state that makes idempotency behavior unsafe

Readiness responses must be generic and non-secret. They may name failing subsystems, but must not include raw DSNs, Redis URLs, AWS credentials, tokens, or secret values.

## Shutdown

Shutdown must be bounded and predictable.

Expected behavior:

- HTTP and gRPC stop accepting new requests.
- In-flight requests receive the configured graceful window where possible.
- Background cleanup stops or exits on context cancellation.
- Database and Redis clients are closed after serving stops.
- Shutdown fallback prevents indefinite hangs.

## Config Reload

Reload remains snapshot-based:

- valid reload replaces the active snapshot
- invalid reload preserves the previous known-good snapshot
- reload status is observable without secret leakage
- restart-required changes remain documented as restart-required

## Verification Expectations

P3 and P4 should add or confirm tests for:

- liveness succeeds when dependencies are unhealthy
- readiness fails when required dependencies are unhealthy
- readiness error bodies do not leak secrets
- config reload failure preserves the previous snapshot
- shutdown completes under bounded contexts
- cleanup failures become operator-visible without accepting unsafe state

## Consequences

- Orchestrators can make better traffic decisions.
- Operators can diagnose dependency problems without reading raw logs first.
- Some readiness checks may be conservative; that is acceptable for a privileged access broker.
