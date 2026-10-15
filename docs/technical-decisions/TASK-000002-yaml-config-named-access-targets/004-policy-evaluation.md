---
title: "Policy evaluation for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-06
updated_at: 2026-10-06
owner: codex
applies_to:
  - TASK-000002
tags:
  - policy
  - authorization
  - access-targets
  - security
---

# Decision: Policy Evaluation For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- [001-contract.md](001-contract.md)
- [002-config-model-and-validation.md](002-config-model-and-validation.md)
- [003-target-resolution.md](003-target-resolution.md)

## Context

After resolving an access target, the server must decide whether a principal is allowed to request it.

This decision covers only policy evaluation:

```text
principal + target -> allow/deny
```

It does not update CLI, gRPC, or AWS behavior.

## Decision

Add a core policy evaluator:

```text
internal/core/policy
```

The evaluator reads validated config and returns a decision.

## Decision Shape

```text
Allowed bool
Reason string
```

## Rules

- Unknown principal is denied.
- Unknown target is denied.
- Principal group membership comes from config.
- A policy allows access when any principal group matches any policy group and the requested target is listed in that policy.
- If no policy matches, deny by default.
- Policy evaluation does not call AWS.
- Policy evaluation does not resolve role ARNs.

## Acceptance Criteria

- Known principal with matching group and target is allowed.
- Unknown principal is denied.
- Unknown target is denied.
- Known principal without matching policy is denied.
- Known principal in wrong group is denied.
- No matching policy denies by default.
- Tests assert no AWS/provider dependency is needed.

