package access

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/audit"
)

type fakeIssuer struct {
	calls int
	grant Grant
	err   error
	got   IssueRequest
}

func (f *fakeIssuer) Issue(ctx context.Context, request IssueRequest) (Grant, error) {
	f.calls++
	f.got = request
	return f.grant, f.err
}

type fakeAuthorizer struct {
	decision AuthorizationDecision
}

func (f fakeAuthorizer) Authorize(principal, target string) AuthorizationDecision {
	return f.decision
}

type fakeResolver struct {
	target ResolvedTarget
	err    error
	calls  int
}

func (f *fakeResolver) ResolveTarget(target string, requestedDuration time.Duration) (ResolvedTarget, error) {
	f.calls++
	return f.target, f.err
}

type fakeAuditRecorder struct {
	events []audit.RecordEventInput
	err    error
}

func (f *fakeAuditRecorder) RecordEvent(ctx context.Context, input audit.RecordEventInput) error {
	f.events = append(f.events, input)
	return f.err
}

type fakeAccessRequestStore struct {
	input      CreateAccessRequestInput
	claimInput ClaimApprovedAccessRequestInput
	claim      *AccessRequest
	err        error
}

func (f *fakeAccessRequestStore) CreateAccessRequest(ctx context.Context, input CreateAccessRequestInput) (AccessRequest, error) {
	f.input = input
	if f.err != nil {
		return AccessRequest{}, f.err
	}
	return AccessRequest{
		ID:                42,
		Principal:         input.Principal,
		Target:            input.Target,
		Provider:          input.Provider,
		Account:           input.Account,
		Role:              input.Role,
		RoleARN:           input.RoleARN,
		RequestedDuration: input.RequestedDuration,
		PolicyReason:      input.PolicyReason,
		Status:            AccessRequestStatusPendingApproval,
		ExpiresAt:         input.ExpiresAt,
	}, nil
}

func (f *fakeAccessRequestStore) ClaimApprovedAccessRequest(ctx context.Context, input ClaimApprovedAccessRequestInput) (AccessRequest, bool, error) {
	f.claimInput = input
	if f.err != nil {
		return AccessRequest{}, false, f.err
	}
	if f.claim == nil {
		return AccessRequest{}, false, nil
	}
	return *f.claim, true, nil
}

func TestService_RequestAccessValidation(t *testing.T) {
	tests := []struct {
		name    string
		request Request
	}{
		{
			name:    "missing principal fails before issuer",
			request: Request{Target: "aws-sandbox-readonly"},
		},
		{
			name:    "missing target fails before issuer",
			request: Request{Principal: "akira"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer := &fakeIssuer{}
			service := NewService(issuer)

			_, err := service.RequestAccess(context.Background(), tt.request)
			if err == nil {
				t.Fatal("expected error")
			}
			var validationErr ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}

			if issuer.calls != 0 {
				t.Fatalf("issuer calls = %d, want 0", issuer.calls)
			}
		})
	}
}

