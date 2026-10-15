---
title: "Security control baseline for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - owasp
  - controls
  - leakage
---

# Decision: Security Control Baseline

## Context

CloudIAM issues short-lived AWS credentials after device login, session validation, policy checks, optional console approval, and audit recording. A security failure can therefore leak AWS credentials, allow an unauthorized principal to request access, approve the wrong request, replay a token, or erase the audit trail needed to investigate an incident.

`TASK-000028` already has an invariant register and concurrency hardening decisions. This decision defines the security control baseline that every remaining hardening slice should use when deciding what to test, what to reject, and what to split into production-hardening follow-up.

## Applicable OWASP Controls

This is the concrete CloudIAM applicability matrix for [OWASP Top 10:2021](https://top10.owasp.org/2021/), [OWASP API Security Top 10:2023](https://api-security.owasp.org/editions/2023/en/0x10-api-security-risks/), and OWASP cheat-sheet guidance for [authorization](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html), [authentication](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html), [session management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html), [logging](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html), and [secrets management](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html).

| OWASP risk or control | CloudIAM surface | Current status | Required action |
|---|---|---|---|
| Broken access control and broken object-level authorization | gRPC access issuance, console approval mutation, device approval, audit reads | Compliant in current hardening scope: server-side session resolution owns principal identity; clients cannot smuggle provider, account, role ARN, approval state, or operator subject; route/domain splits are documented in ADRs 001, 003, and 005. | Keep tests for anonymous, wrong-session, expired-session, wrong-role, and object-state attempts. |
| Broken function-level authorization | `/api/admin/...`, `/api/device/...`, gRPC access methods | Compliant in current scope: console approval requires console cookie session; browser bearer auth is rejected; device and resource approval are separate domains. | Keep route-boundary tests; do not add client-only authorization. |
| Identification, authentication, and session management failures | Console JWT cookie, CloudIAM session token, device codes | Compliant after this ADR update: console session is cookie-backed with `HttpOnly`, `SameSite=Lax`, bounded expiry, and configurable `Secure`; CloudIAM sessions are looked up by token hash; device/session tokens are generated with CSPRNG. | Set `CLOUDIAM_CONSOLE_COOKIE_SECURE=true` in TLS-terminated production deployments. |
| Cryptographic failures and secret management | Password hashes, JWT secret, device/session tokens, AWS credentials, database/Redis credentials | Compliant in current hardening scope: Argon2id is used for persisted console users; durable token lookup stores hashes; no real secret found in C6 scan; scanner tooling is pinned and passing; production mode rejects local/demo secret posture. | Deployment-specific external secret-manager wiring remains outside the MVP app. |
| Injection and mass assignment | JSON HTTP handlers, SQL stores, access request creation, approval mutation | Compliant in current scope: public JSON handlers cap body size and reject unknown fields; stores use parameterized SQL; privileged fields are server resolved or persisted. | Keep forbidden-field tests for every new mutation body. |
| Security misconfiguration and secure headers | Browser dashboard and HTTP APIs | Fixed in this ADR update: HTTP responses now set `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, and CSP; sensitive API paths set `Cache-Control: no-store`. | Revisit CSP if the dashboard later needs external assets or inline scripts. |
| Unrestricted resource consumption | Login/device/approval HTTP paths, request bodies, server lifecycle | Compliant in current scope: Redis-backed rate limiting with local fallback is implemented; request body caps exist; server has `ReadHeaderTimeout`; shutdown is bounded. | Production sizing and distributed rate-limit tuning remain outside Task 28. |
| Security logging and monitoring failures | Audit events and application logs | Compliant for MVP audit: approval, denial, expiration, issuance, and failure paths have durable non-secret audit contracts; idempotency keys are hashed in audit. | Production SIEM/export/alerting is a production-hardening follow-up. |
| SSRF and unsafe API consumption | AWS STS calls and any future outbound HTTP | Not currently exposed to user-controlled URLs; AWS STS target is SDK/config driven. | Any future user-controlled outbound URL feature must get a separate SSRF ADR before implementation. |
| Vulnerable and outdated components | Go module dependencies and scanner tooling | Compliant in current hardening scope: `make server-security-check` passes `go mod verify`, `go vet`, `go test ./...`, `go test -race ./...`, `govulncheck`, `gosec`, and `golangci-lint`. | Keep the gate in release readiness; `govulncheck` requires network access to refresh the vulnerability database. |

Status definitions:

- `Compliant` means the current application has code and tests or ADR-backed invariants for the control.
- `Fixed` means this ADR update included a code change to satisfy the control.
- `Partially compliant` means the app has baseline protection, but production-readiness tooling or operations work remains.
- `Not currently exposed` means no current trust boundary accepts that class of attacker input; adding such a feature requires a new ADR.

## Decision

CloudIAM will use defense in depth with the controls below.

### 1. Identity, Authentication, And Session Controls

- Treat every HTTP, gRPC, CLI, and browser request as untrusted until its session boundary is verified server-side.
- Accept console sessions only through the configured HttpOnly cookie path. Do not accept bearer console sessions from browser APIs.
- Accept CloudIAM access sessions only through the intended gRPC access exchange path.
- Generate tokens and verifiers with cryptographically secure randomness.
- Store only derived verifiers or hashes when durable lookup is needed.
- Enforce expiration, audience, subject, and status checks before privileged mutations.
- Fail closed for malformed, expired, revoked, canceled, or wrong-context sessions.

### 2. Authorization And Object Boundary Controls

- Authorize every privileged mutation at the server boundary and again in the core service where the decision matters.
- Use server-resolved named targets and persisted access-request fields; never trust client-submitted provider, account, role ARN, AWS credential, approval state, policy decision, or operator fields.
- Keep device authorization, console authentication, approval workflow, and AWS credential issuance as separate domains.
- Ensure object-level authorization: callers may mutate only the device authorization, access request, idempotency record, or audit-visible object allowed by their verified identity and route.
- Prefer allow lists over deny lists for target names, state transitions, roles, headers, and public response fields.

### 3. No-Leakage Controls

The following values are classified as secret material:

- Raw device codes and user codes.
- Raw CloudIAM session tokens.
- Raw console JWTs or session tokens.
- Passwords and password reset or bootstrap secrets.
- Raw idempotency keys.
- AWS access key IDs, secret access keys, session tokens, and console federation tokens.
- Signing keys, encryption keys, Redis credentials, database credentials, and other deployment secrets.

Secret material must not appear in:

- API responses, except the explicit CLI credential output path that returns AWS credentials to the authenticated requester.
- Audit records, idempotency records, errors, logs, metrics, traces, docs fixtures, screenshots, or scanner output.
- Browser local storage, session storage, URL parameters, route state, Redux-like stores, or HTML-rendered data.
- Durable database columns unless the column is explicitly a hash, verifier, or encrypted secret field accepted by an ADR.

Allowed references include opaque IDs, hashed tokens, hashed idempotency keys, stable subject IDs, target names, role references already governed by config, timestamps, status values, and non-secret error categories.

### 4. Input, Injection, And Smuggling Controls

- Decode JSON with request body limits on public HTTP handlers.
- Reject unknown or forbidden fields on privileged mutation requests when accepting structured JSON.
- Use typed parsing and validation for UUIDs, durations, statuses, targets, and enum-like values.
- Use parameterized SQL only; never concatenate untrusted input into SQL.
- Do not shell out with attacker-controlled strings. If a command is required later, pass arguments separately and avoid shell interpretation.
- Do not construct outbound AWS or HTTP requests from untrusted URLs unless an ADR adds SSRF controls.
- Return generic external errors for auth, authorization, and persistence failures; keep sensitive diagnostic detail only in non-secret internal logs.

### 5. Browser, Cookie, And Header Controls

- Store console session secrets only in HttpOnly cookies.
- Cookies that carry authenticated session state must set `HttpOnly`, `SameSite`, bounded `MaxAge` or `Expires`, and `Secure` in production TLS deployments.
- Browser APIs must return only safe session metadata and never raw session tokens.
- HTTP responses for browser-facing routes should include security headers appropriate to the deployment: content type protection, frame protection, referrer policy, cache controls for sensitive routes, and a Content Security Policy where feasible.
- The dashboard must not rely on client-side authorization for any privileged decision.

### 6. Replay, Rate Limit, And Resource-Use Controls

- Device codes, session tokens, idempotency keys, and approved requests must be scoped, expiring, and single-use where the product contract says they are single-use.
- Idempotency keys must bind to a stable mutation fingerprint and reject same-key different-fingerprint reuse.
- Rate limits must protect login, device approval, approval mutation, and credential issuance surfaces.
- Redis-backed rate limiting may fall back locally when Redis is unavailable, but fallback must remain bounded, concurrency-safe, and observable.
- Public handlers must enforce body-size, timeout, and state-machine limits to prevent cheap resource exhaustion.

### 7. Cryptography And Secret Storage Controls

- Use Go standard-library or vetted crypto primitives only; do not implement custom cryptography.
- Use Argon2id or an accepted password hashing scheme for persisted passwords.
- Compare secret verifiers with timing-safe comparison where practical.
- Load deployment secrets from environment or an accepted secret manager path, never hardcode production secrets.
- Keep secret lifetimes short where CloudIAM controls the lifetime, especially issued AWS credentials and local bootstrap credentials.

### 8. Audit, Logging, And Monitoring Controls

- Record durable audit events for authentication, authorization, approval, denial, expiration, issuance, and failure decisions that affect access.
- Logs and audit records must support investigation without exposing secrets.
- Log stable correlation identifiers or hashed identifiers, not raw credentials or tokens.
- Preserve audit integrity by writing audit events in the same control path as the security decision when feasible.
- Security-relevant failures should be observable enough to detect abuse, replay, and unusual state-transition attempts.

### 9. Dependency, Build, And Scanner Controls

- Keep `go.sum` committed and verify module checksums with `go mod verify`.
- Run `go test ./...`, `go vet ./...`, and `go test -race ./...` for security-relevant concurrency changes when feasible.
- Keep release-grade scanners pinned and run `make server-security-check` before production readiness.
- Treat vulnerable or abandoned dependencies on authentication, authorization, crypto, HTTP parsing, database access, and AWS issuance paths as security findings.

### 10. Threat Modeling And Severity Controls

- Use STRIDE to classify findings: spoofing, tampering, repudiation, information disclosure, denial of service, and elevation of privilege.
- Use OWASP-style risk thinking and DREAD-like prioritization when deciding fix order.
- Treat credential leakage, auth bypass, approval bypass, double issuance, and audit tampering as high or critical until proven otherwise.
- Treat broad product redesign findings as separate production-hardening tasks instead of hiding them inside Task 28 implementation.

## Verification Requirements

Every Task 28 code slice should choose the smallest useful set from this checklist:

- Unit tests for authentication, authorization, state transition, replay, and no-leakage behavior.
- Handler tests for anonymous, wrong-session, expired-session, wrong-role, forbidden-field, and oversized-body requests.
- Integration tests where Postgres enforces single-use, idempotency, expiration, or audit guarantees.
- Race tests for shared state, rate limiting, config snapshots, shutdown, and concurrent retries.
- Source scans for forbidden response fields, forbidden client-side token storage, and secret-like literals.
- Scanner runs for vulnerabilities, Go security checks, and lint once tooling is pinned or installed.

## Consequences

- Future Task 28 slices should cite this ADR when choosing security checks.
- The invariant register remains the product-specific contract; this ADR is the control baseline used to enforce it.
- The accepted no-leakage rule is stricter than ordinary application logging guidance because CloudIAM handles AWS credential material.
- Scanner tooling is pinned and passing; vulnerability scans still require network access to refresh upstream vulnerability data.
