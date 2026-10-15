package access

import (
	"context"
	"time"
)

type CredentialIssuer interface {
	Issue(ctx context.Context, request IssueRequest) (Grant, error)
}

type Authorizer interface {
	Authorize(principal, target string) AuthorizationDecision
}

type AuthorizationDecision struct {
	Allowed          bool
	Reason           string
	ApprovalRequired bool
}

type TargetResolver interface {
	ResolveTarget(target string, requestedDuration time.Duration) (ResolvedTarget, error)
}

type RequestEvaluator interface {
	EvaluateRequest(principal, target string, requestedDuration time.Duration) (AuthorizationDecision, ResolvedTarget, error)
}

type AccessRequestStore interface {
	CreateAccessRequest(ctx context.Context, input CreateAccessRequestInput) (AccessRequest, error)
	ClaimApprovedAccessRequest(ctx context.Context, input ClaimApprovedAccessRequestInput) (AccessRequest, bool, error)
}

type CreateAccessRequestInput struct {
	Principal         string
	Target            string
	Provider          string
	Account           string
	Role              string
	RoleARN           string
	RequestedDuration time.Duration
	PolicyReason      string
	ExpiresAt         time.Time
}

type ClaimApprovedAccessRequestInput struct {
	Principal         string
	Target            string
	RequestedDuration time.Duration
	IssuedAt          time.Time
}

type ResolvedTarget struct {
	Name     string
	Provider string
	Account  string
	Role     string
	RoleARN  string
	Duration time.Duration
}