func TestService_RequestAccessIssuesApprovedAccessRequest(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	issuer := &fakeIssuer{grant: Grant{
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		SessionToken:    "session-token",
		ExpiresAt:       expiresAt,
	}}
	resolver := &fakeResolver{target: ResolvedTarget{
		Name:     "aws-sandbox-readonly",
		Provider: "aws",
		Account:  "sandbox",
		Role:     "readonly",
		RoleARN:  "arn:aws:iam::000000000000:role/jit/readonly",
		Duration: 15 * time.Minute,
	}}
	approved := AccessRequest{
		ID:                42,
		Principal:         "akira",
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		Status:            AccessRequestStatusIssued,
	}
	store := &fakeAccessRequestStore{claim: &approved}
	auditor := &fakeAuditRecorder{}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{
			Allowed:          true,
			Reason:           "policy matched; approval required",
			ApprovalRequired: true,
		}}),
		WithTargetResolver(resolver),
		WithAccessRequestStore(store),
		WithAuditRecorder(auditor),
	)

	result, err := service.RequestAccess(context.Background(), Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if result.Status != ResultStatusIssued {
		t.Fatalf("status = %q, want issued", result.Status)
	}
	if result.AccessRequest.ID != 42 {
		t.Fatalf("access request id = %d, want 42", result.AccessRequest.ID)
	}
	if issuer.calls != 1 {
		t.Fatalf("issuer calls = %d, want 1", issuer.calls)
	}
	if store.input.Principal != "" {
		t.Fatalf("created new access request unexpectedly: %+v", store.input)
	}
	if store.claimInput.Principal != "akira" || store.claimInput.Target != "aws-sandbox-readonly" {
		t.Fatalf("claim principal/target = %q/%q", store.claimInput.Principal, store.claimInput.Target)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	event := auditor.events[0]
	if event.EventType != audit.EventTypeAccessIssued {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeAccessIssued)
	}
	if event.AccessRequestID == nil || *event.AccessRequestID != 42 {
		t.Fatalf("access request id = %v, want 42", event.AccessRequestID)
	}
	eventText := strings.Join([]string{
		event.Principal,
		event.Target,
		event.Reason,
		event.Provider,
		event.Account,
		event.Role,
	}, " ")
	for _, secret := range []string{"access-key", "secret-key", "session-token"} {
		if strings.Contains(eventText, secret) {
			t.Fatalf("approved access issued audit event contains credential material %q: %+v", secret, event)
		}
	}
}

func TestService_RequestAccessIssuesApprovedAccessRequestWithClaimedRoleARN(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	issuer := &fakeIssuer{grant: Grant{
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		SessionToken:    "session-token",
		ExpiresAt:       expiresAt,
	}}
	resolver := &fakeResolver{target: ResolvedTarget{
		Name:     "aws-sandbox-readonly",
		Provider: "aws",
		Account:  "sandbox",
		Role:     "readonly",
		RoleARN:  "arn:aws:iam::000000000000:role/jit/reloaded",
		Duration: 15 * time.Minute,
	}}
	approved := AccessRequest{
		ID:                42,
		Principal:         "akira",
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/approved",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		Status:            AccessRequestStatusIssued,
	}
	store := &fakeAccessRequestStore{claim: &approved}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{
			Allowed:          true,
			Reason:           "policy matched; approval required",
			ApprovalRequired: true,
		}}),
		WithTargetResolver(resolver),
		WithAccessRequestStore(store),
	)

	_, err := service.RequestAccess(context.Background(), Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if issuer.got.RoleARN != "arn:aws:iam::000000000000:role/jit/approved" {
		t.Fatalf("issued role arn = %q, want approved request role arn", issuer.got.RoleARN)
	}
}

