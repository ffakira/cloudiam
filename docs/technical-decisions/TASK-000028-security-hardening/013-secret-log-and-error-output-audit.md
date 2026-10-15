---
title: "Secret log and error-output audit for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - secrets
  - logging
  - errors
---

# Decision: Secret Log And Error-Output Audit

## Context

S6 covers places where secret material can leave the application outside normal API responses: process stderr, reload status output, CLI output, browser error messages, and operator-facing command output.

CloudIAM intentionally has one raw credential output path: `access request --format env`, which prints short-lived AWS credentials for the authenticated CLI requester. Other paths should avoid raw CloudIAM session tokens, console JWTs, idempotency keys, device/user codes, AWS credentials, database credentials, Redis credentials, and password material.

## Audit Result

| Surface | Result | Action |
|---|---|---|
| HTTP/gRPC API errors | Generic external messages already in place for auth, approval, audit, device, and access handlers. | No code change. Existing tests cover no-leakage. |
| CLI login output | Prints verification URI and user code, then principal after session exchange. Does not print CloudIAM session token. | No code change. Existing tests cover session-token absence. |
| CLI access default output | Prints target, expiry, and a hidden-credentials note. | No code change. Existing tests cover AWS credential absence. |
| CLI access `--format env` | Explicit credential-output mode. | Accepted exception from `TASK-000015`. |
| Browser console client | Uses cookie-backed console sessions and stores only safe session metadata in memory. | No code change. S8 will perform the deeper client/browser storage pass. |
| AWS compliance scanner output | Masks AWS account IDs and does not print credential-looking values. | No code change. Existing tests cover masking. |
| Reload status output | Previously printed raw reload errors. | Fixed by redacting operator-facing reload error details. |
| Startup idempotency cleanup errors | Previously wrapped raw store errors. | Fixed by returning a redacted operator-facing cleanup error. |

## Decision

Operator-facing error output must be redacted at the boundary when it can include lower-level persistence, parsing, or infrastructure errors.

The S6 redaction boundary masks:

- URL userinfo passwords for Postgres and Redis URLs.
- Key/value fields whose keys look like passwords, secrets, tokens, JWTs, device/user codes, idempotency keys, request fingerprints, response bodies, or AWS credential fields.
- AWS access key IDs matching the standard `AKIA...` or `ASIA...` shape.

This is not a replacement for storing secrets safely. It is a final guard for text that leaves the process as stderr or reload status.

## Verification

- Reload status tests now inject secret-shaped error content and assert raw values are absent.
- Idempotency cleanup startup-error tests now inject secret-shaped lower-level error content and assert raw values are absent.
- `go test ./internal/app/server` verifies the changed boundary.
- Existing CLI, HTTP, gRPC, audit, and compliance tests continue to cover their own no-leakage paths.

## Consequences

- Reload output remains useful enough to show failure context without echoing raw secrets.
- Startup cleanup failures still name the failing subsystem, but sensitive row or credential details are masked.
- S8 remains responsible for the deeper browser storage scan; S6 only records the current browser error/output audit.
