---
title: "Resource access request lifecycle"
id: TASK-000017
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "approval-1"
priority: P0
sprint_candidate: true
tags:
  - access
  - approval
  - jit
  - persistence
---

# Task: Resource Access Request Lifecycle

## Outcome

Create a durable resource/JIT access request object when an access request requires console approval, and issue AWS credentials only after that request is approved.

## Scope

- Add a persistent access request table/model.
- Store requester principal, target, requested duration, policy decision context, status, expiration, and timestamps.
- Return a pending approval response to the CLI when approval is required.
- Allow approved requests to continue to AWS STS issuance.
- Expire stale pending access requests.
- Keep AWS credentials out of access request persistence.

## Not In Scope

- Console JWT/auth implementation; covered by `TASK-000016`.
- Approval policy configuration; covered by `TASK-000018`.
- Dashboard UI; covered by `TASK-000009`.

## Technical Decisions

- [Access request lifecycle contract](../../technical-decisions/TASK-000017-resource-access-request-lifecycle/001-access-request-lifecycle-contract.md)
- [Persistence schema and secret boundary](../../technical-decisions/TASK-000017-resource-access-request-lifecycle/002-persistence-schema-and-secret-boundary.md)
- [Access API pending response](../../technical-decisions/TASK-000017-resource-access-request-lifecycle/003-access-api-pending-response.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000017-CREATE-001 | Approval-required access creates a pending resource access request. | Service/store tests cover creation. |
| REQ-000017-ISSUE-001 | Credentials are issued only after the matching request is approved. | Access flow tests cover approved vs pending. |
| REQ-000017-EXPIRE-001 | Expired pending requests cannot be approved or used to issue credentials. | Tests cover expiration. |
| REQ-000017-SECRET-001 | Access request persistence contains no AWS credential values. | Schema/code tests assert secret absence. |

## Result

Added durable resource access requests for approval-required JIT access.

The access service can now return either issued credentials or a `pending_approval` result with an access request ID and expiration. When an approved access request exists for the same principal/target/duration, the service atomically claims it as `issued` before calling AWS STS so the approved request is one-shot.

Postgres persistence now includes `access_requests`, schema readiness checks include the new table, and the CLI/gRPC response shape can represent pending approval without returning credential fields.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make server-test-integration`
