---
title: "CLI to gRPC AWS access slice"
id: TASK-000001
type: feature
status: completed
created_at: 2026-10-06
updated_at: 2026-10-06
completed_at: 2026-10-06
owner: codex
phase: "vertical-slice-1"
current_focus: "CLI -> gRPC -> AWS STS"
outcome: "CLI successfully requested temporary AWS credentials through the local gRPC server using the cloud-iam-broker AWS profile."
tags:
  - aws
  - grpc
  - cli
  - security
  - mvp
---

# Completed Task: CLI to gRPC AWS Access Slice

## Outcome

Build the first usable server-side path:

```text
CLI request -> gRPC AccessService -> Go access service -> AWS STS AssumeRole -> temporary credentials
```

We already configured AWS manually. This task is about connecting that verified AWS setup to a small Go gRPC server and a CLI client.

## Current Status

Done:

- AWS CLI is configured locally.
- Broker IAM user/profile exists locally.
- Target JIT role exists.
- Manual `aws sts assume-role` works.
- AWS setup commands are documented in the runbook below.

Verified:

- Live-tested the smallest server and CLI vertical slice against AWS.
- Server ran with `AWS_PROFILE=cloud-iam-broker`.
- Server assumed `arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly`.
- CLI returned temporary AWS credentials.
- Returned credentials resolved to an assumed-role session for principal `akira`.

Implemented locally:

- `server/` Go module.
- gRPC proto and generated Go code.
- core access service and validation tests.
- AWS STS adapter.
- local gRPC server.
- CLI request command.

## Scope

- Create `server/` as a Go module.
- Add a minimal gRPC API with one access request method.
- Add a minimal CLI command that calls the gRPC server.
- Server calls AWS STS through the broker profile.
- Return temporary credentials to the CLI.
- Keep AWS-specific code out of the core access package.

## Technical Decisions

- [Technical contract for TASK-000001](../../technical-decisions/TASK-000001-cli-to-grpc-aws-access-slice/001-contract.md)
- [Implementation decisions for TASK-000001](../../technical-decisions/TASK-000001-cli-to-grpc-aws-access-slice/002-implementation-decisions.md)
- [Runtime configuration for TASK-000001](../../technical-decisions/TASK-000001-cli-to-grpc-aws-access-slice/003-runtime-configuration.md)

## Non-Goals

- No dashboard.
- No database.
- No YAML policy engine yet.
- No approval workflow.
- No multi-cloud implementation.
- No session recording.
- No long-lived credential storage.

## Masked AWS Context

Sensitive AWS identifiers are intentionally masked.

```text
Account: <AWS_ACCOUNT_ID>
Admin setup user: arn:aws:iam::<AWS_ACCOUNT_ID>:user/<ADMIN_SETUP_USER>
Broker user: arn:aws:iam::<AWS_ACCOUNT_ID>:user/<BROKER_USER_NAME>
Broker local profile: <BROKER_AWS_PROFILE>
Target role: arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly
```

## Implementation Contract

Each requirement has a stable ID so code, tests, and technical decisions can reference it.

Requirement ID format:

```text
REQ-000001-<AREA>-<NUMBER>
```

### Architecture Contract

Use hexagonal architecture with explicit adapters.

Ideal tree for this task:

```text
server/
  go.mod
  go.sum
  README.md

  api/
    proto/
      access/
        v1/
          access.proto

  cmd/
    access-server/
      main.go
    access-cli/
      main.go

  gen/
    access/
      v1/
        access.pb.go
        access_grpc.pb.go

  internal/
    core/
      access/
        errors.go
        ports.go
        service.go
        service_test.go
        types.go

    adapters/
      aws/
        sts_issuer.go
      grpc/
        access_handler.go
      cli/
        request_access.go

    app/
      server/
        app.go
      cli/
        app.go
```

Dependency direction:

```text
cmd -> internal/app -> internal/adapters -> internal/core
```

Requirements:

| ID | Contract | Verification |
|---|---|---|
| REQ-000001-ARCH-001 | Create the server module under `server/`. | `server/go.mod` exists. |
| REQ-000001-ARCH-002 | Use the hexagonal tree shown above. | Directory tree matches the contract for this task. |
| REQ-000001-ARCH-003 | Keep core product behavior in `internal/core/access`. | Core files exist under `internal/core/access`. |
| REQ-000001-ARCH-004 | Keep AWS STS code in `internal/adapters/aws`. | AWS SDK imports appear only in adapter/wiring code, not core. |
| REQ-000001-ARCH-005 | Keep gRPC transport mapping in `internal/adapters/grpc`. | Protobuf types are handled in the gRPC adapter, not core. |
| REQ-000001-ARCH-006 | Keep CLI behavior in `internal/adapters/cli` and `cmd/access-cli`. | CLI package calls the generated gRPC client. |
| REQ-000001-ARCH-007 | Keep `cmd/*` thin. | `cmd/*/main.go` only wires dependencies and starts the app. |
| REQ-000001-ARCH-008 | Do not add database, YAML config, dashboard, Docker, or frontend HTTP adapter in this task. | Review confirms those files/packages are absent. |

