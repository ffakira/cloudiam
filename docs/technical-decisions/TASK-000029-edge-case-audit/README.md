# TASK-000029 Technical Decisions

Task: Edge-case audit.

Decisions:

- [001: Edge-case acceptance invariants](001-edge-case-acceptance-invariants.md)
- [002: Edge-case audit method and deferral rule](002-edge-case-audit-method-and-deferral-rule.md)
- [003: Edge-case audit slice register](003-edge-case-audit-slice-register.md)
- [004: Edge-case matrix and stop rule](004-edge-case-matrix-and-stop-rule.md)
- [005: CLI and gRPC edge-case regressions](005-cli-and-grpc-edge-case-regressions.md)
- [006: HTTP and dashboard edge-case regressions](006-http-and-dashboard-edge-case-regressions.md)
- [007: Persistence state-transition regressions](007-persistence-state-transition-regressions.md)
- [008: Config startup and shutdown regressions](008-config-startup-shutdown-regressions.md)
- [009: AWS credential issuance regressions](009-aws-credential-issuance-regressions.md)
- [010: Edge-case audit closure](010-edge-case-audit-closure.md)

Relevant prior decisions:

- `TASK-000028`: security hardening invariants, closure gate, and residual-risk report.
- `TASK-000003`: device-code login lifecycle and session storage.
- `TASK-000010`: config hot-reload behavior.
- `TASK-000016`: console auth and idempotent approval contract.
- `TASK-000017`: resource access request lifecycle.
- `TASK-000021`: idempotency retention.
- `TASK-000023`: approved-request claim-before-STS semantics.
- `TASK-000024`: cookie-backed console sessions.
