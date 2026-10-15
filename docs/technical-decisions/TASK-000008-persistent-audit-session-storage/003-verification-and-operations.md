---
title: "Verification and operations for TASK-000008"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000008
tags:
  - persistence
  - migrations
  - testing
  - operations
---

# Decision: Verification And Operations

## Decision

Use checked-in SQL migrations and the existing Compose `migrate` service for local persistence setup.

Required local commands:

```bash
make db-up
make migrate-up
make migrate-status
make server-test-integration
```

Regular unit tests must remain runnable without Postgres:

```bash
env GOCACHE=/tmp/cloudiam-go-cache go test ./...
```

Postgres-backed tests use the `integration` build tag and may assume migrations are applied by `make server-test-integration`.

## Runtime Failure Contract

When database-backed mode is enabled but required tables or columns are missing, server startup must fail with a clear schema/migration error. It must not start and then discover missing persistence tables on the first login or access request.

## Acceptance Criteria

- Unit tests do not require a live database.
- Integration tests cover auth/session persistence and audit persistence.
- Startup schema readiness is tested.
- Operator docs include migration and integration-test commands.
