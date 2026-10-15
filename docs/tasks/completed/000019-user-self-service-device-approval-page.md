---
title: "User self-service device approval page"
id: TASK-000019
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-10
completed_at: 2026-10-10
owner: codex
phase: "dashboard-1"
priority: P1
sprint_candidate: true
outcome: "The browser device approval page is integrated, route-separated from console approvals, and covered for success, invalid, expired, invalid-state, deprecated-route, and secret-absence cases."
tags:
  - auth
  - device-login
  - frontend
  - user
---

# Completed Task: User Self-Service Device Approval Page

## Outcome

Replace the manual/dev device approval flow with a user-facing page where users approve their own CLI device login.

## Current State

`TASK-000009` added the browser page and dashboard app. `TASK-000022` deprecated raw `POST /device/approve` and moved browser submission to `POST /api/device/approve`.

Task 19 completed the focused verification and edge-hardening pass for the device approval page.

## Result

The self-service page is served through the dashboard app at `GET /device`, and the browser submits approval requests to `POST /api/device/approve`.

The page and API now have edge coverage for:

- successful approval
- missing or malformed approval input
- unknown user code
- expired authorization
- already-used or otherwise non-pending authorization
- deprecated raw `POST /device/approve`
- absence of device codes, token hashes, session tokens, console JWTs, and AWS credential fields in approval responses

The browser schema trims user code and principal before submission, matching server behavior.

## Scope

- Verify the browser page is served from the HTTP server at `/device`.
- Accept a user code and principal for the MVP demo through `/api/device/approve`.
- Show clear approved, invalid, expired, and invalid-state messages.
- Ensure raw `POST /device/approve` stays deprecated and does not approve devices.
- Assert secret values are absent from API and page states.

## Not In Scope

- Production user identity provider integration.
- Console resource approval.
- Dashboard resource approval UI.

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000019-FORM-001 | User can submit a user code and principal from a browser page. | Handler/UI smoke test covers page and submit. |
| REQ-000019-STATE-001 | Page shows approved, invalid, expired, and invalid-state outcomes clearly. | Handler/UI tests cover state mapping. |
| REQ-000019-SECRET-001 | Page never renders token hashes, CloudIAM session tokens, console JWTs, or AWS credentials. | Tests assert forbidden values absent. |
| REQ-000019-BOUNDARY-001 | Device login approval and console resource approval remain separate. | Tests/docs verify device routes do not approve resources and admin routes do not accept user codes. |
| REQ-000019-DEPRECATED-001 | Raw `POST /device/approve` remains deprecated. | HTTP test covers `410 Gone` and no service call. |

## Verification

- `cd server && GOCACHE=/tmp/segura-go-build go test ./...`
- `cd client && bun run build`
