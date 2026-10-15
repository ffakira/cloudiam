---
title: "Device approval edge-case matrix for TASK-000019"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000019
tags:
  - device-login
  - testing
  - edge-cases
  - frontend
---

# Decision: Device Approval Edge-Case Matrix

## Context

Task 19 is more than rendering a form. Device login has two participants: a CLI polling for session issuance and a browser user approving a code. The browser path must report clear states without exposing secrets.

## Decision

Task 19 implementation and verification should cover this matrix:

| Case | API result | UI expectation | Secret boundary |
|---|---|---|---|
| Valid pending code and principal | `200 OK` | Shows approved state and clears form | No token or hash values rendered |
| Missing user code | `400 Bad Request` | Shows field-level error | No submitted principal leaked into unrelated UI |
| Missing principal | `400 Bad Request` | Shows field-level error | No code hash or token details |
| Unknown user code | `404 Not Found` | Shows invalid/not found state | No lookup internals |
| Expired authorization | `410 Gone` | Shows expired state | No device code hash or session token |
| Already approved/consumed/denied | `409 Conflict` | Shows cannot approve/invalid state | No session token |
| Raw `POST /device/approve` | `410 Gone` | Directs user to `/device` | Does not call approval service |
| Console resource approval route | Admin-only API | No overlap with device approval UI | Does not accept user code |

## Additional Edge Cases

- The page should be reachable by direct navigation and browser refresh.
- The app should not submit while an approval request is already pending.
- The form should trim user code and principal consistently with server behavior.
- JSON bodies with unknown fields should be rejected by the API.
- Non-POST methods for approval should return method errors on the API route.
- Error messages should be specific enough for demo use but must not reveal token hashes or raw session material.

## Verification Expectations

Use a mix of tests:

- HTTP handler tests for API status mapping and deprecated route behavior.
- Frontend build or smoke test for the page route and form wiring.
- Secret-absence assertions against API and rendered error/success states.

## Consequences

- Existing task 9 dashboard behavior can satisfy much of task 19, but task 19 should remain active until its specific edge-case coverage is verified.
- Task 22's route deprecation is part of the task 19 boundary, because the browser page is now the only demo path for device approval.
