---
title: "Access API pending response for TASK-000017"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000017
tags:
  - grpc
  - cli
  - approval
---

# Decision: Access API Pending Response

## Decision

Extend the access response so a request can return either:

- `issued`: includes AWS credential fields
- `pending_approval`: includes access request ID and expiration, but no AWS credential fields

The CLI must display pending approval as a successful request state, not as an internal server error.

## Rationale

Approval-required access is not a failure. The user should see that their request is waiting for admin review.

## Acceptance Criteria

- Pending response contains no credential values.
- Existing direct-issue behavior remains backward-compatible for targets that do not require approval.
- CLI summary output handles pending approval without suggesting credentials were issued.
