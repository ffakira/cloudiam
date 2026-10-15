# CloudIAM Observability Checklist

This checklist maps the AWS-only MVP production flows to the signals currently available to an operator.

It is intentionally not a SIEM, metrics, or alerting implementation. It is the production-readiness map for what exists now and what should be added later.

## Current Signal Types

| Signal | Current Surface | Secret Boundary |
|---|---|---|
| Durable audit events | Postgres `audit_events`, `/audit/events`, `/api/admin/audit/events` | Must not contain AWS credentials, CloudIAM session tokens, raw console JWTs, raw idempotency keys, raw device/user codes, passwords, or hashes. |
| Readiness | `GET /readyz` | May name failing subsystem; redacts secret-shaped dependency errors. |
| Liveness | `GET /healthz` | Process-only signal; no dependency details. |
| Reload status | `SIGHUP` status writer | Reports path, generation, and redacted error text. |
| Startup errors | process startup return/error | Must fail closed for unsafe production config and redact startup cleanup errors. |
| API responses | gRPC/HTTP responses | User-facing errors are generic and do not expose credential or secret material. |

## Critical Flow Matrix

| Flow | Current Signal | Status | Notes / Gap |
|---|---|---|---|
| CLI device login started | durable device authorization state | Partial | Device authorization rows prove the flow started, but no first-class audit event exists for "device login started." Add only if operator needs require auth-flow reporting. |
| Device approved | durable device authorization state; HTTP response | Partial | Device approval state exists. It is not yet mirrored into `audit_events`; acceptable for MVP because resource/JIT approval is the privileged access event. |
| Device denied or expired | durable device authorization state | Partial | State exists. No dedicated audit event yet. |
| CloudIAM session exchange | durable session row with hashed token | Partial | Session state exists without raw token storage. No dedicated audit event yet. |
| Access denied by policy | `access_denied` durable audit event | Covered | Records principal, target, decision, reason. |
| Approval required | `approval_requested` durable audit event and pending `access_requests` row | Covered | Correlates principal, target, provider, account, role, and access request ID. |
| Approval approved | `approval_approved` durable audit event | Covered | Includes operator subject and hashed idempotency-key reference. |
| Approval denied | `approval_denied` durable audit event | Covered | Includes operator subject and hashed idempotency-key reference. |
| Approval expired | `approval_expired` durable audit event on attempted mutation of expired request | Covered for mutation path | Expired state transition is auditable when an operator action observes expiry. |
| Credential issued | `access_issued` durable audit event | Covered | Includes credential expiration and access request ID when issuance came from an approved request. Credential values are not stored. |
| Credential issuance failed | `access_failed` durable audit event when audit can be recorded | Covered with caveat | If durable audit write itself fails, issuance fails closed and returns an operator-visible error path rather than minting credentials. |
| AWS STS dependency failure | `access_failed` audit event plus generic API error | Covered | No readiness STS call; readiness must not perform privileged cloud actions. |
| Audit write failure | request fails closed for issuance paths | Covered for credential issuance | No external alerting yet; P6 runbook should explain how to triage DB/audit failures. |
| Config reload success | reload status line | Covered | Includes config path and generation. |
| Config reload failure | reload status line with redacted error | Covered | Previous known-good config remains active. |
| Postgres dependency failure | `/readyz` dependency failure; startup failure if unavailable at startup | Covered | Readiness response redacts secret-shaped details. |
| Redis dependency failure | `/readyz` dependency failure when Redis configured; request-time limiter falls back locally | Covered | Production requires Redis config. Runtime Redis outage is visible through readiness while local fallback preserves partial abuse protection. |
| Rate limit exceeded | HTTP `429 Too Many Requests` | Partial | No durable audit event or metric yet. Future metrics should count rate-limit hits by route class, not client IP. |
| Startup idempotency cleanup success | startup completion | Partial | No metric/log count yet. P6 runbook should describe expected behavior. |
| Startup idempotency cleanup failure | redacted startup error | Covered | Startup fails before serving when cleanup cannot complete. |
| HTTP liveness | `GET /healthz` | Covered | Does not imply readiness. |
| Runtime readiness | `GET /readyz` | Covered | Checks config, Postgres, and Redis when configured. |
| HTTP/gRPC shutdown | unit regressions | Covered by tests | No external runtime signal yet. P6 runbook should document shutdown expectations. |

## Recommended Future Metrics

These are not required for Task 27 P5 closure, but they are the next production observability layer:

| Metric | Labels | Purpose |
|---|---|---|
| `cloudiam_access_requests_total` | `decision`, `target`, `provider` | Volume of allowed/denied/pending access requests. |
| `cloudiam_access_issuance_total` | `result`, `provider` | Issued versus failed credential grants. |
| `cloudiam_approval_mutations_total` | `action`, `result` | Approval/deny/expired/idempotency replay behavior. |
| `cloudiam_sts_duration_seconds` | `result`, `provider` | STS latency and failure monitoring. |
| `cloudiam_config_reload_total` | `result` | Reload success/failure trends. |
| `cloudiam_rate_limit_total` | `route`, `result` | Abuse throttling behavior without high-cardinality client labels. |
| `cloudiam_readiness_status` | `dependency` | Dependency readiness gauge. |
| `cloudiam_startup_cleanup_total` | `result` | Idempotency cleanup success/failure signal. |

Do not use principal, user ID, raw target from unbounded input, token, idempotency key, client IP, or AWS credential values as metric labels.

## Recommended Future Alerts

| Alert | Trigger |
|---|---|
| Access issuance failures elevated | `access_failed` events or future `cloudiam_access_issuance_total{result="failed"}` above baseline. |
| Readiness failing | `/readyz` returns `not_ready` for Postgres, Redis, or config. |
| Audit write failure | credential issuance fails because audit cannot be recorded. |
| Config reload failures | repeated reload failures over a deployment window. |
| Redis unavailable in production | readiness reports Redis failed while `CLOUDIAM_DEPLOYMENT_ENV=production`. |
| Rate-limit pressure | high rate-limit rejection count on login/device/approval paths. |

## Accepted P5 Gaps

- No Prometheus metrics are implemented in P5.
- No OpenTelemetry tracing is implemented in P5.
- No SIEM/export pipeline is implemented in P5.
- Device-login lifecycle audit events are not first-class `audit_events` yet.
- Rate-limit hits are not durable audit events yet.
- Startup cleanup success count is not externally emitted yet.

These gaps are accepted because P5 is a checklist slice. Implementation belongs in a future observability/platform task unless P6 runbook work reveals that an operator cannot triage a required MVP incident without one.
