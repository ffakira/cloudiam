# Cloud IAM MVP Docs

This repository is now focused on a technical MVP:

```text
CLI login -> user device approval -> named access target -> optional console approval -> AWS STS credentials
```

The old Segura/interview research notes are archived under `docs/archive/research/`.

## Start Here

1. [Project state](PROJECT_STATE.md)
2. [One-week execution plan](ONE_WEEK_PLAN.md)
3. [Completed task: User self-service device approval page](tasks/completed/000019-user-self-service-device-approval-page.md)

## Work Tracking

- [Tasks](tasks/)
- [Active tasks](tasks/active/)
- [Completed tasks](tasks/completed/)
- [Backlog](tasks/backlog/)
- [Technical decisions](technical-decisions/)
- [Archived research](archive/research/)

## Current Rule

Promote one backlog task at a time. Current active task: `TASK-000028` security hardening.

The hardening sequence is:

- `TASK-000028`: Security hardening
- `TASK-000029`: Edge-case audit
- `TASK-000027`: Production hardening

`TASK-000011` multi-account support and `TASK-000012` multi-cloud providers remain deferred while the completed AWS-only MVP is hardened.
