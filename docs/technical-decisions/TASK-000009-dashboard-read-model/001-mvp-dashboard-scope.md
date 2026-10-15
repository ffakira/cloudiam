---
title: "MVP dashboard scope for TASK-000009"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000009
tags:
  - dashboard
  - frontend
  - mvp
---

# Decision: MVP Dashboard Scope

## Decision

Build a small local/demo admin dashboard served by the existing HTTP server.

The dashboard supports:

- user self-service device-login approval
- viewing pending resource/JIT access requests
- approving or denying resource/JIT access requests through secured console APIs
- viewing recent audit events
- seeing basic configured access targets/policy metadata if it fits the slice

The dashboard does not edit governance config in this task. YAML remains the source of truth for principals, targets, and policies.

## Rationale

The demo needs two moments to be visible: the user approves their own device login, and an admin approves resource/JIT access when required. This creates a coherent product story without adding full multi-tenant admin identity, a hosted frontend, or policy editing.

Serving the dashboard from the existing Go HTTP server keeps deployment simple for the one-week MVP and works naturally with Docker Compose.

## Acceptance Criteria

- Dashboard is available from the local/dev HTTP server.
- Dashboard can complete the user device-login approval loop.
- Dashboard can surface pending resource approvals.
- Dashboard reads durable audit history.
- No frontend route exposes raw credentials or bearer tokens.
