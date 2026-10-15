---
title: "Approval decision evaluation for TASK-000018"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000018
tags:
  - policy
  - authorization
  - approval
  - access
---

# Decision: Approval Decision Evaluation

## Context

The access service already accepts `AuthorizationDecision.ApprovalRequired`. Today no production evaluator sets it, so approval-required behavior is dormant outside tests.

`TASK-000018` must connect YAML config to that core access decision.

## Decision

Policy evaluation remains responsible for allow/deny. Target resolution remains responsible for duration and AWS role metadata.

When a request is allowed and the resolved target has `approval_required: true`, the runtime request evaluator returns:

```text
AuthorizationDecision{
  Allowed: true,
  ApprovalRequired: true,
  Reason: "policy <name> matched; approval required"
}
```

When a request is allowed and the target does not require approval, it returns:

```text
AuthorizationDecision{
  Allowed: true,
  ApprovalRequired: false,
  Reason: "policy <name> matched"
}
```

Denied requests must not create pending approvals. Unknown principal, unknown target, and no matching policy continue to deny by default.

## Rationale

Approval is a post-authorization gate. It should not turn a denied request into a pending request, and it should not bypass the existing allow/deny policy model.

Keeping `ApprovalRequired` in the existing access decision lets the access service choose between direct issuance and pending approval without learning config internals.

## Consequences

- The policy package decision shape may need an `ApprovalRequired` field or the snapshot request evaluator can combine policy allow/deny with target approval metadata.
- Tests should cover allowed/direct, allowed/pending, and denied/no-pending paths.
- The reason string is operational metadata only. Task 16 idempotency fingerprints should not depend on exact reason text.
