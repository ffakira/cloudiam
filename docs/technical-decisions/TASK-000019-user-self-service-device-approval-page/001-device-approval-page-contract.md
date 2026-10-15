---
title: "Device approval page contract for TASK-000019"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000019
tags:
  - device-login
  - frontend
  - http
  - security
---

# Decision: Device Approval Page Contract

## Context

The CLI device-login flow returns a verification URI and user code. The MVP needs a browser path where the user can approve their own CLI login without using Postman or a raw JSON endpoint.

`TASK-000009` added a React dashboard that includes the device approval form. `TASK-000022` deprecated raw `POST /device/approve` and moved the JSON approval API under `/api/device/approve`.

## Decision

Use the existing dashboard/browser app as the user self-service approval page:

```http
GET /device
```

The page accepts:

- user code
- MVP principal

The page submits to:

```http
POST /api/device/approve
```

The page is part of the device-login/authentication flow, not the console resource approval flow. Console approval remains under `/api/admin/access-requests/{id}/approve` and `/deny`.

## Security Boundary

The page and API must not render or return:

- raw device code
- device code hash
- user code hash
- CloudIAM session token
- session token hash
- console JWT
- AWS access key ID
- AWS secret access key
- AWS session token

The MVP accepts a typed principal for local/demo use. This is not production authentication and must remain documented as local/demo behavior.

## Rationale

Serving the device approval page from the same local HTTP server keeps the demo path simple: the CLI prints a URI, the user opens it, enters the code and principal, and the CLI receives a CloudIAM session.

Keeping device approval separate from console resource approval prevents a user-login action from being mistaken for privileged JIT access approval.

## Consequences

- Demo docs should instruct users to open the verification URI instead of posting JSON manually.
- UI tests or handler tests should cover success, invalid user code, expired authorization, invalid state, and secret absence.
- Future production identity can replace the MVP principal field without changing the route split.
