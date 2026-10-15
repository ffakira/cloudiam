---
title: "Observability and runbook contract for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - observability
  - runbook
  - operations
---

# Decision: Observability And Runbook Contract

## Context

Production readiness is incomplete if operators cannot answer what happened, why access was allowed or denied, and what to do during common incidents.

The MVP already has durable audit events and several redaction/no-leakage controls. Task 27 should make operational visibility explicit without building a full SIEM or metrics platform.

## Decision

Task 27 must produce an observability checklist and production runbook for the current AWS-only MVP.

## Observability Checklist

The checklist must map these flows to available signals:

| Flow | Required Signal |
|---|---|
| CLI device login started | audit/log or existing device authorization state |
| Device approved/denied/expired | audit/log and durable state |
| CloudIAM session exchange | audit/log or durable state without raw tokens |
| Access denied by policy | durable audit event |
| Approval required | durable `approval_requested` audit event and pending access request |
| Approval approved/denied/expired | durable approval audit event |
| Credential issued | durable `access_issued` audit event correlated to access request when applicable |
| Credential issuance failed | durable `access_failed` audit event or operator-visible failure |
| AWS STS dependency failure | generic API error plus operator-visible signal |
| Config reload success/failure | operator-visible reload status without secrets |
| Rate limiter Redis failure/fallback | operator-visible signal without Redis URL leakage |
| Startup cleanup failure | redacted startup/operator signal |
| Readiness failure | non-secret subsystem status |

Metrics may be documented as expected future counters/gauges if not implemented in this task, but the current signal must be explicit.

## Runbook

The production runbook must cover:

- required configuration and safe production defaults
- startup validation failures and fixes
- health and readiness endpoints
- config reload procedure
- rollback procedure
- diagnosing Postgres failure
- diagnosing Redis/rate-limit failure
- diagnosing AWS STS issuance failure
- diagnosing audit write failure
- interpreting pending/approved/issued/expired access request state
- incident triage without exposing secrets
- accepted residual risks and follow-up tasks

## Non-Goals

Task 27 does not need to implement:

- SIEM export
- alert routing
- dashboards for metrics visualization
- tracing backend
- log aggregation platform
- distributed profiling

It may document where those fit in a future production roadmap.

## Verification Expectations

P5 and P6 should close when:

- every critical flow has a documented current signal
- the runbook gives an operator a concrete action for common failures
- no runbook command or example includes real secrets
- residual risks are listed instead of hidden as TODOs

## Consequences

- The MVP becomes easier to explain as production-aware even before full platform observability exists.
- Future observability implementation can start from the checklist rather than guesswork.
- The interview story can distinguish "what exists now" from "what production platform work comes next."
