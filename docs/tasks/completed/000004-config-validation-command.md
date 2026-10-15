---
title: "Config validation command"
id: TASK-000004
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "configuration-2"
priority: P1
sprint_candidate: true
tags:
  - cli
  - configuration
  - validation
  - mvp
---

# Task: Config Validation Command

## Outcome

Completed a local config validation command:

```bash
go run ./cmd/access-server validate-config --config ./configs/cloudiam.yaml
```

Also added:

```bash
make config-validate
```

The command validates YAML using the existing loader/validation path and does not start gRPC/HTTP servers or load AWS SDK config.

## Original Outcome

Add a local command that validates CloudIAM YAML without starting the server or calling AWS.

## Target Command

```bash
go run ./cmd/access-server validate-config --config ./configs/cloudiam.yaml
```

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000004-CLI-001 | `access-server validate-config --config <path>` validates YAML without starting gRPC/HTTP servers. | CLI/app test covers command path and no listener setup. |
| REQ-000004-CONFIG-001 | Valid config exits successfully and prints a concise success message. | Command test covers valid fixture. |
| REQ-000004-CONFIG-002 | Missing env-expanded AWS account ID fails clearly. | Command test covers missing env var placeholder. |
| REQ-000004-CONFIG-003 | Invalid YAML fails clearly. | Command test covers parse error. |
| REQ-000004-CONFIG-004 | Secret-looking fields fail clearly. | Command test covers forbidden secret key. |
| REQ-000004-AWS-001 | Validation command makes no AWS API/config call. | Implementation calls YAML loader only; test uses invalid AWS env and no AWS mocks. |
