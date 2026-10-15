# Active Tasks

Use this folder for tasks currently in progress.

Each task must include metadata with:

- title
- id
- type
- status
- created_at
- updated_at
- owner
- phase
- tags

Use `type` for the primary work category:

- `feature`
- `epic`
- `fix`
- `docs`
- `chore`
- `decision`

## Requirement IDs

Implementation tasks should express deliverables as stable requirement IDs instead of vague checklist items.

Use this format:

```text
REQ-<task-id>-<area>-<number>
```

Example:

```text
REQ-000001-CORE-001
```

Each requirement should include:

- contract
- verification
- related files when known

This keeps tasks traceable from implementation, tests, reviews, and technical decisions.

Technical decisions should reference tasks by stable task ID, not by relative path, because tasks move from `tasks/active/` to `tasks/completed/`.

When a task is complete, move it to `docs/tasks/completed/` and update its status and completion date.

Suggested filename format:

```text
000001-short-title.md
```
