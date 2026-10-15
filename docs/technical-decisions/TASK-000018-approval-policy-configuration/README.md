# Technical Decisions For TASK-000018

Task: Approval policy configuration.

Decisions:

- [001: Approval configuration shape](001-approval-configuration-shape.md)
- [002: Approval decision evaluation](002-approval-decision-evaluation.md)
- [003: Hot reload and immutable request boundary](003-hot-reload-and-immutable-request-boundary.md)

Downstream decision watch:

- Revisit `TASK-000016` after this lands so approve/deny APIs can rely on the final `approval_required` decision metadata.
- Revisit `TASK-000009` after this lands so the dashboard can display which targets require approval without exposing credential material.
