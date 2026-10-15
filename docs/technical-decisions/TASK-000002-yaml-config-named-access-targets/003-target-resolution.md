---
title: "Access target resolution for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-06
updated_at: 2026-10-06
owner: codex
applies_to:
  - TASK-000002
tags:
  - access-targets
  - configuration
  - duration
  - aws
---

# Decision: Access Target Resolution For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- [001-contract.md](001-contract.md)
- [002-config-model-and-validation.md](002-config-model-and-validation.md)

## Context

After loading and validating YAML config, the server needs a deterministic way to resolve a user-facing target name into the provider details needed by later sections.

This decision covers only target resolution:

```text
target name + optional requested duration -> resolved provider/account/role ARN/duration
```

It does not authorize a principal and does not call AWS.

## Decision

Add a core config resolver:

```text
ResolveTarget(targetName string, requestedDuration time.Duration) (ResolvedTarget, error)
```

Rules:

- Unknown target returns an error.
- `requestedDuration == 0` uses target `default_duration`.
- `requestedDuration < 0` returns an error.
- `requestedDuration > max_duration` returns an error.
- Valid target resolves exactly one provider/account/role ARN.

## Resolved Target Shape

```text
Name
Provider
Account
Role
RoleARN
Duration
```

## Acceptance Criteria

- Valid target resolves to AWS role ARN.
- Missing duration uses default duration.
- Duration override within max is accepted.
- Duration above max is rejected.
- Unknown target is rejected.
- Resolver does not evaluate principal/policy.
- Resolver does not call AWS.

