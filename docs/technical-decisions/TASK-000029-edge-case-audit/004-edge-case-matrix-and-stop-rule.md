---
title: "Edge-case matrix and stop rule for TASK-000029"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - matrix
  - scope
  - testing
---

# Decision: Edge-Case Matrix And Stop Rule

## Context

E1 establishes the initial Task 29 edge-case matrix. The matrix is intentionally boundary-first and finite. It is not a promise to test every possible malformed input; it names the MVP flows where an edge can affect correctness, safety, or operator predictability.

Each later Task 29 slice must update or reference this matrix when it covers, fixes, or defers a row.

## Matrix

| ID | Flow | Boundary | Edge Case | Expected Behavior | Current Evidence | Action |
|---|---|---|---|---|---|---|
| E1-CLI-001 | Access request command | CLI parser | Misplaced flags or extra positional args are interpreted as target input. | Reject with a stable parse error before loading a session or contacting gRPC. | ADR 005 added parser regressions and a narrow parser fix. | Covered. |
| E1-CLI-002 | Access request command | CLI session file | Missing, malformed, expired, or credential-contaminated session file. | Reject before gRPC; never print stored token or AWS credential material. | ADR 005 added regressions for missing, malformed, incomplete, expired, and credential-contaminated session files. | Covered. |
| E1-CLI-003 | Access request command | CLI output | Pending approval accidentally printed in env format. | Reject env output for pending requests; do not print AWS credential exports. | Existing CLI adapter test covers pending env rejection. | Covered. |
| E1-GRPC-001 | Access issuance | gRPC access adapter | Missing, invalid, or stale CloudIAM session token. | Return `Unauthenticated` before access service invocation; no secret details. | Existing access handler tests cover missing/invalid sessions and resolver error no-leakage. | Covered. |
| E1-GRPC-002 | Access issuance | gRPC access adapter | Malformed duration or service validation error. | Return `InvalidArgument`; do not invoke service for negative duration. | Existing access handler tests cover negative duration and validation mapping. | Covered. |
| E1-GRPC-003 | Device login | gRPC auth adapter | Missing device/user code or principal. | Return `InvalidArgument` before service mutation. | Existing auth handler validation tests cover missing values. | Covered. |
| E1-HTTP-001 | Console/device APIs | HTTP JSON/form boundary | Unknown fields, trailing JSON, oversized bodies, no-body mutations. | Reject before mutation; no secret values in responses. | ADR 006 adds focused console HTTP regressions; existing device handler tests cover invalid JSON, unknown fields, trailing JSON, oversized bodies, overlong values, invalid state, and deprecated route behavior. | Covered. |
| E1-HTTP-002 | Dashboard/browser state | Browser state boundary | Retry, failed mutation, or stale form state retains secrets. | Keep secrets only in transient form fields; clear mutation state. | ADR 006 accepts Task 28 S8 dashboard mutation-state scrubbing and build-time scanner as current MVP coverage; no new dashboard bug found in E3. | Covered. |
| E1-DB-001 | Approval mutations | Persistence state transition | Duplicate, mismatched, expired, or concurrent approve/deny. | Single terminal transition; conflict or replay-safe response. | ADR 007 accepts existing access-request integration coverage for concurrent claim, idempotent replay, mismatched fingerprint, approve/deny, approve/expire, and claim-at-expiry behavior. | Covered. |
| E1-DB-002 | Device exchange/session | Persistence state transition | Duplicate exchange, stale token, expired authorization, revoked session. | Single-use exchange; expired/revoked sessions rejected. | ADR 007 adds regressions for expired approved exchange and repeated revoke; existing integration tests cover single-use exchange, concurrent transitions, and revoked-session lookup. | Covered. |
| E1-CONFIG-001 | Config load/reload | Config parser/reloader | Malformed reload, missing required fields, stale partial config. | Preserve known-good config or fail startup; non-secret operator error. | ADR 008 adds canceled reload and AWS-region restart-required regressions; existing tests cover invalid YAML, invalid semantic config, SIGHUP reload, in-flight snapshot stability, and concurrent reader/reload snapshots. | Covered. |
| E1-RUNTIME-001 | Startup/shutdown | Runtime lifecycle | Missing database/Redis/console config, cleanup failure, canceled context. | Fail closed or continue only when ADR permits fallback; non-secret error. | ADR 008 accepts existing startup dependency, Redis fallback/production requirement, production hardening, startup cleanup redaction, gRPC shutdown, and server error reporting tests as current MVP coverage. | Covered. |
| E1-AWS-001 | STS credential issuance | AWS adapter | STS failure, invalid duration, missing role ARN, provider error. | No credentials returned; failure is audited without secret leakage. | ADR 009 adds STS positive-duration, provider-error, missing/incomplete-credential, and tag-boundary regressions; existing gRPC tests hide service error details. | Covered. |
| E1-AUDIT-001 | Security evidence | Audit recorder | Audit write failure during security-relevant decision. | Preserve fail-closed behavior for operations that require durable evidence. | ADR 009 adds issued-access audit fail-closed coverage; existing access and approval tests cover generic failure reasons and no credential material in audit payloads. SIEM/export remains deferred. | Covered for MVP; SIEM/export deferred. |

## Stop Rule

Task 29 should stop when:

- E1 through E7 are completed.
- Each matrix row is `Covered`, fixed, or explicitly deferred.
- No high-severity row remains unaddressed.
- Final `make server-security-check` and `make client-build` pass.
- Closure ADR records any residual risks.

Do not add rows for hypothetical production systems not represented by the current MVP. Those become backlog tasks.

## E1 Output

E1 is complete with this matrix and the accepted invariants from ADRs 001 through 003.
