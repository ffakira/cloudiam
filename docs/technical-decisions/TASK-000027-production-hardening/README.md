# TASK-000027: Production Hardening

Task: [Production hardening](../../tasks/completed/000027-production-hardening.md).

## Decisions

- [001: Production readiness definition](001-production-readiness-definition.md)
- [002: Production hardening acceptance invariants](002-production-hardening-acceptance-invariants.md)
- [003: Production hardening slice register](003-production-hardening-slice-register.md)
- [004: Production config and startup contract](004-production-config-and-startup-contract.md)
- [005: Runtime readiness and lifecycle contract](005-runtime-readiness-and-lifecycle-contract.md)
- [006: Observability and runbook contract](006-observability-and-runbook-contract.md)
- [007: Production config and startup validation](007-production-config-and-startup-validation.md)
- [008: Health readiness and dependency checks](008-health-readiness-and-dependency-checks.md)
- [009: Runtime lifecycle and background work](009-runtime-lifecycle-and-background-work.md)
- [010: Observability checklist](010-observability-checklist.md)
- [011: Production runbook](011-production-runbook.md)
- [012: Closure gate and residual-risk report](012-closure-gate-and-residual-risk-report.md)

## Boundary

`TASK-000027` hardens the AWS-only MVP runtime for safe operation. It does not expand product scope into multi-account, multi-cloud, session recording, policy editing, production SSO/OIDC implementation, SIEM/export implementation, or a full vendor-grade operations platform.
