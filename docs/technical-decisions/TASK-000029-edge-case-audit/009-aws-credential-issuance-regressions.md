---
title: "AWS credential issuance regressions"
type: decision
status: accepted
created_at: 2026-10-11
updated_at: 2026-10-11
owner: codex
applies_to:
  - TASK-000029
tags:
  - edge-cases
  - aws
  - sts
  - credentials
  - audit
  - security
---

# Decision: AWS Credential Issuance Regressions

## Context

E6 covers the AWS STS adapter and the access-service credential issuance boundary. The production-safety question is whether provider failures, malformed final issue requests, partial STS responses, or audit failures can return credentials, skip durable evidence, or leak credential material in audit/status paths.

Existing coverage already proved:

- missing role ARN fails before STS
- resolved role ARN, source identity, session name, and CloudIAM STS tags are used
- STS tag values are bounded and control characters are replaced
- gRPC maps internal issuance failures to generic `Internal` status messages
- access-service failed issuance audit events use a generic reason instead of propagating issuer error details
- issued/approval audit events do not include AWS credential fields

## Decision

Harden the STS adapter as the final AWS boundary:

- reject non-positive final issuance durations before calling STS
- reject missing STS credential payloads
- reject incomplete STS credential payloads, including missing access key ID, secret access key, session token, or expiration
- return a zero credential grant for provider and credential-shape failures

Keep the existing access-service fail-closed audit contract:

- if STS succeeds but recording the `access_issued` audit event fails, do not return credentials to the caller
- if STS fails, record `access_failed` with a generic reason and without credential material

## Implementation

Updated `server/internal/adapters/aws/sts_issuer.go` to validate positive duration and complete STS credential tuples before returning a grant.

Added focused regressions:

- `TestSTSIssuer_IssueRequiresPositiveDuration`
- `TestSTSIssuer_IssueFailsClosedOnProviderErrors`
- `TestSTSIssuer_IssueRejectsMissingOrIncompleteCredentials`
- `TestSessionTagsSanitizeBoundaryValues`
- `TestService_RequestAccessDoesNotReturnCredentialsWhenIssuedAuditFails`

## Verification

Run:

```bash
cd server
GOCACHE=/tmp/cloudiam-go-cache go test ./internal/adapters/aws ./internal/core/access
GOCACHE=/tmp/cloudiam-go-cache go test ./...
```

## Consequences

- `E1-AWS-001` is covered by STS adapter boundary checks, provider-error no-grant behavior, incomplete-credential rejection, and existing gRPC no-leakage mapping.
- `E1-AUDIT-001` is covered for credential issuance: credentials are not returned when the issued-access audit event cannot be durably recorded.
- E6 does not add multi-cloud issuance, external secret-manager integration, STS retry/backoff policy, or provider-specific chaos tests; those remain explicit future production-platform work unless a current MVP bug requires them.
