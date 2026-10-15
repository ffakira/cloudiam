---
title: "Core access flow for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-06
updated_at: 2026-10-06
owner: codex
applies_to:
  - TASK-000002
tags:
  - access
  - policy
  - access-targets
  - ports
---

# Decision: Core Access Flow For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- [003-target-resolution.md](003-target-resolution.md)
- [004-policy-evaluation.md](004-policy-evaluation.md)

## Context

The core access service must combine target resolution and policy evaluation before credential issuing.

This decision covers only core service behavior. It does not update CLI, gRPC, or AWS adapter wiring.

## Decision

Use an issuer request that carries both product metadata and provider metadata:

```text
Principal
Target
Provider
Account
Role
RoleARN
Duration
```

Reason:

- The AWS adapter needs `RoleARN` and `Duration`.
- Future governance/audit needs `Target`, `Provider`, `Account`, and `Role`.
- Core stays cloud-neutral while still passing resolved provider metadata through the port.

## Flow

```text
RequestAccess(principal, target, optional duration)
-> validate required fields
-> evaluate policy principal + target
-> resolve target to provider/account/role ARN/duration
-> call CredentialIssuer once
```

## Rules

- Missing principal fails before policy/issuer.
- Missing target fails before policy/issuer.
- Unauthorized principal fails before target resolution/issuer.
- Unknown target fails before issuer.
- Duration above target max fails before issuer.
- Valid request calls issuer exactly once.

## Acceptance Criteria

- Valid principal + target calls issuer once.
- Issuer receives principal, target, provider, account, role, role ARN, and duration.
- Unauthorized principal does not call issuer.
- Unknown target does not call issuer.
- Duration above max does not call issuer.

