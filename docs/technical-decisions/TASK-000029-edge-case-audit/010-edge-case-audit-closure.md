---
title: "Edge-case audit closure"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - closure
  - residual-risk
  - testing
  - security
---

# Decision: Edge-Case Audit Closure

## Context

`TASK-000029` was bounded by ADRs 001 through 004 and implemented through E1 through E7. The goal was not to discover every possible rare case. The goal was to prove the current AWS-only MVP fails closed and predictably at the boundaries it already owns.

## Closure Decision

Close `TASK-000029` as complete.

All planned slices are complete:

| Slice | Status | Evidence |
|---|---|---|
| E1 | Completed | ADR 004: finite edge-case matrix and stop rule |
| E2 | Completed | ADR 005: CLI parser/session regressions and accepted gRPC malformed/stale-session coverage |
| E3 | Completed | ADR 006: HTTP/dashboard boundary regressions and accepted dashboard scanner/state-scrubbing coverage |
| E4 | Completed | ADR 007: persistence state-transition regressions and accepted Postgres race coverage |
| E5 | Completed | ADR 008: config reload, startup, Redis fallback, production hardening, and shutdown coverage |
| E6 | Completed | ADR 009: AWS STS boundary hardening and credential-issuance audit fail-closed coverage |
| E7 | Completed | This ADR: final gates and residual-risk report |

## Final Gate Results

Passed:

```bash
make server-security-check
make client-build
```

`make server-security-check` passed:

- tool install
- `go mod verify`
- `go vet ./...`
- `go test ./...`
- `go test -race ./...`
- `govulncheck ./...`
- `gosec -quiet -exclude-dir=gen ./...`
- `golangci-lint run --no-config --default=standard ./...`

`govulncheck` reported no called vulnerabilities. It also reported one vulnerability in required modules that current code does not appear to call.

`make client-build` passed the browser-boundary scanner, TypeScript build, and Vite production build.

## Residual Risks

Accepted for the MVP:

- `make client-build` emits a Vite chunk-size warning for one JavaScript chunk above 500 kB. This is performance/build hygiene, not a security or correctness blocker for Task 29.
- Postgres integration tests are not part of the final `make server-security-check` target, but E4 integration coverage was run during the slice and passed against local Postgres.
- SIEM/export/alerting for audit evidence remains outside Task 29. The MVP has durable audit records and no-leakage regressions, not a production event pipeline.
- External secret-manager integration remains outside Task 29. The MVP enforces no secret material in config files and validates production secret strength/boundaries, but it still accepts secrets through environment/config inputs.
- Browser E2E automation remains outside Task 29. Current dashboard coverage is unit/build/scanner focused.
- Full load, chaos, and distributed workflow orchestration are outside Task 29. Current concurrency coverage targets owned state-transition races and snapshot consistency.
- Multi-account and multi-cloud expansion remain deferred by task scope.

## Consequences

- `TASK-000029` can move to completed.
- The next recommended backlog item is `TASK-000027: Production hardening`.
- Do not reopen Task 29 for speculative edges. New findings should become narrow backlog tasks unless they are regressions in the accepted MVP invariants.
