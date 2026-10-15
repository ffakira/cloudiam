---
title: "Edge-case audit method and deferral rule for TASK-000029"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - audit
  - testing
  - deferral
---

# Decision: Edge-Case Audit Method And Deferral Rule

## Context

Task 29 should produce confidence, not churn. The audit needs a repeatable method for deciding whether an edge case is already covered, should be fixed now, or should become future work.

The method must avoid two traps:

- under-hardening, where a real MVP correctness or safety bug is hand-waved away
- over-engineering, where the task grows into production identity, workflow orchestration, load testing, or product expansion

## Decision

Use a boundary matrix for each audited MVP flow.

Each row must include:

| Field | Meaning |
|---|---|
| Flow | The user/system flow being audited. |
| Boundary | CLI, gRPC, HTTP, dashboard, persistence, config, startup/shutdown, AWS adapter, or audit. |
| Edge case | The malformed, duplicate, stale, concurrent, missing, timeout, dependency-failed, or clock-boundary condition. |
| Expected behavior | Reject, no-op, retry-safe response, preserve old state, non-secret operator error, or durable audit event. |
| Current evidence | Existing test, implementation behavior, or documented decision. |
| Action | Covered, fix in slice, add regression only, or defer to backlog. |

## Action Rules

### Covered

Use `Covered` when existing code and tests already prove the expected behavior.

The audit note must name the evidence, such as a test, ADR, or implementation boundary.

### Add Regression Only

Use `Add regression only` when behavior appears correct but there is no focused test for a high-risk edge case.

Regression tests should assert externally visible behavior, not implementation details.

### Fix In Slice

Use `Fix in slice` when the edge case creates one of:

- access granted from ambiguous state
- duplicate approval or duplicate credential issuance
- secret leakage
- corrupted durable state
- stale config replacing known-good config
- confusing success response for a failed security-relevant operation
- panic/crash on ordinary malformed input

Fixes must stay narrow and use existing architecture.

### Defer To Backlog

Use `Defer to backlog` when the finding requires one of:

- new product capability
- new production identity model
- new external platform
- broad load/chaos testing
- multi-account or multi-cloud design
- long-lived token revocation model
- major dashboard workflow expansion

Deferrals must include a trigger that explains when the work becomes necessary.

## Severity Lens

Prioritize by production safety impact:

| Severity | Meaning | Task 29 Handling |
|---|---|---|
| High | Could grant access incorrectly, leak secrets, double-issue credentials, or lose security evidence. | Fix now or block closure. |
| Medium | Could confuse operators, break retry safety, or produce stale/incorrect user-visible state without granting access. | Fix now when narrow; otherwise backlog with rationale. |
| Low | Cosmetic, rare local-demo-only friction, or production-platform concern outside the MVP. | Defer unless it is nearly free and clearly bounded. |

## Test Policy

Task 29 tests should be:

- table-driven where useful
- close to the boundary under test
- deterministic about time and state
- explicit about no-secret output when relevant
- small enough that failures identify the broken contract

Integration tests are appropriate only when the edge case depends on database transaction semantics or persisted state. Otherwise prefer unit or adapter tests.

## Closure Rule

Task 29 can close when:

- the finite slice register is complete
- each audited flow has a matrix row for high-risk edge cases
- every high/medium finding is fixed, covered, or explicitly deferred
- final server and client gates pass
- a closure ADR records residual risk and next backlog items

Do not continue adding rows once the current MVP boundary is covered. New ideas after closure become new tasks.
