---
title: "Closure gate and residual-risk report for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - closure
  - risk
  - verification
---

# Decision: Closure Gate And Residual-Risk Report

## Context

S9 closes `TASK-000028` after the bounded security hardening slice register reached S1 through S8 and the concurrency/context queue reached C1 through C6.

The goal is to stop the security pass with explicit evidence instead of continuing with speculative hardening. New product-expanding security work moves to backlog tasks.

## Closure Gate

Final verification passed on 2026-10-11:

| Gate | Result |
|---|---|
| Server module verification | Passed through `make server-security-check`. |
| Server `go vet` | Passed through `make server-security-check`. |
| Server unit tests | Passed through `make server-security-check`. |
| Server race tests | Passed through `make server-security-check`. |
| Go vulnerability scan | Passed through `make server-security-check`; `govulncheck` reported 0 reachable vulnerabilities. |
| Go SAST | Passed through `make server-security-check`; `gosec` exited cleanly. |
| Go lint | Passed through `make server-security-check`; `golangci-lint` reported 0 issues. |
| Client browser secret-boundary scan | Passed through `make client-build`. |
| Client TypeScript build | Passed through `make client-build`. |
| Client production bundle build | Passed through `make client-build`; Vite emitted only the existing large chunk warning. |

## Requirement Closure

| Requirement | Closure Evidence |
|---|---|
| `REQ-000028-AUTH-001` | ADR 010, console role enforcement, gRPC session resolution, wrong-role/anonymous/invalid-session tests. |
| `REQ-000028-SECRET-001` | ADRs 008, 009, 013, 015, redaction helpers, no-leakage tests, client secret-boundary scanner. |
| `REQ-000028-REPLAY-001` | ADRs 004, 014, idempotency replay tests, single-use device exchange tests, approved-request claim tests, refresh expiry tests. |
| `REQ-000028-AUDIT-001` | ADR 009, approval lifecycle audit tests, access issued/failed correlation, no-secret audit tests. |
| `REQ-000028-CONCURRENCY-001` | ADRs 005 and 006, C1-C6 concurrency closure, `go test -race ./...`. |
| `REQ-000028-ADR-001` | ADRs 001 through 016 document the security invariants, hardening decisions, and closure evidence. |

## Residual Risk

Accepted residual risk for this MVP:

| Risk | Status | Rationale |
|---|---|---|
| Production SSO/OIDC is not implemented. | Accepted out of scope. | The MVP uses persisted console users and Argon2id hashes; production identity-provider integration belongs to production hardening. |
| Long-lived token revocation families are not implemented. | Accepted out of scope. | Current console JWTs are short-lived, signed, cookie-backed tokens; refresh-token family and revocation-list design belongs with production identity. |
| External secret manager integration is not implemented. | Accepted out of scope. | Production startup now fails closed for weak/insecure secret/config combinations, but secret sourcing remains deployment work. |
| SIEM/export/alerting pipelines are not implemented. | Accepted out of scope. | Durable audit records exist and avoid secrets; external export is product/operations expansion. |
| Multi-account and multi-cloud authorization are not implemented. | Accepted out of scope. | Task 28 hardens the AWS-only MVP before provider/account expansion. |
| Client bundle size warning remains. | Accepted non-security risk. | Vite reports a chunk just above 500 kB after minification; this does not affect the security closure gate. |

## Decision

Mark `TASK-000028` complete.

The next security-adjacent work should use the existing backlog sequence:

- `TASK-000029`: edge-case audit
- `TASK-000027`: production hardening
- deferred product expansion: multi-account and multi-cloud support

Do not reopen Task 28 for new broad hardening ideas. Open a new focused task if a future concrete vulnerability is found.
