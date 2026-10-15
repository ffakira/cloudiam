---
title: "Approved access issuance hardening"
id: TASK-000023
type: hardening
status: completed
created_at: 2026-10-08
updated_at: 2026-10-08
completed_at: 2026-10-08
owner: codex
phase: "approval-1"
priority: P0
sprint_candidate: true
tags:
  - approval
  - security
  - persistence
  - hot-reload
  - jit
---

# Task: Approved Access Issuance Hardening

## Outcome

Make approved resource access issuance immutable and review-safe before approval policy configuration starts driving real pending requests.

## Scope

- Persist the resolved AWS role ARN on each resource access request.
- Add this as a forward migration after `000003_access_requests`; do not rely only on editing the original access request table migration.
- Issue approved access from the claimed access request data, not from a freshly resolved hot-reload snapshot.
- Keep raw AWS credentials, CloudIAM session tokens, console JWTs, and device/user codes out of access request persistence.
- Refactor duplicate direct issuance paths through a shared helper where it reduces drift.
- Document the current claim-before-STS semantics so retries and future `TASK-000016` approval APIs do not assume a request remains reusable after a failed STS call.
- Add regression coverage for a hot-reload role ARN change between pending request creation and approved issuance.

## Not In Scope

- Approval policy YAML fields; covered by `TASK-000018`.
- Console approval/denial HTTP APIs; covered by `TASK-000016`.
- Dashboard UI.
- Changing the MVP to a multi-step approval or retry workflow.
- Idempotency record retention; covered by `TASK-000021`.

## Why Now

`TASK-000017` created the durable resource access request lifecycle, and `TASK-000018` will make approval-required decisions configurable. Before configurable approvals create real pending requests, the approved request must freeze the exact AWS permission boundary that was reviewed. The AWS role ARN is not a secret, and it is part of the approved target identity.

## Dependency Watch

This task is a prerequisite for:

- `TASK-000018`, because approval configuration should rely on immutable approved request issuance.
- `TASK-000016`, because console approval APIs should approve a frozen request object by ID.

This task should update the `TASK-000017` technical decisions where the persistence and issuance boundary changed after review.

## Technical Decisions

- [Approved request permission boundary](../../technical-decisions/TASK-000023-approved-access-issuance-hardening/001-approved-request-permission-boundary.md)
- [Claim before STS issuance semantics](../../technical-decisions/TASK-000023-approved-access-issuance-hardening/002-claim-before-sts-semantics.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000023-ARN-001 | Resource access requests persist the resolved AWS role ARN at creation time. | New forward migration, schema check, and store tests cover `role_arn`. |
| REQ-000023-ISSUE-001 | Approved issuance uses the claimed request's persisted role ARN. | Service test covers config changing to a different ARN before issuance. |
| REQ-000023-SECRET-001 | Access request persistence still contains no raw AWS credential values or raw auth tokens. | Existing secret-boundary tests remain green and include `role_arn` as allowed metadata. |
| REQ-000023-SEMANTICS-001 | Claim-before-STS behavior is documented as an MVP concurrency choice. | Technical decision records the retry tradeoff. |
| REQ-000023-DRIFT-001 | Direct issuance paths share the same issue/audit helper where practical. | Unit tests remain green and review confirms reduced duplication. |

## Result

Approved resource access requests now pin the resolved AWS role ARN as non-secret permission-boundary metadata.

The access request store persists and returns `role_arn`, schema readiness checks require the column, and approved issuance now builds the AWS STS request from the claimed access request instead of the latest hot-reload target snapshot. New access requests require a role ARN at the store boundary, while the database migration avoids a hard check constraint so old pending rows are not broken by post-migration status updates.

Direct issuance paths share the same issue/audit helper to reduce drift.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make server-test-integration`
