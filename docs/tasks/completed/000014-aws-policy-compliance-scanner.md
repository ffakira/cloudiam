---
title: "AWS policy compliance scanner"
id: TASK-000014
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "governance-hardening"
priority: P1
sprint_candidate: true
tags:
  - aws
  - iam
  - onboarding
  - compliance
---

# Task: AWS Policy Compliance Scanner

## Outcome

Detect whether customer-provided AWS roles satisfy the CloudIAM managed-role trust-policy contract.

## Scope

- Fetch role trust policy for configured AWS roles.
- Check required STS actions.
- Check required SourceIdentity conditions.
- Check required session tag conditions.
- Produce clear, non-secret remediation output.

## Not In Scope

- Auto-remediation without explicit approval.
- Multi-cloud compliance.
- Broad account-wide scanning.

## Why Now

Task 13 makes the hardened policy contract mandatory for supported roles, but enforcement is still AWS-side. A scanner is a separate product/onboarding feature that explains drift before an access request fails.

## Technical Decisions

- [Scanner contract](../../technical-decisions/TASK-000014-aws-policy-compliance-scanner/001-scanner-contract.md)
- [AWS read boundary](../../technical-decisions/TASK-000014-aws-policy-compliance-scanner/002-aws-read-boundary.md)
- [Verification and output contract](../../technical-decisions/TASK-000014-aws-policy-compliance-scanner/003-verification-and-output-contract.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000014-SCAN-001 | Scanner reports whether a configured role satisfies CloudIAM trust-policy requirements from `TASK-000013`. | Fixture tests cover compliant and missing-condition policies. |
| REQ-000014-CONFIG-001 | Scanner follows configured CloudIAM AWS targets rather than scanning arbitrary account roles. | Tests cover target-to-role resolution from YAML config. |
| REQ-000014-FINDINGS-001 | Scanner emits stable finding IDs for each compliance failure. | Unit tests assert finding IDs from policy fixtures. |
| REQ-000014-DIFF-001 | Scanner output explains missing actions/conditions without exposing secrets. | Golden-output tests cover remediation text and secret redaction. |
| REQ-000014-SAFETY-001 | Scanner does not mutate AWS state. | Default code path uses `iam:GetRole` only for live AWS reads. |
| REQ-000014-REGRESSION-001 | Scanner catches the combined SourceIdentity/session-tag trust-policy failure seen in Task 13. | Fixture test reports `AWS_TRUST_COMBINED_SOURCE_AND_TAG_CONDITIONS`. |

## Result

CloudIAM now has a read-only AWS policy compliance scanner for configured AWS access targets.

The scanner:

- reads target roles from the YAML config
- fetches each role trust policy with `iam:GetRole`
- checks the Task 13 managed-role contract
- emits stable finding IDs
- masks AWS account IDs in output
- exits non-zero when a configured target is non-compliant

Run it locally with:

```bash
make AWS_PROFILE=akira-admin-dev aws-scan-policies
```

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make AWS_PROFILE=akira-admin-dev aws-scan-policies`

The live scanner reported `aws-sandbox-readonly` as compliant and printed the role ARN with the account ID masked.
