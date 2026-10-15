---
title: "Security hardening"
id: TASK-000028
type: chore
status: completed
created_at: 2026-10-10
updated_at: 2026-10-11
completed_at: 2026-10-11
owner: codex
phase: "hardening"
priority: P1
sprint_candidate: true
outcome: "MVP security boundaries were hardened and closed with ADR-backed invariants, server/client security gates, browser secret-boundary checks, replay/concurrency coverage, and a residual-risk report."
tags:
  - security
  - auth
  - audit
---

# Task: Security Hardening

## Outcome

Review the completed MVP security boundaries and harden authentication, authorization, session handling, secret handling, audit integrity, and abuse resistance before adding broader access-management features.

## Scope

- Audit completed tasks `TASK-000001` through `TASK-000026` for security gaps.
- Convert prior ADRs into explicit production security invariants before implementation.
- Verify every privileged mutation requires the correct authentication and authorization boundary.
- Confirm access requests cannot smuggle provider, account, role, credential, or approval state.
- Harden session, JWT, cookie, idempotency, and device-login token handling.
- Add or tighten rate limits, request size limits, replay protections, and timing-safe comparisons where needed.
- Audit context propagation, cancellation, race-sensitive state, and concurrent retry behavior for security-relevant paths.
- Validate that audit events are generated for security-relevant state changes and do not expose secrets.
- Run dependency and secret-scanning checks that fit the repo.

## Non-Goals

- Production SSO or external identity-provider integration.
- Multi-account support.
- Multi-cloud provider support.
- New admin RBAC product model.

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000028-AUTH-001 | Protected routes and gRPC methods enforce the intended auth boundary. | Tests cover anonymous, wrong-session, expired-session, and wrong-role attempts. |
| REQ-000028-SECRET-001 | Secrets, tokens, hashes, and AWS credentials are never logged, rendered, persisted outside intended stores, or returned in API responses. | Tests and scanning checks assert forbidden values are absent. |
| REQ-000028-REPLAY-001 | Idempotency, session, and approval tokens resist replay outside their allowed windows. | Tests cover duplicate, stale, mismatched, and cross-principal replay attempts. |
| REQ-000028-AUDIT-001 | Security-relevant decisions leave durable, non-secret audit trails. | Tests or fixtures prove allow, deny, expire, approve, and failure events. |
| REQ-000028-CONCURRENCY-001 | Race-sensitive auth, approval, access issuance, rate limiting, and config snapshot paths preserve security invariants under concurrent requests and context cancellation. | ADRs define invariants; race, cancellation, concurrency tests, or documented audit checks cover high-risk paths. |
| REQ-000028-ADR-001 | Security hardening decisions summarize prior ADR constraints and define production invariants. | `docs/technical-decisions/TASK-000028-security-hardening/` contains accepted decisions. |

## Security Hardening Slices

Finite security slice register:

- `docs/technical-decisions/TASK-000028-security-hardening/012-security-hardening-slice-register.md` is the source of truth for the remaining Task 28 security slices.
- Security slices S1 through S9 are completed.
- No new Task 28 slices should be added unless a current slice reveals a concrete high-risk security bug that must be fixed before closure. Product-expanding findings become backlog tasks.

Concurrency and context hardening follows `docs/technical-decisions/TASK-000028-security-hardening/006-concurrency-hardening-slices.md`.

Current queue:

- C1: Request context gates. Completed.
- C2: Database single-use concurrency. Completed.
- C3: Expiration and state transition races. Completed.
- C4: Config snapshot concurrency. Completed.
- C5: Shutdown and background work audit. Completed.
- C6: Broad race and security scans. Completed.

Concurrency hardening queue: completed.

Scanner tooling gate:

- Release-grade scanner tooling is pinned in `server/tools`: `govulncheck`, `gosec`, and `golangci-lint`.
- `make server-security-check` runs module verification, vet, tests, race tests, vulnerability scanning, SAST, and lint.
- The gate passes as of the Task 28 scanner-tooling slice.

Security control baseline:

- `docs/technical-decisions/TASK-000028-security-hardening/007-security-control-baseline.md` defines the OWASP-aligned control baseline for no-leakage, auth, authorization, replay, rate limiting, audit, crypto, dependency, and verification work.

Production secret/config boundary:

- `docs/technical-decisions/TASK-000028-security-hardening/008-production-secret-and-config-boundary.md` defines production-mode fail-closed checks for secure cookies, Redis rate limiting, HTTPS verification URIs, database TLS, JWT secret strength, and console seed bootstrap overrides.

Audit coverage and no-leakage gate:

- `docs/technical-decisions/TASK-000028-security-hardening/009-audit-coverage-and-no-leakage.md` defines the concrete CloudIAM audit matrix for denied access, approval lifecycle events, issued credentials, failed issuance, and no-secret audit storage.
- Approved-request credential issuance now carries the access request ID into `access_issued` and `access_failed` audit events so approval and issuance evidence remain directly correlated.

Privileged route/auth boundary:

- `docs/technical-decisions/TASK-000028-security-hardening/010-privileged-route-auth-boundary.md` defines the accepted and rejected authority for gRPC access issuance, console admin routes, device approval, and the deprecated device approval route.
- Console admin routes now enforce the `console_operator` role at the HTTP boundary, and console refresh verifies the refreshed token before setting a new cookie.

Request smuggling and input boundary:

- `docs/technical-decisions/TASK-000028-security-hardening/011-request-smuggling-and-input-boundary.md` defines strict input contracts for privileged HTTP and gRPC surfaces.
- HTTP JSON decoding now rejects trailing JSON values, no-body admin mutations reject request bodies, and gRPC access rejects negative durations before invoking the access service.

Secret log and error-output audit:

- `docs/technical-decisions/TASK-000028-security-hardening/013-secret-log-and-error-output-audit.md` records the S6 output audit for API errors, CLI output, browser error handling, scanner output, reload status, and startup errors.
- Reload status and startup idempotency cleanup errors now redact secret-shaped lower-level error content before writing operator-facing text.

Replay and token lifecycle final regression:

- `docs/technical-decisions/TASK-000028-security-hardening/014-replay-and-token-lifecycle-final-regression.md` records the S7 replay and token lifecycle closure pass.
- Console refresh now derives cookie lifetime from the handler clock, expired console JWTs cannot be refreshed, replayed approval mutations skip duplicate audit events, and existing integration tests cover single-use device exchange and approved-request claim behavior.

Client/browser secret storage boundary:

- `docs/technical-decisions/TASK-000028-security-hardening/015-client-browser-secret-storage-boundary.md` records the S8 browser storage and client-state audit.
- Client build now runs a browser secret-boundary scanner before TypeScript/Vite, and sensitive login/device mutation state is reset after each request so browser-accessible state does not retain passwords or device user codes outside transient form fields.

Closure gate and residual-risk report:

- `docs/technical-decisions/TASK-000028-security-hardening/016-closure-gate-and-residual-risk-report.md` records the S9 closure gate and residual-risk report.
- Final `make server-security-check` and `make client-build` passed on 2026-10-11.
- Accepted residual risks are explicitly deferred to `TASK-000029`, `TASK-000027`, or later product-expansion tasks instead of extending Task 28.
