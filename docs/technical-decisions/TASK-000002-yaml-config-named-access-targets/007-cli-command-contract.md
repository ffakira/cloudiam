---
title: "CLI command contract for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000002
tags:
  - cli
  - access-targets
  - hosted-service
  - security
---

# Decision: CLI Command Contract For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- `TASK-000003`: CLI device-code login.
- [006-grpc-target-contract.md](006-grpc-target-contract.md)

## Context

The CLI is a client for the hosted CloudIAM service. Users should not pass service addresses, AWS account names, roles, role ARNs, or provider internals during access requests.

## Decision

Use this command shape:

```bash
cloudiam access request <target> --principal <principal> [--duration <seconds>]
```

Local development uses the same command shape with `go run ./cmd/access-cli`.

The CLI has no user-facing `--addr`, `--server`, or `--host` flag. Service discovery and login/session routing belong to hosted CloudIAM behavior and are tracked separately by `TASK-000003`.

## Acceptance Criteria

- CLI request target is a positional argument.
- CLI request accepts `--principal`.
- CLI request accepts optional `--duration`.
- CLI request does not accept provider/account/role/role ARN.
- CLI request does not expose a user-facing service address flag.