### gRPC Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000001-GRPC-001 | Define `AccessService` in `api/proto/access/v1/access.proto`. | Proto file contains `service AccessService`. |
| REQ-000001-GRPC-002 | Define `RequestAccess` RPC. | Proto file contains `rpc RequestAccess`. |
| REQ-000001-GRPC-003 | Request includes `principal`, `provider`, `account`, `role`, and `duration_seconds`. | Generated/request proto has those fields. |
| REQ-000001-GRPC-004 | Response includes `access_key_id`, `secret_access_key`, `session_token`, and `expires_at`. | Generated/response proto has those fields. |
| REQ-000001-GRPC-005 | Generate Go protobuf code into `gen/access/v1` using `DEC-000001-IMPL-004`. | Generated `.pb.go` and `_grpc.pb.go` files exist. |

### Core Access Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000001-CORE-001 | Define a core access request type. | `internal/core/access` exposes a request type. |
| REQ-000001-CORE-002 | Define a core grant type. | `internal/core/access` exposes a grant type. |
| REQ-000001-CORE-003 | Define a `CredentialIssuer` port. | Interface exists in core package. |
| REQ-000001-CORE-004 | Validate required fields before calling the issuer. | Unit tests cover missing principal/account/role. |
| REQ-000001-CORE-005 | Reject non-positive duration before calling the issuer. | Unit test covers zero/negative duration. |
| REQ-000001-CORE-006 | Call the issuer exactly once for a valid request. | Unit test uses fake issuer and asserts call count. |
| REQ-000001-CORE-007 | Do not import AWS SDK, gRPC, CLI, or protobuf packages in core. | Import review or automated grep confirms absence. |

### AWS Adapter Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000001-AWS-001 | Load credentials from the AWS SDK default credential chain per `DEC-000001-IMPL-005`. | Running with `AWS_PROFILE=<BROKER_AWS_PROFILE>` works. |
| REQ-000001-AWS-002 | Read the target role ARN from the required server flag per `DEC-000001-IMPL-006`. | Server fails at startup if role ARN is missing. |
| REQ-000001-AWS-003 | Call STS `AssumeRole`. | Manual CLI-to-server request returns STS credentials. |
| REQ-000001-AWS-004 | Return temporary credentials and expiration. | CLI output includes all credential fields and expiration. |
| REQ-000001-AWS-005 | Do not log secret material. | Review confirms no `SecretAccessKey` or `SessionToken` logging. |

### Server Adapter Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000001-SERVER-001 | Start a local gRPC server on `127.0.0.1:50051` by default per `DEC-000001-IMPL-002`. | Server listens on loopback. |
| REQ-000001-SERVER-002 | Register `AccessService`. | CLI can call `RequestAccess`. |
| REQ-000001-SERVER-003 | Convert protobuf request to core request. | Handler code maps request fields explicitly. |
| REQ-000001-SERVER-004 | Convert core grant to protobuf response. | Handler code maps grant fields explicitly. |
| REQ-000001-SERVER-005 | Map validation failures to gRPC errors. | Invalid CLI request returns a non-OK gRPC status. |

### CLI Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000001-CLI-001 | Connect to local gRPC server on `127.0.0.1:50051` by default per `DEC-000001-IMPL-002`. | CLI reaches running server. |
| REQ-000001-CLI-002 | Add `request-access` command using the standard library `flag` package per `DEC-000001-IMPL-003`. | Command exists and displays help. |
| REQ-000001-CLI-003 | Accept principal, provider, account, role, and duration flags. | Command rejects missing required flags. |
| REQ-000001-CLI-004 | Print credentials as shell exports. | Successful command prints export statements. |

Expected CLI shape:

```bash
AWS_PROFILE=<BROKER_AWS_PROFILE> go run ./cmd/access-server \
  --addr 127.0.0.1:50051 \
  --role-arn arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly
```

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

## Security Invariants

The task is not complete if any invariant fails.

These invariants are grounded in:

