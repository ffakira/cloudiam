# Project State

## What We Are Building

An AWS-only Cloud IAM broker MVP.

The first useful flow is:

```text
engineer uses CLI
-> CLI starts device login
-> user approves their own device and receives a one-hour CloudIAM session
-> CLI calls gRPC server with a named access target and CloudIAM session
-> server resolves the session principal and target from config/database
-> server authorizes principal against config policy
-> if required, a console operator approves the resource/JIT access request
-> server calls AWS STS AssumeRole using broker credentials
-> CLI receives short-lived AWS credentials
```

## What This Is Not Yet

- Not a full production dashboard product.
- Not a multi-cloud system.
- Not a policy engine.
- Not a clone of any vendor product.

## Last Completed Task

[TASK-000027: Production hardening](tasks/completed/000027-production-hardening.md)

## Active Task

None.

## Planned Follow-Up Task

None. `TASK-000011` multi-account support and `TASK-000012` multi-cloud providers remain deferred expansion work.

## Security Baseline

Security requirements for the MVP are source-backed and tracked in:

[TASK-000001 technical contract](technical-decisions/TASK-000001-cli-to-grpc-aws-access-slice/001-contract.md)

## AWS Status

AWS has been manually bootstrapped locally:

- AWS CLI works.
- Broker IAM user/profile exists.
- Target JIT role exists.
- Manual `aws sts assume-role` works.

Sensitive AWS identifiers are masked in docs. The local `cloud/` folder contains policy templates with placeholders.

## Implementation Direction

Use hexagonal architecture with explicit adapters for the first server implementation:

```text
server/
  api/proto/access/v1/
  cmd/
    access-server/
    access-cli/
  gen/access/v1/
  internal/
    core/access/
    adapters/aws/
    adapters/grpc/
    adapters/cli/
    app/server/
    app/cli/
```

Core rule:

```text
internal/core/access must not import AWS SDK, gRPC, CLI, or protobuf packages.
```

Adapter boundaries:

```text
internal/adapters/aws  -> outbound AWS STS adapter
internal/adapters/grpc -> inbound gRPC adapter
internal/adapters/cli  -> inbound CLI adapter/client
```

## Current Completed Baseline

The MVP now has:

- CLI to gRPC AWS access requests.
- YAML named access targets and policy validation.
- CLI device-code login with persisted sessions.
- Persistent audit events.
- Docker service packaging for local demo/runtime use.
- AWS SourceIdentity and CloudIAM session tags.
- AWS managed-role compliance scanning.
- Safer default CLI credential output.
- Runtime YAML governance config reload through `SIGHUP`.
- Immutable approved resource access requests that pin the resolved AWS role ARN.
- Target-level `approval_required` config for JIT approval gating.
- Console JWT authentication for resource approval APIs.
- Idempotent console approve/deny mutations for pending resource access requests.
- Cookie-backed console dashboard sessions with Zustand-managed safe metadata.
- Persisted console users with Argon2id password hashes and local/demo startup seeding.
- Core dashboard auth and JIT approval logic split into `consoleauth` and `approval` packages.
- MVP local/demo dashboard served by the Go HTTP server with self-service device approval, console-authenticated resource approval/denial, pending request read model, and recent audit event inspection.
- First-class approval lifecycle audit events for requested, approved, denied, and expired resource access requests.
- Startup cleanup for expired console approval idempotency records after their 24-hour replay window.
- Browser-based device-login approval through `/device` and `/api/device/approve`; raw `POST /device/approve` is deprecated.
- Edge-case coverage for browser device approval success, invalid, expired, invalid-state, deprecated-route, and secret-absence cases.
- Security hardening closure for auth boundaries, secret handling, audit coverage, replay protections, rate limiting, browser secret storage, concurrency/context invariants, scanner tooling, and residual-risk reporting.
- Edge-case audit closure for CLI, gRPC, HTTP, dashboard state, persistence transitions, config reload/startup/shutdown, AWS STS issuance, final server security gates, client build gates, and residual-risk reporting.
- Production hardening closure for startup validation, readiness, lifecycle behavior, observability mapping, production runbook coverage, final verification gates, and residual-risk reporting.

## Backlog Recommendation

No immediate hardening follow-up is promoted.

Keep `TASK-000011` and `TASK-000012` deferred unless the product direction explicitly shifts from AWS-only MVP hardening into account or provider expansion.
