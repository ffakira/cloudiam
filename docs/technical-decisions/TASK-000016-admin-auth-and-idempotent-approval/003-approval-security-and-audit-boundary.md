---
title: "Approval security and audit boundary for TASK-000016"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000016
tags:
  - approval
  - audit
  - security
  - resource-access
---

# Decision: Resource Approval Security And Audit Boundary

## Decision

Dashboard console approval APIs approve or deny JIT resource access requests by access request ID. They do not approve devices.

Device approval/login is a user self-service authentication flow. A user can approve their own device to establish a CloudIAM session. Console approval is reserved for resource/JIT access requests that need human authorization before credentials are issued.

Minimum endpoints for this task:

```http
POST /api/admin/login
GET  /api/admin/access-requests?status=pending
POST /api/admin/access-requests/{id}/approve
POST /api/admin/access-requests/{id}/deny
```

Approve/deny bodies may be empty for MVP because the pending access request already contains the principal, target, requested duration, and policy context.

Approval and denial must verify:

- console JWT is valid
- idempotency key is valid
- target access request exists
- target is still pending
- target is not expired
- target is a resource access request, not a device authorization

Approval and denial must not re-resolve the target from the latest hot-reload config and must not require the current `approval_required` setting to still be true. The access request row is the immutable approval object. It already contains the requester, target, requested duration, policy reason/context, provider/account/role metadata, and pinned `role_arn` from request creation time.

Approval responses must not include:

- raw device code
- device code hash
- user code hash
- session token hash
- console JWT
- AWS credential values

Approval responses should include only safe state metadata such as access request ID, status, requester principal, target, requested duration, expiration, and approved/denied timestamp.

## Audit Boundary

Approval/denial records must include:

- console operator subject
- action
- resource access request ID
- requesting principal
- target
- pinned role ARN or safe reference to the persisted request metadata
- timestamp
- idempotency key or a safe hash/reference to it

They must not include raw JWTs, passwords, raw device codes, raw user codes, CloudIAM session tokens, or AWS credentials.

## Relationship To Existing `/device/approve`

`TASK-000022` deprecated raw `POST /device/approve` for the demo path. Device login approval is user self-service through the browser page at `GET /device`, backed by `POST /api/device/approve`.

The dashboard console API must not use device approval routes for resource approvals.

## Acceptance Criteria

- Dashboard console approval path is for resource/JIT requests, not device login.
- User device approval remains separate from console resource approval.
- Console operator identity is attached to approval/denial metadata.
- Approve/deny by immutable access request ID and do not rebuild approval semantics from current YAML config.
- Invalid state maps to clear 4xx responses.
- Secret values are absent from API responses and persistence records.
