---
title: "Mandatory managed-role boundary for TASK-000013"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000013
tags:
  - aws
  - iam
  - governance
  - managed-roles
---

# Decision: Mandatory Managed-Role Boundary

## Decision

CloudIAM-supported AWS JIT roles must use the CloudIAM trust-policy contract.

For MVP, this means:

- hardened SourceIdentity conditions are mandatory for supported CloudIAM-managed/onboarded roles
- hardened session tag key/value conditions are mandatory for supported CloudIAM-managed/onboarded roles
- existing customer roles that do not satisfy the contract are out of the supported path until updated
- AWS is the enforcement boundary in this slice; malformed requests fail through `AssumeRole`

CloudIAM will not silently support arbitrary pre-existing role trust policies as first-class managed roles.

## Not In This Task

Task 13 does not build a policy scanner, drift detector, or automatic remediation flow.

Those capabilities belong in a separate onboarding/compliance task because they need:

- role discovery
- policy parsing
- explainable diff output
- customer-safe remediation prompts
- possibly account-wide read permissions

## Acceptance Criteria

- Task 13 templates express the mandatory role trust-policy contract.
- Documentation states that non-conforming roles are unsupported until updated.
- A backlog task exists for policy compliance/drift detection.
