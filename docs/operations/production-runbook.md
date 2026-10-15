# CloudIAM Production Runbook

This runbook covers the AWS-only CloudIAM MVP runtime. It is written for operators and reviewers who need to start, validate, reload, roll back, and triage the service without expanding the product scope.

CloudIAM production readiness means the current MVP can run safely with explicit limitations. It does not mean multi-cloud, production SSO, SIEM export, session recording, password rotation, or a complete platform operations stack are implemented.

## 1. Required Production Configuration

Production mode is enabled with:

```bash
CLOUDIAM_DEPLOYMENT_ENV=production
```

In production mode, startup must fail closed unless the following are true:

| Setting | Requirement |
|---|---|
| `CLOUDIAM_DATABASE_URL` | Must be set and must not use `sslmode=disable`, unless `CLOUDIAM_ALLOW_INSECURE_DATABASE_IN_PRODUCTION=true` is deliberately set. |
| `CLOUDIAM_REDIS_URL` | Must be set so rate limits are shared across replicas. |
| `CLOUDIAM_AUTH_VERIFICATION_URI` | Must be explicit and use `https://`. |
| `CLOUDIAM_CONSOLE_COOKIE_SECURE` | Must be `true`. |
| `CLOUDIAM_CONSOLE_JWT_SECRET` | If set, must be at least 32 characters and not a placeholder. |
| `CLOUDIAM_CONSOLE_SEED_PASSWORD` | Must be unset unless `CLOUDIAM_ALLOW_CONSOLE_SEED_IN_PRODUCTION=true`; even then it must be strong and non-placeholder. |
| AWS runtime identity | Prefer provider-native runtime identity. Do not bake AWS credentials into images. |

Never put raw AWS credentials, CloudIAM session tokens, console JWTs, database passwords, Redis passwords, idempotency keys, device codes, or user codes in docs, tickets, or screenshots.

## 2. Startup Procedure

1. Validate the YAML config before starting:

```bash
make config-validate
```

2. Apply migrations:

```bash
make migrate-up
make migrate-status
```

3. Start the service through the deployment system.

For the local Compose runtime:

```bash
make service-up
```

4. Check liveness:

```bash
curl -i http://127.0.0.1:18080/healthz
```

Expected:

```text
HTTP/1.1 204 No Content
```

5. Check readiness:

```bash
curl -s http://127.0.0.1:18080/readyz
```

Expected:

```json
{"status":"ready","checks":[{"name":"config","status":"ok"},{"name":"postgres","status":"ok"},{"name":"redis","status":"ok"}]}
```

Readiness may include only the dependencies configured in the current runtime.

## 3. Startup Failure Triage

| Symptom | Meaning | Action |
|---|---|---|
| `config path is required` | Server has no YAML config path. | Start with `--config ./configs/cloudiam.yaml` or the deployment's mounted config path. |
| `CLOUDIAM_CONSOLE_COOKIE_SECURE must be true` | Production cookies would be sent without `Secure`. | Set `CLOUDIAM_CONSOLE_COOKIE_SECURE=true` behind TLS. |
| `CLOUDIAM_REDIS_URL is required` | Production shared rate limiting is not configured. | Configure Redis and verify `/readyz`. |
| `CLOUDIAM_AUTH_VERIFICATION_URI is required` | Production would otherwise fall back to local device URL. | Set the public HTTPS device approval URL. |
| `CLOUDIAM_AUTH_VERIFICATION_URI must use https` | Production device approval URL is insecure. | Use an HTTPS URL. |
| `CLOUDIAM_DATABASE_URL must not use sslmode=disable` | Database TLS is disabled in production. | Use TLS or intentionally set the override only in a controlled exception. |
| `postgres persistence schema is missing ...` | Migrations are not applied. | Run migrations and restart. |
| `cleanup expired idempotency records on startup ...` | Startup cleanup failed. | Check Postgres health and migration state; do not serve traffic until cleanup succeeds. |
| `load aws config ...` | AWS SDK runtime config failed. | Check runtime identity, region, and mounted/provider credentials. |

Startup errors should not expose raw secrets. If you see a raw secret in an error, treat it as a bug.

## 4. Health And Readiness

Use `GET /healthz` for process liveness.

Use `GET /readyz` for traffic readiness.

Readiness checks:

- `config`: active config snapshot exists
- `postgres`: Postgres ping succeeds when database-backed runtime is configured
- `redis`: Redis ping succeeds when Redis-backed rate limiting is configured

Readiness failure examples:

```json
{"status":"not_ready","checks":[{"name":"redis","status":"failed","error":"redis rate limiter ping failed: ..."}]}
```

The error may name the subsystem, but must not include raw DSNs, Redis credentials, AWS credentials, tokens, or passwords.

## 5. Config Reload

CloudIAM reloads YAML governance config on `SIGHUP`.

Reloadable:

- principals
- access targets
- `approval_required`
- policies
- AWS account IDs and role ARNs used by target resolution

Restart-required:

