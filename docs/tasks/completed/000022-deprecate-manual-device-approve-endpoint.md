---
title: "Deprecate manual device approve endpoint"
id: TASK-000022
type: chore
status: completed
created_at: 2026-10-07
updated_at: 2026-10-10
completed_at: 2026-10-10
owner: codex
phase: "auth-cleanup"
priority: P2
sprint_candidate: false
outcome: "The demo device-login approval path now uses the browser page and `/api/device/approve`; raw `POST /device/approve` is deprecated."
tags:
  - auth
  - device-login
  - cleanup
---

# Completed Task: Deprecate Manual Device Approve Endpoint

## Outcome

The self-service browser page at `GET /device` remains the demo path for device-login approval. The page submits through `POST /api/device/approve`.

Raw `POST /device/approve` now returns `410 Gone` with a message that directs users to open `/device` in a browser. It does not call the device approval service.

Console resource approval remains separate under `/api/admin/access-requests/{id}/approve` and `/api/admin/access-requests/{id}/deny`.

## Verification

- `cd server && GOCACHE=/tmp/segura-go-build go test ./...`
- `cd client && bun run build`
