---
title: "gRPC target request contract for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000002
tags:
  - grpc
  - access-targets
  - authorization
  - security
---

# Decision: gRPC Target Request Contract For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- [001-contract.md](001-contract.md)
- [003-target-resolution.md](003-target-resolution.md)
- [004-policy-evaluation.md](004-policy-evaluation.md)
- [005-core-access-flow.md](005-core-access-flow.md)

## Context

The first technical slice exposed provider/account/role fields through gRPC so the CLI could prove the AWS STS path.

For the product slice, those fields are internal server configuration. A client request must name an access target, then the server must resolve that target through config and policy.

## Decision

`RequestAccessRequest` carries only:

```text
principal
target
duration_seconds
```

It does not carry:

```text
provider
account
role
role_arn
```

## Security Rationale

Provider/account/role/ARN are not accepted at the transport boundary because they would let a caller attempt to bypass named-target policy resolution.

The gRPC adapter must map transport input into the core access request and let the core service enforce authorization, target resolution, and duration bounds before any credential issuer call.

## Acceptance Criteria

- gRPC request includes `principal`, `target`, and optional `duration_seconds`.
- gRPC request does not include provider/account/role/role ARN.
- gRPC handler passes `principal`, `target`, and duration to the core access service.
- gRPC handler does not reconstruct provider/account/role from client input.
- Invalid target-flow input returns a gRPC validation error before issuing credentials.