- gRPC address
- HTTP address
- AWS region
- database URL
- auth workspace/endpoint/verification URI
- AWS SDK runtime client construction

Reload procedure:

1. Validate the candidate config:

```bash
make config-validate
```

2. Deploy the config file.

3. Send `SIGHUP` to the running server process through the deployment system.

4. Confirm reload status output:

```text
config reload succeeded path="..." generation=N
```

If reload fails:

```text
config reload failed path="..." generation=N error=...
```

The previous known-good config remains active. Fix the candidate config and retry. If the error says a field requires restart, perform a restart/rollout instead of another reload.

## 6. Rollback Procedure

For config-only rollback:

1. Restore the previous known-good YAML config.
2. Validate it with `make config-validate`.
3. Send `SIGHUP`.
4. Confirm reload success and generation increment.

For binary/container rollback:

1. Stop routing new traffic to the bad version.
2. Roll back to the previous image/release.
3. Confirm `/healthz`.
4. Confirm `/readyz`.
5. Confirm a low-risk access flow or dashboard read-only view.

Do not roll back database migrations blindly. If migration rollback is required, use the migration procedure and verify `make migrate-status`.

## 7. Incident Triage

### Access Denied Unexpectedly

Check recent audit events:

```http
GET /audit/events?limit=50
```

Look for:

- `event_type=access_denied`
- `principal`
- `target`
- `reason`

Common causes:

- principal missing from config
- group membership not matching policy
- target name typo
- config reload failed and old policy is still active

### Approval Request Not Visible

Check:

- CLI response includes `pending_approval` and `access_request_id`
- dashboard is authenticated as `console_operator`
- `/api/admin/access-requests?status=pending`
- audit event `approval_requested`
- `/readyz` for Postgres health

### Approval Mutation Fails

Check:

- console session cookie is valid
- operator role is `console_operator`
- `Idempotency-Key` is a valid UUIDv4
- access request status is still `pending_approval`
- request is not expired

Expected durable events:

- `approval_approved`
- `approval_denied`
- `approval_expired`

### Credential Issuance Fails

Check audit events:

- `access_failed`
- target/provider/account/role
- access request ID if approval-gated

Likely causes:

- AWS STS failure
- missing `sts:AssumeRole`
- missing `sts:SetSourceIdentity`
- missing `sts:TagSession`
- target role trust policy not matching CloudIAM conditions
- audit write failure causing fail-closed issuance

Use the AWS compliance scanner for configured roles:

```bash
AWS_PROFILE=<BROKER_PROFILE> make aws-scan-policies
```

### Audit Events Missing

Check:

- `/readyz` Postgres status
- migration status
- audit write errors from access issuance path

Credential issuance should fail closed if the durable issued-access audit event cannot be recorded.

### Redis Readiness Fails

Check:

- `CLOUDIAM_REDIS_URL`
- Redis service health
- network policy/security groups
- `/readyz` output

Request-time rate limiting falls back locally if Redis operations fail, but production should treat Redis readiness failure as degraded because limits are no longer shared across replicas.

### Rate Limit Exceeded

HTTP response:

```text
429 Too Many Requests
```

Current MVP signal is the HTTP response. Future metrics should count rate-limit hits by route class without client IP or token labels.

### Config Reload Fails

Check the reload status line. Common causes:

- invalid YAML
- missing required config
- changing restart-required fields
- AWS region change
- canceled reload context during shutdown

The active config remains the previous known-good generation.

## 8. Shutdown Expectations

On shutdown:

- HTTP stops accepting new requests and uses a bounded shutdown context.
- gRPC tries graceful stop and then forces stop after the timeout.
- reload signal handler stops.
- Redis and Postgres clients are closed through server cleanup.
- startup cleanup does not run in the background after serving begins.

If shutdown hangs, capture stack traces through the deployment platform if available and treat it as a lifecycle bug.

## 9. Current Residual Risks

Accepted for the MVP:

- no production SSO/OIDC implementation
- no external secret manager integration
- no SIEM/export or alert routing
- no Prometheus metrics or OpenTelemetry tracing
- no session recording
- no password delivery or rotation
- no multi-account or multi-cloud runtime
- no full load/chaos/disaster-recovery program
- device-login lifecycle is partially observable through durable state, not first-class audit events
- rate-limit hits are not durable audit events

Triggers to implement follow-up work:

- multiple replicas or production traffic: add metrics and alerts
- compliance/reporting requirement: add audit export/SIEM pipeline
- enterprise identity requirement: add SSO/OIDC and token revocation model
- multi-account/provider roadmap: add account/provider authorization model after production hardening remains green

## 10. Safe Communication During Incidents

Share:

- event type
- target
- provider/account/role metadata
- access request ID
- operator subject
- readiness dependency name
- redacted error category

Do not share:

- AWS access keys or session tokens
- CloudIAM session tokens
- console JWTs
- passwords or password hashes
- raw device/user codes
- raw idempotency keys
- full database or Redis URLs with credentials
