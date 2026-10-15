---
title: "Cookie-backed console dashboard session"
id: TASK-000024
type: hardening
status: completed
created_at: 2026-10-08
updated_at: 2026-10-08
completed_at: 2026-10-08
owner: codex
phase: "dashboard-security-2"
priority: P0
sprint_candidate: true
tags:
  - dashboard
  - console
  - auth
  - cookies
  - frontend
  - security
---

# Task: Cookie-Backed Console Dashboard Session

## Outcome

Move the dashboard console session away from browser-managed bearer tokens and local storage. The server owns the console JWT in an HttpOnly cookie, while the client stores only safe session metadata in memory.

## Scope

- Add cookie-backed console login for the dashboard.
- Add `GET /api/admin/me` for session validation.
- Add `POST /api/admin/refresh` for session refresh/rotation.
- Add a logout endpoint if needed to clear the HttpOnly cookie.
- Update existing console APIs to authenticate from the console session cookie.
- Install Zustand and use it only for safe client-side session state.
- Remove local-storage persistence of console access tokens.
- Ensure dashboard requests use cookie credentials instead of an `Authorization` bearer token.

## Not In Scope

- Production SSO/OIDC.
- Multi-user console management.
- Refresh-token database persistence.
- Multi-tenant dashboard sessions.

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000024-COOKIE-001 | Console login sets an HttpOnly cookie and does not return the raw JWT in JSON. | HTTP handler tests cover cookie and response body. |
| REQ-000024-ME-001 | `/api/admin/me` validates the cookie and returns safe session metadata. | HTTP handler tests cover valid and missing cookie. |
| REQ-000024-REFRESH-001 | `/api/admin/refresh` validates the cookie, rotates the JWT cookie, and returns safe session metadata. | HTTP handler tests cover valid refresh. |
| REQ-000024-FRONTEND-001 | Client uses Zustand for safe session metadata and no local storage token persistence. | Build passes and source scan confirms no `localStorage` session token usage. |
| REQ-000024-API-001 | Dashboard API calls send cookies and do not attach bearer tokens. | Client API code uses `credentials: "include"`. |

## Result

Console dashboard authentication now uses a server-owned HttpOnly cookie instead of browser-managed bearer tokens.

`POST /api/admin/login` sets the cookie and returns only safe session metadata. `GET /api/admin/me` validates the cookie, `POST /api/admin/refresh` rotates it, and `POST /api/admin/logout` clears it. Dashboard console APIs authenticate from the cookie.

The frontend uses Zustand for in-memory session metadata only. It no longer stores the console JWT in local storage, no longer reads an `access_token`, and no longer sends `Authorization` headers from dashboard code.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `bun run build`
- `rg "localStorage|access_token|Authorization|Bearer|\\.access_token" client/src client/package.json`
