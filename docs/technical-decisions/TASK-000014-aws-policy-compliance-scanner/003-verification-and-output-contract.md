---
title: "Verification and output contract for TASK-000014"
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
  - testing
  - output
---

# Decision: Verification And Output Contract

## Sources

- [AWS IAM: Policy evaluation logic](https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_evaluation-logic.html)
- [AWS IAM API: SimulatePrincipalPolicy](https://docs.aws.amazon.com/IAM/latest/APIReference/API_SimulatePrincipalPolicy.html)
- [AWS IAM: Pass session tags in AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_session-tags.html)

## Decision

Scanner output must be explainable, stable, and non-secret.

The default output should be human-readable and grouped by configured target:

- target name
- role ARN with account ID masked in docs/tests
- overall status: compliant or non-compliant
- finding IDs
- short remediation hints

Finding IDs must be stable so task requirements, tests, and future dashboard views can reference the same behavior. Initial finding IDs:

- `AWS_TRUST_MISSING_ASSUME_ROLE`
- `AWS_TRUST_MISSING_SET_SOURCE_IDENTITY`
- `AWS_TRUST_SOURCE_IDENTITY_CONDITION`
- `AWS_TRUST_MISSING_TAG_SESSION`
- `AWS_TRUST_TAG_KEYS`
- `AWS_TRUST_PROVIDER_TAG`
- `AWS_TRUST_REQUIRED_REQUEST_TAG`
- `AWS_TRUST_TRANSITIVE_TAGS`
- `AWS_TRUST_COMBINED_SOURCE_AND_TAG_CONDITIONS`

The scanner must not print AWS access keys, secret access keys, session tokens, CloudIAM session tokens, raw device codes, or full temporary credential exports.

## Verification

Use fixture-driven unit tests for:

- compliant split-statement trust policy
- missing `sts:SetSourceIdentity`
- missing `sts:TagSession`
- incorrect `sts:SourceIdentity` condition
- extra or missing `aws:TagKeys`
- wrong `cloudiam_provider`
- transitive tag allowance
- combined SourceIdentity and TagSession conditions that can fail live AWS authorization

Live AWS verification is optional for Task 14 and must not be required for unit tests.

## Acceptance Criteria

- Scanner findings are deterministic from policy fixtures.
- Golden-output tests prove remediation output is stable.
- Output tests prove secrets and temporary credentials are not printed.
- The known Task 13 live failure mode is covered by a fixture.
