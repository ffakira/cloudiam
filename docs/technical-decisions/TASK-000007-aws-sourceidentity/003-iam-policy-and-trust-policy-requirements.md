---
title: "IAM policy and trust policy requirements for TASK-000007"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000007
tags:
  - aws
  - iam
  - sts
  - security
---

# Decision: IAM Policy And Trust Policy Requirements

## Sources

- [AWS IAM: Monitor and control actions taken with assumed roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_control-access_monitor.html)
- [AWS IAM: Pass session tags in AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_session-tags.html)
- [AWS STS service authorization reference](https://docs.aws.amazon.com/service-authorization/latest/reference/list_sts.html)

## Decision

Update local AWS policy templates for the broker and JIT role trust policy.

Broker permission policy must allow:

- `sts:AssumeRole`
- `sts:SetSourceIdentity`
- `sts:TagSession`

JIT role trust policy must allow the broker principal to perform:

- `sts:AssumeRole`
- `sts:SetSourceIdentity`
- `sts:TagSession`

Keep placeholders masked:

- `<AWS_ACCOUNT_ID>`
- `<BROKER_USER_NAME>`

In this MVP, do not add restrictive IAM conditions for exact source identity or tag values yet. CloudIAM principal names are still local/demo data, and overly strict AWS-side conditions will slow iteration. Add that as a future hardening backlog item after the principal model is stable.

## Acceptance Criteria

- `cloud/broker-assume-policy.json` includes required STS actions.
- `cloud/jit-readonly-trust-policy.json` includes required STS actions.
- Docs explain that missing permissions cause AssumeRole to fail.
- No personal AWS account IDs are committed.
