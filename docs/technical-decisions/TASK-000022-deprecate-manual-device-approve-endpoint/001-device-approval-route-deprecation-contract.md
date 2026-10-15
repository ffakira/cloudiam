---
title: "Device approval route deprecation contract for TASK-000022"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000022
tags:
  - device-login
  - http
  - cleanup
  - dashboard
---

# Decision: Device Approval Route Deprecation Contract

## Context

The MVP originally documented raw JSON approval with `POST /device/approve`. After `TASK-000009`, users have a browser page at `GET /device` for self-service device login approval. Keeping raw JSON approval in the demo path invites confusion between user device approval and console resource approval.

## Decision

Keep the browser route:

```http
GET /device
```

Move the browser form's JSON API to:

```http
POST /api/device/approve
```

Deprecate raw manual approval:

```http
POST /device/approve -> 410 Gone
```

The deprecated route must not call the device approval service and must direct users to open `/device` in a browser.

## Rationale

This preserves the local/demo browser flow while making the old Postman-style approval path visibly obsolete. It also keeps API routing consistent: JSON APIs live under `/api`, while `/device` is a browser page.

## Edge Cases

- `GET /device` must continue to serve the browser app.
- `POST /api/device/approve` must preserve validation and state-specific errors.
- `POST /device/approve` must not approve devices, even with a valid body.
- Console resource approval must remain under `/api/admin/access-requests/{id}/approve` and `/deny`.
- Error responses must not expose raw device codes, token hashes, CloudIAM session tokens, console JWTs, or AWS credentials.

## Consequences

- Local demo docs point users at the browser verification URI.
- Existing scripts or manual Postman steps using `/device/approve` must move to the browser flow or `/api/device/approve` if they intentionally test the JSON API.
- A later production identity provider can replace the MVP principal entry without changing the route boundary.
