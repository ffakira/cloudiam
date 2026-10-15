---
title: "Failure handling and observability for TASK-000010"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000010
tags:
  - configuration
  - observability
  - safety
---

# Decision: Failure Handling And Observability

## Decision

Reload is fail-closed for the candidate config and fail-open for the running service.

That means:

- invalid candidate config is rejected
- the last valid config remains active
- the server continues serving requests
- the reload trigger reports failure clearly

Reload status must include:

- whether reload succeeded
- config path
- active config version or generation number
- error reason when reload fails

Reload status must not include:

- AWS access keys
- AWS secret access keys
- AWS session tokens
- CloudIAM session tokens
- raw device codes

## Edge Cases

- YAML parse error: reject reload, keep old config.
- Config validation error: reject reload, keep old config.
- Missing file: reject reload, keep old config.
- Environment variable expands to empty value: reject reload through existing validation.
- Reload requested during shutdown: return context/shutdown error and do not swap.
- Multiple reloads overlap: serialize reload attempts or allow only one active reload at a time.

## Acceptance Criteria

- Failed reload leaves the previous config active.
- Operator output/logs explain the failure without secrets.
- Concurrent reload attempts cannot corrupt active config state.
