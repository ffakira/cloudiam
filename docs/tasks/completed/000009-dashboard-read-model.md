---
title: "MVP console dashboard"
id: TASK-000009
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-08
completed_at: 2026-10-08
owner: codex
phase: "dashboard-1"
priority: P1
sprint_candidate: true
outcome: "Local/demo dashboard served by the Go HTTP server with device approval, console-authenticated resource approval/denial, pending request read model, recent audit events, and a Bun/React client."
tags:
  - dashboard
  - read-model
  - frontend
  - console
  - device-login
---

# Completed Task: MVP Console Dashboard

## Outcome

The MVP dashboard is implemented in the top-level `client/` Bun workspace with React 19, React Aria Components, Zod, React Hook Form, TanStack Query, TanStack Router, Tailwind CSS v4, and `tailwind-merge`.

The existing Go HTTP server can serve the compiled dashboard assets. The UI supports self-service device login approval, console login/logout/refresh through the cookie-backed console session, pending resource approval review, idempotent approve/deny mutations, and recent audit event inspection.

Dashboard data APIs expose pending resource access metadata and audit event metadata without raw CloudIAM session tokens, raw device codes, AWS access keys, AWS secret access keys, or AWS session tokens. Console resource mutations use the secured console APIs from `TASK-000016`, `TASK-000024`, `TASK-000025`, and `TASK-000026`.

## Verification

- `cd client && bun run build`
- `cd server && env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
