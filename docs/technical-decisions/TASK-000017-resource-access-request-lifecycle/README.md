# Technical Decisions For TASK-000017

Task: Resource access request lifecycle.

Decisions:

- [001: Access request lifecycle contract](001-access-request-lifecycle-contract.md)
- [002: Persistence schema and secret boundary](002-persistence-schema-and-secret-boundary.md)
- [003: Access API pending response](003-access-api-pending-response.md)

Downstream decision watch:

- Revisit persistence/issuance decisions through `TASK-000023` so approved requests pin the resolved role ARN before Task 18 creates real pending approvals.
- Revisit `TASK-000016` idempotency fingerprints after access request IDs and statuses are implemented.
- Revisit `TASK-000018` approval config to ensure it can drive the lifecycle without changing persistence shape.
