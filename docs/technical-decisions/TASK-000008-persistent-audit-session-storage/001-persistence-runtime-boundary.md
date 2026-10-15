---
title: "Persistence runtime boundary for TASK-000008"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000008
tags:
  - persistence
  - postgres
  - runtime
---

# Decision: Persistence Runtime Boundary

## Decision

Use Postgres as the durable persistence backend for server-side CloudIAM sessions, device authorization state, and access audit events.

Database-backed runtime is enabled when `CLOUDIAM_DATABASE_URL` is set. In that mode the server must:

- open Postgres during startup
- verify required persistence schema is present
- register auth/device-login gRPC and HTTP routes
- use the same database connection pool for auth/session storage and audit storage
- record access audit events through the access service

When `CLOUDIAM_DATABASE_URL` is not set, keep the narrow no-database gRPC access path available for local smoke tests. That mode does not register device-login routes and does not provide persistent sessions or audit history.

## Rationale

Postgres is already present in the local Compose stack and migrations already exist for device authorizations, sessions, and audit events. Reusing one database keeps the MVP operationally simple while giving dashboard and read-model work a durable foundation.

The startup schema check gives operators a clear migration error before serving requests. Without it, a missing migration would appear later as an auth, audit, or request failure.

## Acceptance Criteria

- `CLOUDIAM_DATABASE_URL` controls database-backed mode.
- Database-backed startup verifies schema readiness.
- Auth sessions and audit events share the configured Postgres pool.
- No-database mode remains limited to access requests that do not require CloudIAM sessions.
