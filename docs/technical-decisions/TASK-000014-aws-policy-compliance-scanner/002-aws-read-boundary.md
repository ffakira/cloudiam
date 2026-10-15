---
title: "AWS read boundary for TASK-000014"
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
  - least-privilege
  - read-only
---

# Decision: AWS Read Boundary

## Sources

- [AWS IAM API: GetRole](https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetRole.html)
- [AWS IAM API: SimulatePrincipalPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_SimulatePrincipalPolicy.html)
- [AWS IAM: How to simulate policies](https://docs.aws.amazon.com/IAM/latest/UserGuide/policies_policy-simulator-how-to.html)

## Decision

Task 14 must not mutate AWS state.

The first scanner implementation should require only:

- `iam:GetRole` for each configured target role

Do not require `iam:UpdateAssumeRolePolicy`, `iam:AttachUserPolicy`, `iam:CreatePolicyVersion`, or any other write API.

Policy simulator support is optional for this task. If included, it must be a secondary verification mode, not the scanner's only source of truth. The local trust-policy parser remains required because it gives deterministic tests and clear remediation output without calling AWS.

If simulator support is added, use read-only simulator APIs only:

- `iam:SimulatePrincipalPolicy`
- optionally `iam:GetContextKeysForPrincipalPolicy`

## Acceptance Criteria

- The default scanner code path is read-only.
- Unit tests cover policy evaluation without AWS credentials.
- Any live AWS check is explicitly labeled as online verification.
- Documentation lists the exact AWS read permissions needed.
