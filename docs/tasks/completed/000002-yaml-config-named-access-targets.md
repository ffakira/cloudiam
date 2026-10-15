---
title: "YAML config and named access targets"
id: TASK-000002
type: feature
status: completed
created_at: 2026-10-06
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "configuration-1"
current_focus: "Ory-style config -> named access target -> policy check -> AWS STS"
outcome: "CLI successfully requested AWS access through named YAML target resolution, policy authorization, gRPC, and AWS STS without exposing provider/account/role request fields."
tags:
  - yaml
  - configuration
  - access-targets
  - policy
  - aws
  - cli
  - mvp
  - security
---

# Task: YAML Config And Named Access Targets

## Outcome

Make the CLI user-facing by replacing internal AWS request fields with named access targets.

Target user flow:

```bash
cloudiam access request aws-sandbox-readonly --principal akira
```

Server flow:

```text
target name -> YAML config -> policy check -> AWS role ARN -> STS AssumeRole
```

## Context

`TASK-000001` proved the technical path:

```text
CLI -> gRPC -> AWS STS -> temporary credentials
```

But the CLI still exposes internal fields:

```text
provider, account, role, duration
```

This task introduces Ory-style file configuration so businesses can define named access targets and policies.

Security review notes:

- Authorization must be checked on every access request.
- Requests must fail closed by default.
- YAML is governance/config data, not a place for AWS secrets.
- CLI request fields must not allow users to smuggle provider/account/role/ARN values around the named-target policy model.

Current status:

- Section 1 is implemented: config model, YAML loader, validation, example config, and tests.
- Section 2 is implemented: access target resolution and duration enforcement.
- Section 3 is implemented: core policy evaluator.
- Section 4 is implemented: core access service combines authorization, target resolution, and issuer call.
- Section 5 is implemented: gRPC request contract carries target instead of provider/account/role.
- Section 6 is implemented: CLI request command uses positional target and no user-facing service address flag.
- Section 7 is implemented: server loads YAML config and wires policy + target resolution into access service.
- Section 8 is implemented: legacy core provider/account/role request fields are removed.

## Scope

- Add server `--config` flag.
- Load YAML config from disk.
- Define named access targets.
- Define principals, groups, and policies.
- Map target name to AWS role ARN.
- Reject unauthorized principals before AWS call.
- Reject unknown targets before AWS call.
- Keep credentials unpersisted.
- Keep database/dashboard out of scope.
- Treat the CLI as a client for the hosted CloudIAM service.

## Technical Decisions

- [Technical contract for TASK-000002](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/001-contract.md)
- [Config model and validation](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/002-config-model-and-validation.md)
- [Access target resolution](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/003-target-resolution.md)
- [Policy evaluation](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/004-policy-evaluation.md)
- [Core access flow](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/005-core-access-flow.md)
- [gRPC target request contract](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/006-grpc-target-contract.md)
- [CLI command contract](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/007-cli-command-contract.md)
- [Server config wiring](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/008-server-config-wiring.md)
- [Remove legacy access request fields](../../technical-decisions/TASK-000002-yaml-config-named-access-targets/009-remove-legacy-access-request.md)

## Non-Goals

- No database.
- No dashboard.
- No Docker packaging yet.
- No multi-cloud provider implementation.
- No approval workflow.
- No audit persistence.
- No policy editor.
- No hot reload.
- No CLI login/session flow. That is tracked by `TASK-000003`.
- No user-facing endpoint/address flag.

## Implementation Contract

Each requirement has a stable ID.

Requirement ID format:

```text
REQ-000002-<AREA>-<NUMBER>
```

### Config Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000002-CONFIG-001 | Add `--config` to `access-server`. | Server accepts config path. |
| REQ-000002-CONFIG-002 | Load YAML config from disk. | Valid config starts server. |
| REQ-000002-CONFIG-003 | Fail fast when config path is missing. | Server exits before listening. |
| REQ-000002-CONFIG-004 | Fail fast when config is invalid. | Invalid config returns clear error. |
| REQ-000002-CONFIG-005 | Do not add `.env`, Viper, or database-backed config. | Review confirms absence. |
| REQ-000002-CONFIG-006 | Config includes tenant/company identifier. | Example config includes `tenant`. |
| REQ-000002-CONFIG-007 | Config rejects unknown top-level sections/fields. | Unit test covers unknown field. |
| REQ-000002-CONFIG-008 | Config must not contain AWS access keys, secret keys, or session tokens. | Unit test covers secret-looking fields. |

### Access Target Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000002-TARGET-001 | Define named access targets in config. | Example config includes `aws-sandbox-readonly`. |
| REQ-000002-TARGET-002 | CLI request uses target name instead of provider/account/role flags. | CLI command accepts positional target. |
| REQ-000002-TARGET-003 | Unknown target is rejected before AWS call. | Unit test proves target resolution fails before issuer exists. |
| REQ-000002-TARGET-004 | Target maps to provider/account/role and AWS role ARN. | Unit test resolves target to role ARN. |

