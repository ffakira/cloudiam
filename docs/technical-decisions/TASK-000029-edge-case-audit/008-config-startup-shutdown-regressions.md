---
title: "Config startup and shutdown regressions"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - config
  - startup
  - shutdown
  - testing
---

# Decision: Config Startup And Shutdown Regressions

## Context

E5 covers config reload, startup dependency checks, and shutdown/background-work behavior. The production-safety question is whether malformed reloads, canceled contexts, restart-required config changes, missing dependencies, or shutdown paths can leave the server in a partial or unsafe state.

The existing server app tests already covered:

- missing config path
- missing auth database URL for authenticated access
- invalid YAML and semantically invalid config validation
- forbidden secret-bearing config fields
- config reload preserving the previous generation on invalid YAML or invalid semantic config
- complete atomic snapshots under concurrent reload/read pressure
- in-flight requests staying on their original snapshot
- SIGHUP reload path
- graceful gRPC shutdown with forced stop fallback
- non-blocking server error reporting
- reload/startup error redaction
- Redis rate-limit local fallback when Redis is not configured
- invalid Redis URL fail-fast behavior
- production-only hardening requirements for secure cookies, Redis, HTTPS verification URI, database TLS, weak JWT secrets, and console seeding

## Decision

Accept E5 as covered by the existing lifecycle contract plus two added reload regressions:

- canceled reload contexts must fail before replacing the active snapshot
- AWS region changes must be rejected as restart-required, like gRPC bind address changes

No new runtime subsystem is needed for Task 29. Redis fallback remains allowed outside production; production mode requires `CLOUDIAM_REDIS_URL`. Full process supervision, health orchestration, and distributed config rollout are outside Task 29 unless a concrete MVP bug appears.

## Implementation

Added focused regressions in `server/internal/app/server/app_test.go`:

- `TestConfigStoreReloadRejectsAWSRegionChange`
- `TestConfigStoreReloadCanceledContextKeepsOldSnapshot`

No production code change was required for E5.

## Verification

Run:

```bash
cd server
GOCACHE=/tmp/cloudiam-go-cache go test ./internal/app/server
GOCACHE=/tmp/cloudiam-go-cache go test ./...
```

## Consequences

- `E1-CONFIG-001` is covered by config validation, reload failure preservation, restart-required field checks, canceled-context behavior, and concurrent snapshot tests.
- `E1-RUNTIME-001` is covered by startup dependency validation, production hardening validation, Redis fallback/production requirement tests, startup cleanup error redaction, and shutdown tests.
- E5 does not add process-manager, Kubernetes, distributed config, or production secret-manager work; those remain production-platform concerns outside this bounded audit.
