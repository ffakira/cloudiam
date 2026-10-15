---
title: "Claim before STS issuance semantics for TASK-000023"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000023
tags:
  - approval
  - concurrency
  - retry
  - sts
---

# Decision: Claim Before STS Issuance Semantics

## Decision

Keep the MVP behavior where an approved access request is atomically claimed and marked `issued` before the service calls AWS STS.

Document this as a deliberate concurrency tradeoff: a failed STS call after the claim consumes the approved request. A later task may introduce an `issuing` state, retry leases, or an outbox-style issuance flow if operational retries become required.

## Rationale

The immediate MVP risk is double issuing credentials from the same approved request under concurrent client retries. Claiming first gives the access service a simple single-use guarantee with one SQL update.

## Consequences

- Task 16 idempotent approval APIs must not assume that an approved request remains reusable after the user retries access.
- Audit should continue recording failed issuance attempts without storing credential material.
- A future retry-safe flow should be explicit rather than accidentally changing this single-use contract.
