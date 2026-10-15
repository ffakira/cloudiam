---
title: "Security hardening slice register for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - planning
  - scope
---

# Decision: Security Hardening Slice Register

## Context

`TASK-000028` covers broad security hardening. Without a finite implementation queue, the task can expand indefinitely because every security pass can suggest another useful check.

The accepted audit plan defines stages, and the control baseline defines what good looks like. This decision freezes the remaining Task 28 implementation slices and defines when the task is done.

## Decision

Use this finite slice register for the remainder of `TASK-000028`.

| Slice | Name | Status | Primary requirement | Output |
|---|---|---|---|---|
| S1 | Security invariant extraction and stale decision reconciliation | Completed | `REQ-000028-ADR-001` | ADRs 001, 002, 003 |
| S2 | Abuse resistance, scanner gate, and production config boundary | Completed | `REQ-000028-SECRET-001`, `REQ-000028-REPLAY-001` | ADRs 004, 007, 008; `make server-security-check` |
| S3 | Audit coverage and no-leakage gate | Completed | `REQ-000028-AUDIT-001`, `REQ-000028-SECRET-001` | ADR 009; audit correlation and no-leakage tests |
| S4 | Privileged route and auth boundary | Completed | `REQ-000028-AUTH-001` | ADR 010; console role and gRPC session boundary tests |
| S5 | Request smuggling and input boundary | Completed | `REQ-000028-AUTH-001`, `REQ-000028-SECRET-001` | ADR 011; strict JSON, no-body mutation, and gRPC adapter validation tests |
| S6 | Secret, log, and error-output audit | Completed | `REQ-000028-SECRET-001` | ADR 013; redacted reload/startup error-output tests and CLI/browser output audit |
| S7 | Replay and token lifecycle final regression | Completed | `REQ-000028-REPLAY-001` | ADR 014; refresh expiry, replay audit suppression, cookie lifetime, and existing single-use exchange/claim regressions |
| S8 | Client/browser secret storage boundary | Completed | `REQ-000028-SECRET-001`, `REQ-000028-AUTH-001` | ADR 015; build-time browser secret-boundary scanner and mutation-state scrubbing |
| S9 | Closure gate and residual-risk report | Completed | all Task 28 requirements | ADR 016; final server/client gates, residual-risk report, and task completion |

Concurrency/context hardening is intentionally tracked separately in ADR 006 and is already completed as C1 through C6.

## Stop Rule

`TASK-000028` is complete when:

- S1 through S9 are completed.
- C1 through C6 remain completed.
- `make server-security-check` passes.
- Any remaining findings are either fixed or moved to a specific backlog task.

Do not add new Task 28 slices unless the current slice reveals a concrete high-risk security bug that must be fixed before closure. Product-expanding findings must become backlog tasks, not new Task 28 slices.

## Out Of Scope For This Register

These are explicitly outside the finite Task 28 slice queue:

- Production SSO/OIDC.
- External secret-manager integration.
- SIEM/export/alerting pipelines.
- Multi-account or multi-cloud provider support.
- A new admin RBAC product model.
- Long-lived token revocation infrastructure beyond the current MVP session contracts.

## Consequences

- The Task 28 security slice register is complete.
- Each slice must either produce code/tests, an ADR-backed no-op audit result, or a backlog task for out-of-scope work.
- Once S9 passes, Task 28 should move to completed instead of continuing with speculative hardening.
