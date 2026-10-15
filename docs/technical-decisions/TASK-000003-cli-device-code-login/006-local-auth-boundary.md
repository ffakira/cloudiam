---
title: "Local auth boundary for TASK-000003"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000003
tags:
  - auth
  - local-dev
  - governance
  - security
---

# Decision: Local Auth Boundary For TASK-000003

## Related Work

- `TASK-000003`: CLI device-code login.
- [001-contract.md](001-contract.md)
- [003-database-persistence.md](003-database-persistence.md)
- [005-device-authorization-schema.md](005-device-authorization-schema.md)

## Context

The product will eventually need real hosted authentication, organization/workspace membership, and governance controls.

That is not part of this task. `TASK-000003` proves the device-code login shape and CloudIAM session behavior so the CLI can act like a hosted-service client.

Without a clear boundary, a local browser approval could be mistaken for production authentication.

## Decision

Local browser approval is an MVP/dev simulation.

For local development, the approval step may assign a principal by:

- using a default local principal such as `akira`, or
- accepting a submitted local principal in the approval form.

The resulting principal is not trusted as production identity.

The access request must still be authorized by the server-side YAML policy from `TASK-000002` before AWS is called.

## Non-Goals

This task does not implement:

- passwords
- production identity provider integration
- OAuth/OIDC provider login
- MFA
- organization membership management
- user lifecycle management
- dashboard governance UI
- production authorization administration

## Future Replacement

Future hosted auth should replace the source of the approved principal, not the overall device/session flow.

The DB schema remains compatible with future auth because device authorizations and sessions already carry:

```text
workspace
principal
endpoint
expires_at
```

## Security Rules

- Local approval is clearly labeled as local/dev behavior.
- A browser-approved principal is still subject to YAML policy authorization.
- The CLI must not pass provider/account/role/role ARN.
- The CLI must not store AWS credentials.
- Server-side sessions expire after one hour.

## Acceptance Criteria

- Local auth simulation is documented as non-production.
- Implementation keeps principal approval separate from AWS authorization.
- Access request still checks YAML policy before AWS.
- Future real auth can replace local approval without changing the CLI command shape.
