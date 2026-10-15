---
title: "Edge-case audit slice register for TASK-000029"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - planning
  - scope
  - reliability
---

# Decision: Edge-Case Audit Slice Register

## Context

`TASK-000029` covers edge cases across the completed MVP. Without a finite queue, the task can expand forever because every boundary has more rare cases.

This register freezes the implementation slices and defines what each slice must produce.

## Decision

Use this finite slice register for Task 29.

| Slice | Name | Status | Primary requirement | Output |
|---|---|---|---|---|
| E1 | Edge-case matrix and stop rule | Completed | `REQ-000029-MATRIX-001`, `REQ-000029-FOLLOWUP-001` | ADR 004; initial MVP flow matrix, severity/action rules, and explicit stop rule |
| E2 | CLI and gRPC malformed/stale session behavior | Completed | `REQ-000029-TESTS-001` | ADR 005; CLI parser/session regressions and existing gRPC malformed/stale session coverage |
| E3 | HTTP and dashboard form/API edge cases | Completed | `REQ-000029-TESTS-001` | ADR 006; focused HTTP boundary regressions plus existing dashboard state scrubbing and client scanner coverage |
| E4 | Persistence and state-transition oddities | Completed | `REQ-000029-RACE-001` | ADR 007; focused Postgres regressions plus existing approval/device/session race and transaction-boundary coverage |
| E5 | Config reload, startup, and shutdown edge cases | Completed | `REQ-000029-TESTS-001` | ADR 008; focused config reload regressions plus existing startup, Redis fallback, production hardening, and shutdown coverage |
| E6 | AWS adapter and credential issuance edge cases | Completed | `REQ-000029-TESTS-001` | ADR 009; STS boundary hardening, provider/incomplete-credential regressions, and credential-issuance audit fail-closed coverage |
| E7 | Closure gate and residual-risk report | Completed | all Task 29 requirements | ADR 010; final server/client gates, completed task update, and residual-risk report |

## Stop Rule

`TASK-000029` is complete when:

- E1 through E7 are completed.
- Each high-risk MVP flow has matrix coverage.
- Each finding is marked covered, fixed, regression-only, or deferred with rationale.
- `make server-security-check` passes.
- `make client-build` passes.
- Residual risks are recorded in the closure ADR.

Do not add new Task 29 slices unless a current slice reveals a concrete high-risk correctness or safety bug that must be fixed before closure.

## Out Of Scope For This Register

These are outside Task 29 unless they are needed to fix a concrete current MVP bug:

- production SSO/OIDC
- refresh-token family and revocation-list infrastructure
- external secret manager integration
- SIEM/export/alerting pipelines
- full browser E2E automation
- full load testing or chaos testing
- distributed workflow orchestration
- multi-account authorization
- multi-cloud provider support
- dashboard policy editing or new dashboard product workflows

## Consequences

- Task 29 has seven slices and a hard closure path.
- Each slice must produce code/tests, an ADR-backed no-op audit result, or a backlog task for deferred work.
- Once E7 passes, Task 29 should move to completed instead of continuing with speculative edge-case hunting.
