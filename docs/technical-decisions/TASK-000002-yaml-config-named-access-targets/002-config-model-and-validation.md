---
title: "Config model and validation for TASK-000002"
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
  - validation
  - security
---

# Decision: Config Model And Validation For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- [001-contract.md](001-contract.md)

## Context

`TASK-000002` introduces YAML as governance configuration. Before changing the server behavior, we need a precise config model and validation contract.

This decision covers only Section 1:

```text
config model -> YAML loader -> validation tests
```

It does not implement CLI, gRPC, policy evaluation, or AWS changes.

## Decision

Split config into:

```text
internal/core/config
internal/adapters/config/yaml
```

Reason:

- `internal/core/config` owns the product config model and validation rules.
- `internal/adapters/config/yaml` owns reading YAML from disk.
- YAML parsing is an adapter detail.

## Initial File Shape

```text
server/
  configs/
    cloudiam.yaml

  internal/
    core/
      config/
        types.go
        validate.go
        validate_test.go

    adapters/
      config/
        yaml/
          loader.go
          loader_test.go
```

## YAML Shape

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

## Model Decisions

### Maps for named resources

Use maps for unique named resources:

```text
accounts
roles
users
targets
```

Reason:

- Names are identifiers.
- YAML duplicate-key behavior is parser-dependent, so the structure should not rely on list duplicate detection for primary objects.
- Lookup by name is direct.

### Lists for policies

Use a list for policies:

```yaml
policies:
  - name: developers-sandbox-readonly
```

Reason:

- Policies are ordered enough for humans to read as rules.
- Validation must reject duplicate policy names.

### Durations

Use Go duration strings in YAML:

```text
15m
1h
```

Validation rules:

- `default_duration` must parse with `time.ParseDuration`.
- `max_duration` must parse with `time.ParseDuration`.
- both must be positive.
- `default_duration <= max_duration`.

### Version

Only this version is accepted:

```text
v1
```

## Validation Contract

The config is invalid when:

- `version` is empty or not `v1`.
- `tenant` is empty.
- `serve.grpc.addr` is empty.
- AWS provider region is empty.
- an AWS account ID is empty.
- an AWS role ARN is empty.
- a user references an empty group.
- a target references an unknown provider.
- a target references an unknown account.
- a target references an unknown role.
- a policy has an empty name.
- a policy name is duplicated.
- a policy references an unknown group.
- a policy references an unknown target.
- a target has invalid duration strings.
- a target has non-positive durations.
- a target default duration exceeds max duration.

## Security Validation Contract

The YAML loader must reject unknown fields.

The config must reject secret-looking fields anywhere in the YAML tree, including keys such as:

```text
access_key
access_key_id
secret_access_key
session_token
aws_access_key_id
aws_secret_access_key
aws_session_token
cloudiam_token
token
password
secret
```

Rationale:

- YAML config is governance data.
- AWS credentials and CloudIAM session tokens belong in the hosted service secret store or AWS credential chain, not config.

Role ARNs are allowed because they are identifiers, not credentials.

## Section 1 Acceptance Criteria

- `server/configs/cloudiam.yaml` exists with masked placeholders.
- YAML loader reads config from disk.
- YAML loader rejects unknown fields.
- YAML loader rejects secret-looking fields.
- Config validation checks required fields.
- Config validation checks references.
- Config validation checks durations.
- Tests cover valid config.
- Tests cover unknown field.
- Tests cover secret-looking field.
- Tests cover unknown target/account/role/group references.
- Tests cover invalid duration.

## Explicitly Deferred

- CLI command changes.
- gRPC request shape changes.
- policy evaluator service behavior.
- AWS adapter changes.
- Docker packaging.
- hosted-service login.
