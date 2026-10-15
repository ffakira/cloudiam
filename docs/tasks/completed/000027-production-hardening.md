---
title: "Production hardening"
id: TASK-000027
type: chore
status: completed
created_at: 2026-10-10
updated_at: 2026-10-12
completed_at: 2026-10-12
owner: codex
phase: "hardening"
priority: P1
sprint_candidate: true
tags:
  - production
  - reliability
  - operations
---

# Task: Production Hardening

## Outcome

Review the completed MVP tasks as a production-readiness surface and harden runtime behavior, deployment defaults, operational visibility, and failure handling without expanding the product scope.

Completed with a bounded production-readiness definition, fail-closed production startup validation, dependency readiness checks, runtime lifecycle coverage, an observability checklist, a production runbook, final verification gates, and an explicit residual-risk report.

## Scope

- Audit completed tasks `TASK-000001` through `TASK-000026` for production gaps.
- Define startup and shutdown behavior for HTTP, gRPC, database, and background cleanup paths.
- Ensure required production configuration has explicit validation and safe failures.
- Add health/readiness checks that distinguish process liveness from dependency readiness.
- Confirm Docker and local/demo packaging do not hide production-required secrets or defaults.
- Verify observability signals for authentication, authorization, approval, AWS STS calls, config reloads, and cleanup jobs.
- Document a production runbook for startup, config reload, rollback, and incident triage.

## Non-Goals

- Multi-account support.
- Multi-cloud provider support.
- New dashboard product features.
- Production identity provider integration beyond documenting the boundary.

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000027-CONFIG-001 | Production-required config is validated at startup and unsafe defaults fail closed. | Unit/integration tests cover missing and invalid production config. |
| REQ-000027-RUNTIME-001 | HTTP, gRPC, database, and cleanup paths have predictable lifecycle behavior. | Tests or runbook exercises cover startup, readiness, graceful shutdown, and cleanup scheduling. |
| REQ-000027-OBS-001 | Critical flows emit enough logs/audit/metrics for production triage. | Observability checklist maps each flow to its signal. |
| REQ-000027-DOCS-001 | Operators have a production hardening runbook. | README or docs include production startup, reload, rollback, and troubleshooting steps. |

## Production-Readiness Decisions

Task 27 is bounded by:

- `docs/technical-decisions/TASK-000027-production-hardening/001-production-readiness-definition.md`
- `docs/technical-decisions/TASK-000027-production-hardening/002-production-hardening-acceptance-invariants.md`
- `docs/technical-decisions/TASK-000027-production-hardening/003-production-hardening-slice-register.md`
- `docs/technical-decisions/TASK-000027-production-hardening/004-production-config-and-startup-contract.md`
- `docs/technical-decisions/TASK-000027-production-hardening/005-runtime-readiness-and-lifecycle-contract.md`
- `docs/technical-decisions/TASK-000027-production-hardening/006-observability-and-runbook-contract.md`

The task is not "make CloudIAM a complete enterprise platform." It is "make the current AWS-only MVP safe and diagnosable enough to operate with explicit residual risks."

## Production Hardening Slices

Finite slice register:

- P1: Production readiness contract. Completed.
- P2: Production config and startup validation. Completed.
- P3: Health, readiness, and dependency checks. Completed.
- P4: Runtime lifecycle and background work. Completed.
- P5: Observability checklist. Completed.
- P6: Production runbook. Completed.
- P7: Closure gate and residual-risk report. Completed.

No new Task 27 slices should be added unless a current slice reveals a concrete unsafe production behavior that must be fixed before closure. Product-expanding findings become backlog tasks.

## P1 Closure: Production Readiness Contract

Completed on 2026-10-12.

P1 establishes the production-readiness boundary before implementation:

- `001-production-readiness-definition.md` defines production ready for this AWS-only MVP as safe startup, predictable dependency failure, operator visibility, and no insecure production defaults.
- `002-production-hardening-acceptance-invariants.md` defines the stop rule, fail-closed behavior, liveness/readiness split, no-leakage requirement, documentation requirement, and residual-risk rule.
- `003-production-hardening-slice-register.md` defines the finite Task 27 sequence from P1 through P7.

Accepted P1 deferrals:

- production SSO/OIDC implementation
- external secret-manager implementation
- SIEM/export implementation
- multi-account and multi-cloud expansion
- session recording
- password delivery and rotation
- threat analytics and user behavior analytics
- dashboard policy editor or product expansion
- broad load, chaos, and disaster-recovery program

Next slice: P2, production config and startup validation.

## P2 Closure: Production Config And Startup Validation

Completed on 2026-10-12.

