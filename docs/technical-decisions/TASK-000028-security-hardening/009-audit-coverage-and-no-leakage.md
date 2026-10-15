---
title: "Audit coverage and no-leakage gate for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - audit
  - leakage
  - owasp
---

# Decision: Audit Coverage And No-Leakage Gate

## Context

CloudIAM's access path makes high-value decisions: deny access, create approval requests, approve or deny requests, expire stale requests, issue AWS credentials, and report issuance failure. OWASP logging guidance is useful only when translated into those product events.

The Task 28 invariant register already requires durable, non-secret audit trails for security-relevant decisions. This decision defines the concrete coverage matrix and the leakage gate for the current application.

## Decision

Audit coverage is required for these CloudIAM events:

| Event | Source of truth | Required fields | Secret boundary | Current status |
|---|---|---|---|---|
| `access_denied` | Access service authorization decision | principal, target, decision, reason | no credentials or tokens | Covered by core tests. |
| `approval_requested` | Access service pending request creation | principal, target, provider, account, role, access request ID, pending decision | no user code, device code, JWT, idempotency key, or AWS credential | Covered by core tests. |
| `approval_approved` | Console approval mutation | access request ID, operator subject, idempotency-key hash, persisted target metadata | raw idempotency key and console JWT must not be stored | Covered by handler tests. |
| `approval_denied` | Console denial mutation | access request ID, operator subject, idempotency-key hash, persisted target metadata | raw idempotency key and console JWT must not be stored | Covered by handler tests. |
| `approval_expired` | Console mutation encountering an expired request | access request ID, operator subject, idempotency-key hash, persisted target metadata, expired reason | raw idempotency key and console JWT must not be stored | Covered by handler tests. |
| `access_issued` | Successful credential issuance | principal, target, provider, account, role, credential expiration; access request ID when issuance came from an approved request | AWS access key ID, secret access key, and session token must not be stored | Covered by core tests; Task 28 now preserves approved-request correlation. |
| `access_failed` | Credential issuer failure after authorization | principal, target, provider, account, role, generic failure reason; access request ID when failure came from an approved request | issuer error details and AWS credential material must not be stored | Covered by core tests. |

Device login approval and exchange remain governed by the hash-only token-storage and single-use session ADRs. They are not added to the current audit-event contract in this slice because the current audit schema is access-request oriented. Expanding audit to device authorization events should be a separate task if product requirements require operator-visible login audit history.

### No-Leakage Gate

Audit events and audit HTTP responses must not contain:

- raw device codes or user codes
- raw CloudIAM session tokens
- raw console JWTs
- passwords or password hashes
- raw idempotency keys
- AWS access key IDs, secret access keys, or session tokens
- database, Redis, signing, or deployment secrets

Allowed audit references are:

- stable principal and operator subjects
- target, provider, account, and role labels
- access request IDs
- idempotency-key hashes
- credential expiration timestamps
- non-secret, categorical failure reasons

Issuer failures must use a generic reason rather than the upstream error string, because SDK or provider errors can contain request metadata or credentials.

## Implementation Requirement

When a credential is issued from a claimed approved access request, `access_issued` and `access_failed` audit events must include that access request ID. Direct policy-allowed issuance has no access request ID.

## Verification

- `server/internal/core/access` tests assert deny, pending approval, direct issuance, approved-request issuance, and failure audit events.
- Approved-request issuance tests assert the emitted `access_issued` event carries `access_request_id`.
- Failure tests assert issuer error details containing credential-looking material do not enter the audit event.
- Console handler tests assert approval lifecycle events hash idempotency keys and avoid raw session or credential values.
- Audit handler tests assert serialized audit responses expose only the safe response fields.
- Postgres integration tests assert the audit schema has no forbidden secret columns.

## Consequences

- Repudiation coverage is now a concrete matrix tied to CloudIAM behavior rather than a generic OWASP checklist.
- Audit records are useful for incident investigation without becoming a second secret store.
- Device-login audit expansion is intentionally left as a scoped future decision instead of being mixed into access-request hardening.
