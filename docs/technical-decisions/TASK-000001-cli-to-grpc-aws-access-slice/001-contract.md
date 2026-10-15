---
title: "Technical contract for TASK-000001"
type: decision
status: accepted
created_at: 2026-10-06
updated_at: 2026-10-06
owner: codex
applies_to:
  - TASK-000001
tags:
  - aws
  - grpc
  - cli
  - security
  - persistence
  - mvp
---

# Decision: Technical Contract For TASK-000001

## Related Work

- `TASK-000001`: CLI to gRPC AWS access slice.

## Context

The first MVP slice must prove one usable flow:

```text
CLI request -> gRPC server -> core access service -> AWS STS AssumeRole -> temporary credentials
```

The CLI must not call AWS directly. The server owns the AWS provider integration and credential issuing flow.

## Sources

- [OWASP API Security Top 10 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10/)
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
- [OWASP Authorization Patterns Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Patterns_Cheat_Sheet.html)
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)

## Architecture Decision

Use hexagonal architecture with explicit adapters:

```text
server/
  api/proto/access/v1/
  cmd/access-server/
  cmd/access-cli/
  gen/access/v1/
  internal/core/access/
  internal/adapters/aws/
  internal/adapters/grpc/
  internal/adapters/cli/
  internal/app/server/
  internal/app/cli/
```

Dependency rule:

```text
cmd -> internal/app -> internal/adapters -> internal/core
```

Core rule:

```text
internal/core/access must not import AWS SDK, gRPC, CLI, or protobuf packages.
```

## gRPC Contract

Define one gRPC service for this task:

```proto
service AccessService {
  rpc RequestAccess(RequestAccessRequest) returns (RequestAccessResponse);
}
```

Request fields:

```proto
string principal = 1;
string provider = 2;
string account = 3;
string role = 4;
int32 duration_seconds = 5;
```

Response fields:

```proto
string access_key_id = 1;
string secret_access_key = 2;
string session_token = 3;
string expires_at = 4;
```

`expires_at` must be an RFC3339 timestamp.

## Behavior Contract

- Empty `principal` is rejected.
- Empty `provider` is rejected.
- Empty `account` is rejected.
- Empty `role` is rejected.
- `duration_seconds <= 0` is rejected.
- Validation failures must happen before AWS is called.
- Valid requests are passed to the core access service.
- The gRPC adapter must not contain AWS SDK calls.
- The gRPC adapter must not make authorization decisions.
- The AWS adapter must not make authorization decisions beyond issuing the configured role.
- The CLI receives temporary credentials only through the gRPC response.

## Error Mapping

| Condition | gRPC code |
|---|---|
| Missing required field | `InvalidArgument` |
| Non-positive duration | `InvalidArgument` |
| Core validation failure | `InvalidArgument` |
| AWS provider failure | `Unavailable` for transient failures, `Internal` otherwise |

## Security Decision

Use these non-negotiable rules for the MVP:

1. Deny by default.
2. Enforce least privilege for the AWS broker identity.
3. Validate required request fields before any provider call.
4. Keep authorization decisions in the access layer, not in provider adapters.
5. Never log or persist AWS temporary secret material.
6. Treat AWS temporary credentials as secrets with expiration.
7. Test denied access paths, not only the happy path.

Security invariants:

- The server must not log `SecretAccessKey` or `SessionToken`.
- The CLI may print credentials because that is the requested output.
- The server must not write credentials to files.
- The server must not persist credentials.
- The broker identity must only have permission to assume allowed JIT roles.
- The server must not require IAM admin permissions.
- Empty required request fields must fail before calling AWS.
- Non-positive duration must fail before calling AWS.
- Returned credentials must include an expiration.
- Requests must fail closed by default.

## Persistence Decision

Do not create a database schema in `TASK-000001`.

The first slice must not add:

- SQLite
- Postgres
- migrations
- session tables
- audit tables
- user tables
- policy tables
- persistence packages

Temporary credentials are returned to the CLI but not stored.

## Acceptance Criteria

- `RequestAccess` succeeds for a valid request and returns all credential fields.
- `RequestAccess` rejects missing required fields without calling the provider.
- `RequestAccess` rejects non-positive duration without calling the provider.
- The CLI can call `RequestAccess` and print shell exports.
- The server calls AWS STS through the broker profile.
- No credential secret material is logged by the server.
- No database dependency, migration tooling, or persistence package is added.
- `internal/core/access` has no AWS SDK, gRPC, CLI, or protobuf imports.

## Deferred

- `CheckAccess`
- `ListAuditEvents`
- streaming APIs
- dashboard HTTP API
- authentication between CLI and server
- TLS/mTLS
- YAML policy config
- audit/session persistence
- Docker packaging
- multi-account support
- multi-cloud support
