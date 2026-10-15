---
title: "Approval state transition persistence for TASK-000016"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000016
tags:
  - approval
  - persistence
  - state-machine
  - idempotency
  - transactions
---

# Decision: Approval State Transition Persistence

## Context

`TASK-000017` introduced durable resource access request statuses. `TASK-000023` made the access request an immutable approval object by pinning the resolved `role_arn`. `TASK-000018` made `approval_required` target-level config that controls future request creation only.

`TASK-000016` now needs concrete approve/deny persistence semantics.

## Decision

Store admin approve/deny state on the existing `access_requests` row for the MVP.

Add admin metadata fields directly to `access_requests`:

- `approved_by`
- `denied_by`

The existing timestamp fields remain the status timestamps:

- `approved_at`
- `denied_at`

Persist idempotency separately in an `idempotency_records` table. The idempotency record stores replay/conflict metadata and references the access request ID; it does not duplicate access request metadata.

Approve/deny must run in one database transaction:

1. Validate or reserve the idempotency key for the console mutation.
2. Lock the target `access_requests` row.
3. Verify the access request is still `pending_approval`.
4. Verify `expires_at` is in the future.
5. Update status to `approved` or `denied`.
6. Write admin metadata and transition timestamp.
7. Store the replayable idempotency result.
8. Commit.

## State Transitions

Console mutations may perform only these transitions:

```text
pending_approval -> approved
pending_approval -> denied
```

Console mutations must not transition:

```text
approved -> denied
denied -> approved
issued -> approved
issued -> denied
expired -> approved
expired -> denied
```

Expired pending access requests cannot be approved or denied. The implementation may either mark the row `expired` during the failed mutation attempt or return a clear conflict response while leaving cleanup to a later task. The response must make clear that no approval was applied.

## Idempotency Result Shape

The idempotency record should store a stable replay result for approve/deny:

- HTTP status
- action
- access request ID
- resulting access request status
- console operator subject
- transition timestamp, when applied
- conflict/error category, when not applied

It must not store raw JWTs, raw passwords, CloudIAM session tokens, device/user codes, or AWS credential values.

## Rationale

The access request row is already the durable approval object and contains the immutable target metadata needed for review. Storing admin transition metadata on the same row keeps the MVP simple and makes listing pending/approved/denied requests straightforward.

Separate idempotency storage keeps retry semantics independent from the domain row and avoids overloading access request columns with replay concerns.

The single transaction prevents double approval, approve/deny races, and idempotency records that disagree with the applied state.

## Consequences

- `TASK-000020` added richer append-only approval audit events without changing the MVP transition contract.
- Dashboard list/read APIs can source basic approval state directly from `access_requests`.
- `TASK-000021` added startup cleanup for expired idempotency records after the replay window.
- Tests should cover replay, fingerprint conflict, approve/deny race safety, expired request conflict, and invalid terminal-state mutations.
