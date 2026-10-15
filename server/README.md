# Cloud IAM MVP Server

This module implements the Cloud IAM MVP server slice: CLI to gRPC to YAML policy/target resolution to AWS STS.

## Run Server

```bash
CLOUDIAM_AWS_ACCOUNT_ID=<AWS_ACCOUNT_ID> \
AWS_PROFILE=<BROKER_AWS_PROFILE> \
go run ./cmd/access-server \
  --config ./configs/cloudiam.yaml
```

For local auth/device-login work, prefer running from the repository root so `.env.local` is loaded:

```bash
make server-run
```

## Request Access

Login first:

```bash
go run ./cmd/access-cli login
```

Open the verification URI printed by the CLI, normally `http://127.0.0.1:8080/device`, and approve the user code from the browser page.

The old raw JSON `POST /device/approve` path is deprecated and does not approve devices in the demo path. The browser page submits through `POST /api/device/approve`.

Then request access by target name:

```bash
go run ./cmd/access-cli \
  access request aws-sandbox-readonly \
  --duration 900
```

By default, the CLI prints a non-secret success summary. To intentionally print shell exports for the temporary AWS credentials:

```bash
go run ./cmd/access-cli \
  access request aws-sandbox-readonly \
  --duration 900 \
  --format env
```

List recent local/dev audit events:

```http
GET http://127.0.0.1:8080/audit/events?limit=50
```

Audit events record governance metadata such as principal, target, decision, reason, provider, account, role, access request ID, console operator subject, hashed idempotency-key reference, and credential expiration. They do not store AWS secret access keys, AWS session tokens, CloudIAM session tokens, raw console JWTs, raw idempotency keys, or raw device/user codes.

The current MVP is AWS-only. The audit schema already carries provider and account metadata so a reviewed request can be traced precisely, but multiple AWS accounts and non-AWS providers remain backlog work rather than supported runtime modes.

## AWS Traceability

Access requests set AWS STS `SourceIdentity` from the CloudIAM principal and pass non-secret CloudIAM session tags:

- `cloudiam_principal`
- `cloudiam_target`
- `cloudiam_provider`
- `cloudiam_account`
- `cloudiam_role`

The broker permission policy and JIT role trust policy must allow:

- `sts:AssumeRole`
- `sts:SetSourceIdentity`
- `sts:TagSession`

If either policy is missing `sts:SetSourceIdentity` or `sts:TagSession`, AWS will reject the assume-role request.

CloudIAM-supported AWS JIT roles must also use the hardened trust-policy conditions from `TASK-000013`: `SourceIdentity` must match `cloudiam-*`, session tag keys must be limited to the CloudIAM tag set, `cloudiam_provider` must equal `aws`, and transitive session tags must be absent. Keep SourceIdentity and session-tag conditions in separate trust-policy statements so AWS can authorize `sts:SetSourceIdentity` and `sts:TagSession` independently.

## Local Database

From the repository root:

```bash
cp .env.example .env.local
make db-up
make redis-up
make migrate-up
make migrate-status
```

Run database-backed integration tests:

```bash
make server-test-integration
```

This target applies pending migrations before running the Postgres integration tests.

The gRPC server registers auth/device-login methods when `CLOUDIAM_DATABASE_URL` is set.
In that database-backed mode, startup verifies the required persistence schema for device authorizations, sessions, audit events, access requests, idempotency records, and console users. If migrations are missing, the server fails fast with a clear migration error instead of accepting requests with a partial persistence setup.

Console approve/deny idempotency records use a 24-hour MVP replay window. Startup removes expired idempotency records where `expires_at <= cleanup_time`, before the server accepts requests. Cleanup deletes only idempotency rows; access requests and audit events remain durable.

HTTP abuse throttling uses Redis when `CLOUDIAM_REDIS_URL` is set. If Redis is unset or unavailable at request time, the server falls back to an in-process fixed-window limiter so local development and transient Redis outages still retain partial protection. Production deployments should configure Redis so limits are shared across server replicas.

Browser-facing HTTP responses include baseline security headers. Console session cookies are `HttpOnly` and `SameSite=Lax`; set `CLOUDIAM_CONSOLE_COOKIE_SECURE=true` in TLS-terminated deployments so those cookies also include `Secure`.

## Health And Readiness

The HTTP server exposes separate liveness and readiness probes:

