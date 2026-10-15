---
title: "Reload trigger and scope for TASK-000010"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000010
tags:
  - configuration
  - operations
  - scope
---

# Decision: Reload Trigger And Scope

## Decision

Use an explicit reload trigger for this task. Do not add a filesystem watcher.

The preferred MVP trigger is process-local `SIGHUP` for the running server. If signal-based testing becomes awkward, expose the same reload function behind a small internal hook and keep the production trigger as `SIGHUP`.

Reloadable fields:

- `principals`
- `access.targets`
- `access.policies`
- AWS account IDs and role ARNs used by target resolution

Restart-required fields:

- `serve.grpc.addr`
- HTTP/device/audit server address
- database URL
- auth workspace/endpoint/verification URI
- AWS SDK runtime region/client construction

If a reload changes restart-required fields, the reload must fail clearly or keep using the old runtime value. It must not half-apply runtime settings.

## Rationale

Filesystem watchers introduce extra edge cases: partial writes, editor temp files, duplicate events, debounce timing, and platform differences. A signal keeps the MVP operationally useful while making the async boundary explicit.

## Acceptance Criteria

- Runtime reload has one explicit trigger.
- No filesystem watcher is introduced.
- Docs state which fields are reloadable and which require restart.
- Non-reloadable changes do not silently half-apply.
