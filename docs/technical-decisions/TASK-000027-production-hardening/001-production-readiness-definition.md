---
title: "Production readiness definition for TASK-000027"
type: decision
status: accepted
created_at: 2026-10-12
updated_at: 2026-10-12
owner: codex
applies_to:
  - TASK-000027
tags:
  - production
  - reliability
  - operations
  - scope
---

# Decision: Production Readiness Definition

## Context

`TASK-000027` can become unbounded if "production ready" means every capability expected from a mature Cloud IAM platform.

CloudIAM is still an AWS-only MVP. The production hardening pass should prove that the current runtime can be operated safely with explicit limitations. It should not become multi-cloud expansion, a full identity-provider integration, a SIEM product, or a complete dashboard platform.

## Definition

For this project, production ready means:

> CloudIAM can run in a real environment, fail safely under expected dependency and configuration problems, expose enough operational signals to diagnose incidents, and avoid insecure defaults that were acceptable only for local/demo use.

This definition applies to the current AWS-only MVP surface:

- CLI device login and access request flow.
- gRPC access API.
- HTTP dashboard/device/console APIs.
- Postgres-backed sessions, access requests, idempotency records, console users, and audit events.
- Redis-backed rate limiting when production mode requires shared counters.
- AWS STS credential issuance.
- Runtime config validation and `SIGHUP` reload behavior.
- Startup cleanup and graceful shutdown behavior.

## Decision

Treat `TASK-000027` as a production safety and operations pass, not a product expansion pass.

The task should close when these production readiness properties are true for the current MVP:

1. Unsafe production configuration fails closed at startup.
2. Liveness and readiness are distinct and useful.
3. HTTP, gRPC, database, Redis, cleanup jobs, and config reload have predictable lifecycle behavior.
4. Operators can diagnose critical auth, authorization, approval, STS, audit, reload, rate-limit, and cleanup failures without secret leakage.
5. Local/demo conveniences are blocked, disabled, or made explicitly non-production in production mode.
6. A runbook exists for startup, readiness, reload, rollback, and incident triage.
7. Residual risks are documented with clear triggers for future implementation.

## Not Required For Production Readiness In This Task

The following are valuable, but not required for `TASK-000027` closure:

- Multi-account support.
- Multi-cloud provider support.
- Production SSO/OIDC implementation.
- External secret-manager implementation.
- SIEM/export pipeline implementation.
- Session recording.
- Password delivery or password rotation.
- Threat analytics or user behavior analytics.
- Full browser E2E suite.
- Full load, chaos, or disaster-recovery program.
- Dashboard product expansion or policy editor.

## Consequences

- Task 27 can say "not now" to broad platform ideas while still being honest about production gaps.
- Implementation should focus on runtime safety, verification, and operator guidance.
- Product expansion must become later backlog work instead of being hidden inside the hardening pass.
