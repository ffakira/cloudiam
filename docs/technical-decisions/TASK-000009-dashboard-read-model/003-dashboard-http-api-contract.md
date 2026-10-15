---
title: "Dashboard HTTP API contract for TASK-000009"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000009
tags:
  - dashboard
  - http
  - api
  - device-login
---

# Decision: Dashboard HTTP API Contract

## Decision

Add dashboard APIs under `/api/admin`.

Minimum endpoints:

```http
GET  /device
POST /api/device/approve
GET  /api/admin/access-requests?status=pending
GET  /api/admin/audit/events?limit=50
```

Device approval is user self-service and belongs to the login/authentication flow. Console resource approval and denial endpoints are defined by `TASK-000016` because they require console JWT validation and idempotency-key semantics.

Pending resource access request responses include:

- id
- requesting principal
- target
- requested duration
- workspace
- status
- expires at
- created at

They do not include:

- CloudIAM session token
- session token hash
- AWS credential values

## Relationship To Existing Endpoint

`TASK-000022` deprecated the raw `POST /device/approve` endpoint. User self-service device login uses the browser page at `GET /device`, which submits to `POST /api/device/approve`.

The dashboard console resource approval flow must use the secured ID-based console APIs from `TASK-000016` and must not rely on raw device/user codes.

## Acceptance Criteria

- API responses are JSON.
- Invalid state maps to clear 4xx responses.
- Missing database-backed persistence returns clear service errors rather than partial dashboard data.
- API tests assert secret fields are absent.
