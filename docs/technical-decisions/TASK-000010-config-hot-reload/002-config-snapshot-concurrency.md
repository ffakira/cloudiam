---
title: "Config snapshot concurrency contract for TASK-000010"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000010
tags:
  - configuration
  - concurrency
  - atomicity
---

# Decision: Config Snapshot Concurrency Contract

## Decision

Use immutable config snapshots and atomic replacement.

The server must never mutate the active config in place. Reload builds a complete candidate snapshot, validates it, prepares any derived policy/target state, and then swaps the active pointer in one step.

Access requests must use a single snapshot for the full authorization and target-resolution path. A request must not authorize against one config version and resolve the target against another config version.

Implementation expectation:

- introduce a small runtime config store
- store the current config snapshot behind `atomic.Value` or an equivalent lock-protected pointer
- expose a method that returns the current snapshot for a request
- ensure authorizer and target resolver read from the same snapshot

## Edge Cases

- Reload happens while a request is authorizing: the request continues with the snapshot it loaded.
- Reload happens while a request is assuming the AWS role: the AWS call continues with the already-resolved role ARN.
- Reload removes a target: new requests fail policy/target checks, existing requests continue with their snapshot.
- Reload changes a role ARN: new requests use the new ARN, existing requests continue with the old resolved ARN.
- Reload changes policy membership: new requests use the new policy, existing requests keep the old decision path.

## Non-Goals

This task does not implement transactional cancellation of in-flight AWS STS calls. Temporary credentials already have bounded duration and audit events record the target metadata used for the request.

## Acceptance Criteria

- No request can mix authorization from one config with target resolution from another.
- The reload path does not mutate maps referenced by in-flight requests.
- Concurrency tests cover request/reload overlap.
