---
title: "CLI credential output safety"
id: TASK-000015
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "cli-safety"
priority: P1
sprint_candidate: true
tags:
  - cli
  - security
  - credentials
  - developer-experience
---

# Task: CLI Credential Output Safety

## Outcome

Make AWS temporary credential output explicit and harder to leak during testing, demos, and support workflows.

## Scope

- Change the default `access request` output to a non-secret success summary.
- Keep shell-export credential output available through an explicit flag or format.
- Add structured JSON output only if it can avoid accidental logging by default.
- Ensure tests prove the default output does not include AWS access keys, secret access keys, or session tokens.
- Keep audit/log storage free of raw AWS credentials and CloudIAM session tokens.

## Non-Goals

- Building OS keychain integration.
- Copying credentials to clipboard.
- Changing how credentials are issued by the server.
- Changing AWS STS duration or policy behavior.

## Why Now

The current CLI prints live AWS temporary credentials directly into terminals by default. That is useful for a first proof, but unsafe for demos, support, task logs, and chat-driven testing.

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000015-DEFAULT-001 | Default access request output contains no raw AWS credential values. | CLI output tests assert absence of access key, secret key, and session token values. |
| REQ-000015-EXPLICIT-001 | Operators can still request shell-export output intentionally. | CLI tests cover the explicit credential-output mode. |
| REQ-000015-LOGGING-001 | Credential values are not written to logs, audit events, or task docs. | Tests or code review cover output and audit boundaries. |

## Result

The CLI now defaults to a non-secret access summary for `access request`.

Default output includes:

- access status
- target name
- credential expiration
- a note that credentials are hidden

Shell export output remains available only when explicitly requested:

```bash
go run ./cmd/access-cli access request aws-sandbox-readonly --duration 900 --format env
```

`--format json` was intentionally not added in this task. It would introduce another raw credential output path without helping the current demo flow.

## Verification

- `env GOCACHE=/tmp/cloudiam-go-cache go test ./internal/adapters/cli ./internal/app/cli`
- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`

The default-output test asserts that access key, secret key, session token, and AWS shell-export variable names are absent from normal CLI output.
