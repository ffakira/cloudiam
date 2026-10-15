---
title: "Production config and startup contract for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - config
  - startup
  - fail-closed
---

# Decision: Production Config And Startup Contract

## Context

Local/demo behavior is intentionally convenient. Production behavior must be stricter.

Task 28 already hardened several production security boundaries. Task 27 turns those boundaries into an operational startup contract so unsafe deployments fail before serving privileged traffic.

## Decision

Production startup must validate required runtime configuration and fail closed before accepting HTTP or gRPC traffic when critical production requirements are missing or unsafe.

The production startup contract covers:

- deployment mode is explicit enough to distinguish local/demo from production
- console cookie security settings are production-safe
- JWT/session signing secrets meet production strength requirements
- demo console user seeding is disabled or explicitly blocked in production
- Postgres connection is configured and required schema is present
- Redis is configured when production shared rate limiting is required
- AWS region/runtime config is present for STS issuance
- verification URLs and device approval URLs do not rely on insecure local-only assumptions
- config files do not contain secret material
- startup cleanup failures are redacted and handled predictably

## Accepted Local/Demo Exceptions

Local/demo mode may keep convenience behavior when it is clearly outside production:

- insecure local HTTP callback or device URLs
- local fallback rate limiter
- demo console user seeding
- insecure cookie setting for local HTTP development
- local Docker Compose defaults

These exceptions must not silently apply in production.

## Rejected Behavior

Production mode must not:

- start with weak or default signing secrets
- serve console admin routes with insecure cookies
- rely on process-local rate limiting when Redis is required
- auto-seed demo console users without explicit non-production mode
- accept partial or missing persistence schema
- expose raw connection strings, Redis URLs, secrets, tokens, or AWS credentials in startup errors

## Verification Expectations

P2 should add or confirm tests for:

- valid production startup config
- missing/weak secret failure
- insecure cookie failure
- missing Redis failure when production requires Redis
- demo seeding rejection in production
- missing schema failure
- redacted startup errors

## Consequences

- Production behavior is stricter than local/demo behavior.
- Some deployments that previously started will intentionally fail until configured safely.
- The runbook must explain each production startup failure and the expected fix.
