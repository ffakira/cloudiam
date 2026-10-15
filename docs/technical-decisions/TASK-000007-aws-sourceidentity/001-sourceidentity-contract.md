---
title: "SourceIdentity contract for TASK-000007"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000007
tags:
  - aws
  - sts
  - sourceidentity
  - governance
---

# Decision: SourceIdentity Contract

## Sources

- [AWS IAM: Monitor and control actions taken with assumed roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp_control-access_monitor.html)
- [AWS STS API: AssumeRole](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
- [AWS CloudTrail: userIdentity element](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-event-reference-user-identity.html)

## Decision

Set STS `SourceIdentity` on every AWS AssumeRole call using the CloudIAM principal.

The value must:

- be derived only from the authenticated CloudIAM principal
- follow the STS SourceIdentity character constraints
- be 2 to 64 characters
- not begin with `aws:`
- not include secrets, tokens, email-like external identifiers unless they are already the principal ID

CloudIAM will reuse the same sanitizer family as `RoleSessionName`, but keep this as a separate helper so SourceIdentity rules can evolve independently.

## Acceptance Criteria

- AssumeRole input includes `SourceIdentity`.
- Empty or invalid principals fall back to a stable non-secret CloudIAM value.
- Unit tests cover sanitization, truncation, and reserved `aws:` prefix handling.
- Documentation states that AWS CloudTrail records SourceIdentity on STS assume-role calls and subsequent role-session activity.
