---
title: "Console auth and approval package boundaries for TASK-000025"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000025
tags:
  - architecture
  - naming
  - auth
  - approval
---

# Decision: Console Auth And Approval Package Boundaries

## Decision

Do not use `admin` as a core domain package.

Use:

- `internal/core/consoleauth` for dashboard/operator console authentication: credential validation, signed session token creation, token verification, and refresh.
- `internal/core/approval` for resource/JIT approval semantics: approve/deny actions, idempotency fingerprints, mutation inputs/results, store contract, and transition errors.

The HTTP adapter may continue serving `/api/admin/...` during the MVP because that route prefix is an external API label. It must not force the core domain model to be called `admin`.

## Rationale

`admin` is a role or actor label, not a bounded context. Leaving session auth, approval mutations, idempotency, and dashboard behavior in a package named `admin` would invite unrelated privileged behavior to collect there.

`consoleauth` describes the browser console session boundary. `approval` describes the JIT resource approval transition boundary. These names make the next feature less likely to land in the wrong package.

## Acceptance Criteria

- No `internal/core/admin` package remains.
- Approval mutation storage depends on `internal/core/approval`, not console session auth.
- Dashboard session code depends on `internal/core/consoleauth`, not approval mutation semantics.
- `/api/admin/...` remains stable until a separate HTTP contract migration is planned.
