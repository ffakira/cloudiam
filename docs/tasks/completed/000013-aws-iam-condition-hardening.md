---
title: "AWS IAM condition hardening"
id: TASK-000013
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
  - security
  - hardening
---

# Task: AWS IAM Condition Hardening

## Outcome

Constrain AWS-side trust and permission policies with IAM conditions once CloudIAM principal and workspace naming are stable.

## Scope

- Add conditions for expected `sts:SourceIdentity` format.
- Add conditions for allowed `aws:RequestTag/*` keys and values.
- Add conditions for `aws:TagKeys`.
- Consider `sts:TransitiveTagKeys` only if role chaining becomes part of the product.
- Keep placeholders masked in committed templates.

## Not In Scope

- Changing Task 000007 behavior.
- Supporting multi-account or multi-cloud.
- Adding role chaining or transitive session tags.

## Why Now

Task 000007 sent SourceIdentity and session tags without strict AWS-side value conditions. Task 000013 makes those fields mandatory for CloudIAM-supported AWS JIT roles while keeping exact tenant/principal enforcement out of scope for the MVP.

## Technical Decisions

- [SourceIdentity condition contract](../../technical-decisions/TASK-000013-aws-iam-condition-hardening/001-sourceidentity-condition-contract.md)
- [Session tag condition contract](../../technical-decisions/TASK-000013-aws-iam-condition-hardening/002-session-tag-condition-contract.md)
- [Verification and rollout contract](../../technical-decisions/TASK-000013-aws-iam-condition-hardening/003-verification-and-rollout-contract.md)
- [Mandatory managed-role boundary](../../technical-decisions/TASK-000013-aws-iam-condition-hardening/004-mandatory-managed-role-boundary.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000013-SOURCE-001 | Trust policy can restrict acceptable `sts:SourceIdentity` values. | Template and docs include masked condition examples. |
| REQ-000013-TAGS-001 | Trust policy can restrict acceptable CloudIAM session tag keys. | Template and docs include `aws:TagKeys` / `aws:RequestTag` examples. |
| REQ-000013-DOCS-001 | Failure modes are documented clearly. | README or task docs explain expected AWS access-denied behavior. |
| REQ-000013-VERIFY-001 | IAM simulator and live access request still allow the happy path. | Simulator returns `allowed`; CLI request succeeds without credential output in docs. |
| REQ-000013-BOUNDARY-001 | Supported CloudIAM-managed/onboarded roles require the hardened trust-policy contract. | ADR and docs state non-conforming roles are unsupported until updated. |

## Result

CloudIAM now prefixes AWS SourceIdentity values with `cloudiam-`, and the AWS trust-policy template requires that prefix for supported JIT roles.

The trust policy is split into separate statements:

- `sts:AssumeRole` / `sts:SetSourceIdentity` require `sts:SourceIdentity` to match `cloudiam-*`.
- `sts:TagSession` restricts tag keys to CloudIAM's tag set, requires `cloudiam_provider=aws`, and rejects transitive session tags.

Root scripts now apply and debug the hardened policy without pasting command heredocs into the shell:

- `aws-task13-apply.sh`
- `aws-task13-debug.sh`

## Verification

- `jq . cloud/broker-assume-policy.json`
- `jq . cloud/jit-readonly-trust-policy.json`
- `jq . cloud/jit-readonly-permissions.json`
- `bash -n aws-task13-apply.sh`
- `bash -n aws-task13-debug.sh`
- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `AWS_PROFILE=akira-admin-dev ./aws-task13-apply.sh`
- `go run ./cmd/access-cli access request aws-sandbox-readonly --duration 900`

The live CLI request succeeded. Temporary AWS credentials were intentionally not copied into this task document.
