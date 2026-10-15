---
title: "Audit event schema for TASK-000005"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000005
tags:
  - audit
  - database
  - security
  - governance
---

# Decision: Audit Event Schema For TASK-000005

## Related Work

- `TASK-000005`: Persistent audit events.
- `TASK-000003`: CLI device-code login.

## Sources

- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
- [OWASP Developer Guide: Logging and Monitoring](https://devguide.owasp.org/en/04-design/02-web-app-checklist/09-logging-monitoring/)

## Decision

Persist access governance events in Postgres.

The schema records:

- when the event happened
- event type
- principal
- target
- decision/outcome
- reason
- expiration time when AWS credentials are issued
- bounded provider/account/role metadata

The schema must not contain:

- AWS secret access keys
- AWS session tokens
- CloudIAM session tokens
- raw device codes
- raw user codes

OWASP logging guidance says security logs should contain enough context to reconstruct what happened, while excluding secrets such as tokens/passwords and sensitive data not required for investigation. This task follows that boundary.

## SQL Shape

```sql
CREATE TABLE audit_events (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  event_type TEXT NOT NULL,
  principal TEXT NOT NULL,
  target TEXT NOT NULL,
  decision TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',

  provider TEXT NOT NULL DEFAULT '',
  account TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT '',
  credential_expires_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

  CONSTRAINT audit_events_event_type_check
    CHECK (event_type IN ('access_requested', 'access_denied', 'access_issued', 'access_failed')),
  CONSTRAINT audit_events_decision_check
    CHECK (decision IN ('allowed', 'denied', 'failed')),
  CONSTRAINT audit_events_principal_check
    CHECK (length(trim(principal)) > 0),
  CONSTRAINT audit_events_target_check
    CHECK (length(trim(target)) > 0)
);

CREATE INDEX audit_events_occurred_at_idx
  ON audit_events (occurred_at DESC);

CREATE INDEX audit_events_principal_occurred_at_idx
  ON audit_events (principal, occurred_at DESC);

CREATE INDEX audit_events_target_occurred_at_idx
  ON audit_events (target, occurred_at DESC);
```

## Rules

- Audit recording failure must not mint credentials if the event is the final `access_issued` event.
- Denied access attempts must be recorded before returning the denial.
- Audit event reads are local/dev only in this slice.
- Keep event data scalar and bounded; no arbitrary request dumps.

## Acceptance Criteria

- Allowed access request records an `access_issued` event.
- Denied access request records an `access_denied` event.
- Failed access request can record `access_failed` when the request reached the issuer path.
- Audit events can be listed newest-first.
- Schema contains no secret credential columns.
