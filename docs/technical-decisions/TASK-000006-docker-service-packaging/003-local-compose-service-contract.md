---
title: "Local Compose service contract for TASK-000006"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000006
tags:
  - docker
  - compose
  - local-runtime
---

# Decision: Local Compose Service Contract

## Decision

Add a `cloudiam-server` Compose service for local MVP demos.

The service:

- builds from `server/Dockerfile`
- depends on healthy Postgres
- mounts `./server/configs/cloudiam.yaml` to `/etc/cloudiam/cloudiam.yaml:ro`
- mounts `${HOME}/.aws` to `/aws:ro`
- sets `HOME=/home/nonroot`
- runs as the host UID/GID for local demos
- uses a read-only filesystem with `/tmp` as tmpfs
- drops Linux capabilities
- sets `no-new-privileges`
- binds container gRPC on `50051`, with host port controlled by `CLOUDIAM_SERVICE_GRPC_PORT`
- binds container local/dev HTTP on `8080`, with host port controlled by `CLOUDIAM_SERVICE_HTTP_PORT`
- defaults the host gRPC port to `50051` so the CLI can keep using its internal service address
- defaults the host HTTP port to `18080` to avoid colliding with native local dev on `127.0.0.1:8080`
- uses the Compose-network Postgres URL: `postgres://cloudiam:cloudiam@postgres:5432/cloudiam?sslmode=disable`

## Acceptance Criteria

- `make service-up` starts Postgres, applies migrations, and starts the server.
- `make service-logs` follows server logs.
- `make service-down` stops the Compose stack.
- Local CLI can connect to the Docker service through `127.0.0.1:50051`.
- Local device approval uses the browser page at `http://127.0.0.1:18080/device` by default for the Docker service.
- Container healthcheck reaches `/healthz`.
