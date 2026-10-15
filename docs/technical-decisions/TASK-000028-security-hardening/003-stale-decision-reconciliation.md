---
title: "Stale decision reconciliation for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - adr
  - production
---

# Decision: Stale Decision Reconciliation

## Context

The MVP was built through small tasks. Some early ADRs intentionally kept local/demo shortcuts, while later tasks hardened or superseded parts of those decisions.

Task 28 must respect accepted decisions but should not preserve stale assumptions that were only safe before the dashboard, console sessions, persisted users, and approval lifecycle existed.

## Decision

Use this reconciliation when auditing and hardening the codebase.

## Still Binding

These decisions remain production-security constraints:

- `TASK-000003`: device authorization exchange is single-use and stores only token hashes.
- `TASK-000008`: persistent storage excludes raw auth tokens, raw user/device codes, and AWS credentials.
- `TASK-000013`: supported AWS roles must conform to the CloudIAM SourceIdentity and session-tag trust-policy contract.
- `TASK-000014`: compliance scanner output must be stable and non-secret.
- `TASK-000016`: console approval mutates only resource access requests, requires valid console auth, and uses idempotency.
- `TASK-000017`: access request rows are durable lifecycle objects.
- `TASK-000018`: approval policy changes affect future requests only.
- `TASK-000019`: browser device approval is separate from console resource approval and must not expose secrets.
- `TASK-000020`: approval lifecycle audit events use safe metadata and idempotency-key hashes.
- `TASK-000021`: idempotency records have a bounded replay window and cleanup path.
- `TASK-000023`: approved issuance uses the persisted role ARN and consumes the approved request before STS.
- `TASK-000024`: console JWTs are server-owned HttpOnly cookies, not browser-managed bearer tokens.
- `TASK-000025`: `consoleauth` and `approval` are separate core domains.
- `TASK-000026`: console passwords use persisted Argon2id hashes.

## Superseded Or Demo-Only Assumptions

These assumptions must not be treated as production-complete:

- The early `TASK-000016` bearer-token/JWT response shape is superseded by cookie-backed console sessions.
- The typed device principal on `/device` is local/demo identity, not production identity proof.
- Environment-seeded console users are a bootstrap/demo path, not a full production user lifecycle.
- Claim-before-STS intentionally prevents double issuance, but it is not retry-friendly after AWS transient failures.
- Single-account AWS config is an MVP constraint, not a production multi-account model.
- AWS-only provider support remains the current product boundary.

## Newly Explicit Invariants For Task 28

The earlier ADRs imply these, but task 28 should now test or document them directly:

- Subject binding: an authenticated CloudIAM session principal must be the only principal used for gRPC access decisions.
- Smuggling resistance: clients cannot influence provider, account, role ARN, approval status, audit decision, credential expiration, or operator subject through request payloads.
- Fail-closed auth: missing, malformed, expired, wrong-role, wrong-audience, or tampered auth material produces denial before state mutation.
- Replay resistance: stale device codes, consumed authorizations, duplicate approved requests, expired idempotency keys, and mismatched idempotency fingerprints do not create new authority.
- Non-secret observability: errors, audit rows, logs, scanner output, API responses, and frontend state remain useful without exposing credential material.
- State freshness: hot reload may change future authorization decisions but cannot change already-created approval objects.

## Consequences

- Task 28 may update older docs when they cause ambiguity, but should prefer new clarifying decisions over rewriting history.
- Any production feature implied by these invariants but outside the current MVP, such as production SSO or retry leases for STS issuance, should become a separate backlog task.
