---
title: "Console JWT session contract for TASK-000016"
type: decision
status: superseded
created_at: 2026-10-07
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000016
tags:
  - console
  - auth
  - jwt
  - security
---

# Decision: Console JWT Session Contract

## Decision

Use a short-lived signed console JWT for MVP dashboard APIs.

This decision originally used env-only admin credentials and bearer-token responses. That part is superseded by:

- `TASK-000024`: HttpOnly cookie-backed console sessions with `/me`, `/refresh`, and `/logout`.
- `TASK-000025`: `consoleauth` and `approval` domain boundaries.
- `TASK-000026`: persisted console users with Argon2id password hashes.

The local/demo console credential is seeded outside YAML:

```text
CLOUDIAM_CONSOLE_SEED_USERNAME
CLOUDIAM_CONSOLE_SEED_PASSWORD
CLOUDIAM_CONSOLE_JWT_SECRET
CLOUDIAM_CONSOLE_JWT_TTL
```

Console login:

```http
POST /api/admin/login
Content-Type: application/json

{
  "username": "admin",
  "password": "..."
}
```

Successful login returns:

```json
{
  "authenticated": true,
  "subject": "admin",
  "role": "console_operator",
  "expires_at": "2026-10-07T18:30:00Z"
}
```

The JWT is stored only in an HttpOnly session cookie. Browser clients do not store it in local storage and do not send it through an `Authorization` header.

JWT claims:

- `iss`: `cloudiam`
- `aud`: `cloudiam-console`
- `sub`: console username
- `role`: `console_operator`
- `iat`
- `exp`
- `jti`

Default token TTL is short and configurable by environment if needed.

Password verification uses persisted Argon2id password hashes. The MVP can seed one local/demo console operator, but raw password material must not appear in YAML, database rows, API responses, audit records, idempotency records, or logs.

## Rationale

A static bearer token is too blunt for approval actions. A JWT gives the dashboard a normal login/session shape, expiration, a subject to attach to approval audit metadata, and a token ID for future revocation or investigation.

This remains MVP/demo auth, not production identity. It intentionally avoids multi-user lifecycle, password reset, OIDC, or RBAC.

## Acceptance Criteria

- JWT signature, expiration, issuer, audience, and role are verified.
- Expired or malformed tokens are rejected.
- Console password material is not stored in YAML.
- Raw console password and JWT secret are never logged or returned.
