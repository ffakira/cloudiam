package access

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/audit"
)

type Service struct {
	issuer             CredentialIssuer
	authorizer         Authorizer
	resolver           TargetResolver
	evaluator          RequestEvaluator
	accessRequests     AccessRequestStore
	auditor            audit.Recorder
	now                func() time.Time
	approvalRequestTTL time.Duration
}

const accessFailureAuditReason = "credential issuance failed"

type Option func(*Service)

func WithAuthorizer(authorizer Authorizer) Option {
	return func(s *Service) {
		s.authorizer = authorizer
	}
}

func WithTargetResolver(resolver TargetResolver) Option {
	return func(s *Service) {
		s.resolver = resolver
	}
}

func WithRequestEvaluator(evaluator RequestEvaluator) Option {
	return func(s *Service) {
		s.evaluator = evaluator
	}
}

func WithAccessRequestStore(store AccessRequestStore) Option {
	return func(s *Service) {
		s.accessRequests = store
	}
}

func WithAuditRecorder(auditor audit.Recorder) Option {
	return func(s *Service) {
		s.auditor = auditor
	}
}

func WithApprovalRequestTTL(ttl time.Duration) Option {
	return func(s *Service) {
		s.approvalRequestTTL = ttl
	}
}

func NewService(issuer CredentialIssuer, opts ...Option) *Service {
	service := &Service{
		issuer:             issuer,
		now:                func() time.Time { return time.Now().UTC() },
		approvalRequestTTL: 15 * time.Minute,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *Service) RequestAccess(ctx context.Context, request Request) (Result, error) {
	if strings.TrimSpace(request.Principal) == "" {
		return Result{}, ValidationError{Field: "principal", Reason: "is required"}
	}
	if strings.TrimSpace(request.Target) == "" {
		return Result{}, ValidationError{Field: "target", Reason: "is required"}
	}
	if err := contextError(ctx); err != nil {
		return Result{}, err
	}
	if s.evaluator != nil {
		return s.requestAccessWithEvaluator(ctx, request)
	}
	if s.authorizer == nil {
		return Result{}, fmt.Errorf("authorizer is required")
	}
	if s.resolver == nil {
		return Result{}, fmt.Errorf("target resolver is required")
	}

	decision := s.authorizer.Authorize(request.Principal, request.Target)
	if !decision.Allowed {
		if err := s.recordAuditEvent(ctx, audit.RecordEventInput{
			EventType: audit.EventTypeAccessDenied,
			Principal: request.Principal,
			Target:    request.Target,
			Decision:  audit.DecisionDenied,
			Reason:    decision.Reason,
		}); err != nil {
			return Result{}, err
		}
		return Result{}, fmt.Errorf("%w: %s", ErrAccessDenied, decision.Reason)
	}

	resolved, err := s.resolver.ResolveTarget(request.Target, request.Duration)
	if err != nil {
		return Result{}, err
	}
	if decision.ApprovalRequired {
		return s.requestApprovalOrIssueApproved(ctx, request.Principal, decision, resolved)
	}

	grant, err := s.issueResolvedGrant(ctx, request.Principal, decision, resolved, nil)
	if err != nil {
		return Result{}, err
	}
	return Result{Status: ResultStatusIssued, Grant: grant}, nil
}

func (s *Service) requestAccessWithEvaluator(ctx context.Context, request Request) (Result, error) {
	decision, resolved, err := s.evaluator.EvaluateRequest(request.Principal, request.Target, request.Duration)
	if err != nil {
		return Result{}, err
	}
	if !decision.Allowed {
		if err := s.recordAuditEvent(ctx, audit.RecordEventInput{
			EventType: audit.EventTypeAccessDenied,
			Principal: request.Principal,
			Target:    request.Target,
			Decision:  audit.DecisionDenied,
			Reason:    decision.Reason,
		}); err != nil {
			return Result{}, err
		}
		return Result{}, fmt.Errorf("%w: %s", ErrAccessDenied, decision.Reason)
	}
	if decision.ApprovalRequired {
		return s.requestApprovalOrIssueApproved(ctx, request.Principal, decision, resolved)
	}

	grant, err := s.issueResolvedGrant(ctx, request.Principal, decision, resolved, nil)
	if err != nil {
		return Result{}, err
	}
	return Result{Status: ResultStatusIssued, Grant: grant}, nil
}

func (s *Service) createPendingAccessRequest(ctx context.Context, principal string, decision AuthorizationDecision, resolved ResolvedTarget) (Result, error) {
	if s.accessRequests == nil {
		return Result{}, fmt.Errorf("access request store is required for approval-required access")
	}
	if err := contextError(ctx); err != nil {
		return Result{}, err
	}
	pending, err := s.accessRequests.CreateAccessRequest(ctx, CreateAccessRequestInput{
		Principal:         principal,
		Target:            resolved.Name,
		Provider:          resolved.Provider,
		Account:           resolved.Account,
		Role:              resolved.Role,
		RoleARN:           resolved.RoleARN,
		RequestedDuration: resolved.Duration,
		PolicyReason:      decision.Reason,
		ExpiresAt:         s.now().Add(s.approvalRequestTTL),
	})
	if err != nil {
		return Result{}, fmt.Errorf("create access request: %w", err)
	}
	if err := s.recordAuditEvent(ctx, audit.RecordEventInput{
		EventType:       audit.EventTypeApprovalRequested,
		Principal:       pending.Principal,
		Target:          pending.Target,
		Decision:        audit.DecisionPending,
		Reason:          pending.PolicyReason,
		Provider:        pending.Provider,
		Account:         pending.Account,
		Role:            pending.Role,
		AccessRequestID: &pending.ID,
	}); err != nil {
		return Result{}, err
	}
	return Result{
		Status:        ResultStatusPendingApproval,
		AccessRequest: pending,
	}, nil
}

func (s *Service) requestApprovalOrIssueApproved(ctx context.Context, principal string, decision AuthorizationDecision, resolved ResolvedTarget) (Result, error) {
	if s.accessRequests == nil {
		return Result{}, fmt.Errorf("access request store is required for approval-required access")
	}
	if err := contextError(ctx); err != nil {
		return Result{}, err
	}
	claimed, ok, err := s.accessRequests.ClaimApprovedAccessRequest(ctx, ClaimApprovedAccessRequestInput{
		Principal:         principal,
		Target:            resolved.Name,
		RequestedDuration: resolved.Duration,
		IssuedAt:          s.now(),
	})
	if err != nil {
		return Result{}, fmt.Errorf("claim approved access request: %w", err)
	}
	if !ok {
		return s.createPendingAccessRequest(ctx, principal, decision, resolved)
	}

	grant, err := s.issueAccessRequestGrant(ctx, claimed)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Status:        ResultStatusIssued,
		Grant:         grant,
		AccessRequest: claimed,
	}, nil
}

