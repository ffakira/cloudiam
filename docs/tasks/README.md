# Tasks

Task tracking is grouped by status:

```text
tasks/
  active/
  completed/
  backlog/
```

Use stable task IDs. Do not encode dates in filenames.

## Sprint Rule

For the current one-week sprint, active work should stay narrow:

- Finish one active task at a time.
- Pull from backlog only when the active task is complete.
- Prefer small security/product slices over broad platform work.
- Anything requiring multi-cloud support, production admin identity, or dashboard config editing is deferred unless explicitly promoted.

## Current Sprint Order

1. `TASK-000017`: Resource access request lifecycle. Completed.
2. `TASK-000023`: Approved access issuance hardening. Completed.
3. `TASK-000018`: Approval policy configuration. Completed.
4. `TASK-000016`: Console auth and idempotent resource approvals. Completed.
5. `TASK-000019`: User self-service device approval page. Completed.
6. `TASK-000009`: MVP console dashboard. Completed.
7. `TASK-000020`: Console approval audit events. Completed.
8. `TASK-000021`: Idempotency record retention. Completed.
9. `TASK-000022`: Deprecate manual device approve endpoint. Completed.
10. `TASK-000028`: Security hardening. Completed.
11. `TASK-000029`: Edge-case audit. Completed.
12. `TASK-000027`: Production hardening. Completed.

`TASK-000011` multi-account support and `TASK-000012` multi-cloud providers remain deferred. No active hardening task is promoted by default; new work should be selected only if it has a concrete acceptance boundary.
