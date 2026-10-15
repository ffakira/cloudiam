---
title: "Client browser secret storage boundary for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - browser
  - secrets
  - sessions
---

# Decision: Client Browser Secret Storage Boundary

## Context

S8 verifies the browser client side of `REQ-000028-SECRET-001` and `REQ-000028-AUTH-001`.

The accepted console session model is cookie-backed:

- the server returns the raw console JWT only through an `HttpOnly` cookie
- the browser receives safe session metadata in JSON
- privileged console requests use `credentials: "include"`
- the client must not move JWTs, CloudIAM CLI session tokens, device codes, idempotency keys, or AWS credentials into browser-accessible state or storage

The browser still has two intentional transient form inputs:

- console login username/password
- device approval user code/principal

Those values can exist in controlled form fields while the user is submitting them, but they must not be persisted, cached as API response state, stored in Zustand, stored in Web Storage, or copied into explicit bearer-token plumbing.

## Audit Result

| Surface | Result | Action |
|---|---|---|
| Console session state | Zustand stores only `authenticated`, `subject`, `role`, and `expires_at`. | No secret fields added. |
| Console API calls | `fetch` uses `credentials: "include"` and does not construct `Authorization: Bearer ...` headers. | No code change. |
| Login mutation state | React Query mutation state could retain submitted `username` and `password` after the request. | Mutation state is reset after each login attempt; UI keeps only safe error text. |
| Device approval mutation state | React Query mutation state could retain submitted `user_code` after the request. | Mutation state is reset after each device approval attempt; success/error UI uses safe local status. |
| Access approval mutation | Idempotency keys are generated inside the request function and are not returned, persisted, or placed in component state. | Added build-time scanner coverage for idempotency-key fields. |
| Browser storage APIs | No current use of `localStorage`, `sessionStorage`, `indexedDB`, `document.cookie`, or Zustand persistence. | Added build-time scanner coverage. |
| API response types | Client response types do not include JWTs, CloudIAM session tokens, raw device codes, idempotency keys, or AWS credential fields. | Added build-time scanner coverage. |

## Decision

Keep the browser client as a cookie-only session consumer:

- Do not add browser-readable token storage.
- Do not add bearer-token auth for console routes.
- Do not persist Zustand session state.
- Do not add response fields for raw JWTs, CloudIAM session tokens, raw device codes, idempotency keys, or AWS credentials.
- Clear React Query mutation state after sensitive form submissions.

The client build now runs `security:browser-boundary` before TypeScript and Vite. The scanner fails if browser source introduces:

- `localStorage`, `sessionStorage`, or `indexedDB`
- `document.cookie`
- Zustand persistence middleware
- explicit `Authorization` or `Bearer` token plumbing
- secret-shaped token, device-code, idempotency-key, or AWS credential fields

## Verification

- `bun run security:browser-boundary`
- `bun run typecheck`
- `bun run build`

## Consequences

- Browser-accessible state remains limited to safe display metadata and transient form values.
- Any future client feature that needs browser-readable credentials must update this ADR and the scanner intentionally.
- The remaining Task 28 security work is the S9 closure gate and residual-risk report.