P2 reviewed the production startup contract against existing runtime checks and closed the remaining production verification URL gap:

- production mode now requires `CLOUDIAM_AUTH_VERIFICATION_URI` to be explicitly set
- production mode still requires `CLOUDIAM_AUTH_VERIFICATION_URI` to use `https://`
- local/demo mode can continue using the local HTTP device approval default

Existing P2 startup gates remain accepted:

- secure console cookies are required in production
- Redis URL is required in production for shared rate limiting
- database `sslmode=disable` is rejected in production unless explicitly overridden
- weak or placeholder console JWT secrets are rejected
- console seed password is disabled in production unless explicitly overridden
- weak production console seed passwords are rejected even with override
- persistence schema verification fails startup before serving partial database-backed state
- startup cleanup errors are redacted

Verification:

```bash
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./internal/app/server
```

P2 decision record:

- `docs/technical-decisions/TASK-000027-production-hardening/007-production-config-and-startup-validation.md`

Next slice: P3, health, readiness, and dependency checks.

## P3 Closure: Health, Readiness, And Dependency Checks

Completed on 2026-10-12.

P3 keeps liveness and readiness separate:

- `GET /healthz` remains liveness and returns `204 No Content` when the HTTP process can answer.
- `GET /readyz` is readiness and returns JSON status for active config, Postgres, and Redis checks.
- readiness returns `503 Service Unavailable` when a required dependency check fails.
- readiness errors are redacted before returning JSON.
- Redis readiness is registered only when Redis-backed rate limiting is configured.

Verification:

```bash
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./internal/app/server ./internal/adapters/http
```

P3 decision record:

- `docs/technical-decisions/TASK-000027-production-hardening/008-health-readiness-and-dependency-checks.md`

Next slice: P4, runtime lifecycle and background work.

## P4 Closure: Runtime Lifecycle And Background Work

Completed on 2026-10-12.

P4 confirms the runtime lifecycle model:

- HTTP shutdown is bounded and returns after timeout even when a handler is still blocked.
- gRPC shutdown uses graceful stop with forced stop fallback.
- config reload preserves the previous known-good snapshot on invalid/canceled/restart-required reloads.
- concurrent config reload/read paths see complete snapshots.
- `SIGHUP` reload status is emitted with redacted errors.
- startup idempotency cleanup runs before serving and redacts lower-level error details.
- server error reporting does not block on a full error channel.

Verification:

```bash
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./internal/app/server
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./...
```

P4 decision record:

- `docs/technical-decisions/TASK-000027-production-hardening/009-runtime-lifecycle-and-background-work.md`

Next slice: P5, observability checklist.

## P5 Closure: Observability Checklist

Completed on 2026-10-12.

P5 adds the production observability map for the current AWS-only MVP:

- durable audit events
- liveness and readiness endpoints
- redacted reload status
- startup errors
- generic API responses
- lifecycle test coverage

Checklist:

- `docs/operations/observability-checklist.md`

P5 decision record:

- `docs/technical-decisions/TASK-000027-production-hardening/010-observability-checklist.md`

Accepted P5 gaps:

- Prometheus metrics are not implemented yet.
- OpenTelemetry tracing is not implemented yet.
- SIEM/export and alert routing are not implemented yet.
- Device-login lifecycle audit events are not first-class `audit_events` yet.
- Rate-limit hits are not durable audit events yet.
- Startup cleanup success count is not externally emitted yet.

Next slice: P6, production runbook.

## P6 Closure: Production Runbook

Completed on 2026-10-12.

P6 adds the production runbook for the current AWS-only MVP:

- required production configuration
- startup procedure
- startup failure triage
- liveness and readiness checks
- config reload procedure
- rollback procedure
- incident triage
- shutdown expectations
- accepted residual risks
- safe communication boundaries for incidents

Runbook:

- `docs/operations/production-runbook.md`

P6 decision record:

- `docs/technical-decisions/TASK-000027-production-hardening/011-production-runbook.md`

P6 handed off to P7 for final closure verification.

## P7 Closure: Closure Gate And Residual-Risk Report

Completed on 2026-10-12.

P7 closes Task 27 after final verification:

- server unit/integration suite passed
- client production build passed, including browser secret-boundary assertion
- server security gate passed, including module verification, vet, tests, race tests, `govulncheck`, `gosec`, and `golangci-lint`
- residual risks are documented as explicit scope boundaries, not hidden TODOs

Verification:

```bash
cd server
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./...
cd ..
make client-build
make server-security-check
```

P7 decision record:

- `docs/technical-decisions/TASK-000027-production-hardening/012-closure-gate-and-residual-risk-report.md`

Task 27 is complete.
