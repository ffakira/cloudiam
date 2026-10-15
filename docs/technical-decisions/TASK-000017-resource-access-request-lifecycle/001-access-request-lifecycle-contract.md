---
title: "Access request lifecycle contract for TASK-000017"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000017
tags:
  - access
  - approval
  - lifecycle
---

# Decision: Access Request Lifecycle Contract

## Decision

Introduce a durable resource access request lifecycle.

Statuses:

- `pending_approval`
- `approved`
- `denied`
- `expired`
- `issued`

When an authorized access request requires approval, the access service creates a `pending_approval` resource access request and does not call AWS STS.

When a resource access request is approved, a later access flow can issue credentials from the approved request and then mark it `issued`.

## Rationale

Console approval needs a durable object to approve. Device authorization is the wrong object: it proves a user authenticated a CLI session, while resource access requests represent JIT access to a governed target.

## Task 16 Watch

Task 16 approval APIs should approve or deny this access request ID. Its idempotency fingerprint should include action, access request ID, admin subject, and any approval payload.
