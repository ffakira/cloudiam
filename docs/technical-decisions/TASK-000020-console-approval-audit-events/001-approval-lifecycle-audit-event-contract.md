---
title: "Approval lifecycle audit event contract for TASK-000020"
type: decision
status: accepted
created_at: 2026-10-09
updated_at: 2026-10-09
owner: codex
applies_to:
  - TASK-000020
tags:
  - audit
  - approval
  - security
  - persistence
---

# Decision: Approval Lifecycle Audit Event Contract

## Context

Access audit events existed before console resource approvals had first-class lifecycle evidence. The dashboard could show recent audit history, but approval-required requests needed explicit events for request creation, operator approval, operator denial, and expiration.

The accepted `TASK-000016` audit boundary requires console operator identity and a safe idempotency-key reference for approval/denial records. The implementation must keep raw JWTs, raw idempotency keys, CloudIAM session tokens, raw device/user codes, and AWS credentials out of audit storage.

## Decision

Extend the existing `audit_events` table and audit model instead of creating a separate approval-audit table.

Add event types:

- `approval_requested`
- `approval_approved`
- `approval_denied`
- `approval_expired`

Add approval metadata fields:

- `access_request_id`
- `operator_subject`
- `idempotency_key_hash`

Use `decision = pending` for `approval_requested`, `decision = allowed` for `approval_approved`, and `decision = denied` for `approval_denied` and `approval_expired`.

Store a SHA-256 hex digest of the idempotency key for console approval mutations. Do not store the raw idempotency key.

## Rationale

Keeping approval lifecycle events in `audit_events` preserves one local/demo read path for the dashboard and operators. The access request row remains the source of immutable reviewed target metadata, while audit events become the append-only timeline.

The schema already carries provider/account/role metadata. That is enough to make the current AWS-only MVP traceable and keeps future multi-account work from needing to reinterpret older events.

## Consequences

- Dashboard and `/audit/events` consumers can inspect access and approval lifecycle events from the same endpoint.
- Approval audit rows contain operator and retry-reference metadata without raw secrets.
- Multi-account support can reuse the existing account metadata field, but `TASK-000011` still needs expanded config, tests, and runtime behavior.
- Multi-cloud providers remain deferred under `TASK-000012`; this decision only preserves provider metadata in audit history.
