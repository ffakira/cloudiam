---
title: "Edge-case audit"
id: TASK-000029
type: chore
status: completed
created_at: 2026-10-10
updated_at: 2026-10-11
completed_at: 2026-10-11
owner: codex
phase: "hardening"
priority: P1
sprint_candidate: true
tags:
  - testing
  - edge-cases
  - reliability
---

# Task: Edge-Case Audit

## Outcome

Systematically review the completed MVP flows for edge cases that can break correctness, confuse operators, or weaken the security posture, then add focused tests and fixes.

Completed with finite edge-case coverage across CLI, gRPC, HTTP, dashboard state, persistence transitions, config reload/startup/shutdown, AWS STS issuance, and final closure gates. Residual risks are recorded in `docs/technical-decisions/TASK-000029-edge-case-audit/010-edge-case-audit-closure.md`.

## Scope

- Audit completed tasks `TASK-000001` through `TASK-000026` for missing edge-case coverage.
- Cover malformed input, missing config, duplicate requests, stale state, race-prone state transitions, clock skew, and dependency failures.
- Verify CLI, gRPC, HTTP, dashboard, persistence, config reload, approval lifecycle, and AWS STS adapter behavior at boundaries.
- Convert discovered edge cases into small fixes or follow-up tasks when they are too large for one hardening pass.
- Keep tests behavior-focused and avoid large unrelated refactors.

## Non-Goals

- Multi-account support.
- Multi-cloud provider support.
- New dashboard features.
- Load testing beyond small concurrency/race checks.

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000029-MATRIX-001 | Each completed MVP flow has an edge-case matrix. | Docs map flows to edge cases and expected behavior. |
| REQ-000029-TESTS-001 | High-risk edge cases have automated coverage. | Unit/integration tests cover selected cases. |
| REQ-000029-RACE-001 | Approval and session state transitions are checked for duplicate and stale operations. | Tests cover duplicate approve/deny/login/session poll scenarios. |
| REQ-000029-FOLLOWUP-001 | Large discovered gaps become explicit backlog tasks instead of hidden TODOs. | Backlog contains follow-up tasks with scope and done conditions. |

## Acceptance Invariants

Task 29 is bounded by:

- `docs/technical-decisions/TASK-000029-edge-case-audit/001-edge-case-acceptance-invariants.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/002-edge-case-audit-method-and-deferral-rule.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/003-edge-case-audit-slice-register.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/004-edge-case-matrix-and-stop-rule.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/006-http-and-dashboard-edge-case-regressions.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/007-persistence-state-transition-regressions.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/008-config-startup-shutdown-regressions.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/009-aws-credential-issuance-regressions.md`
- `docs/technical-decisions/TASK-000029-edge-case-audit/010-edge-case-audit-closure.md`

The task is not "find every edge case." It is "prove the MVP fails closed and predictably at the boundaries we already own."

## Edge-Case Audit Slices

Finite slice register:

- E1: Edge-case matrix and stop rule. Completed.
- E2: CLI and gRPC malformed/stale session behavior. Completed.
- E3: HTTP and dashboard form/API edge cases. Completed.
- E4: Persistence and state-transition oddities. Completed.
- E5: Config reload, startup, and shutdown edge cases. Completed.
- E6: AWS adapter and credential issuance edge cases. Completed.
- E7: Closure gate and residual-risk report. Completed.

No new Task 29 slices should be added unless a current slice reveals a concrete high-risk correctness or safety bug that must be fixed before closure. Product-expanding or production-platform findings become backlog tasks.

CLI and gRPC malformed/stale session behavior:

- `docs/technical-decisions/TASK-000029-edge-case-audit/005-cli-and-grpc-edge-case-regressions.md` records the E2 audit and regressions.
- CLI malformed positional input now fails before session loading or gRPC access, and CLI session-store tests cover missing, malformed, incomplete, expired, and credential-contaminated session files.
- Existing gRPC adapter tests remain the accepted coverage for malformed/stale CloudIAM sessions, negative duration, auth validation, and no-secret status messages.

HTTP and dashboard form/API edge cases:

- `docs/technical-decisions/TASK-000029-edge-case-audit/006-http-and-dashboard-edge-case-regressions.md` records the E3 audit and regressions.
- Console HTTP tests now prove malformed login bodies fail before auth, no-body mutations reject positive-length and chunked request bodies before auth/store access, malformed mutation paths do not mutate or audit, and invalid list filters fail before store reads.
- Existing device handler tests and Task 28 dashboard state-scrubbing/client scanner remain the accepted coverage for device approval and browser state edge cases.

Persistence and state-transition oddities:

- `docs/technical-decisions/TASK-000029-edge-case-audit/007-persistence-state-transition-regressions.md` records the E4 audit and regressions.
- Existing Postgres integration tests cover approval idempotency, concurrent approve/deny, approve/expire, claim-at-expiry, single-use access issuance, single-use device-code exchange, and concurrent device authorization transitions.
- New Postgres integration regressions prove expired approved device-code exchange creates no session/consume marker, and repeated session revocation fails closed.

Config reload, startup, and shutdown edge cases:

- `docs/technical-decisions/TASK-000029-edge-case-audit/008-config-startup-shutdown-regressions.md` records the E5 audit and regressions.
- Config reload tests now prove AWS region changes are restart-required and canceled reload contexts preserve the active snapshot.
- Existing server app tests remain the accepted coverage for invalid config, secret-field rejection, SIGHUP reload, concurrent snapshot safety, missing startup dependencies, Redis fallback/production requirement, startup cleanup error redaction, and graceful shutdown fallback.

AWS adapter and credential issuance edge cases:

- `docs/technical-decisions/TASK-000029-edge-case-audit/009-aws-credential-issuance-regressions.md` records the E6 audit and regressions.
- STS adapter tests now prove non-positive final durations fail before STS, provider errors return no grant, missing/incomplete STS credential payloads return no grant, and CloudIAM STS tags are bounded/sanitized.
- Access service tests now prove credentials are not returned when durable issued-access audit recording fails.
- Existing gRPC and audit tests remain the accepted coverage for generic external error responses and no credential material in audit payloads.

Closure gate and residual-risk report:

- `docs/technical-decisions/TASK-000029-edge-case-audit/010-edge-case-audit-closure.md` records the E7 closure gate.
- `make server-security-check` passed, including module verification, vet, unit tests, race tests, govulncheck, gosec, and golangci-lint.
- `make client-build` passed, including the browser-boundary scanner, TypeScript build, and Vite production build.
- Residual risks are accepted for the MVP and explicitly scoped out or forwarded to production hardening where appropriate.
