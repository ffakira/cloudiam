---
title: "SourceIdentity condition contract for TASK-000013"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000013
tags:
  - aws
  - iam
  - sourceidentity
  - hardening
---

# Decision: SourceIdentity Condition Contract

## Sources

- [AWS IAM: Monitor and control actions taken with assumed roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_control-access_monitor.html)
- [AWS STS API: AssumeRole](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
- [AWS STS service authorization reference](https://docs.aws.amazon.com/service-authorization/latest/reference/list_sts.html)

## Decision

Require `sts:SourceIdentity` on broker-to-JIT role assumptions.

The trust policy must only allow source identities that match CloudIAM's current sanitized principal format:

```json
"StringLike": {
  "sts:SourceIdentity": "cloudiam-*"
}
```

To make this useful, CloudIAM must prefix generated source identities with `cloudiam-`.

Keep `sts:AssumeRole` / `sts:SetSourceIdentity` authorization in a separate trust-policy statement from `sts:TagSession` conditions. AWS evaluates these permissions separately during one AssumeRole call, and session-tag conditions should not gate SourceIdentity authorization.

This task does not bind the source identity to a specific named principal in IAM. Exact per-principal enforcement belongs in a later tenancy/principal model, not in this MVP hardening slice.

## Acceptance Criteria

- CloudIAM-generated SourceIdentity starts with `cloudiam-`.
- Trust policy contains a `sts:SourceIdentity` condition.
- Unit tests assert the SourceIdentity prefix and sanitization behavior.
- Live AWS assume-role still succeeds after policy update.