- [OWASP API Security Top 10 2023](https://api-security.owasp.org/editions/2023/en/0x11-t10/)
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
- [OWASP Authorization Patterns Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Patterns_Cheat_Sheet.html)
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
- [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html)
- [Technical contract](../../technical-decisions/TASK-000001-cli-to-grpc-aws-access-slice/001-contract.md)

- The server must not log `SecretAccessKey` or `SessionToken`.
- The CLI may print credentials because that is the requested output.
- The server must not write credentials to files.
- The server must not persist credentials.
- The broker identity must only have permission to assume allowed JIT roles.
- The server must not require IAM admin permissions.
- The core access package must not import AWS SDK packages.
- The AWS provider must not make authorization decisions beyond calling the configured role.
- Empty required request fields must fail before calling AWS.
- Non-positive duration must fail before calling AWS.
- Returned credentials must include an expiration.
- Requests must fail closed by default.

## Tests And Verification

### Automated tests for this task

- [x] Unit test: missing principal fails before provider call.
- [x] Unit test: missing account fails before provider call.
- [x] Unit test: missing role fails before provider call.
- [x] Unit test: non-positive duration fails before provider call.
- [x] Unit test: access service calls credential issuer for valid request.
- [x] Unit test: access package has no AWS SDK imports, or code review confirms it.

### Manual AWS checks for this task

- [x] Broker profile can assume the intended target role.
- [ ] Broker profile cannot create an IAM user.
- [ ] Broker profile cannot attach a policy.
- [x] CLI can request credentials through gRPC.
- [x] Returned credentials can call AWS STS identity lookup as the assumed role.

Suggested negative checks:

```bash
aws iam create-user \
  --profile <BROKER_AWS_PROFILE> \
  --user-name should-not-work

aws iam attach-user-policy \
  --profile <BROKER_AWS_PROFILE> \
  --user-name <BROKER_USER_NAME> \
  --policy-arn arn:aws:iam::aws:policy/AdministratorAccess
```

Expected result:

```text
AccessDenied
```

## Done When

- [x] Server starts locally.
- [x] CLI calls the server over gRPC.
- [x] Server calls AWS STS.
- [x] CLI receives temporary credentials.
- [x] Security invariants above are satisfied for implemented scope.
- [x] Technical contract acceptance criteria are satisfied.
- [x] Implementation decisions are followed.
- [x] Tests for request validation pass.
- [x] No database, dashboard, or YAML policy engine is added.

## Backlog

Pick these as separate future tasks. Do not implement them in this task.

- YAML policy config.
- In-memory audit events.
- Persistent audit/session storage.
- gRPC health service.
- Config validation command.
- SourceIdentity support.
- Session policy support.
- Role ARN lookup from config instead of fixed MVP mapping.
- Dashboard read model.
- Docker service packaging.
- Multi-account support.
- Multi-cloud provider support.

## AWS Bootstrap Runbook

This section records setup commands already used for the local AWS environment.

### 1. Confirm admin identity

```bash
aws sts get-caller-identity
```

Expected shape:

```json
{
  "UserId": "<AWS_USER_ID>",
  "Account": "<AWS_ACCOUNT_ID>",
  "Arn": "arn:aws:iam::<AWS_ACCOUNT_ID>:user/<ADMIN_SETUP_USER>"
}
```

### 2. Create broker user

```bash
aws iam create-user \
  --user-name <BROKER_USER_NAME>
```

### 3. Create broker assume-role policy

Policy file:

```text
cloud/broker-assume-policy.json
```

Create policy:

```bash
aws iam create-policy \
  --policy-name CloudIamBrokerAssumeJitRoles \
  --policy-document file://broker-assume-policy.json
```

Attach policy:

```bash
aws iam attach-user-policy \
  --user-name <BROKER_USER_NAME> \
  --policy-arn arn:aws:iam::<AWS_ACCOUNT_ID>:policy/CloudIamBrokerAssumeJitRoles
```

### 4. Create target role

Trust policy file:

```text
cloud/jit-readonly-trust-policy.json
```

Create role:

```bash
aws iam create-role \
  --path /jit/ \
  --role-name readonly \
  --assume-role-policy-document file://jit-readonly-trust-policy.json
```

### 5. Attach target role permissions

Permission policy file:

```text
cloud/jit-readonly-permissions.json
```

Attach inline policy:

```bash
aws iam put-role-policy \
  --role-name readonly \
  --policy-name JitReadonlyDemoPermissions \
  --policy-document file://jit-readonly-permissions.json
```

### 6. Create broker access key and local profile

```bash
aws iam create-access-key \
  --user-name <BROKER_USER_NAME>
```

```bash
aws configure --profile <BROKER_AWS_PROFILE>
```

Use:

```text
Default region name: us-east-1
Default output format: json
```

### 7. Verify broker identity

```bash
aws sts get-caller-identity \
  --profile <BROKER_AWS_PROFILE>
```

Expected ARN shape:

```text
arn:aws:iam::<AWS_ACCOUNT_ID>:user/<BROKER_USER_NAME>
```

### 8. Verify broker can assume target role

```bash
aws sts assume-role \
  --profile <BROKER_AWS_PROFILE> \
  --role-arn arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly \
  --role-session-name local-dev-test \
  --duration-seconds 900
```

Expected result:

```text
AWS returns temporary credentials with AccessKeyId, SecretAccessKey, SessionToken, and Expiration.
```
