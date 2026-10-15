---
title: "Redis-backed rate limiting with local fallback for TASK-000028"
type: decision
status: accepted
created_at: 2026-10-10
updated_at: 2026-10-10
owner: codex
applies_to:
  - TASK-000028
tags:
  - security
  - rate-limiting
  - redis
  - abuse-resistance
---

# Decision: Redis-Backed Rate Limiting With Local Fallback

## Context

`TASK-000028` requires abuse resistance for unauthenticated and security-sensitive routes. Device approval and console login are especially sensitive because attackers can repeatedly guess user codes or credentials before a trusted session exists.

Process-local rate limits are useful for development and single-instance fallback, but they do not coordinate across multiple server replicas. Production deployments need a shared counter.

## Related Work

- `TASK-000003`: device authorization codes are single-use and stored only as hashes.
- `TASK-000024`: console sessions are cookie-backed and server-owned.
- `TASK-000026`: console passwords use Argon2id hashes.
- `TASK-000028`: security hardening requires request-size limits, replay protection, and abuse resistance.

## Options

- Local-only fixed-window rate limits. Simple and dependency-free, but ineffective across multiple replicas.
- Postgres-backed counters. Durable, but high-write and not ideal for hot-path abuse throttling.
- Redis-backed counters with local fallback. Shared across replicas when Redis is available and still protective in local/dev or Redis outage scenarios.

## Decision

Use Redis-backed fixed-window counters for HTTP abuse throttling when `CLOUDIAM_REDIS_URL` is configured.

If Redis is not configured or a Redis operation fails, fall back to an in-process fixed-window limiter. Fallback is intentionally not a hard production replacement for Redis; it preserves partial protection and availability during local development or transient Redis outages.

Rate-limit keys must be server-derived and must not include raw user codes, passwords, tokens, idempotency keys, or AWS credentials. The initial key shape is route plus client remote address.

Rate-limit responses must be generic `429 Too Many Requests` responses and must not echo submitted credentials, codes, principals, passwords, tokens, or client IPs.

## Consequences

- Production deployments should configure `CLOUDIAM_REDIS_URL` so rate limits coordinate across server replicas.
- Local development works without Redis, but local fallback limits only the current server process.
- Redis outages fail over to local limiting instead of disabling the route or exposing raw backend errors.
- Future slices should apply the same limiter to console login and any other unauthenticated mutation routes.
