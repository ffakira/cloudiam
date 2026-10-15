---
title: "Docker service packaging"
id: TASK-000006
type: feature
status: completed
created_at: 2026-10-07
updated_at: 2026-10-07
completed_at: 2026-10-07
owner: codex
phase: "packaging-1"
priority: P1
sprint_candidate: true
tags:
  - docker
  - deployment
  - mvp
---

# Task: Docker Service Packaging

## Outcome

Package the server as a Docker service for repeatable local demos without baking AWS credentials or local secrets into the image.

## Result

The server now builds as `cloudiam-server:local` with a distroless nonroot runtime, runs through Compose with mounted config and read-only AWS profile files, applies migrations before startup, and can issue access through the containerized gRPC service.

## Scope

- Build server image.
- Mount YAML config.
- Pass AWS credentials through the standard AWS SDK credential chain.
- Wire the service into local Compose alongside the existing Postgres/migrate services.
- Keep dashboard out of scope.

## Technical Decisions

- [Container image build and runtime](../../technical-decisions/TASK-000006-docker-service-packaging/001-container-image-build-and-runtime.md)
- [Runtime configuration and AWS credential boundary](../../technical-decisions/TASK-000006-docker-service-packaging/002-runtime-configuration-and-aws-credential-boundary.md)
- [Local Compose service contract](../../technical-decisions/TASK-000006-docker-service-packaging/003-local-compose-service-contract.md)

## Done When

| ID | Contract | Verification |
|---|---|---|
| REQ-000006-IMAGE-001 | Docker image builds the Go server binary with a multi-stage build and distroless nonroot runtime. | `make service-build` succeeds. |
| REQ-000006-RUNTIME-001 | Container starts with mounted YAML config at a stable in-container path. | Compose service command uses `/etc/cloudiam/cloudiam.yaml`; `make service-up` starts the service. |
| REQ-000006-AWS-001 | AWS credentials are available only at runtime through the AWS SDK credential chain. | Dockerfile contains no AWS secret `ARG`/`ENV`; Compose mounts host AWS config read-only and runs as host UID/GID for local demos. |
| REQ-000006-DB-001 | Container uses the Compose network database URL when Postgres-backed auth/audit is enabled. | Compose service sets `CLOUDIAM_DATABASE_URL` to `postgres:5432`. |
| REQ-000006-PORTS-001 | Docker service host ports are configurable while keeping CLI-compatible gRPC defaults. | Compose maps gRPC from `CLOUDIAM_SERVICE_GRPC_PORT` and HTTP from `CLOUDIAM_SERVICE_HTTP_PORT`, defaulting to `50051` and `18080`. |
| REQ-000006-DOCS-001 | README documents build/run commands and the credential boundary. | Server README includes Docker service usage. |

## Verification

- `make service-build`
- `CLOUDIAM_SERVICE_GRPC_PORT=15051 make service-up`
- Container HTTP endpoint check: `GET /audit/events?limit=1`
- Container healthcheck reaches `GET /healthz`.
- Docker image default user is nonroot UID `65532`.
- Docker image uses distroless static runtime; image history scan found no `apk add`, `/bin/sh`, or AWS credential fields.
- Safe AWS config diagnostic loads profile headers from `/aws` without printing credential values.
- Mounted YAML config readable inside container.
- Mounted AWS profile config readable inside container.
- Containerized gRPC access request reached AWS STS successfully; credential output was discarded.
- `env GOCACHE=/tmp/cloudiam-go-cache go test ./...`
