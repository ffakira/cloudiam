---
title: "Test matrix for TASK-000010"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000010
tags:
  - configuration
  - testing
  - concurrency
---

# Decision: Test Matrix

## Decision

Task 10 must be test-first around the reload boundary. The important behavior is not that reload exists, but that reload does not create mixed-policy or partial-config states.

Required tests:

| Scenario | Expected behavior |
|---|---|
| Valid policy change reloads | New request observes new policy decision. |
| Invalid YAML reload | Old config remains active. |
| Invalid semantic config reload | Old config remains active. |
| Target removed during reload | Existing in-flight request keeps old snapshot; new request fails. |
| Role ARN changed during reload | New request resolves new ARN; old request keeps resolved ARN. |
| Restart-required field changed | Reload rejects or reports restart-required without half-applying. |
| Concurrent reload requests | Active config remains one complete valid generation. |
| Request overlaps reload | Authorization and target resolution come from the same generation. |
| Reload output | Status is clear and contains no credential/session material. |

## Verification Commands

Run at minimum:

```bash
env GOCACHE=/tmp/cloudiam-go-cache go test ./...
```

If the implementation introduces goroutines beyond the signal handler, run the race detector for the touched packages:

```bash
env GOCACHE=/tmp/cloudiam-go-cache go test -race ./internal/app/server ./internal/core/...
```

## Acceptance Criteria

- Tests cover success, failure, and request/reload overlap.
- Race-sensitive code is covered by package-level concurrency tests.
- No test relies on timing sleeps when a deterministic channel/barrier can express the same condition.
