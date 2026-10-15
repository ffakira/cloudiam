---
title: "Verification and rollout contract for TASK-000013"
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
  - verification
  - rollout
---

# Decision: Verification And Rollout Contract

## Sources

- [AWS STS service authorization reference](https://docs.aws.amazon.com/service-authorization/latest/reference/list_sts.html)
- [AWS IAM: Pass session tags in AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_session-tags.html)

## Decision

Keep the rollout explicit and simulator-first.

Verification order:

1. Update local templates.
2. Validate JSON with `jq`.
3. Run unit tests.
4. Apply policies with `aws-task13-apply.sh`.
5. Run IAM simulator for `sts:AssumeRole`, `sts:SetSourceIdentity`, and `sts:TagSession`.
6. Run the live CLI access request and discard temporary AWS credential output.

The task must include a debug script so failures can be inspected without retyping long IAM commands.

## Acceptance Criteria

- Root-level apply/debug scripts exist for Task 13.
- Scripts do not print AWS secrets or temporary credentials.
- Documentation tells the operator to expect AccessDenied when required conditions do not match.
