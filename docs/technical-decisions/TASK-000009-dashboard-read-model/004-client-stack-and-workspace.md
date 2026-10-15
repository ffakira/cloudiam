---
title: "Client stack and workspace for TASK-000009"
type: decision
status: accepted
created_at: 2026-10-08
updated_at: 2026-10-08
owner: codex
applies_to:
  - TASK-000009
tags:
  - dashboard
  - frontend
  - react
  - bun
---

# Decision: Client Stack And Workspace

## Decision

Create a top-level `client/` workspace for the MVP dashboard.

The client uses:

- Bun as the package manager and script runner.
- React 19.
- React Aria Components for accessible UI primitives.
- Zod, `@hookform/resolvers`, and React Hook Form for form validation.
- TanStack React Query for server state.
- TanStack React Router for client routing.
- Tailwind CSS v4 for styling.
- `tailwind-merge` for class merging where component variants need it.

## Rationale

The dashboard needs a real frontend foundation now that console approval mutations are authenticated and idempotent. Keeping the client as a top-level workspace avoids coupling React build tooling to the Go module while still letting the Go server serve the compiled assets later in the task.

Bun keeps install and local scripts fast for the MVP. React Query and TanStack Router keep API state and routing explicit without requiring a larger application framework. React Aria Components gives the dashboard accessible controls for forms, buttons, tabs, and future dialogs.

## Acceptance Criteria

- `client/package.json` defines Bun-compatible scripts.
- Client dependencies include React 19, React Aria Components, Zod, `@hookform/resolvers`, React Hook Form, TanStack React Query, TanStack React Router, Tailwind CSS v4, and `tailwind-merge`.
- The client can run locally with `bun run dev`.
- The client can build with `bun run build`.
- The dashboard remains a single-tenant MVP and does not introduce multi-tenant frontend assumptions.
