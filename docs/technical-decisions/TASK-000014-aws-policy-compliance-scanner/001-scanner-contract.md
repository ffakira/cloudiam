---
title: "Scanner contract for TASK-000014"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000014
tags:
  - aws
  - iam
  - scanner
  - compliance
---

# Decision: Scanner Contract

## Sources

- [AWS IAM API: GetRole](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetRole.html)
- [AWS IAM: Policy evaluation logic](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_evaluation-logic.html)
- [AWS STS service authorization reference](https://docs.aws.amazon.com/service-authorization/latest/reference/list_sts.html)

## Decision

Build the scanner as a read-only compliance check for configured CloudIAM AWS targets.

For this task, the scanner checks only the target role trust policy returned by `iam:GetRole`. It does not try to prove the full effective permission graph for a customer account.

The scanner must validate the CloudIAM-managed role contract from `TASK-000013`:

- trust policy has an allow statement for the configured broker principal
- `sts:AssumeRole` is allowed
- `sts:SetSourceIdentity` is allowed with `sts:SourceIdentity` matching `cloudiam-*`
- `sts:TagSession` is allowed in a separate statement from SourceIdentity conditions
- `aws:TagKeys` is restricted to the CloudIAM tag set
- `aws:RequestTag/cloudiam_provider` equals `aws`
- required CloudIAM request tag keys are present
- `sts:TransitiveTagKeys` is absent for the supported path

Do not scan arbitrary AWS roles in this task. The scanner follows the existing YAML target configuration, because that is the MVP boundary we already operate.

## Acceptance Criteria

- Scanner can evaluate a trust-policy document without calling AWS.
- Scanner can load configured AWS target role ARNs.
- Scanner reports pass/fail per configured role.
- Scanner result includes stable finding IDs that can be tested and documented.