- `GET /healthz` returns `204 No Content` when the process can answer HTTP requests. It does not check Postgres, Redis, AWS, config reload status, or cleanup state.
- `GET /readyz` returns JSON readiness state. It reports `200 OK` with `{"status":"ready"}` when required runtime dependencies are healthy enough to accept traffic, and `503 Service Unavailable` with `{"status":"not_ready"}` when a dependency check fails.

Readiness currently checks:

- active config snapshot
- Postgres ping when database-backed runtime is configured
- Redis ping when Redis-backed rate limiting is configured

Readiness responses may name failing subsystems, but they redact secret-shaped error details before returning JSON.

Production startup checks are enabled with `CLOUDIAM_DEPLOYMENT_ENV=production`. In that mode, startup fails unless:

- `CLOUDIAM_CONSOLE_COOKIE_SECURE=true`
- `CLOUDIAM_REDIS_URL` is set for shared rate limiting
- `CLOUDIAM_AUTH_VERIFICATION_URI` is explicitly set and uses `https://`
- `CLOUDIAM_DATABASE_URL` does not use `sslmode=disable`, unless `CLOUDIAM_ALLOW_INSECURE_DATABASE_IN_PRODUCTION=true`
- `CLOUDIAM_CONSOLE_JWT_SECRET`, when set, is not a placeholder and is at least 32 characters
- `CLOUDIAM_CONSOLE_SEED_PASSWORD` is unset, unless `CLOUDIAM_ALLOW_CONSOLE_SEED_IN_PRODUCTION=true` and the seed password is non-placeholder and at least 16 characters

## Config Reload

The running server reloads its YAML governance config when it receives `SIGHUP`.

Reloadable fields:

- `principals`
- `access.targets`
- `access.targets.*.approval_required`
- `access.policies`
- AWS account IDs and role ARNs used by target resolution

Restart-required fields:

- `serve.grpc.addr`
- `providers.aws.region`
- local/dev HTTP address
- database URL
- auth workspace, endpoint, and verification URI
- AWS SDK runtime region/client construction

Reload is fail-safe for the running process: the candidate YAML is fully loaded and validated first, and the active config generation is swapped only after validation succeeds. Failed reloads keep the previous valid generation active and print a concise status line with the config path, generation, and error.

Regenerate protobuf/gRPC code after changing files under `server/proto`:

```bash
make proto
```

Validate YAML config without starting the server or calling AWS:

```bash
make config-validate
```

Scan configured AWS JIT role trust policies for the CloudIAM managed-role contract:

```bash
AWS_PROFILE=akira-admin-dev make aws-scan-policies
```

The scanner reads configured target roles with `iam:GetRole`, masks account IDs in output, and exits non-zero when a role is non-compliant.

Run the server security gate:

```bash
make server-security-check
```

This installs pinned scanner tools from `server/tools` into `/tmp/cloudiam-go-tools/bin`, then runs module verification, vet, tests, race tests, `govulncheck`, `gosec`, and `golangci-lint`. `govulncheck` requires network access to refresh the Go vulnerability database.

The local Postgres URL defaults to:

```text
postgres://cloudiam:cloudiam@127.0.0.1:6000/cloudiam?sslmode=disable
```

## Docker Service

From the repository root, build and run the local server container:

```bash
make service-build
make service-up
```

The image uses a distroless nonroot runtime with no shell, package manager, or Go toolchain. The Compose service mounts:

- `./server/configs/cloudiam.yaml` to `/etc/cloudiam/cloudiam.yaml:ro`
- `${HOME}/.aws` to `/aws:ro`

AWS credentials are not baked into the image. For local demos, set `AWS_PROFILE` and `CLOUDIAM_AWS_ACCOUNT_ID` in `.env.local`; the server uses the standard AWS SDK credential chain at runtime.

The local Compose service runs as your host UID/GID so the container can read standard `0600` AWS profile files without changing their permissions. Production deployments should prefer provider-native runtime identity over mounted developer credentials.
Compose sets `AWS_CONFIG_FILE` and `AWS_SHARED_CREDENTIALS_FILE` to the mounted read-only files so distroless runtime discovery is deterministic.

By default, the Docker service maps:

- gRPC: `127.0.0.1:50051`
- local/dev HTTP: `127.0.0.1:18080`

Override these with `CLOUDIAM_SERVICE_GRPC_PORT` and `CLOUDIAM_SERVICE_HTTP_PORT` in `.env.local`.

Compose also starts Redis for shared HTTP rate limiting. Override `CLOUDIAM_REDIS_URL` in `.env.local` when using a different Redis endpoint.

Follow logs:

```bash
make service-logs
```

Stop the stack:

```bash
make service-down
```
