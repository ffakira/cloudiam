---
title: "Runtime configuration and AWS credential boundary for TASK-000006"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000006
tags:
  - docker
  - aws
  - security
  - configuration
---

# Decision: Runtime Configuration And AWS Credential Boundary

## Sources

- [Docker Docs: Build secrets](https://docs.docker.com/build/building/secrets/)
- [Docker Docs: Build variables](https://docs.docker.com/build/building/variables/)

## Decision

Do not pass AWS credentials through Dockerfile `ARG` or `ENV`.

For local demos, Compose mounts the developer AWS configuration directory read-only into the runtime container and sets `AWS_PROFILE`. This keeps broker credentials in the standard AWS SDK credential chain while avoiding credentials in image layers, image metadata, or source-controlled config.

Because AWS config and credential files are usually `0600`, the local Compose service runs with the host UID/GID supplied by `make`. That lets the container read the mounted profile without loosening file permissions.

Compose also sets the standard AWS SDK file path variables explicitly:

- `AWS_SDK_LOAD_CONFIG=1`
- `AWS_CONFIG_FILE=/aws/config`
- `AWS_SHARED_CREDENTIALS_FILE=/aws/credentials`

This keeps distroless runtime profile discovery deterministic without copying credentials into the image.

Runtime configuration is split:

- YAML config is mounted read-only at `/etc/cloudiam/cloudiam.yaml`.
- Non-secret runtime environment values come from `.env.local`.
- AWS account ID may be supplied through `.env.local` for YAML expansion.
- AWS credentials remain outside the image.

## Acceptance Criteria

- Dockerfile contains no AWS credential fields.
- `.dockerignore` excludes `.env*`, `.aws`, `.git`, and local build output.
- Compose mounts `${HOME}/.aws` read-only for local demos.
- Compose runs the service as the host UID/GID for local credential-file readability.
- Compose uses explicit AWS SDK config and credentials file paths.
- Compose mounts AWS files at `/aws` instead of under `/home/nonroot`, because the distroless nonroot home is not traversable by arbitrary host UID overrides.
- README states that real credentials must not be committed or baked into images.