### Policy Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000002-POLICY-001 | Config defines principals and groups. | Example config includes `akira` in `developers`. |
| REQ-000002-POLICY-002 | Config defines policies that allow groups to request targets. | Example config allows `developers` to request `aws-sandbox-readonly`. |
| REQ-000002-POLICY-003 | Unauthorized principal is denied before AWS call. | Unit test proves issuer is not called. |
| REQ-000002-POLICY-004 | Missing principal is denied before AWS call. | Unit test proves issuer is not called. |
| REQ-000002-POLICY-005 | Requests fail closed by default. | No matching policy means denied. |
| REQ-000002-POLICY-006 | Authorization is checked on every access request. | Handler/service path always evaluates policy before issuer. |
| REQ-000002-POLICY-007 | Principal and target are evaluated inside the same tenant/company config. | Unit test covers tenant mismatch or missing tenant context. |

### Duration Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000002-DURATION-001 | Target defines `default_duration` and `max_duration`. | Example config includes both. |
| REQ-000002-DURATION-002 | CLI can omit duration and use target default. | Target resolver uses default duration. |
| REQ-000002-DURATION-003 | CLI can request duration override. | Target resolver accepts duration within max duration. |
| REQ-000002-DURATION-004 | Duration above max is denied before AWS call. | Unit test proves target resolution fails before issuer exists. |

### CLI Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000002-CLI-001 | CLI command shape is `access request <target>`. | Command works with target positional arg. |
| REQ-000002-CLI-002 | CLI no longer requires provider/account/role flags. | Those flags are removed or ignored for this command. |
| REQ-000002-CLI-003 | CLI still prints shell exports on success. | Output remains compatible with `eval`. |
| REQ-000002-CLI-004 | CLI does not expose a user-facing service address flag. | User command has no `--addr`, `--server`, or `--host`. |
| REQ-000002-CLI-005 | CLI request does not accept provider/account/role/role ARN in named-target flow. | Command rejects or lacks those flags. |

### gRPC Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000002-GRPC-001 | gRPC request accepts target name instead of provider/account/role. | Proto request has `target` and no provider/account/role fields. |
| REQ-000002-GRPC-002 | gRPC handler forwards principal, target, and duration to core service. | Unit test or code review covers adapter mapping. |
| REQ-000002-GRPC-003 | gRPC handler does not accept role ARN from clients. | Proto request has no role ARN field. |

### Security Contract

| ID | Contract | Verification |
|---|---|---|
| REQ-000002-SEC-001 | No AWS credentials or CloudIAM tokens are stored in YAML config. | Config tests reject secret-looking keys. |
| REQ-000002-SEC-002 | Role ARNs may appear in YAML because they are identifiers, not secrets. | Example config includes role ARN only. |
| REQ-000002-SEC-003 | Denied requests do not call AWS. | Unit tests use fake issuer call count. |
| REQ-000002-SEC-004 | Denial messages do not expose credentials or hidden policy internals. | Error string review/test. |
| REQ-000002-SEC-005 | OWASP authorization and input-validation guidance is reflected in tests. | Tests cover deny-by-default, unknown target, unknown principal, unknown config field. |

## Expected Commands

Hosted service behavior:

```text
CLI -> hosted CloudIAM service -> AWS STS
```

Product CLI:

```bash
cloudiam access request aws-sandbox-readonly
```

Current local development server:

```bash
CLOUDIAM_AWS_ACCOUNT_ID=<AWS_ACCOUNT_ID> \
AWS_PROFILE=cloud-iam-broker \
go run ./cmd/access-server \
  --config ./configs/cloudiam.yaml
```

Current local development CLI:

```bash
go run ./cmd/access-cli \
  access request aws-sandbox-readonly \
  --principal akira
```

Optional duration override:

```bash
go run ./cmd/access-cli \
  access request aws-sandbox-readonly \
  --principal akira \
  --duration 900
```

## Tests And Verification

Automated:

- [x] valid config loads
- [x] invalid config fails
- [x] config with unknown top-level field fails
- [x] config with AWS access key/secret-like fields fails
- [x] unknown target reference fails config validation
- [x] unknown target request fails target resolution
- [x] unknown principal denied before issuer call
- [x] unauthorized principal denied before issuer call
- [x] invalid duration fails config validation
- [x] duration above max denied before issuer call
- [x] valid target resolves default duration
- [x] valid target resolves duration override
- [x] valid principal + target calls issuer once
- [x] gRPC request carries target instead of provider/account/role
- [x] provider/account/role/role ARN request flags are not accepted in named-target command
- [x] CLI request adapter sends target to gRPC
- [x] server app requires config path
- [x] server app builds access service with authorizer and target resolver
- [x] core access request has no provider/account/role/role ARN fields

Manual:

- [x] server starts with `--config ./configs/cloudiam.yaml`
- [x] CLI can request `aws-sandbox-readonly`
- [x] returned credentials still resolve to assumed-role session
- [x] user-facing CLI examples do not include service address flags

## Done When

- [x] CLI requests a named access target.
- [x] Server resolves the target from YAML config.
- [x] Server authorizes principal through config policy.
- [x] Server calls AWS only after policy allows the request.
- [x] Credentials still return as shell exports.
- [x] No database, dashboard, or Docker packaging is added.
- [x] Endpoint/login behavior is deferred to `TASK-000003`.

## Backlog Migration

Follow-up work moved to [tasks/backlog](../backlog/) and the active `TASK-000003` login task.
