---
title: "AWS SourceIdentity and session tags"
id: TASK-000007
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "governance-2"
priority: P1
sprint_candidate: true
tags:
  - aws
  - audit
  - security
---

# Task: AWS SourceIdentity And Session Tags

## Outcome

Set AWS STS SourceIdentity and bounded CloudIAM session tags when assuming a role so CloudTrail and AWS-side controls can trace temporary credentials back to CloudIAM governance context.

## Result

CloudIAM now sends AWS STS `SourceIdentity` and bounded non-secret session tags on access requests. The broker IAM policy and JIT role trust policy templates include `sts:SetSourceIdentity` and `sts:TagSession`, and the live AWS access path was verified after applying the updated policies.

## Scope

- Set `SourceIdentity` from the CloudIAM principal.
- Pass bounded session tags for CloudIAM context.
- Keep tag values non-secret and deterministic.
- Update broker permission and role trust policy templates.
- Add tests around the STS AssumeRole request.
- Verify with AWS without printing temporary credentials.

## Risk

This requires updating AWS IAM policies. Source identity requires `sts:SetSourceIdentity`; session tags require `sts:TagSession`. Roles without these trust-policy permissions will reject the AssumeRole request.

## Technical Decisions

- [SourceIdentity contract](../../technical-decisions/TASK-000007-aws-sourceidentity/001-sourceidentity-contract.md)
- [Session tag contract](../../technical-decisions/TASK-000007-aws-sourceidentity/002-session-tag-contract.md)
- [IAM policy and trust policy requirements](../../technical-decisions/TASK-000007-aws-sourceidentity/003-iam-policy-and-trust-policy-requirements.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000007-SOURCE-001 | STS AssumeRole request includes a valid `SourceIdentity` derived from the CloudIAM principal. | Unit test asserts `AssumeRoleInput.SourceIdentity`. |
| REQ-000007-TAGS-001 | STS AssumeRole request includes bounded CloudIAM session tags. | Unit test asserts expected tag keys and values. |
| REQ-000007-IAM-001 | Broker policy template includes `sts:SetSourceIdentity` and `sts:TagSession` for JIT roles. | Cloud policy file updated with masked placeholders. |
| REQ-000007-TRUST-001 | JIT role trust policy template allows `sts:SetSourceIdentity` and `sts:TagSession`. | Cloud trust policy file updated with masked placeholders. |
| REQ-000007-DOCS-001 | README or task docs explain the AWS-side policy requirement and failure mode. | Documentation includes required IAM actions and source links. |
| REQ-000007-VERIFY-001 | Real AWS assume-role path still works without leaking credentials. | Manual/container verification discards temporary credential output. |

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `jq` parsed AWS policy templates.
- IAM simulator returned `allowed` for `sts:AssumeRole`, `sts:SetSourceIdentity`, and `sts:TagSession`.
- Live CLI access request succeeded after AWS policy/trust-policy update.
- Temporary AWS credential output was not pasted into task docs.
