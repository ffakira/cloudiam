---
title: "Runtime configuration for TASK-000001"
type: decision
status: accepted
created_at: 2026-10-06
updated_at: 2026-10-06
owner: codex
applies_to:
  - TASK-000001
tags:
  - configuration
  - cli
  - aws
  - mvp
---

# Decision: Runtime Configuration For TASK-000001

## Related Work

- `TASK-000001`: CLI to gRPC AWS access slice.
- [001-contract.md](001-contract.md)
- [002-implementation-decisions.md](002-implementation-decisions.md)

## Context

The first vertical slice needs enough runtime configuration to run locally, but not enough machinery to become a configuration system.

This task must not introduce YAML, `.env`, Viper, config files, or a database-backed settings model.

## Decision

Use command flags for application configuration.

Use the AWS SDK default credential chain for AWS credentials.

## Server Configuration

The server accepts:

```text
--addr
--role-arn
```

Defaults:

```text
--addr 127.0.0.1:50051
```

Required:

```text
--role-arn
```

The server must fail fast if `--role-arn` is empty.

Example:

```bash
AWS_PROFILE=<BROKER_AWS_PROFILE> go run ./cmd/access-server \
  --addr 127.0.0.1:50051 \
  --role-arn arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly
```

## CLI Configuration

The CLI accepts:

```text
--addr
```

Defaults:

```text
--addr 127.0.0.1:50051
```

The `request-access` command accepts:

```text
--principal
--provider
--account
--role
--duration
```

Required:

```text
--principal
--provider
--account
--role
--duration
```

`--duration` is seconds.

Example:

```bash
go run ./cmd/access-cli \
  --addr 127.0.0.1:50051 \
  request-access \
  --principal <LOCAL_USER> \
  --provider aws \
  --account sandbox \
  --role readonly \
  --duration 900
```

## AWS Credential Configuration

AWS credentials are not passed as application flags.

The server uses the AWS SDK default credential chain.

For local testing, set:

```bash
AWS_PROFILE=<BROKER_AWS_PROFILE>
```

The CLI does not need broker AWS credentials.

## Explicitly Deferred

- YAML config
- `.env` loading
- Viper
- Cobra
- config validation command
- multiple role mappings
- provider/account/role lookup tables
- policy config

## Acceptance Criteria

- Server defaults to `127.0.0.1:50051`.
- CLI defaults to `127.0.0.1:50051`.
- Server fails fast when `--role-arn` is missing.
- CLI fails fast when required request flags are missing.
- AWS credentials are loaded only through the AWS SDK default credential chain.
- No YAML, `.env`, Viper, or config file loader is added in this task.

