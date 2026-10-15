---
title: "Replay and token lifecycle final regression for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - replay
  - sessions
  - idempotency
---

# Decision: Replay And Token Lifecycle Final Regression

## Context

S7 closes the `REQ-000028-REPLAY-001` pass for CloudIAM's current MVP replay boundaries:

- console approval/denial idempotency
- console session refresh
- device-code exchange
- approved access-request claiming
- stale or expired tokens

Earlier tasks already established the main contracts:

- Device authorization exchange is single-use and persists only hashes.
- Console approval idempotency keys are scoped to the console operator and request fingerprint.
- Approved access requests are claimed before STS issuance and cannot be claimed again.
- Console JWTs are cookie-backed, short-lived, signed with HMAC-SHA256, and externally returned only through `Set-Cookie`.

S7 verifies that those contracts still hold after the Task 28 hardening work and patches the concrete lifecycle gap found during the pass.

## Audit Result

| Boundary | Result | Action |
|---|---|---|
| Console approval idempotency replay | Existing store tests cover duplicate replay, mismatched fingerprint conflict, concurrent replay, and cross-action conflict. | Added HTTP regression proving replayed mutations set `Idempotency-Replayed: true` and do not emit duplicate approval audit events. |
| Console session refresh | Existing HTTP tests cover cookie rotation, no token in JSON body, wrong-role rejection, and no-body refresh. | Added service regression proving expired tokens cannot be refreshed. |
| Console session cookie lifetime | Refresh handler used an injectable auth clock but cookie `MaxAge` used process wall clock. In controlled-clock tests and deterministic runtimes this could emit an immediately stale cookie for an otherwise fresh session. | Fixed cookie `MaxAge` to use the handler clock and added a regression test. |
| Device-code exchange | Existing service and Postgres integration tests cover pending, denied, expired, consumed, not found, and concurrent single-use exchange. | No code change. |
| Approved access-request claim | Existing Postgres integration tests cover single-use claim, concurrent single-use claim, expired approved request rejection, and expiry-boundary race behavior. | No code change. |
| gRPC stale CloudIAM sessions | The gRPC access adapter resolves only active persisted sessions and rejects missing or invalid session tokens before invoking access issuance. | No code change. Existing adapter tests cover invalid session rejection. |

## Decision

Keep the current replay architecture:

- Device-code replay is prevented by the persisted `consumed` state and `FOR UPDATE` exchange transaction.
- Console mutation replay is allowed only for the exact same operator/key/fingerprint within the idempotency retention window.
- Console mutation replay must not create a second audit event, because the first mutation already recorded the security-relevant decision.
- Approved access-request replay is prevented by atomically moving one approved request to `issued` before STS credentials are minted.
- Console JWT refresh must verify the presented token before issuing a new token, and expired tokens must fail closed.
- Browser-visible session JSON must continue to omit raw JWTs; tokens travel only in `HttpOnly` cookies.

## Verification

- `go test ./internal/adapters/http ./internal/core/consoleauth`
- Existing integration tests cover:
  - `TestAuthStoreIntegration_ConcurrentExchangeApprovedDeviceAuthorizationIsSingleUse`
  - `TestAccessRequestStoreIntegration_ConcurrentClaimApprovedAccessRequestIsSingleUse`
  - `TestAccessRequestStoreIntegration_DoesNotClaimExpiredApprovedAccessRequest`
  - `TestAccessRequestStoreIntegration_MutateAccessRequestIsIdempotent`
  - `TestAccessRequestStoreIntegration_ConcurrentMutateAccessRequestIsIdempotent`
  - `TestAccessRequestStoreIntegration_ConcurrentMutateAccessRequestConflictsOnMismatchedFingerprint`

## Consequences

- The remaining replay work for Task 28 is closed for the current MVP scope.
- Long-lived token revocation lists, refresh-token families, and production SSO/OIDC replay protections remain out of scope for Task 28 and should be handled with a future identity-provider integration task.