func TestService_RequestAccessCreatesPendingApprovalRequest(t *testing.T) {
	issuer := &fakeIssuer{}
	resolver := &fakeResolver{target: ResolvedTarget{
		Name:     "aws-sandbox-readonly",
		Provider: "aws",
		Account:  "sandbox",
		Role:     "readonly",
		RoleARN:  "arn:aws:iam::000000000000:role/jit/readonly",
		Duration: 15 * time.Minute,
	}}
	store := &fakeAccessRequestStore{}
	auditor := &fakeAuditRecorder{}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{
			Allowed:          true,
			Reason:           "policy matched; approval required",
			ApprovalRequired: true,
		}}),
		WithTargetResolver(resolver),
		WithAccessRequestStore(store),
		WithAuditRecorder(auditor),
		WithApprovalRequestTTL(10*time.Minute),
	)

	result, err := service.RequestAccess(context.Background(), Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if result.Status != ResultStatusPendingApproval {
		t.Fatalf("status = %q, want pending approval", result.Status)
	}
	if result.AccessRequest.ID != 42 {
		t.Fatalf("access request id = %d, want 42", result.AccessRequest.ID)
	}
	if issuer.calls != 0 {
		t.Fatalf("issuer calls = %d, want 0", issuer.calls)
	}
	if store.input.Principal != "akira" || store.input.Target != "aws-sandbox-readonly" {
		t.Fatalf("store principal/target = %q/%q", store.input.Principal, store.input.Target)
	}
	if store.input.Provider != "aws" || store.input.Account != "sandbox" || store.input.Role != "readonly" {
		t.Fatalf("store provider/account/role = %q/%q/%q", store.input.Provider, store.input.Account, store.input.Role)
	}
	if store.input.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("store role arn = %q", store.input.RoleARN)
	}
	if store.input.PolicyReason != "policy matched; approval required" {
		t.Fatalf("policy reason = %q", store.input.PolicyReason)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	event := auditor.events[0]
	if event.EventType != audit.EventTypeApprovalRequested {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeApprovalRequested)
	}
	if event.Decision != audit.DecisionPending {
		t.Fatalf("decision = %q, want %q", event.Decision, audit.DecisionPending)
	}
	if event.AccessRequestID == nil || *event.AccessRequestID != 42 {
		t.Fatalf("access request id = %v, want 42", event.AccessRequestID)
	}
	if event.Principal != "akira" || event.Target != "aws-sandbox-readonly" {
		t.Fatalf("event principal/target = %q/%q", event.Principal, event.Target)
	}
	if event.Provider != "aws" || event.Account != "sandbox" || event.Role != "readonly" {
		t.Fatalf("event provider/account/role = %q/%q/%q", event.Provider, event.Account, event.Role)
	}
	eventText := strings.Join([]string{
		event.Principal,
		event.Target,
		event.Reason,
		event.Provider,
		event.Account,
		event.Role,
		event.OperatorSubject,
	}, " ")
	for _, secret := range []string{"secret-key", "session-token", "access-key", "device-code", "user-code", "jwt"} {
		if strings.Contains(eventText, secret) {
			t.Fatalf("approval requested audit event contains secret %q: %+v", secret, event)
		}
	}
}

func TestService_RequestAccessCanceledContextStopsMutations(t *testing.T) {
	issuer := &fakeIssuer{}
	resolver := &fakeResolver{target: ResolvedTarget{
		Name:     "aws-sandbox-readonly",
		Provider: "aws",
		Account:  "sandbox",
		Role:     "readonly",
		RoleARN:  "arn:aws:iam::000000000000:role/jit/readonly",
		Duration: 15 * time.Minute,
	}}
	store := &fakeAccessRequestStore{}
	auditor := &fakeAuditRecorder{}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{
			Allowed:          true,
			Reason:           "policy matched; approval required",
			ApprovalRequired: true,
		}}),
		WithTargetResolver(resolver),
		WithAccessRequestStore(store),
		WithAuditRecorder(auditor),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := service.RequestAccess(ctx, Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls = %d, want 0", resolver.calls)
	}
	if store.claimInput.Principal != "" || store.input.Principal != "" {
		t.Fatalf("store mutated after cancellation: claim=%+v create=%+v", store.claimInput, store.input)
	}
	if issuer.calls != 0 {
		t.Fatalf("issuer calls = %d, want 0", issuer.calls)
	}
	if len(auditor.events) != 0 {
		t.Fatalf("audit events = %d, want 0", len(auditor.events))
	}
}

