---
title: "Production security invariant register for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - invariants
  - production
---

# Decision: Production Security Invariant Register

## Context

Earlier tasks built the AWS-only MVP in small slices. Those slices already made security decisions about device login, session storage, audit events, approval idempotency, cookie-backed console sessions, AWS trust-policy enforcement, and approved-request issuance.

`TASK-000028` should not invent a new security model. It should harden the accepted model by making the cross-cutting invariants explicit and then testing them.

## Decision

Use the following invariants as the security hardening contract.

### Identity And Session Invariants

- A CloudIAM session is created only through the device-code exchange path.
- A console operator session is created only through `consoleauth` login and is stored in an HttpOnly cookie.
- Browser code may store safe console session metadata, but must not store console JWTs or CloudIAM session tokens.
- Expired, malformed, revoked, or wrong-audience sessions must fail closed.
- MVP typed device principals remain local/demo behavior, not production identity.

### Route And Domain Boundary Invariants

- Device approval approves only device login.
- Console approval approves or denies only resource/JIT access requests.
- `/api/device/approve` must not mutate resource access requests.
- `/api/admin/access-requests/{id}/approve` and `/deny` must not accept user codes or mutate device authorizations.
- Core `consoleauth` code must not own approval transition semantics, and core `approval` code must not own console credential verification.

### Secret And Credential Invariants

- Durable storage must not contain raw device codes, raw user codes, raw CloudIAM session tokens, raw console JWTs, raw passwords, AWS access key IDs, AWS secret access keys, or AWS session tokens.
- Hash columns may store token digests when lookup requires them.
- API responses, audit events, idempotency records, logs, scanner output, and docs fixtures must not expose credential material.
- CLI credential output may print AWS credentials only through the explicit credential-output mode that already exists for the local workflow.

### Approval And Replay Invariants

- Resource approvals and denials require a valid console session and a UUIDv4 idempotency key.
- Idempotency records are scoped to stable mutation fingerprints and must reject same-key different-fingerprint reuse.
- Pending resource access requests are immutable approval objects; approve/deny must not re-resolve target role metadata from the current hot-reload config.
- Approved requests are single-use for issuance under the accepted claim-before-STS contract.
- Device authorizations are single-use for CloudIAM session creation; consumed, denied, expired, or already-approved invalid states must not issue a second session.

### Concurrency And Context Invariants

- Request-scoped HTTP and gRPC work must propagate context into stores, Redis, and AWS calls.
- Canceled request contexts must stop new privileged state mutations when cancellation is observable before mutation.
- Device authorization exchange, approval idempotency, approved-request claiming, config snapshot reads, and local rate-limit counters must be safe under concurrent requests.
- Database transactions must preserve single-use and idempotency guarantees under concurrent retries.
- Redis rate limiting may fall back locally on Redis errors, but fallback must remain concurrency-safe and must not disable abuse protection.
- Background work that intentionally outlives a request must be explicit and must not carry raw request credentials or mutable request state.

### Authorization And AWS Boundary Invariants

- Access requests are resolved by named target and session principal; clients must not smuggle provider, account, role ARN, AWS credential, approval state, or policy decision fields.
- Authorization must happen before AWS STS is called.
- Supported AWS roles must satisfy the CloudIAM trust-policy contract for SourceIdentity and session tags.
- Issuance after approval must use the persisted role ARN from the access request, not a fresh config lookup.

### Audit Invariants

- Security-relevant decisions must leave durable audit evidence: deny, approval requested, approval approved, approval denied, approval expired, access issued, and access failed.
- Audit records may include scalar governance metadata and safe references such as access request ID, operator subject, and idempotency-key hash.
- Audit records must not include raw tokens, raw passwords, raw device or user codes, raw idempotency keys, or AWS credential values.

## Consequences

- Task 28 implementation should be driven by tests that encode these invariants.
- Race-sensitive invariants should be verified with focused concurrency tests and `go test -race ./...` when feasible.
- Existing ADRs remain accepted unless explicitly superseded.
- Any discovered gap that changes product scope must become a separate backlog task instead of being hidden inside security hardening.
