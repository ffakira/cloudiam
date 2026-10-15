---
title: "Server config wiring for TASK-000002"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000002
tags:
  - server
  - configuration
  - dependency-injection
  - access-targets
---

# Decision: Server Config Wiring For TASK-000002

## Related Work

- `TASK-000002`: YAML config and named access targets.
- [002-config-model-and-validation.md](002-config-model-and-validation.md)
- [004-policy-evaluation.md](004-policy-evaluation.md)
- [005-core-access-flow.md](005-core-access-flow.md)

## Context

The server must use YAML as the governance source for named targets, principals, groups, policies, and AWS role ARNs.

`--role-arn` was useful for the first AWS proof, but it bypasses the named-target model and must not remain in the TASK-000002 server path.

## Decision

`access-server` accepts:

```bash
--config ./configs/cloudiam.yaml
```

Server startup loads and validates YAML before opening the listener.

At the app composition root, the loaded config is adapted into:

```text
policy evaluator -> access authorizer port
target resolver -> access target resolver port
AWS STS issuer -> access credential issuer port
```

AWS SDK configuration still comes from the default credential chain. The YAML AWS region is passed into the AWS SDK client configuration.

Runtime-specific identifiers, such as AWS account IDs, may be supplied through shell environment variables referenced by YAML. This keeps committed config generic while still producing a concrete role ARN before the server starts.

## Acceptance Criteria

- Server config path is required.
- Invalid config fails before the server listens.
- Server no longer accepts `--role-arn`.
- Access service is built with authorizer and target resolver.
- AWS role ARN comes from resolved target config, not CLI input.
- Unresolved or placeholder AWS account IDs/role ARNs fail config validation before AWS is called.
