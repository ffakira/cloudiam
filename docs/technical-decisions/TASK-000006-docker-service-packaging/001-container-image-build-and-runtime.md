---
title: "Container image build and runtime for TASK-000006"
type: decision
status: accepted
created_at: 2026-10-07
updated_at: 2026-10-07
owner: codex
applies_to:
  - TASK-000006
tags:
  - docker
  - golang
  - packaging
---

# Decision: Container Image Build And Runtime

## Sources

- [Docker Docs: Multi-stage builds](https://docs.docker.com/build/building/multi-stage/)
- [Docker Docs: Go language-specific guide](https://docs.docker.com/guides/golang/)

## Decision

Build the server image from `server/Dockerfile` using a multi-stage Go build:

- builder stage: official Go image
- runtime stage: distroless static Debian nonroot image
- binary: `/usr/local/bin/access-server`
- default config path: `/etc/cloudiam/cloudiam.yaml`
- default gRPC bind address inside the container: `0.0.0.0:50051`
- healthcheck command: `/usr/local/bin/access-server healthcheck --url http://127.0.0.1:8080/healthz`

The runtime image has no package manager, no shell, and no Go toolchain. It packages only the static server binary and the distroless base runtime material. It does not package the React client.

Compose may override the container user to the host UID/GID for local AWS profile readability. Production deployments should use the image default nonroot user and provider-native runtime identity where possible.

## Acceptance Criteria

- `make service-build` builds the image.
- The final image does not require the Go toolchain at runtime.
- The final image has no shell or package manager.
- The final image runs as nonroot by default.
- The final image declares a binary-based healthcheck.
- The server command can be overridden by Compose or `docker run`.
