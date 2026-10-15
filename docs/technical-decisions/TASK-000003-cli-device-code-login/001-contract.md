---
title: "Device-code login contract for TASK-000003"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000003
tags:
  - cli
  - auth
  - device-code
  - security
---

# Decision: Device-Code Login Contract For TASK-000003

## Related Work

- `TASK-000003`: CLI device-code login.
- [TASK-000002 gRPC target request contract](../TASK-000002-yaml-config-named-access-targets/006-grpc-target-contract.md)
- [TASK-000002 CLI command contract](../TASK-000002-yaml-config-named-access-targets/007-cli-command-contract.md)

## Sources

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://www.rfc-editor.org/info/rfc8628/)
- [OWASP OAuth2 Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)

## Context

The CLI is becoming a hosted-service client. Users should log in once, then request named access targets without passing service endpoints or AWS internals.

RFC 8628 defines the product direction: the CLI starts a device authorization, displays a verification URI and user code, then polls until the user completes browser authentication or the device code expires.

For the local MVP, the browser-hosted CloudIAM app does not exist yet. We still need the internal shape to match the future hosted flow so this does not rot.

## Decision

Use device-code semantics as the product contract:

```text
cloudiam login
-> receive verification_uri, user_code, device_code, expires_in, interval
-> display verification_uri and user_code
-> poll token endpoint using device_code
-> store short-lived CloudIAM session
```

The session token lifetime is one hour.

The access request command remains:

```bash
cloudiam access request <target>
```

No user-facing access command may accept:

```text
--addr
--server
--host
provider
account
role
role_arn
```

## Local MVP Boundary

Until a hosted browser/device endpoint exists, local development may simulate the authorization response and issue a local CloudIAM session.

That simulation must still preserve the same boundaries:

- generate codes/tokens with cryptographically secure randomness
- expire device codes
- expire sessions after one hour
- never store AWS credentials
- never print raw session tokens in normal output

## Acceptance Criteria

- CLI has `login` command.
- Login output shows verification URI and user code.
- Login creates a CloudIAM session with one-hour expiry.
- Access request requires a valid CloudIAM session.
- Expired sessions are rejected.
- Session token is generated with `crypto/rand`.
- Session data contains no AWS access key, secret key, or session token.
- Endpoint discovery remains internal, not a user-facing flag.
