---
title: "Session tag contract for TASK-000007"
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
  - session-tags
  - governance
---

# Decision: Session Tag Contract

## Sources

- [AWS IAM: Pass session tags in AWS STS](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_session-tags.html)
- [AWS STS API: AssumeRole](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html)
- [AWS STS API: Tag](https://docs.aws.amazon.com/STS/latest/APIReference/API_Tag.html)

## Decision

Pass a small, stable set of non-secret session tags on AWS AssumeRole:

| Key | Value |
|---|---|
| `cloudiam_principal` | CloudIAM principal |
| `cloudiam_target` | CloudIAM access target name |
| `cloudiam_provider` | Cloud provider, currently `aws` |
| `cloudiam_account` | CloudIAM configured account alias |
| `cloudiam_role` | CloudIAM configured role alias |

Use underscore-separated tag keys instead of colon-separated keys. AWS STS allows colons in tag keys, but underscore keys are easier to use in IAM conditions, CLI commands, log searches, and future documentation examples.

No raw CloudIAM session token, AWS secret access key, AWS session token, raw device code, or raw user code may appear in a session tag.

Do not mark tags transitive in this slice. Role chaining is not part of the current MVP, and non-transitive tags keep blast radius smaller.

## Acceptance Criteria

- AssumeRole input includes the selected tag keys.
- Tag values are bounded to AWS STS tag value limits.
- Tags are not transitive in this task.
- Tests assert no credential-like values are passed as tags.
