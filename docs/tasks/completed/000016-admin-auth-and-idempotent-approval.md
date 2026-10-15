---
title: "Console auth and idempotent resource approvals"
id: TASK-000016
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-08
completed_at: 2026-10-08
owner: codex
phase: "dashboard-security-1"
priority: P0
sprint_candidate: true
tags:
  - dashboard
  - console
  - auth
  - security
  - idempotency
  - resource-access
---

# Task: Console Auth And Idempotent Resource Approvals

## Outcome

Secure the console resource-approval path before building the dashboard UI: console operators authenticate, receive a short-lived signed console JWT, and approve/deny pending JIT resource access requests through idempotent HTTP APIs.

## Scope

- Add simple MVP console authentication for one local/demo operator.
- Issue a short-lived console JWT after successful console login.
- Require the console JWT on dashboard approval APIs.
- Require an `Idempotency-Key` UUIDv4 header for approve/deny mutations.
- Persist idempotency records so retries do not double-apply approvals or denials.
- Approve/deny by resource access request ID, not by raw device code, user code, or token hash.
- Record enough metadata to prove which console operator approved or denied a request.
- Keep raw console secrets, raw CloudIAM session tokens, raw device codes, and AWS credentials out of responses and logs.

## Not In Scope

- Multi-user console user management.
- Production SSO/OIDC.
- Password reset or email flows.
- Fine-grained RBAC.
- Dashboard frontend UI beyond minimal API smoke support.
- Editing policies, targets, or config.
- End-user device approval/login. Users approve their own devices as part of authentication; console approval is for resource/JIT access.

## Why Now

The resource approval flow is a privileged action. Before building UI affordances, the HTTP mutation path needs authentication, replay protection, and a clear audit trail. Device login approval remains a user self-service authentication step, while console approval governs resource/JIT access.

## Dependency Watch

This task depends on:

- `TASK-000017` for the durable resource access request object and statuses.
- `TASK-000023` for the immutable approved request permission boundary and claim-before-STS semantics.
- `TASK-000018` for the policy/config decision that marks an access request as approval-required.

Tasks 17, 23, and 18 are now complete. Implement this task against the durable access request object, immutable persisted role ARN metadata, claim-before-STS semantics, and target-level `approval_required` config.

## Technical Decisions

- [Console JWT session contract](../../technical-decisions/TASK-000016-admin-auth-and-idempotent-approval/001-admin-jwt-session-contract.md)
- [Idempotent approval contract](../../technical-decisions/TASK-000016-admin-auth-and-idempotent-approval/002-idempotent-approval-contract.md)
- [Approval security and audit boundary](../../technical-decisions/TASK-000016-admin-auth-and-idempotent-approval/003-approval-security-and-audit-boundary.md)
- [Approval state transition persistence](../../technical-decisions/TASK-000016-admin-auth-and-idempotent-approval/004-approval-state-transition-persistence.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000016-AUTH-001 | Console login returns a short-lived signed JWT only after valid credentials. | Handler/service tests cover valid and invalid login. |
| REQ-000016-AUTH-002 | Console APIs reject missing, expired, malformed, or invalid JWTs. | Handler tests cover auth failure modes. |
| REQ-000016-IDEMP-001 | Approve/deny APIs require `Idempotency-Key` as a UUIDv4 value. | Tests cover missing, malformed, and valid keys. |
| REQ-000016-IDEMP-002 | Retrying the same approve/deny request with the same key returns the original result without reapplying state changes. | Integration or store tests cover replay behavior. |
| REQ-000016-IDEMP-003 | Reusing an idempotency key with a different request payload is rejected as a conflict. | Tests cover fingerprint mismatch. |
| REQ-000016-APPROVE-001 | Console operator approves pending resource access request by ID. | API/store tests cover state transition. |
| REQ-000016-DENY-001 | Console operator denies pending resource access request by ID. | API/store tests cover state transition. |
| REQ-000016-TXN-001 | Approve/deny state transition and idempotency result are committed atomically. | Store transaction tests cover success, replay, and conflict paths. |
| REQ-000016-AUDIT-001 | Approval/denial records console operator identity and does not store raw tokens or credentials. | Tests assert metadata and secret absence. |

## Result

Console resource approval now has an MVP HTTP API and persistence boundary.

The original env-only admin credential and bearer-token response were superseded by `TASK-000024`, `TASK-000025`, and `TASK-000026`. The current implementation uses cookie-backed console sessions, persisted console users, and Argon2id password hashes.

Pending resource access requests can now be listed, approved, or denied by immutable access request ID. Approval and denial store the console operator subject on the request row, use the pinned persisted role ARN from the request object, and commit the state transition plus replayable idempotency record in one database transaction.

This task intentionally does not replace the existing user self-service device approval endpoint; console approval is only for resource/JIT access requests.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make server-test-integration`
