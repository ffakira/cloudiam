---
title: "Console approval audit events"
id: TASK-000020
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-09
completed_at: 2026-10-09
owner: codex
phase: "approval-1"
priority: P1
sprint_candidate: true
outcome: "Resource approval lifecycle actions now emit first-class audit events with non-secret request, operator, and idempotency-reference metadata."
tags:
  - audit
  - approval
  - console
---

# Completed Task: Console Approval Audit Events

## Outcome

Resource approval lifecycle actions now write durable audit events:

- `approval_requested`
- `approval_approved`
- `approval_denied`
- `approval_expired`

Approval audit events include requester principal, target, provider/account/role metadata, access request ID, console operator subject when applicable, and a SHA-256 idempotency-key reference for console mutations.

Raw console JWTs, CloudIAM session tokens, raw device/user codes, raw idempotency keys, and AWS credential values are not stored in audit events.

## Verification

- `cd server && GOCACHE=/tmp/segura-go-build go test ./...`
