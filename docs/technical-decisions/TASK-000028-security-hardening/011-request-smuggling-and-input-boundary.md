---
title: "Request smuggling and input boundary for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - input-validation
  - http
  - grpc
  - smuggling
---

# Decision: Request Smuggling And Input Boundary

## Context

CloudIAM's privileged endpoints intentionally accept very small request shapes. Security hardening should preserve that narrow contract: clients must not be able to append a second JSON object, send ignored mutation bodies that look authoritative, or pass obviously invalid values deeper into the core service.

The risk is ambiguity. Even when ignored fields do not affect current behavior, they create review confusion and can become exploitable if later code starts reading them.

## Decision

Use strict adapter-level input contracts for privileged surfaces:

| Surface | Accepted input | Rejected input |
|---|---|---|
| `/api/admin/login` | exactly one JSON object with `username` and `password` | unknown fields, oversized body, malformed JSON, trailing JSON values |
| `/api/admin/logout` | no request body | any request body |
| `/api/admin/refresh` | no request body; session from cookie only | any request body |
| `/api/admin/access-requests/{id}/approve` and `/deny` | path ID/action plus `Idempotency-Key` header; session from cookie | any request body, malformed path/action, missing or malformed idempotency key |
| `/api/device/approve` | exactly one JSON object with `user_code` and `principal` | unknown fields, oversized body, malformed JSON, trailing JSON values, overlong fields |
| gRPC `RequestAccess` | session token, target, and duration seconds greater than or equal to zero | missing or invalid session, client principal smuggling when session resolver is enabled, negative durations |

HTTP JSON decoding must require a single object. A valid first object followed by another JSON value is invalid.

Admin POST routes that derive authority from cookies, paths, and headers must reject request bodies when the route contract has no body. This prevents ignored JSON fields such as `principal`, `role_arn`, `user_code`, or approval state from becoming ambiguous input.

gRPC adapters must reject structurally invalid values before invoking the core service when the check is adapter-specific and cheap.

## Verification

- Console login tests reject trailing JSON without setting cookies.
- Console logout, refresh, approve, and deny tests reject request bodies before calling protected stores.
- Device approval tests reject unknown fields, trailing JSON, oversized bodies, missing fields, and overlong fields.
- gRPC access tests reject negative durations before calling the access service.
- Existing no-leakage assertions verify rejected input is not reflected in responses.

## Consequences

- Privileged request shapes remain intentionally small and reviewable.
- Future mutation routes must choose whether they accept a body; if they do, they need a typed request struct with strict decoding.
- Clients that accidentally send bodies to no-body admin mutations now receive `400 Bad Request`.
