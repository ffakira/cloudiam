---
title: "Observability checklist for TASK-000027"
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
  - audit
  - operations
  - verification
---

# Decision: Observability Checklist

## Context

P5 implements the checklist portion of ADR 006.

The MVP already has durable security audit events, readiness/liveness probes, redacted reload status, and fail-closed startup behavior. It does not yet have a full metrics, tracing, SIEM, alerting, or dashboard platform.

P5 must make the current signal surface explicit without pretending that production observability is complete.

## Decision

Add `docs/operations/observability-checklist.md` as the source of truth for Task 27's current observability map.

The checklist maps critical flows to available signals:

- device login lifecycle
- CloudIAM session exchange
- access denied
- approval required
- approval approved/denied/expired
- credential issued
- credential issuance failed
- AWS STS dependency failure
- audit write failure
- config reload success/failure
- Postgres and Redis dependency failure
- rate limiting behavior
- startup cleanup
- liveness/readiness
- shutdown behavior

The checklist also records accepted P5 gaps and future metrics/alerts so they do not become hidden TODOs.

## Signal Boundary

P5 accepts these current signal types:

- durable audit events
- readiness and liveness endpoints
- reload status output
- startup errors
- generic API responses
- focused lifecycle tests

P5 does not implement:

- Prometheus metrics
- OpenTelemetry tracing
- SIEM/export
- alert routing
- dashboard visualizations
- log aggregation

## Verification

Documentation verification completed on 2026-10-12:

- checklist exists at `docs/operations/observability-checklist.md`
- each critical flow from ADR 006 has a current signal or explicit accepted gap
- future metrics avoid high-cardinality labels and secret material
- accepted gaps are listed explicitly

No code tests were required for P5 because this slice changes documentation only.

## Consequences

- Operators and interview reviewers can see exactly what can be diagnosed today.
- P6 can build the production runbook from this checklist.
- Future observability work can implement metrics/alerts from a concrete map rather than inventing scope during Task 27.
