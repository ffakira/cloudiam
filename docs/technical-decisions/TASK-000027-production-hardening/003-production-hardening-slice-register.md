---
title: "Production hardening slice register for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - slice-register
  - planning
  - scope
---

# Decision: Production Hardening Slice Register

## Context

The user asked for a finite production-readiness definition before implementation. Prior hardening tasks used bounded slice registers to prevent never-ending work.

## Decision

Use this finite slice register for `TASK-000027`.

| Slice | Name | Outcome | Closure Evidence |
|---|---|---|---|
| P1 | Production readiness contract | Define production ready, stop rule, and accepted deferrals. | ADRs 001 through 003 are accepted and task doc links them. |
| P2 | Production config and startup validation | Production mode rejects unsafe defaults and missing critical config before serving traffic. | Tests cover accepted startup failures and safe local/demo exceptions. |
| P3 | Health, readiness, and dependency checks | Liveness and readiness are distinct; readiness reports required dependency state without secrets. | HTTP tests or app tests cover healthy and unhealthy readiness cases. |
| P4 | Runtime lifecycle and background work | HTTP/gRPC shutdown, DB/Redis clients, config reload, and cleanup jobs have bounded behavior. | Tests or runbook exercises cover shutdown/reload/cleanup paths. |
| P5 | Observability checklist | Critical flows map to logs/audit/metrics/readiness signals. | Docs map auth, authorization, approval, STS, audit, reload, rate limit, cleanup, DB, Redis. |
| P6 | Production runbook | Operators can start, validate, reload, roll back, and triage common incidents. | Runbook exists and references exact config, commands, signals, and residual risks. |
| P7 | Closure gate and residual-risk report | Verify implementation and document what remains out of scope. | Security/build/test gates pass or failures are explicitly documented; closure ADR records residual risk. |

## Non-Slices

Do not add these to Task 27:

- production SSO/OIDC implementation
- external secret-manager implementation
- SIEM/export implementation
- multi-account or multi-cloud expansion
- session recording
- password delivery/rotation
- threat analytics/user behavior analytics
- policy editor or dashboard product expansion
- broad load/chaos/disaster-recovery program

These can be follow-up tasks if the interview or product roadmap needs them.

## Consequences

- The user no longer has to ask "what is next?" for Task 27.
- Implementation can proceed in order from P2 through P7.
- New findings are classified as immediate fixes only if they violate the accepted production readiness invariants.
