---
title: "Hot reload and immutable request boundary for TASK-000018"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000018
tags:
  - hot-reload
  - approval
  - configuration
  - security
---

# Decision: Hot Reload And Immutable Request Boundary

## Context

`TASK-000010` established immutable config snapshots for hot reload. `TASK-000023` established that an approved access request issues from persisted request metadata, including the pinned role ARN, rather than from the latest config snapshot.

Approval configuration is governance config and should be reloadable.

## Decision

Treat `approval_required` as reloadable governance config.

Candidate reload behavior:

- A valid reload can turn approval on for future requests to a target.
- A valid reload can turn approval off for future requests to a target.
- In-flight requests continue using the snapshot they loaded.
- Already-created pending or approved access requests keep their persisted target metadata and pinned role ARN.
- Issuing an already-approved request must not re-resolve the role ARN from the current config.

## Rationale

Operators need to tighten or loosen approval requirements without restarting the demo service. At the same time, approved requests must remain immutable approval objects so hot reload cannot change the permission boundary after review.

## Consequences

- Reload tests should cover changing `approval_required` and verify subsequent requests observe the new behavior.
- Existing `TASK-000010` snapshot tests remain relevant: one request must not mix authorization from one snapshot with target approval metadata from another.
- Task 16 console approval APIs should continue approving a request ID, not a target name rebuilt from current config.
