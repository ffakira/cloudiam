---
title: "CLI device-code login"
id: TASK-000003
type: feature
status: completed
created_at: 2026-10-06
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "auth-1"
current_focus: "Hosted CloudIAM login -> one-hour CLI session"
priority: P0
sprint_candidate: true
tags:
  - cli
  - auth
  - device-code
  - postgres
  - hosted-service
  - mvp
---

# Task: CLI Device-Code Login

## Outcome

Completed an end-to-end local MVP login and access flow:

- CLI starts a device-code login.
- Local/Postman approval completes the device authorization over HTTP.
- CLI stores a one-hour CloudIAM session locally.
- Access requests use the CloudIAM session token instead of `--principal`.
- Server resolves the CloudIAM session to a principal before enforcing YAML policy and issuing AWS STS credentials.

## Original Outcome

Make the CLI behave like a hosted-service client, similar to Stripe/GitHub CLI.

Target flow:

```bash
cloudiam login
```

CLI shows:

```text
Open: https://cloudiam.abacaxi.dev/device
Code: ABCD-1234
```

User signs in through the browser. The CLI receives a short-lived CloudIAM session token.

## Scope

- Add `cloudiam login` flow.
- Generate or request a one-time device code.
- Store a short-lived CloudIAM session token locally.
- Persist server-side device authorization and CloudIAM session state in Postgres.
- Run local Postgres through Docker Compose on host port `6000`.
- Session token lifetime: 1 hour.
- Use the session token for `cloudiam access request`.
- Resolve hosted service endpoint from the authenticated workspace/company.

## Technical Decisions

- [Device-code login contract](../../technical-decisions/TASK-000003-cli-device-code-login/001-contract.md)
- [CLI session storage](../../technical-decisions/TASK-000003-cli-device-code-login/002-cli-session-storage.md)
- [Database persistence](../../technical-decisions/TASK-000003-cli-device-code-login/003-database-persistence.md)
- [Local Docker Compose runtime](../../technical-decisions/TASK-000003-cli-device-code-login/004-local-docker-compose-runtime.md)
- [Device authorization schema](../../technical-decisions/TASK-000003-cli-device-code-login/005-device-authorization-schema.md)
- [Local auth boundary](../../technical-decisions/TASK-000003-cli-device-code-login/006-local-auth-boundary.md)

## Implementation Contract

Each requirement has a stable ID.

Requirement ID format:

```text
REQ-000003-<AREA>-<NUMBER>
```

### Login Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000003-LOGIN-001 | CLI supports `login` command. | CLI parser test covers command. |
| REQ-000003-LOGIN-002 | Login displays verification URI and user code. | CLI output test covers both. |
| REQ-000003-LOGIN-003 | Device/user/session tokens use cryptographically secure randomness. | Unit test covers generated token length and uniqueness; code review confirms `crypto/rand`. |
| REQ-000003-LOGIN-004 | Login creates a CloudIAM session that expires after one hour. | Unit test covers expiry timestamp. |

### Session Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000003-SESSION-001 | Session is stored under user config directory. | Unit test uses injected temp config dir. |
| REQ-000003-SESSION-002 | Session file permission is `0600`. | Unit test checks mode. |
| REQ-000003-SESSION-003 | Expired sessions are rejected. | Unit test covers expired session. |
| REQ-000003-SESSION-004 | Session store rejects AWS credential-looking fields. | Unit test covers forbidden fields. |
| REQ-000003-SESSION-005 | Session token is not printed during normal access request. | CLI output test confirms absence. |

### Database Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000003-DB-001 | Local Postgres runs through Docker Compose. | `docker compose up -d postgres` starts service. |
| REQ-000003-DB-002 | Local Postgres host port is `6000`. | Compose file maps `6000:5432`. |
| REQ-000003-DB-003 | Device authorization state is persisted. | Repository test creates and reads pending authorization. |
| REQ-000003-DB-004 | CloudIAM sessions are persisted server-side. | Repository test creates and reads session by token hash. |
| REQ-000003-DB-005 | Raw session tokens and device codes are not stored. | Schema/repository review and tests use hashes. |
| REQ-000003-DB-006 | AWS credentials are not stored. | Schema review confirms absence of AWS credential columns. |

