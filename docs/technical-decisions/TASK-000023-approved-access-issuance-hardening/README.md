# Technical Decisions For TASK-000023

Task: Approved access issuance hardening.

Decisions:

- [001: Approved request permission boundary](001-approved-request-permission-boundary.md)
- [002: Claim before STS issuance semantics](002-claim-before-sts-semantics.md)

Downstream decision watch:

- Revisit `TASK-000018` after this lands so approval-required config creates requests that pin the resolved permission boundary.
- Revisit `TASK-000016` after this lands so admin approve/deny APIs treat access request IDs as immutable approval objects.
