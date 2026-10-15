---
title: "Split console auth and approval core domains"
id: TASK-000025
type: refactor
status: completed
created_at: 2026-10-08
updated_at: 2026-10-08
completed_at: 2026-10-08
owner: codex
phase: "dashboard-security-2"
priority: P0
sprint_candidate: true
tags:
  - architecture
  - naming
  - dashboard
  - approval
  - auth
---

# Task: Split Console Auth And Approval Core Domains

## Outcome

Remove the broad `admin` core package name before it becomes a long-lived dumping ground.

## Scope

- Replace `internal/core/admin` with responsibility-based packages.
- Use `internal/core/consoleauth` for dashboard/operator session login, verification, refresh, and token errors.
- Use `internal/core/approval` for JIT resource approval mutations, idempotency fingerprinting, approval store contracts, and approval state errors.
- Rename the HTTP adapter implementation from `AdminHandler` to `ConsoleHandler` while preserving the current `/api/admin/...` route contract.
- Keep external environment variable names and route paths stable for the MVP.

## Technical Decisions

- [Console auth and approval package boundaries](../../technical-decisions/TASK-000025-console-auth-and-approval-domain-split/001-consoleauth-and-approval-boundaries.md)

## Result

The core package split now reflects actual bounded contexts:

- `consoleauth`: browser console session authentication.
- `approval`: resource/JIT approval transition semantics.

The `/api/admin` route prefix remains unchanged because that is an external MVP HTTP contract, not the internal domain model.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `bun run build`
- Source scan confirmed no `internal/core/admin`, `package admin`, `AdminHandler`, or `AdminAuthService` references remain.
