---
title: "Technical contract for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-06
updated_at: 2026-10-06
owner: codex
applies_to:
  - TASK-000002
tags:
  - yaml
  - configuration
  - policy
  - access-targets
  - aws
  - cli
  - security
---

# Decision: Technical Contract For TASK-000002

## Related Work

- `TASK-000001`: CLI to gRPC AWS access slice.
- `TASK-000002`: YAML config and named access targets.

## Context

The first task proved the broker can issue temporary AWS credentials. The next task should make the interface business-friendly by hiding AWS internals behind named access targets.

## Sources

- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
- [OWASP Authorization Patterns Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Patterns_Cheat_Sheet.html)
- [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)
- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [OWASP API Security Top 10 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10/)

## Decision

Use an Ory-style YAML config file as the server source of truth for:

- tenant/company identity
- server address
- AWS accounts and role ARNs
- principals and groups
- named access targets
- policies
- default/max durations

Do not introduce a database yet.

## Initial Config Shape

```yaml
version: v1
tenant: "<COMPANY_UUID>"

serve:
  grpc:
    addr: 127.0.0.1:50051

providers:
  aws:
    region: us-east-1
    accounts:
      sandbox:
        id: "<AWS_ACCOUNT_ID>"
        roles:
          readonly:
            arn: "arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly"

principals:
  users:
    akira:
      groups:
        - developers

access:
  targets:
    aws-sandbox-readonly:
      provider: aws
      account: sandbox
      role: readonly
      default_duration: 15m
      max_duration: 1h

  policies:
    - name: developers-sandbox-readonly
      groups:
        - developers
      targets:
        - aws-sandbox-readonly
```

## Behavior Contract

- Request authorization is evaluated for every access request.
- Requests are denied by default.
- Unknown target is denied before AWS call.
- Unknown principal is denied before AWS call.
- Principal without matching policy is denied before AWS call.
- Principal and target must belong to the same tenant/company config.
- Missing duration uses target `default_duration`.
- Requested duration must be less than or equal to target `max_duration`.
- Valid request resolves to one AWS role ARN.
- AWS adapter receives resolved role ARN and duration.
- Credentials are not stored.

## Security Contract

- YAML config must not contain AWS access keys, AWS secret keys, session tokens, or CloudIAM session tokens.
- YAML config may contain AWS role ARNs because role ARNs are identifiers, not credentials.
- Config validation must use allowlisted known fields; unknown top-level sections are rejected.
- Config validation must reject duplicate target names, duplicate user names, and duplicate policy names when represented as lists.
- Config validation must reject references to unknown groups, targets, accounts, or roles.
- Request payload must not accept provider/account/role/role ARN for the named-target flow.
- The server resolves provider/account/role/role ARN from config only.
- Denial errors should explain the failed condition without leaking credentials or hidden policy internals.
- The config file is governance data; secrets belong in the hosted service secret store or AWS credential chain, not YAML.

## CLI Contract

Use:

```bash
cloudiam access request <target>
```

Temporary development form before `TASK-000003` login exists:

```bash
go run ./cmd/access-cli access request aws-sandbox-readonly --principal akira
```

Duration override:

```bash
go run ./cmd/access-cli access request aws-sandbox-readonly --principal akira --duration 900
```

The user-facing CLI must not expose service endpoint flags such as:

```text
--addr
--server
--host
```

The CLI is a client for the hosted CloudIAM service, not a self-hosted backend.

Authentication and endpoint discovery are deferred to `TASK-000003`.

## Acceptance Criteria

- Config loader validates references between targets, accounts, roles, groups, and policies.
- Config loader rejects unknown sections/fields.
- Config loader rejects secret-looking AWS credential fields.
- Config includes tenant/company identifier.
- CLI no longer requires provider/account/role flags for the request flow.
- CLI does not expose user-facing service endpoint flags.
- Denied requests do not call AWS.
- Allowed requests still return temporary AWS credentials.
- No database or dashboard is added.
