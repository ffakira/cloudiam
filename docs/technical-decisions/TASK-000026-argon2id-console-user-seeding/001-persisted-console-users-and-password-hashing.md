---
title: "Persisted console users and Argon2id password hashing"
id: TASK-000026-DEC-001
task: TASK-000026
date: 2026-10-08
status: accepted
---

# Decision

Console login credentials are persisted as `console_users` rows and password hashes use Argon2id with per-password random salts. Verification must reject malformed hashes and Argon2id cost parameters outside the application-defined bounds before deriving a key.

The server may seed a local/demo console operator from explicit environment variables:

- `CLOUDIAM_CONSOLE_SEED_USERNAME`
- `CLOUDIAM_CONSOLE_SEED_PASSWORD`
- `CLOUDIAM_CONSOLE_JWT_SECRET`
- `CLOUDIAM_CONSOLE_JWT_TTL`

The raw seed password is only startup input. It is not stored in the database, not returned by APIs, and not written to audit events.

# Rationale

The earlier SHA-256 shortcut was unacceptable even for an MVP. SHA-256 is fast and cheap to brute-force, and using an env-only credential path would make the demo architecture look weaker than the security story we are trying to tell.

Argon2id is intentionally slow and memory-hard, which is the right default for password verification. Persisting users also gives us a clean boundary for future lifecycle work without coupling console auth to process environment.

# Consequences

- A database and migrations are required for console login.
- Local demos must provide seed env vars at startup if no console user already exists.
- Reseeding the same username replaces the password hash and active flag.
- Production should replace env seeding with an operator bootstrap or SSO/OIDC flow.

# Verification

- Unit tests cover Argon2id hash generation, verification, random salts, and malformed hash rejection.
- Unit tests cover rejection of tampered out-of-bounds Argon2id costs.
- Console auth service tests verify persisted-user login and invalid credential handling.
- PostgreSQL integration tests cover seed, reseed, read, and non-Argon2id hash rejection.
