---
title: "Argon2id console user seeding"
id: TASK-000026
type: hardening
status: completed
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
phase: "dashboard-security-2"
priority: P0
sprint_candidate: true
tags:
  - security
  - auth
  - dashboard
  - password-hashing
  - persistence
---

# Task: Argon2id Console User Seeding

## Outcome

Replace the weak env-only SHA-256 console password shortcut with persisted console users and Argon2id password hashes.

## Scope

- Add a `console_users` persistence table.
- Hash console passwords with Argon2id using per-password random salts.
- Verify password hashes with constant-time comparison.
- Seed a local/demo console user at startup from explicit seed env vars.
- Keep raw passwords out of database rows, API responses, audit records, and logs.
- Update local env documentation away from `sha256:` password hashes.

## Not In Scope

- Multi-user lifecycle UI.
- Password reset.
- Production SSO/OIDC.
- Role hierarchy beyond the MVP console operator role.

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000026-HASH-001 | Console passwords use Argon2id hashes with random salts. | Unit tests cover hash/verify and malformed hashes. |
| REQ-000026-SEED-001 | Local startup can seed a console user without storing raw password material. | Store tests or startup wiring tests cover seed path. |
| REQ-000026-LOGIN-001 | Console login validates against persisted console user credentials. | Service tests cover valid and invalid login. |
| REQ-000026-SCHEMA-001 | Schema readiness checks require console user persistence. | Schema tests remain green. |
| REQ-000026-DOCS-001 | Env docs no longer recommend SHA-256 password hashes. | `.env.example` documents seed envs. |

## Result

Console authentication now uses persisted `console_users` rows with Argon2id password hashes and per-password random salts. Verification rejects malformed hashes and out-of-bounds Argon2id cost parameters before deriving a key.

The server can seed a local/demo console operator from `CLOUDIAM_CONSOLE_SEED_USERNAME` and `CLOUDIAM_CONSOLE_SEED_PASSWORD`, and it uses `CLOUDIAM_CONSOLE_JWT_SECRET` for signed console sessions. The local `.env.local` now seeds a demo `admin` user for development, while the database stores only the Argon2id hash.

The console JWT audience and role were renamed to `cloudiam-console` and `console_operator`, and the HTTP/core code now uses console/approver naming instead of growing the old admin domain.

## Verification

- `cd client && bun run build`
- `cd server && env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make config-validate`
- `make server-test-integration`
