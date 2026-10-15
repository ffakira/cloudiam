---
title: "Persistent audit and session storage"
id: TASK-000008
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "persistence-1"
priority: P1
sprint_candidate: true
tags:
  - database
  - audit
  - sessions
---

# Task: Persistent Audit And Session Storage

## Outcome

Persist audit events and CloudIAM sessions.

## Scope

- Treat Postgres as the persistent store for server-side CloudIAM sessions and audit events.
- Keep raw AWS credentials, raw device codes, raw user codes, and raw CloudIAM session tokens out of database storage.
- Fail clearly at server startup when database-backed auth/audit is enabled but required migrations are missing.
- Keep local no-database access mode available only for the narrow gRPC access demo path.
- Document the operational commands for starting Postgres, applying migrations, and running persistence tests.

## Not In Scope

- Dashboard read APIs or UI.
- Multi-tenant user management.
- Background retention jobs.
- Encryption-at-rest orchestration beyond relying on the deployment database.
- Changing the CLI local session file format.

## Why Now

The broker now has device login, audit events, Docker packaging, and config hot reload. The next product slice needs a durable session/audit baseline so dashboard read models can build on stable storage instead of in-memory assumptions.

## Technical Decisions

- [Persistence runtime boundary](../../technical-decisions/TASK-000008-persistent-audit-session-storage/001-persistence-runtime-boundary.md)
- [Schema and secret storage boundary](../../technical-decisions/TASK-000008-persistent-audit-session-storage/002-schema-and-secret-boundary.md)
- [Verification and operations](../../technical-decisions/TASK-000008-persistent-audit-session-storage/003-verification-and-operations.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000008-SESSION-001 | Server-side CloudIAM sessions are stored durably and looked up by token hash. | Auth store integration test covers create/exchange/find/revoke. |
| REQ-000008-AUDIT-001 | Access decisions and issued/failed access outcomes are persisted as audit events. | Audit store integration test covers record/list. |
| REQ-000008-SECRET-001 | Persistent tables do not store raw AWS credentials, raw CloudIAM session tokens, raw device codes, or raw user codes. | Schema tests assert forbidden columns are absent and token hash constraints exist. |
| REQ-000008-MIGRATION-001 | Database-backed server startup fails clearly when required persistence migrations are missing. | Unit or integration test covers schema readiness check. |
| REQ-000008-OPS-001 | Docs explain local Postgres, migrations, service startup, and persistence test commands. | README/project docs reference commands. |

## Result

Postgres is the durable store for CloudIAM device authorization state, server-side sessions, and access audit events when `CLOUDIAM_DATABASE_URL` is configured.

The server now verifies the required persistence schema during database-backed startup. If migrations are missing or partial, startup fails clearly with a `run migrations` error instead of accepting requests with broken auth/audit persistence.

Sensitive values remain outside durable storage: raw device codes, raw user codes, raw CloudIAM session tokens, AWS access key IDs, AWS secret access keys, and AWS session tokens are not stored in the persistence schema.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make server-test-integration`
