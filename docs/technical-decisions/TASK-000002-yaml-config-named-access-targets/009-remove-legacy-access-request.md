---
title: "Remove legacy access request fields for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000002
tags:
  - access
  - cleanup
  - security
---

# Decision: Remove Legacy Access Request Fields For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- [005-core-access-flow.md](005-core-access-flow.md)
- [006-grpc-target-contract.md](006-grpc-target-contract.md)
- [008-server-config-wiring.md](008-server-config-wiring.md)

## Context

The first AWS proof accepted provider/account/role directly in the core access request. The product flow now resolves those values from YAML after authorization.

Keeping the legacy request fields creates an avoidable bypass-shaped API inside the core.

## Decision

`access.Request` carries only:

```text
Principal
Target
Duration
```

Provider/account/role/role ARN remain only in resolved target data and issuer requests produced by the server after policy evaluation.

The AWS STS adapter no longer accepts a constructor fallback role ARN. It requires the resolved role ARN in each issuer request.

## Acceptance Criteria

- Core access request has no provider/account/role/role ARN fields.
- Core service has no legacy provider/account/role request branch.
- Missing target fails before authorization, resolution, or issuer.
- STS issuer uses only the resolved role ARN from `IssueRequest`.
