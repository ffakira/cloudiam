---
title: "Health readiness and dependency checks for TASK-000027"
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
  - health
  - dependencies
  - verification
---

# Decision: Health, Readiness, And Dependency Checks

## Context

P3 implements the runtime readiness contract from ADR 005.

Before P3, the HTTP server exposed `/healthz` as a simple liveness endpoint. That was useful for "is the process answering HTTP?" but not enough for production traffic decisions. A process can be alive while Postgres, Redis, or active config state is not healthy enough for privileged access traffic.

## Decision

Keep `/healthz` as liveness and add `/readyz` as readiness.

`GET /healthz`:

- returns `204 No Content`
- checks only that the HTTP process can answer
- does not require Postgres, Redis, AWS STS, config reload success, or cleanup success

`GET /readyz`:

- returns `200 OK` with JSON status `ready` when required runtime dependencies pass
- returns `503 Service Unavailable` with JSON status `not_ready` when any dependency fails
- rejects non-GET methods with `405 Method Not Allowed`
- redacts secret-shaped error details before returning JSON

Current readiness checks:

- active config snapshot exists
- Postgres ping succeeds when database-backed runtime is configured
- Redis ping succeeds when Redis-backed rate limiting is configured

AWS STS is not called from readiness. The current readiness contract treats AWS runtime configuration as startup/config validation work, while actual STS dependency failures remain observable through access issuance failures and audit behavior. A future production platform may add a bounded AWS identity check if needed, but readiness must not mint credentials or perform privileged cloud actions.

## Verification

Focused verification passed on 2026-10-12:

```bash
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./internal/app/server ./internal/adapters/http
```

Regression coverage added:

- readiness checker reports `ready` when dependencies pass
- readiness checker reports `not_ready` when a dependency fails
- readiness dependency errors are redacted before JSON output
- readiness handler returns `503 Service Unavailable` when not ready
- readiness handler rejects non-GET methods
- Redis-backed rate limiter exposes a readiness check without changing local fallback behavior

## Consequences

- Orchestrators can distinguish process liveness from traffic readiness.
- Production operators can identify failing subsystems without leaking DSNs, Redis credentials, JWTs, device codes, idempotency keys, or AWS credential material.
- Local/demo mode continues to work without Redis; Redis readiness is registered only when Redis-backed rate limiting is configured.
- P3 closes without adding metrics, SIEM/export, external alerting, or a broad observability platform; those belong to P5/P6 or later production work.
