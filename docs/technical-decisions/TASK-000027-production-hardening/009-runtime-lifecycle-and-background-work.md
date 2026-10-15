---
title: "Runtime lifecycle and background work for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - lifecycle
  - shutdown
  - reload
  - cleanup
  - verification
---

# Decision: Runtime Lifecycle And Background Work

## Context

P4 implements the lifecycle portion of ADR 005.

The prior security and edge-case passes already covered most runtime lifecycle behavior:

- config reload preserves the previous known-good snapshot on invalid YAML, invalid semantic config, restart-required field changes, AWS region changes, and canceled contexts
- concurrent reloads and concurrent readers see complete immutable snapshots
- `SIGHUP` triggers reload and reports redacted status
- gRPC shutdown uses graceful stop with forced stop fallback
- server error reporting does not block when the error channel is full
- startup idempotency cleanup is synchronous, bounded by caller context, and redacts lower-level secret-shaped errors
- Redis close and Postgres close are wired through server startup defers

P4 found one missing regression: HTTP shutdown was bounded by code, but not directly tested.

## Decision

Keep the existing lifecycle model:

- HTTP and gRPC servers are stopped through bounded shutdown paths.
- gRPC first tries `GracefulStop`, then calls `Stop` when the graceful window expires.
- HTTP shutdown uses a timeout context and returns after the configured graceful window.
- Config reload remains snapshot-based and preserves the active snapshot on failure.
- Startup cleanup runs before serving traffic and fails startup with redacted errors.
- Postgres and Redis clients are closed through server shutdown defers.

Add a focused HTTP shutdown regression that uses an in-memory `net.Pipe` listener. The test proves `shutdownHTTPServer` returns after its timeout even when a request handler remains blocked, without requiring OS socket permissions.

## Verification

Focused and full server verification passed on 2026-10-12:

```bash
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./internal/app/server
GOCACHE=/tmp/cloudiam-go-build-cache GOMODCACHE=/tmp/cloudiam-go-mod-cache go test ./...
```

Regression added:

- `TestShutdownHTTPServerReturnsAfterTimeout`

Existing accepted P4 coverage:

- `TestConfigStoreReloadInvalidYAMLKeepsOldSnapshot`
- `TestConfigStoreReloadInvalidSemanticConfigKeepsOldSnapshot`
- `TestConfigStoreReloadRejectsRestartRequiredFieldChange`
- `TestConfigStoreReloadRejectsAWSRegionChange`
- `TestConfigStoreReloadCanceledContextKeepsOldSnapshot`
- `TestConfigStoreReloadKeepsInFlightRequestOnOldSnapshot`
- `TestConfigStoreConcurrentReloadsKeepCompleteGeneration`
- `TestConfigStoreConcurrentReadersAndReloadsSeeCompleteSnapshots`
- `TestReloadSignalHandlerReloadsOnSIGHUP`
- `TestStopGRPCServerStopsAfterTimeout`
- `TestStopGRPCServerDoesNotForceStopWhenGracefulCompletes`
- `TestReportServerErrorDoesNotBlockWhenChannelFull`
- `TestWriteReloadStatusDoesNotIncludeCredentialMaterial`
- `TestCleanupExpiredIdempotencyRecordsOnStartup`
- `TestCleanupExpiredIdempotencyRecordsOnStartupPropagatesRedactedError`

## Consequences

- HTTP and gRPC shutdown behavior are now both covered by focused unit regressions.
- Background reload and startup cleanup behavior remain simple and bounded; no workflow engine, queue, or scheduler is introduced in Task 27.
- P4 closes without adding metrics, alerting, external job orchestration, load testing, or chaos testing.