func (s *Service) issueAccessRequestGrant(ctx context.Context, request AccessRequest) (Grant, error) {
	accessRequestID := request.ID
	return s.issueResolvedGrant(ctx, request.Principal, AuthorizationDecision{
		Allowed: true,
		Reason:  request.PolicyReason,
	}, ResolvedTarget{
		Name:     request.Target,
		Provider: request.Provider,
		Account:  request.Account,
		Role:     request.Role,
		RoleARN:  request.RoleARN,
		Duration: request.RequestedDuration,
	}, &accessRequestID)
}

func (s *Service) issueResolvedGrant(ctx context.Context, principal string, decision AuthorizationDecision, resolved ResolvedTarget, accessRequestID *int64) (Grant, error) {
	if err := contextError(ctx); err != nil {
		return Grant{}, err
	}
	issueRequest := IssueRequest{
		Principal: principal,
		Target:    resolved.Name,
		Provider:  resolved.Provider,
		Account:   resolved.Account,
		Role:      resolved.Role,
		RoleARN:   resolved.RoleARN,
		Duration:  resolved.Duration,
	}
	grant, err := s.issuer.Issue(ctx, issueRequest)
	if err != nil {
		if auditErr := s.recordAuditEvent(ctx, audit.RecordEventInput{
			EventType:       audit.EventTypeAccessFailed,
			Principal:       principal,
			Target:          resolved.Name,
			Decision:        audit.DecisionFailed,
			Reason:          accessFailureAuditReason,
			Provider:        resolved.Provider,
			Account:         resolved.Account,
			Role:            resolved.Role,
			AccessRequestID: accessRequestID,
		}); auditErr != nil {
			return Grant{}, auditErr
		}
		return Grant{}, err
	}
	if err := s.recordAuditEvent(ctx, audit.RecordEventInput{
		EventType:           audit.EventTypeAccessIssued,
		Principal:           principal,
		Target:              resolved.Name,
		Decision:            audit.DecisionAllowed,
		Reason:              decision.Reason,
		Provider:            resolved.Provider,
		Account:             resolved.Account,
		Role:                resolved.Role,
		AccessRequestID:     accessRequestID,
		CredentialExpiresAt: &grant.ExpiresAt,
	}); err != nil {
		return Grant{}, err
	}
	return grant, nil
}

func (s *Service) recordAuditEvent(ctx context.Context, input audit.RecordEventInput) error {
	if s.auditor == nil {
		return nil
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = s.now()
	}
	if err := s.auditor.RecordEvent(ctx, input); err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}
	return nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
