---
title: "Production config and startup validation for TASK-000027"
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
  - verification
---

# Decision: Production Config And Startup Validation

## Context

P2 implements the production config and startup validation contract from ADR 004.

Task 28 already added most production fail-closed checks:

- secure console cookies in production
- Redis required in production
- database TLS must not be disabled in production unless explicitly overridden
- weak or placeholder console JWT secrets rejected
- console seed password disabled in production unless explicitly overridden
- weak production console seed password rejected
- persistence schema verified before serving database-backed traffic
- startup cleanup errors redacted

P2 reviewed those checks against the Task 27 startup contract and found one gap: production allowed an empty `CLOUDIAM_AUTH_VERIFICATION_URI`. Later startup wiring would default the verification URI to local HTTP, `http://127.0.0.1:8080/device`, which is acceptable for local/demo mode but not production.

## Decision

Production mode now requires `CLOUDIAM_AUTH_VERIFICATION_URI` to be explicit and HTTPS.

Accepted behavior:

- local/demo mode may omit `CLOUDIAM_AUTH_VERIFICATION_URI` and use the local device approval default
- production mode fails closed when `CLOUDIAM_AUTH_VERIFICATION_URI` is empty
- production mode fails closed when `CLOUDIAM_AUTH_VERIFICATION_URI` uses `http://`
- production mode accepts an explicit `https://` verification URI

## Verification

Focused verification passed on 2026-10-12:

```bash
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./internal/app/server
```

Regression added:

- `TestValidateProductionSecretConfigRequiresExplicitVerificationURI`

Existing accepted P2 coverage remains:

- `TestValidateProductionSecretConfigAcceptsHardenedConfig`
- `TestValidateProductionSecretConfigRequiresSecureCookie`
- `TestValidateProductionSecretConfigRequiresRedis`
- `TestValidateProductionSecretConfigRequiresHTTPSVerificationURI`
- `TestValidateProductionSecretConfigRejectsDisabledDatabaseTLS`
- `TestValidateProductionSecretConfigAllowsExplicitDatabaseTLSOverride`
- `TestValidateProductionSecretConfigRejectsWeakJWTSecret`
- `TestValidateProductionSecretConfigRejectsConsoleSeedByDefault`
- `TestValidateProductionSecretConfigAllowsStrongConsoleSeedWithOverride`
- `TestValidateProductionSecretConfigRejectsWeakConsoleSeedWithOverride`
- startup schema and cleanup behavior covered by existing app/Postgres tests and prior hardening regressions

## Consequences

- Production startup cannot silently use the local device approval URL.
- Operators must configure the externally reachable HTTPS device verification URL before enabling production mode.
- P2 closes without adding production SSO, secret-manager integration, readiness endpoints, or SIEM/export behavior; those remain outside this slice.
