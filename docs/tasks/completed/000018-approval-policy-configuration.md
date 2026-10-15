---
title: "Approval policy configuration"
id: TASK-000018
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-08
completed_at: 2026-10-08
owner: codex
phase: "approval-1"
priority: P0
sprint_candidate: true
tags:
  - config
  - policy
  - approval
  - jit
---

# Task: Approval Policy Configuration

## Outcome

Make console approval requirements configurable in YAML so some targets or policies can require approval before JIT credentials are issued.

## Scope

- Add explicit approval configuration to access targets or policies.
- Validate approval fields in `validate-config`.
- Ensure hot reload treats approval rules as reloadable governance config.
- Make policy evaluation return whether approval is required.
- Keep default behavior backward-compatible: no approval unless configured.
- Preserve the immutable approved-request boundary from `TASK-000023`; approval config creates future requests, but approved issuance uses persisted request metadata.

## Not In Scope

- Approved access issuance hardening; covered by `TASK-000023`.
- Console auth and approval mutation APIs.
- Dashboard UI.
- Multi-step approval chains.
- Multiple approver groups.

## Dependency Watch

This task starts after `TASK-000023` pinned the approved request permission boundary. Approval-required config will create pending requests, so the request lifecycle must continue storing and issuing from immutable resolved target metadata.

## Technical Decisions

- [Approval configuration shape](../../technical-decisions/TASK-000018-approval-policy-configuration/001-approval-configuration-shape.md)
- [Approval decision evaluation](../../technical-decisions/TASK-000018-approval-policy-configuration/002-approval-decision-evaluation.md)
- [Hot reload and immutable request boundary](../../technical-decisions/TASK-000018-approval-policy-configuration/003-hot-reload-and-immutable-request-boundary.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000018-CONFIG-001 | YAML can mark a target or policy as approval-required. | Config load/validation tests cover valid config. |
| REQ-000018-DEFAULT-001 | Existing configs continue to allow direct issue when approval is not configured. | Existing access tests remain green. |
| REQ-000018-POLICY-001 | Policy evaluation exposes approval-required decisions to the access service. | Policy/access tests cover required vs not required. |
| REQ-000018-RELOAD-001 | Approval config changes hot reload safely with governance config. | Reload tests cover approval field change. |
| REQ-000018-BOUNDARY-001 | Approval-required config does not weaken the immutable approved-request issuance boundary from `TASK-000023`. | Hot-reload tests remain green with persisted request role ARN. |

## Result

Approval requirements are now configurable per access target with `approval_required`.

The field defaults to `false` when omitted, validates as a boolean through YAML decoding, and is reloadable governance config. Policy evaluation now exposes `ApprovalRequired` to the access service, so allowed requests to approval-required targets enter the pending approval lifecycle instead of direct credential issuance. Denied requests remain denied and do not create pending approvals.

The sample config documents `approval_required: false` without changing local demo behavior.

## Verification

- `make config-validate`
- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
- `make server-test-integration`
