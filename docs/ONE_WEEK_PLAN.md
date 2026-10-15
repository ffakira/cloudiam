# One-Week Execution Plan

## Goal

By the end of the week, demonstrate a usable AWS-only CloudIAM MVP:

```text
CLI login -> user self-approves device -> short-lived CloudIAM session -> named access target -> policy check -> optional admin resource approval -> AWS STS credentials
```

The demo should prove:

- users do not pass AWS provider/account/role internals through the CLI
- the server resolves access from YAML governance config
- authorization happens before AWS is called
- AWS credentials are temporary and not persisted
- the CLI behaves like a hosted-service client
- users can approve their own device login from a local/demo page
- an admin can approve or deny pending resource/JIT access requests from a local/demo dashboard
- recent access decisions are visible from durable audit history

## Completed Foundation

- `TASK-000001`: CLI to gRPC to AWS STS proof.
- `TASK-000002`: YAML config, named targets, policy checks, and legacy request cleanup.
- `TASK-000003`: CLI device-code login and one-hour CloudIAM session.
- `TASK-000004`: Config validation command.
- `TASK-000005`: Persistent audit events.
- `TASK-000006`: Docker service packaging.
- `TASK-000007`: AWS SourceIdentity and session tags.
- `TASK-000008`: Persistent audit and session storage baseline.
- `TASK-000010`: Config hot reload.
- `TASK-000013`: AWS IAM condition hardening.
- `TASK-000014`: AWS policy compliance scanner.
- `TASK-000015`: CLI credential output safety.

## Active Task

- `TASK-000028`: Security hardening.

## Sprint Candidate Backlog

1. `TASK-000029`: Edge-case audit.
2. `TASK-000027`: Production hardening.

## Deferred This Sprint

These are explicitly out of the one-week scope:

- multi-cloud providers
- production admin identity/RBAC
- dashboard config editing
- multi-account expansion until the completed MVP is hardened

## Daily Shape

### Day 1-2: Login Contract And Local Session

- Define the CLI login/session contract.
- Decide where the local one-hour session token lives.
- Keep endpoint flags out of user-facing commands.

### Day 3-4: Login Implementation Slice

- Implement `cloudiam login` equivalent in the current CLI.
- Generate or simulate device code for local MVP.
- Store a short-lived local CloudIAM session.
- Attach the session to access requests.

### Day 5: Security Review And Tests

- Expired session is rejected.
- Missing session is rejected.
- Access requests do not accept provider/account/role.
- Token storage does not include AWS credentials.

### Day 6: Secure Admin Resource Approval

- Add admin login and short-lived JWT verification.
- Require UUIDv4 idempotency keys for approve/deny.
- Approve/deny pending resource/JIT access requests by ID.
- Persist replay-safe idempotency records.

### Day 7: MVP Dashboard And Demo Polish

- Serve a small admin dashboard from the HTTP server.
- Add user self-service device-login APIs.
- Add user self-service device approval.
- Use the secured admin approve/deny actions for resource requests.
- Show recent audit events.
- Update README commands.
- Run the end-to-end demo.
- Demo the full loop: CLI login waits, user approves device, CLI receives session, resource request waits if approval is required, admin approves resource access, access request returns temporary AWS credentials.
- Close completed task.
- Promote the next backlog item deliberately.