### Access Request Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000003-ACCESS-001 | Access request requires a valid CloudIAM session. | CLI/app test covers missing session. |
| REQ-000003-ACCESS-002 | User-facing access request has no endpoint flags. | CLI parser test rejects `--addr`, `--server`, `--host`. |
| REQ-000003-ACCESS-003 | User-facing access request has no provider/account/role flags. | CLI parser test rejects provider/account/role flags. |

### Local Auth Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000003-AUTH-001 | Browser approval is local/dev simulation, not production identity. | ADR and approval UI text make this explicit. |
| REQ-000003-AUTH-002 | Approved principal is still checked against YAML policy before AWS. | End-to-end or service test covers policy denial after login. |
| REQ-000003-AUTH-003 | No passwords, IdP, MFA, or org membership management are added. | Review confirms absence. |

## Non-Goals

- No AWS credential storage.
- No self-hosted backend configuration.
- No user-facing `--addr`, `--server`, or `--host`.
- No long-lived API keys.
- No dashboard implementation.
- No audit persistence or dashboard read-model schema in this task.
- No production identity provider or organization membership implementation.

## Product Model

The CLI talks to our hosted service:

```text
cloudiam.internal.<company-uuid>.abacaxi.dev
```

Users should not pass that endpoint on every command. It is discovered or stored during login.

## Local Database

Local environment files:

```bash
cp .env.example .env.local
```

```bash
make db-up
```

Local database URL:

```text
postgres://cloudiam:cloudiam@127.0.0.1:6000/cloudiam?sslmode=disable
```

Apply migrations:

```bash
make migrate-up
```

Check migration state:

```bash
make migrate-status
```

If a different local database URL is needed, override `CLOUDIAM_DATABASE_URL`. Avoid using a generic `DATABASE_URL`; other projects may already set it.

Run database-backed repository tests:

```bash
make server-test-integration
```

## Implementation Progress

- [x] Local Postgres runtime through Docker Compose.
- [x] Postgres 18-compatible volume mount.
- [x] Versioned migration for device authorizations and sessions.
- [x] Make targets for DB, migrations, and integration tests.
- [x] Go auth domain types and repository port.
- [x] Postgres auth repository adapter.
- [x] Integration tests for approve/exchange/session/revoke flow.
- [x] Integration test for 32-byte hash schema enforcement.
- [x] Core login service starts device login with secure device/user codes.
- [x] Core login service approves local/dev browser login.
- [x] Core login service exchanges approved device code for one-hour session.
- [x] Core login service stores hashes only, never raw device/session tokens.
- [x] Proto source files restored for generated gRPC code.
- [x] Auth gRPC contract added for start/approve/exchange device login.
- [x] Auth gRPC handler maps domain states to explicit gRPC status codes.
- [x] Server registers DB-backed AuthService when `CLOUDIAM_DATABASE_URL` is configured.
- [x] CLI local session store writes `0600` JSON under user config path.
- [x] CLI local session store rejects expired sessions.
- [x] CLI local session store rejects AWS credential-looking fields.
- [x] CLI login adapter starts device login, prints URL/code, polls exchange, and saves session.
- [x] CLI parser accepts `login` without exposing address flags.
- [x] Access gRPC contract accepts CloudIAM session token.
- [x] Server resolves CloudIAM session token to principal before AWS access request.
- [x] CLI `access request` loads local CloudIAM session instead of accepting `--principal`.
- [x] User-facing access command rejects provider/account/role/principal/service-address flags.
- [x] Local dev HTTP endpoint approves device login from Postman.
- [x] HTTP approval endpoint is local by default on `127.0.0.1:8080`.

## Done When

- [x] CLI has `login` command.
- [x] Login service gives caller a browser URL and one-time code.
- [x] Device login state is stored in Postgres through reviewed migrations.
- [x] Login service issues a CloudIAM session valid for 1 hour.
- [x] Access request uses CloudIAM session token.
- [x] User-facing access request command does not include service endpoint flags.
- [x] Local/dev approval can be completed over HTTP without dashboard or browser UI.
