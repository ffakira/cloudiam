---
title: "Production secret and config boundary for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - secrets
  - production
  - configuration
---

# Decision: Production Secret And Config Boundary

## Context

CloudIAM local development uses `.env.local`, Docker Compose defaults, local Postgres, local Redis, AWS profiles, and optional console-user seeding. Those defaults are useful for the MVP, but they are unsafe as implicit production behavior.

The Task 28 security baseline requires no secret leakage and explicit production controls. Scanner tooling is now pinned and passing, so the remaining secret-management gap is making local/demo configuration impossible to mistake for production.

## Decision

Use an explicit production switch:

- `CLOUDIAM_DEPLOYMENT_ENV=production`
- `CLOUDIAM_DEPLOYMENT_ENV=prod`

When production mode is enabled, server startup must fail closed unless the effective configuration satisfies the checks below.

### Required Production Checks

- Console session cookies must be marked `Secure`: `CLOUDIAM_CONSOLE_COOKIE_SECURE=true`.
- Shared HTTP rate limiting must be configured: `CLOUDIAM_REDIS_URL` must be non-empty.
- The device verification URI must use HTTPS: `CLOUDIAM_AUTH_VERIFICATION_URI=https://...`.
- The Postgres URL must not use `sslmode=disable`.
- If console JWT signing is enabled, `CLOUDIAM_CONSOLE_JWT_SECRET` must be non-placeholder and at least 32 characters.
- Console seed password bootstrap is disabled by default.

### Explicit Bootstrap Overrides

Two overrides exist because real deployments sometimes bootstrap infrastructure in stages:

- `CLOUDIAM_ALLOW_CONSOLE_SEED_IN_PRODUCTION=true`
- `CLOUDIAM_ALLOW_INSECURE_DATABASE_IN_PRODUCTION=true`

When console seeding is explicitly allowed in production, the seed password still must be non-placeholder and at least 16 characters.

Overrides are intentionally verbose so they are visible in manifests, runbooks, and reviews.

### Local Development Boundary

`.env.example`, Docker Compose defaults, local AWS profile usage, local Postgres credentials, local Redis, and `sslmode=disable` are local/demo behavior only.

Local mode may keep:

- `CLOUDIAM_DEPLOYMENT_ENV=development`
- `CLOUDIAM_CONSOLE_COOKIE_SECURE=false`
- `CLOUDIAM_DATABASE_URL` with `sslmode=disable`
- local seed user credentials for demos
- local AWS profile mounting for Compose

## Verification

- Unit tests cover production-mode acceptance and fail-closed checks for secure cookies, Redis, HTTPS verification URI, database TLS, weak JWT secrets, disabled console seeding, explicit seed override, and weak seed passwords.
- `go test ./internal/app/server` passes.
- `go test ./...` passes.

## Consequences

- Production deployments must declare themselves explicitly.
- Local/demo defaults remain ergonomic, but they no longer define acceptable production posture.
- Secret-manager integration remains deployment-specific, but the application now rejects obvious local/demo secret patterns in production mode.