func TestService_RequestAccessTarget(t *testing.T) {
	validGrant := Grant{
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		SessionToken:    "session-token",
		ExpiresAt:       time.Now().Add(15 * time.Minute),
	}

	tests := []struct {
		name          string
		request       Request
		decision      AuthorizationDecision
		resolved      ResolvedTarget
		wantCalls     int
		wantResolver  int
		wantErr       bool
		wantIssueData IssueRequest
	}{
		{
			name: "valid principal and target calls issuer with resolved metadata",
			request: Request{
				Principal: "akira",
				Target:    "aws-sandbox-readonly",
			},
			decision: AuthorizationDecision{Allowed: true, Reason: "policy matched"},
			resolved: ResolvedTarget{
				Name:     "aws-sandbox-readonly",
				Provider: "aws",
				Account:  "sandbox",
				Role:     "readonly",
				RoleARN:  "arn:aws:iam::000000000000:role/jit/readonly",
				Duration: 15 * time.Minute,
			},
			wantCalls:    1,
			wantResolver: 1,
			wantIssueData: IssueRequest{
				Principal: "akira",
				Target:    "aws-sandbox-readonly",
				Provider:  "aws",
				Account:   "sandbox",
				Role:      "readonly",
				RoleARN:   "arn:aws:iam::000000000000:role/jit/readonly",
				Duration:  15 * time.Minute,
			},
		},
		{
			name: "unauthorized principal fails before resolver and issuer",
			request: Request{
				Principal: "unknown",
				Target:    "aws-sandbox-readonly",
			},
			decision: AuthorizationDecision{Allowed: false, Reason: "principal not found"},
			wantErr:  true,
		},
		{
			name: "unknown target fails before issuer",
			request: Request{
				Principal: "akira",
				Target:    "missing",
			},
			decision:     AuthorizationDecision{Allowed: true, Reason: "policy matched"},
			resolved:     ResolvedTarget{},
			wantResolver: 1,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer := &fakeIssuer{grant: validGrant}
			resolver := &fakeResolver{target: tt.resolved}
			auditor := &fakeAuditRecorder{}
			if tt.name == "unknown target fails before issuer" {
				resolver.err = errors.New("target not found")
			}
			service := NewService(
				issuer,
				WithAuthorizer(fakeAuthorizer{decision: tt.decision}),
				WithTargetResolver(resolver),
				WithAuditRecorder(auditor),
			)

			_, err := service.RequestAccess(context.Background(), tt.request)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if issuer.calls != tt.wantCalls {
				t.Fatalf("issuer calls = %d, want %d", issuer.calls, tt.wantCalls)
			}
			if resolver.calls != tt.wantResolver {
				t.Fatalf("resolver calls = %d, want %d", resolver.calls, tt.wantResolver)
			}
			if tt.wantCalls == 1 && issuer.got != tt.wantIssueData {
				t.Fatalf("issue request = %+v, want %+v", issuer.got, tt.wantIssueData)
			}
		})
	}
}

func TestService_RequestAccessRecordsIssuedAuditEvent(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	issuer := &fakeIssuer{grant: Grant{
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		SessionToken:    "session-token",
		ExpiresAt:       expiresAt,
	}}
	resolver := &fakeResolver{target: ResolvedTarget{
		Name:     "aws-sandbox-readonly",
		Provider: "aws",
		Account:  "sandbox",
		Role:     "readonly",
		RoleARN:  "arn:aws:iam::000000000000:role/jit/readonly",
		Duration: 15 * time.Minute,
	}}
	auditor := &fakeAuditRecorder{}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{Allowed: true, Reason: "policy matched"}}),
		WithTargetResolver(resolver),
		WithAuditRecorder(auditor),
	)

	_, err := service.RequestAccess(context.Background(), Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	event := auditor.events[0]
	if event.EventType != audit.EventTypeAccessIssued {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeAccessIssued)
	}
	if event.Decision != audit.DecisionAllowed {
		t.Fatalf("decision = %q, want %q", event.Decision, audit.DecisionAllowed)
	}
	if event.Principal != "akira" || event.Target != "aws-sandbox-readonly" {
		t.Fatalf("event principal/target = %q/%q", event.Principal, event.Target)
	}
	if event.Provider != "aws" || event.Account != "sandbox" || event.Role != "readonly" {
		t.Fatalf("event provider/account/role = %q/%q/%q", event.Provider, event.Account, event.Role)
	}
	if event.CredentialExpiresAt == nil || !event.CredentialExpiresAt.Equal(expiresAt) {
		t.Fatalf("credential expiration = %v, want %v", event.CredentialExpiresAt, expiresAt)
	}
	eventText := strings.Join([]string{
		event.Principal,
		event.Target,
		event.Reason,
		event.Provider,
		event.Account,
		event.Role,
	}, " ")
	for _, secret := range []string{"secret-key", "session-token", "access-key"} {
		if strings.Contains(eventText, secret) {
			t.Fatalf("audit event contains credential material %q: %+v", secret, event)
		}
	}
}

