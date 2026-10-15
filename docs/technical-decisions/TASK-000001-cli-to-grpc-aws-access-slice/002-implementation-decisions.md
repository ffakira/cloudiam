---
title: "Implementation decisions for TASK-000001"
type: decision
status: accepted
created_at: 2026-10-06
updated_at: 2026-10-06
owner: codex
applies_to:
  - TASK-000001
tags:
  - go
  - grpc
  - protobuf
  - aws
  - cli
---

# Decision: Implementation Decisions For TASK-000001

## Related Work

- `TASK-000001`: CLI to gRPC AWS access slice.
- [001-contract.md](001-contract.md)

## Sources

- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers Go Generated Code Guide](https://protobuf.dev/reference/go/go-generated/)
- [AWS SDK for Go v2 documentation](https://docs.aws.amazon.com/sdk-for-go/)
- [AWS STS Go SDK package documentation](https://docs.aws.amazon.com/sdk-for-go/api/service/sts/)

## Decisions

### DEC-000001-IMPL-001: Go module path

Use:

```text
module github.com/ffakira/cloudiam/server
```

Reason:

- The repository root is `/home/akira/segura`.
- The Go module is scoped to `server/`.
- The canonical repository path is `github.com/ffakira/cloudiam`.

### DEC-000001-IMPL-002: gRPC bind address

The server listens on:

```text
127.0.0.1:50051
```

The CLI connects to:

```text
127.0.0.1:50051
```

Both may be overridden by flags:

```text
access-server --addr 127.0.0.1:50051
access-cli --addr 127.0.0.1:50051 request-access ...
```

Reason:

- This task is local-only.
- Binding to loopback avoids exposing the unauthenticated MVP server to the network.

### DEC-000001-IMPL-003: CLI library

Use the Go standard library `flag` package for this task.

Do not add Cobra yet.

Reason:

- The CLI has one command.
- Avoid extra dependencies until the command tree is real.
- Cobra can be introduced when we add multiple command groups.

### DEC-000001-IMPL-004: protobuf generation

Use `protoc` with:

```text
protoc-gen-go
protoc-gen-go-grpc
```

Generated code goes under:

```text
server/gen/access/v1
```

`access.proto` must set:

```proto
option go_package = "github.com/ffakira/cloudiam/server/gen/access/v1;accessv1";
```

Generation command:

```bash
protoc \
  --proto_path=api/proto \
  --go_out=. \
  --go_opt=module=github.com/ffakira/cloudiam/server \
  --go-grpc_out=. \
  --go-grpc_opt=module=github.com/ffakira/cloudiam/server \
  api/proto/access/v1/access.proto
```

Reason:

- The official Go protobuf generator requires a Go import path.
- The `module` option strips the module prefix so generated files land under `gen/access/v1`.

### DEC-000001-IMPL-005: AWS SDK

Use AWS SDK for Go v2.

The AWS adapter must load configuration with the default credential chain.

Local execution uses:

```bash
AWS_PROFILE=<BROKER_AWS_PROFILE> go run ./cmd/access-server
```

Reason:

- The broker profile is already verified manually.
- The CLI must not receive broker credentials.
- The server owns AWS STS calls.

### DEC-000001-IMPL-006: target role configuration

For this task, pass the target role ARN to the server as a required flag:

```bash
access-server \
  --addr 127.0.0.1:50051 \
  --role-arn arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly
```

Do not add YAML config yet.

Reason:

- The task proves the vertical slice.
- YAML policy config is backlog.
- A required flag is explicit and easy to verify.

### DEC-000001-IMPL-007: request duration

The CLI sends duration in seconds.

The server rejects:

```text
duration_seconds <= 0
```

For manual testing, use:

```text
900
```

Reason:

- AWS STS accepts duration in seconds.
- This avoids duration string parsing in the gRPC contract.

### DEC-000001-IMPL-008: deadlines

Use explicit timeouts:

- CLI gRPC call timeout: `10s`
- AWS STS call timeout: `10s`

Reason:

- External calls must not hang indefinitely.
- Cancellation should propagate from CLI to gRPC handler to AWS adapter.

### DEC-000001-IMPL-009: credential output

The CLI prints shell exports:

```bash
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...
export AWS_SESSION_EXPIRATION=...
```

The server must not log these values.

Reason:

- The purpose of the MVP is to hand short-lived credentials to the user.
- Shell exports make the result immediately testable.

## Acceptance Criteria

- The implementation follows all decisions above.
- The active task requirements reference these decisions when behavior would otherwise be ambiguous.
- Any deviation must update this file before code is changed.
