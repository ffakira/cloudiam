---
title: "HTTP and dashboard edge-case regressions"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - http
  - dashboard
  - security
  - testing
---

# Decision: HTTP And Dashboard Edge-Case Regressions

## Context

E3 covers the browser-facing and HTTP API boundaries for the MVP:

- console login/logout/refresh/session endpoints
- console approval list and mutation endpoints
- console audit-event endpoint
- device approval endpoint
- dashboard API consumption and build-time secret scanner

The edge-case risk is not that these endpoints need broad new product behavior. The risk is that malformed browser/API input could reach auth, persistence, audit, or mutation paths before validation; or that validation errors could echo secret-bearing request material.

## Decision

Treat E3 as covered by strict HTTP boundary behavior plus dashboard build verification.

The accepted invariants are:

- Login JSON must reject unknown fields, trailing JSON, and oversized bodies before calling the auth service.
- Login validation failures must not set a console session cookie or echo the submitted password/token-shaped fields.
- Logout, refresh, approve, and deny endpoints remain no-body mutations. Positive-length and chunked bodies must be rejected before auth/store mutation.
- Approval mutation routes must parse to a positive request ID and supported action before store mutation or approval audit writes.
- Access-request list status filters must reject invalid values before store reads.
- Device approval endpoints continue to reject invalid JSON, unknown fields, trailing JSON, oversized bodies, overlong values, invalid state, and deprecated route usage without secret leakage.
- Dashboard behavior remains represented by the existing Task 28 mutation-state scrubbing and client build-time secret scanner; E3 does not add browser E2E automation because no current bug or security gap requires it.

## Implementation

Added focused HTTP adapter regressions in `server/internal/adapters/http/console_handler_test.go`:

- `TestConsoleHandlerLoginRejectsMalformedBodiesBeforeAuth`
- stronger auth-call assertion in `TestConsoleHandlerLoginRejectsTrailingJSON`
- `TestConsoleHandlerRejectsChunkedBodiesOnNoBodyMutationsBeforeAuth`
- `TestConsoleHandlerRejectsMalformedMutationPathBeforeStore`
- `TestConsoleHandlerRejectsInvalidListStatusBeforeStore`

No production HTTP handler change was required for E3. The existing handlers already failed closed for the audited cases.

## Verification

Run:

```bash
cd server
GOCACHE=/tmp/cloudiam-go-cache go test ./internal/adapters/http
GOCACHE=/tmp/cloudiam-go-cache go test ./...
cd ..
make client-build
```

## Consequences

- `E1-HTTP-001` is covered for the MVP API boundary.
- `E1-HTTP-002` remains covered by the Task 28 dashboard state scrubbing and client scanner, with no E3-triggered patch needed.
- Full browser E2E coverage remains out of scope for Task 29 unless a later slice reveals a concrete regression that cannot be proved at the unit/build layer.
