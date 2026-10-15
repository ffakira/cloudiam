---
title: "Approval configuration shape for TASK-000018"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000018
tags:
  - configuration
  - approval
  - yaml
  - validation
---

# Decision: Approval Configuration Shape

## Context

The current YAML config defines named access targets and group-based policies. `TASK-000017` introduced the durable pending approval lifecycle, and `TASK-000023` pinned the resolved AWS role ARN on each access request.

`TASK-000018` needs a minimal, demo-friendly way to mark access as approval-required without introducing multi-step approvals, approver groups, or dashboard config editing.

## Decision

Add approval configuration at the access target level:

```yaml
access:
  targets:
    aws-sandbox-readonly:
      provider: aws
      account: sandbox
      role: readonly
      default_duration: 15m
      max_duration: 1h
      approval_required: true
```

Default behavior is backward-compatible: omitted `approval_required` is `false`.

Validation rules:

- `approval_required` is optional.
- Only boolean values are valid.
- Existing target validation for provider/account/role/durations remains unchanged.
- No approver identity, approver group, approval reason template, or SLA field is added in this task.

## Rationale

Target-level approval is the smallest useful governance control for the MVP. It answers the question “does this target require console approval before credentials are issued?” without changing the existing policy matching model.

Policy-level approval is deferred because it introduces precedence questions when multiple policies match the same principal/target pair.

## Consequences

- Some users may need approval for a target even if different groups eventually should not. That is acceptable for the MVP.
- Future work can add policy-level overrides if the product needs group-specific approval behavior.
- Config docs and validation tests should show both omitted/default and explicit `true` cases.
