# TASK-000028 Technical Decisions

Task: Security hardening.

Decisions:

- [001: Production security invariant register](001-production-security-invariant-register.md)
- [002: Security hardening audit plan](002-security-hardening-audit-plan.md)
- [003: Stale decision reconciliation](003-stale-decision-reconciliation.md)
- [004: Redis-backed rate limiting with local fallback](004-redis-backed-rate-limiting.md)
- [005: Concurrency and context security invariants](005-concurrency-and-context-security.md)
- [006: Concurrency hardening implementation slices](006-concurrency-hardening-slices.md)
- [007: Security control baseline](007-security-control-baseline.md)
- [008: Production secret and config boundary](008-production-secret-and-config-boundary.md)
- [009: Audit coverage and no-leakage gate](009-audit-coverage-and-no-leakage.md)
- [010: Privileged route auth boundary](010-privileged-route-auth-boundary.md)
- [011: Request smuggling and input boundary](011-request-smuggling-and-input-boundary.md)
- [012: Security hardening slice register](012-security-hardening-slice-register.md)
- [013: Secret log and error-output audit](013-secret-log-and-error-output-audit.md)
- [014: Replay and token lifecycle final regression](014-replay-and-token-lifecycle-final-regression.md)
- [015: Client browser secret storage boundary](015-client-browser-secret-storage-boundary.md)
- [016: Closure gate and residual-risk report](016-closure-gate-and-residual-risk-report.md)

Relevant prior decisions:

- `TASK-000003`: device authorization status, single-use exchange, and hash-only token storage.
- `TASK-000008`: persistence and secret-storage boundary.
- `TASK-000013`: mandatory AWS managed-role trust-policy boundary.
- `TASK-000014`: non-secret AWS policy scanner output.
- `TASK-000016`: console approval authentication, idempotency, and audit boundary.
- `TASK-000019`: browser device approval page route and secret boundary.
- `TASK-000020`: approval lifecycle audit event contract.
- `TASK-000023`: approved-request permission boundary and claim-before-STS semantics.
- `TASK-000024`: HttpOnly cookie-backed console sessions.
- `TASK-000025`: `consoleauth` and `approval` domain split.
- `TASK-000026`: persisted console users and Argon2id password hashing.
