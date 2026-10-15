---
title: "Production runbook for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - runbook
  - operations
  - incident-response
  - verification
---

# Decision: Production Runbook

## Context

P6 implements the runbook portion of ADR 006.

P5 created the observability checklist. P6 turns that checklist into concrete operator actions for startup, readiness, reload, rollback, shutdown, and incident triage.

## Decision

Add `docs/operations/production-runbook.md` as the Task 27 production runbook.

The runbook covers:

- required production configuration
- startup procedure
- startup failure triage
- health and readiness usage
- config reload procedure
- rollback procedure
- incident triage for access denial, approval issues, credential issuance failure, audit failure, Redis readiness failure, rate limiting, and reload failure
- shutdown expectations
- accepted residual risks
- safe communication boundaries during incidents

## Verification

Documentation verification completed on 2026-10-12:

- runbook exists at `docs/operations/production-runbook.md`
- runbook references concrete commands and endpoints available in the repo
- runbook distinguishes liveness from readiness
- runbook explains fail-closed startup checks
- runbook explains rollback and config reload behavior
- runbook lists residual risks instead of hiding them as TODOs
- examples avoid real secrets and use placeholders where needed

No code tests were required for P6 because this slice changes documentation only.

## Consequences

- Operators have an MVP production guide without requiring a full observability platform.
- Interview walkthroughs can show that production readiness includes operations, not only code.
- P7 can close Task 27 by verifying final gates and recording residual risk.
