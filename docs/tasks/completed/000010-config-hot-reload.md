---
title: "Config hot reload"
id: TASK-000010
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "configuration-runtime"
priority: P1
sprint_candidate: true
tags:
  - configuration
  - operations
  - concurrency
---

# Task: Config Hot Reload

## Outcome

Reload the YAML governance config while the server is running, without interrupting in-flight requests or accepting invalid partial config.

## Scope

- Add an explicit runtime reload trigger.
- Validate the candidate YAML config before making it active.
- Atomically swap the active governance snapshot only after validation succeeds.
- Keep in-flight requests on the snapshot they started with.
- Keep the previous valid config when reload fails.
- Document which config fields require server restart.

## Not In Scope

- Filesystem watcher reload.
- Hot-changing gRPC/HTTP listener addresses.
- Hot-changing database/auth runtime settings.
- Hot-changing AWS SDK region/client construction.
- Dashboard UI for reloads.

## Why Now

The demo will involve config-driven access targets and policy checks. A safe reload path lets us change governance rules without bouncing the local server, but the async edge cases must be explicit before implementation.

## Technical Decisions

- [Reload trigger and scope](../../technical-decisions/TASK-000010-config-hot-reload/001-reload-trigger-and-scope.md)
- [Config snapshot concurrency contract](../../technical-decisions/TASK-000010-config-hot-reload/002-config-snapshot-concurrency.md)
- [Failure handling and observability](../../technical-decisions/TASK-000010-config-hot-reload/003-failure-handling-and-observability.md)
- [Test matrix](../../technical-decisions/TASK-000010-config-hot-reload/004-test-matrix.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000010-TRIGGER-001 | Server exposes an explicit reload trigger without using a filesystem watcher. | Unit or integration test covers trigger path. |
| REQ-000010-VALIDATE-001 | Candidate config is fully loaded and validated before it becomes active. | Test invalid YAML/invalid config keeps old snapshot. |
| REQ-000010-ATOMIC-001 | In-flight access requests use one consistent config snapshot. | Concurrency test covers request during reload. |
| REQ-000010-FAILSAFE-001 | Failed reload does not interrupt the server or replace the last good config. | Test reload failure and subsequent request still succeeds with old config. |
| REQ-000010-SCOPE-001 | Non-reloadable runtime fields are rejected or ignored with clear output. | Test documents restart-required fields. |
| REQ-000010-OBS-001 | Reload success/failure is visible to operators without printing secrets. | Output/log tests assert status and no credential/session values. |

## Result

The server now supports explicit process-local config reload through `SIGHUP`.

Reload builds and validates a complete candidate YAML config before publishing it. The active governance config is held as an immutable snapshot behind an atomic pointer, and access requests capture one snapshot for authorization and target resolution so in-flight requests cannot mix generations.

Failed reloads keep the last valid config active and report a concise operator status with config path, generation, and error reason. Reload rejects restart-required changes for runtime fields that cannot be safely half-applied.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `env GOCACHE=/tmp/cloudiam-go-cache go test -race ./internal/app/server ./internal/core/access`
