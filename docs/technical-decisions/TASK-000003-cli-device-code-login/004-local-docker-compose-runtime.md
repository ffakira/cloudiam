---
title: "Local Docker Compose runtime for TASK-000003"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000003
tags:
  - docker
  - postgres
  - local-dev
---

# Decision: Local Docker Compose Runtime For TASK-000003

## Related Work

- `TASK-000003`: CLI device-code login.
- [003-database-persistence.md](003-database-persistence.md)

## Context

Local development needs a repeatable Postgres database without conflicting with ports already in use.

## Decision

Use Docker Compose with the official Postgres image.

Expose Postgres on host port:

```text
6000
```

Use container port:

```text
5432
```

Example connection URL:

```text
postgres://cloudiam:cloudiam@127.0.0.1:6000/cloudiam?sslmode=disable
```

Do not build a custom database Dockerfile.

Do not bake AWS credentials or CloudIAM session secrets into images or Compose files.

## Acceptance Criteria

- `docker compose up -d postgres` starts Postgres.
- Host port is `6000`.
- Database name/user/password are local-dev values only.
- Application reads database URL from environment/config.
- No AWS secrets are present in Compose files.
