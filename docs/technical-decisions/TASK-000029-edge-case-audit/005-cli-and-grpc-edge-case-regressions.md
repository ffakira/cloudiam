---
title: "CLI and gRPC edge-case regressions for TASK-000029"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - cli
  - grpc
  - testing
---

# Decision: CLI And gRPC Edge-Case Regressions

## Context

E2 covers malformed and stale session behavior at the CLI and gRPC boundaries.

The E1 matrix identified two CLI gaps:

- misplaced flags or extra positional values could be accepted as access-request targets
- malformed/missing session-file cases needed clearer regression coverage

The gRPC access/auth adapters already had focused coverage for the high-risk rows: missing/invalid CloudIAM sessions, negative duration rejection, validation mapping, device-login validation, and no-secret status messages.

## Decision

Harden the CLI parser and add regressions at the CLI boundary.

Accepted CLI behavior:

- `access request` requires exactly one target.
- A target that starts with `-` is treated as a missing target, not as a real target name.
- Extra positional arguments after the target are rejected.
- Missing, malformed, expired, incomplete, or credential-contaminated session files fail before gRPC access is requested.

Accepted gRPC behavior:

- Missing or invalid CloudIAM sessions return `Unauthenticated` before the access service is called.
- Negative access duration returns `InvalidArgument` before the access service is called.
- Auth/device-login validation errors return `InvalidArgument` before mutating service state.
- Adapter status messages do not include raw session tokens, device codes, user codes, AWS credentials, or lower-level service error details.

## Verification

- `go test ./internal/app/cli ./internal/adapters/cli ./internal/adapters/grpc`

New regressions:

- CLI parser rejects misplaced duration flags where a target is required.
- CLI parser rejects extra positional arguments after the target.
- Session store rejects missing files, malformed JSON, missing access tokens, and multiple AWS/session credential field spellings.

Existing gRPC regressions remain the evidence for:

- missing session token
- invalid/stale session token
- negative duration
- hidden resolver/service error details
- missing auth/device-login request fields
- mapped device authorization states

## Consequences

- CLI malformed input now fails before session loading or gRPC calls.
- E2 does not add new gRPC implementation because the existing tests already cover the accepted invariants.
- Broader CLI UX changes, such as allowing flags before the target, are out of scope; this slice only rejects ambiguous malformed input.
