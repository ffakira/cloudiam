# Technical Decisions For TASK-000016

Task: Console auth and idempotent resource approvals.

Decisions:

- [001: Console JWT session contract](001-admin-jwt-session-contract.md)
- [002: Idempotent approval contract](002-idempotent-approval-contract.md)
- [003: Approval security and audit boundary](003-approval-security-and-audit-boundary.md)
- [004: Approval state transition persistence](004-approval-state-transition-persistence.md)

Reviewed against completed dependencies:

- `TASK-000017`: approve/deny operates on durable access request IDs and lifecycle statuses.
- `TASK-000023`: approval is for immutable persisted request metadata, including pinned `role_arn`; issuance may consume an approved request before STS succeeds.
- `TASK-000018`: target-level `approval_required` controls future request creation only and must not be re-evaluated during admin approve/deny.
