---
title: "Persistent audit events"
id: TASK-000005
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "governance-1"
priority: P1
sprint_candidate: true
tags:
  - audit
  - governance
  - security
  - mvp
  - postgres
---

# Task: Persistent Audit Events

## Outcome

Access decisions now write durable Postgres audit events, and the local/dev HTTP listener exposes recent events for governance inspection.

## Scope

- Persist request time, principal, target, decision, reason, and expiration when issued.
- Persist enough metadata to answer who requested what, when, and what happened.
- Do not store AWS secret credentials, AWS session tokens, CloudIAM session tokens, or raw device codes.
- Add a local/dev HTTP read endpoint for recent audit events.
- Keep dashboard/read-model work out of scope.

## Technical Decisions

- [Audit event persistence schema](../../technical-decisions/TASK-000005-persistent-audit-events/001-audit-event-schema.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000005-SCHEMA-001 | Audit events are persisted in Postgres through a versioned migration. | Migration exists and integration test writes/reads events. |
| REQ-000005-ACCESS-001 | Allowed access requests create audit events. | Core service test and/or integration test covers issued event. |
| REQ-000005-ACCESS-002 | Denied access requests create audit events. | Core service test covers denied event. |
| REQ-000005-SECURITY-001 | Audit events do not include AWS secret access key, AWS session token, CloudIAM session token, or raw device code. | Schema review and tests confirm no secret fields. |
| REQ-000005-HTTP-001 | Local/dev HTTP endpoint returns recent audit events. | HTTP handler test covers JSON response. |
| REQ-000005-HTTP-002 | Audit read endpoint is local/dev only in this slice. | Server wiring uses existing local HTTP listener. |

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make server-test-integration`
