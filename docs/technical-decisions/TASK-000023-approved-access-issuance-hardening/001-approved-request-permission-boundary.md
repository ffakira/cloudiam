---
title: "Approved request permission boundary for TASK-000023"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000023
tags:
  - approval
  - security
  - persistence
  - hot-reload
---

# Decision: Approved Request Permission Boundary

## Decision

Persist the resolved AWS role ARN on each resource access request when the pending request is created.

When an approved request is later claimed for issuance, the access service must issue credentials from the claimed request's persisted target metadata, including the persisted role ARN. It must not substitute a newly resolved role ARN from the current hot-reload config snapshot.

## Rationale

Console approval is for a concrete resource access request. The role ARN is not secret; it is the AWS permission boundary that will be assumed. If config hot reload changes the role ARN behind the same logical target/role name after approval, issuing from the latest snapshot could grant a different permission boundary than the one represented by the approved request.

## Consequences

- Access request persistence stores one additional non-secret metadata field: `role_arn`.
- Hot reload can still change future requests.
- Already-created pending or approved requests remain tied to the role ARN resolved at request creation time.
- Task 16 approval APIs should approve or deny by access request ID and should not rebuild the approval object from current config.
