---
title: "Session tag condition contract for TASK-000013"
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
  - session-tags
  - hardening
---

# Decision: Session Tag Condition Contract

## Sources

- [AWS IAM: Pass session tags in AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_session-tags.html)
- [AWS STS service authorization reference](https://docs.aws.amazon.com/service-authorization/latest/reference/list_sts.html)
- [AWS IAM: Controlling access using tags](https://docs.aws.amazon.com/IAM/latest/UserGuide/access_tags.html)
- [AWS IAM: Single-valued vs. multivalued context keys](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_condition-single-vs-multi-valued-context-keys.html)

## Decision

Constrain session tags to the CloudIAM tag set introduced in `TASK-000007`.

Allowed tag keys:

- `cloudiam_principal`
- `cloudiam_target`
- `cloudiam_provider`
- `cloudiam_account`
- `cloudiam_role`

The trust policy should keep session-tag authorization in a separate statement from `sts:AssumeRole` / `sts:SetSourceIdentity`, because AWS evaluates `sts:TagSession` as its own permission when tags are passed.

The session-tag statement should:

- require these keys through `aws:TagKeys`
- restrict `aws:RequestTag/cloudiam_provider` to `aws`
- prevent transitive tags by requiring `sts:TransitiveTagKeys` to be absent or empty

Do not enforce exact values for `cloudiam_principal`, `cloudiam_target`, `cloudiam_account`, or `cloudiam_role` yet. Those values are still local MVP names and will become stronger once tenancy and account models settle.

## Acceptance Criteria

- Trust policy restricts accepted tag keys.
- Trust policy restricts provider tag to `aws`.
- CloudIAM continues to send no transitive tags.
- IAM simulator allows the happy path.
