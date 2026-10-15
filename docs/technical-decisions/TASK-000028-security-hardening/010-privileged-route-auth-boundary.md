---
title: "Privileged route auth boundary for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - auth
  - authorization
  - http
  - grpc
---

# Decision: Privileged Route Auth Boundary

## Context

CloudIAM has three privileged request surfaces in the MVP:

- gRPC access issuance
- browser console administration
- browser device-login approval

The Task 28 control baseline requires server-side authorization and explicit domain boundaries. The risk is not just anonymous access; it is cross-domain reuse, such as a valid non-operator console principal reaching admin mutation code, a client-supplied gRPC principal overriding the session principal, or a device approval request smuggling resource-approval fields.

## Decision

Use these route-boundary rules as the S4 regression contract.

| Surface | Accepted authority | Rejected authority | Mutation boundary |
|---|---|---|---|
| gRPC `RequestAccess` | CloudIAM session token resolved by token hash | missing, unknown, expired, or resolver-failed session; client-supplied principal when session resolver is enabled | access service is not called unless the session is valid; service principal comes from session |
| `/api/admin/*` console routes | verified console cookie whose principal has `console_operator` role | bearer header, missing cookie, invalid cookie, valid non-operator principal | approval and audit stores are not called unless authentication and role checks pass |
| `/api/device/approve` | bounded JSON body containing only `user_code` and `principal` | admin cookie semantics, bearer semantics, unknown fields, oversized body, missing fields, overlong fields | calls only device-login approval service |
| `/device/approve` legacy route | none | all mutation attempts | always returns `410 Gone` and never calls the service |

Console route authorization must be enforced at the HTTP handler boundary even though the concrete `consoleauth` service also rejects non-operator JWT claims. The handler depends on an interface, so it must fail closed if a future or test implementation returns a non-operator principal.

Console refresh must verify the refreshed session token and role before writing a new session cookie.

## Verification

- Console handler tests cover missing session, bearer-only session, invalid token, wrong role, and no store/audit calls on wrong role.
- Console refresh tests assert a wrong-role refreshed principal is rejected before a refreshed cookie is set.
- gRPC access handler tests assert missing or invalid CloudIAM sessions fail before calling access issuance.
- Existing gRPC tests assert client-supplied principal is ignored when session resolution is enabled.
- Device handler tests assert unknown fields, oversized body, overlong fields, deprecated route, rate limiting, and no secret leakage.

## Consequences

- A valid token is not sufficient for console admin access unless it carries the accepted operator role.
- Future admin routes should use the same `authenticate` boundary rather than checking cookies ad hoc.
- New request surfaces must define their accepted authority and rejected authority before implementation.
