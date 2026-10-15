---
title: "CLI session storage for TASK-000003"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000003
tags:
  - cli
  - session
  - storage
  - security
---

# Decision: CLI Session Storage For TASK-000003

## Related Work

- `TASK-000003`: CLI device-code login.
- [001-contract.md](001-contract.md)

## Sources

- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)

## Context

The CLI needs a local session so users do not run `login` before every access request.

This session is a CloudIAM session only. It is not an AWS credential cache.

## Decision

Store a local session JSON file under the user config directory:

```text
<os.UserConfigDir>/cloudiam/session.json
```

Session file shape:

```json
{
  "version": "v1",
  "workspace": "<company-uuid>",
  "principal": "akira",
  "endpoint": "cloudiam.internal.<company-uuid>.abacaxi.dev",
  "access_token": "<opaque-cloudiam-token>",
  "expires_at": "2026-10-07T12:00:00Z"
}
```

File permissions:

```text
0600
```

The session token is opaque and generated with `crypto/rand`.

## Rules

- Do not store AWS credentials in the session file.
- Do not log or print the raw CloudIAM session token.
- Treat missing session as unauthenticated.
- Treat expired session as unauthenticated.
- The session token lifetime is one hour.

## Acceptance Criteria

- Session store writes `0600` files.
- Session store reads valid non-expired sessions.
- Session store rejects expired sessions.
- Session store rejects files containing AWS credential-looking fields.
- Tests use temp directories and do not write to the real user config directory.