func TestService_RequestAccessRecordsDeniedAuditEvent(t *testing.T) {
	issuer := &fakeIssuer{}
	resolver := &fakeResolver{}
	auditor := &fakeAuditRecorder{}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{Allowed: false, Reason: "principal not found"}}),
		WithTargetResolver(resolver),
		WithAuditRecorder(auditor),
	)

	_, err := service.RequestAccess(context.Background(), Request{
		Principal: "unknown",
		Target:    "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if issuer.calls != 0 {
		t.Fatalf("issuer calls = %d, want 0", issuer.calls)
	}
	if resolver.calls != 0 {
		t.Fatalf("resolver calls = %d, want 0", resolver.calls)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	event := auditor.events[0]
	if event.EventType != audit.EventTypeAccessDenied {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeAccessDenied)
	}
	if event.Decision != audit.DecisionDenied {
		t.Fatalf("decision = %q, want %q", event.Decision, audit.DecisionDenied)
	}
	if event.Reason != "principal not found" {
		t.Fatalf("reason = %q", event.Reason)
	}
}

func TestService_RequestAccessRecordsFailedAuditEvent(t *testing.T) {
	issuerErr := errors.New("assume role failed: access-key secret-key session-token")
	issuer := &fakeIssuer{err: issuerErr}
	resolver := &fakeResolver{target: ResolvedTarget{
		Name:     "aws-sandbox-readonly",
		Provider: "aws",
		Account:  "sandbox",
		Role:     "readonly",
		RoleARN:  "arn:aws:iam::000000000000:role/jit/readonly",
		Duration: 15 * time.Minute,
	}}
	auditor := &fakeAuditRecorder{}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{Allowed: true, Reason: "policy matched"}}),
		WithTargetResolver(resolver),
		WithAuditRecorder(auditor),
	)

	_, err := service.RequestAccess(context.Background(), Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if !errors.Is(err, issuerErr) {
		t.Fatalf("error = %v, want issuer error", err)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditor.events))
	}
	event := auditor.events[0]
	if event.EventType != audit.EventTypeAccessFailed {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeAccessFailed)
	}
	if event.Decision != audit.DecisionFailed {
		t.Fatalf("decision = %q, want %q", event.Decision, audit.DecisionFailed)
	}
	if event.Reason != accessFailureAuditReason {
		t.Fatalf("reason = %q", event.Reason)
	}
	eventText := strings.Join([]string{
		event.Principal,
		event.Target,
		event.Reason,
		event.Provider,
		event.Account,
		event.Role,
	}, " ")
	for _, secret := range []string{"access-key", "secret-key", "session-token"} {
		if strings.Contains(eventText, secret) {
			t.Fatalf("failed audit event contains credential material %q: %+v", secret, event)
		}
	}
}

func TestService_RequestAccessDoesNotReturnCredentialsWhenIssuedAuditFails(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	issuer := &fakeIssuer{grant: Grant{
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		SessionToken:    "session-token",
		ExpiresAt:       expiresAt,
	}}
	resolver := &fakeResolver{target: ResolvedTarget{
		Name:     "aws-sandbox-readonly",
		Provider: "aws",
		Account:  "sandbox",
		Role:     "readonly",
		RoleARN:  "arn:aws:iam::000000000000:role/jit/readonly",
		Duration: 15 * time.Minute,
	}}
	auditor := &fakeAuditRecorder{err: errors.New("audit database unavailable")}
	service := NewService(
		issuer,
		WithAuthorizer(fakeAuthorizer{decision: AuthorizationDecision{Allowed: true, Reason: "policy matched"}}),
		WithTargetResolver(resolver),
		WithAuditRecorder(auditor),
	)

	result, err := service.RequestAccess(context.Background(), Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "record audit event") {
		t.Fatalf("error = %v, want audit failure", err)
	}
	if result.Grant != (Grant{}) {
		t.Fatalf("grant = %+v, want no credentials returned", result.Grant)
	}
	if len(auditor.events) != 1 {
		t.Fatalf("audit events = %d, want 1 attempted event", len(auditor.events))
	}
	if auditor.events[0].EventType != audit.EventTypeAccessIssued {
		t.Fatalf("event type = %q, want %q", auditor.events[0].EventType, audit.EventTypeAccessIssued)
	}
}
