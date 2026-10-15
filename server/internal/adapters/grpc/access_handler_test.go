package grpcadapter

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	accessv1 "github.com/ffakira/cloudiam/server/gen/access/v1"
	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeAccessService struct {
	request access.Request
	result  access.Result
	calls   int
	err     error
}

func (f *fakeAccessService) RequestAccess(ctx context.Context, request access.Request) (access.Result, error) {
	f.calls++
	f.request = request
	return f.result, f.err
}

type fakeSessionResolver struct {
	hash    auth.Hash
	session auth.Session
	ok      bool
	err     error
}

func (f *fakeSessionResolver) FindSessionByTokenHash(ctx context.Context, hash auth.Hash) (auth.Session, bool, error) {
	f.hash = hash
	return f.session, f.ok, f.err
}

func TestAccessHandler_RequestAccessMapsTargetRequest(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	service := &fakeAccessService{
		result: access.Result{
			Status: access.ResultStatusIssued,
			Grant: access.Grant{
				AccessKeyID:     "access-key-id",
				SecretAccessKey: "secret-access-key",
				SessionToken:    "session-token",
				ExpiresAt:       expiresAt,
			},
		},
	}
	handler := NewAccessHandler(service)

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		Principal:       "akira",
		Target:          "aws-sandbox-readonly",
		DurationSeconds: 900,
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if service.request.Principal != "akira" {
		t.Fatalf("principal = %q, want akira", service.request.Principal)
	}
	if service.request.Target != "aws-sandbox-readonly" {
		t.Fatalf("target = %q, want aws-sandbox-readonly", service.request.Target)
	}
	if service.request.Duration != 15*time.Minute {
		t.Fatalf("duration = %s, want 15m", service.request.Duration)
	}
	if response.GetAccessKeyId() != "access-key-id" {
		t.Fatalf("access key id = %q", response.GetAccessKeyId())
	}
	if response.GetStatus() != string(access.ResultStatusIssued) {
		t.Fatalf("status = %q, want issued", response.GetStatus())
	}
	if response.GetExpiresAt() != expiresAt.Format(time.RFC3339) {
		t.Fatalf("expires at = %q", response.GetExpiresAt())
	}
}

func TestAccessHandler_RequestAccessMapsPendingApproval(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 15, 0, 0, time.UTC)
	service := &fakeAccessService{
		result: access.Result{
			Status: access.ResultStatusPendingApproval,
			AccessRequest: access.AccessRequest{
				ID:        42,
				ExpiresAt: expiresAt,
			},
		},
	}
	handler := NewAccessHandler(service)

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if response.GetStatus() != string(access.ResultStatusPendingApproval) {
		t.Fatalf("status = %q, want pending approval", response.GetStatus())
	}
	if response.GetAccessRequestId() != 42 {
		t.Fatalf("access request id = %d, want 42", response.GetAccessRequestId())
	}
	if response.GetApprovalExpiresAt() != expiresAt.Format(time.RFC3339) {
		t.Fatalf("approval expires at = %q", response.GetApprovalExpiresAt())
	}
	if response.GetAccessKeyId() != "" || response.GetSecretAccessKey() != "" || response.GetSessionToken() != "" {
		t.Fatalf("pending response leaked credential fields: %+v", response)
	}
}

func TestAccessHandler_RequestAccessUsesSessionPrincipal(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	service := &fakeAccessService{
		result: access.Result{
			Status: access.ResultStatusIssued,
			Grant: access.Grant{
				AccessKeyID:     "access-key-id",
				SecretAccessKey: "secret-access-key",
				SessionToken:    "session-token",
				ExpiresAt:       expiresAt,
			},
		},
	}
	resolver := &fakeSessionResolver{
		ok: true,
		session: auth.Session{
			Principal: "akira",
		},
	}
	handler := NewAccessHandlerWithSessions(service, resolver)

	_, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "cloudiam-session",
		Principal:            "spoofed",
		Target:               "aws-sandbox-readonly",
		DurationSeconds:      900,
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if service.request.Principal != "akira" {
		t.Fatalf("principal = %q, want akira", service.request.Principal)
	}
	if resolver.hash != sha256.Sum256([]byte("cloudiam-session")) {
		t.Fatal("session token hash was not used for lookup")
	}
}

func TestAccessHandler_RequestAccessRequiresSessionToken(t *testing.T) {
	service := &fakeAccessService{}
	handler := NewAccessHandlerWithSessions(service, &fakeSessionResolver{})

	_, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		Target: "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("code = %s, want %s", code, codes.Unauthenticated)
	}
	if service.calls != 0 {
		t.Fatalf("service calls = %d, want 0", service.calls)
	}
}

func TestAccessHandler_RequestAccessRejectsInvalidSessionBeforeService(t *testing.T) {
	service := &fakeAccessService{}
	handler := NewAccessHandlerWithSessions(service, &fakeSessionResolver{})

	_, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "unknown-session",
		Principal:            "spoofed",
		Target:               "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("code = %s, want %s", code, codes.Unauthenticated)
	}
	if service.calls != 0 {
		t.Fatalf("service calls = %d, want 0", service.calls)
	}
	assertStatusMessageDoesNotContainCredentialMaterial(t, err)
}

func TestAccessHandler_RequestAccessRejectsNegativeDurationBeforeService(t *testing.T) {
	service := &fakeAccessService{}
	handler := NewAccessHandlerWithSessions(service, &fakeSessionResolver{
		ok:      true,
		session: auth.Session{Principal: "akira"},
	})

	_, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
		DurationSeconds:      -1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", code, codes.InvalidArgument)
	}
	if service.calls != 0 {
		t.Fatalf("service calls = %d, want 0", service.calls)
	}
	assertStatusMessageDoesNotContainCredentialMaterial(t, err)
}

func TestAccessHandler_RequestAccessHidesSessionResolverErrorDetails(t *testing.T) {
	handler := NewAccessHandlerWithSessions(&fakeAccessService{}, &fakeSessionResolver{
		err: errors.New("database failed with cloudiam-session secret-access-key session-token"),
	})

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if response != nil {
		t.Fatalf("response = %+v, want nil", response)
	}
	if code := status.Code(err); code != codes.Internal {
		t.Fatalf("code = %s, want %s", code, codes.Internal)
	}
	assertStatusMessageDoesNotContainCredentialMaterial(t, err)
}

func TestAccessHandler_RequestAccessMapsValidationError(t *testing.T) {
	handler := NewAccessHandler(&fakeAccessService{
		err: access.ValidationError{Field: "target", Reason: "is required"},
	})

	_, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		Principal: "akira",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", code, codes.InvalidArgument)
	}
	assertStatusMessageDoesNotContainCredentialMaterial(t, err)
}

func TestAccessHandler_RequestAccessMapsDeniedWithoutPolicyDetails(t *testing.T) {
	handler := NewAccessHandlerWithSessions(&fakeAccessService{
		err: fmt.Errorf("%w: policy mentioned secret-access-key", access.ErrAccessDenied),
	}, &fakeSessionResolver{
		ok:      true,
		session: auth.Session{Principal: "akira"},
	})

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if response != nil {
		t.Fatalf("response = %+v, want nil", response)
	}
	if code := status.Code(err); code != codes.PermissionDenied {
		t.Fatalf("code = %s, want %s", code, codes.PermissionDenied)
	}
	if message := status.Convert(err).Message(); message != "access denied" {
		t.Fatalf("message = %q, want access denied", message)
	}
	assertStatusMessageDoesNotContainCredentialMaterial(t, err)
}

func TestAccessHandler_RequestAccessMapsCredentialIssuanceDenied(t *testing.T) {
	handler := NewAccessHandlerWithSessions(&fakeAccessService{
		err: fmt.Errorf("%w: sts:SetSourceIdentity arn:aws:iam::000000000000:role/jit/readonly", access.ErrCredentialIssuanceDenied),
	}, &fakeSessionResolver{
		ok:      true,
		session: auth.Session{Principal: "akira"},
	})

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if response != nil {
		t.Fatalf("response = %+v, want nil", response)
	}
	if code := status.Code(err); code != codes.FailedPrecondition {
		t.Fatalf("code = %s, want %s", code, codes.FailedPrecondition)
	}
	if message := status.Convert(err).Message(); message != "credential issuance denied by cloud provider" {
		t.Fatalf("message = %q", message)
	}
	assertStatusMessageDoesNotContainCredentialMaterial(t, err)
}

func TestAccessHandler_RequestAccessHidesServiceErrorDetails(t *testing.T) {
	handler := NewAccessHandlerWithSessions(&fakeAccessService{
		err: errors.New("assume role failed: access-key-id secret-access-key session-token cloudiam-session"),
	}, &fakeSessionResolver{
		ok:      true,
		session: auth.Session{Principal: "akira"},
	})

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if response != nil {
		t.Fatalf("response = %+v, want nil", response)
	}
	if code := status.Code(err); code != codes.Internal {
		t.Fatalf("code = %s, want %s", code, codes.Internal)
	}
	assertStatusMessageDoesNotContainCredentialMaterial(t, err)
}

func assertStatusMessageDoesNotContainCredentialMaterial(t *testing.T, err error) {
	t.Helper()

	message := status.Convert(err).Message()
	for _, forbidden := range []string{
		"access-key-id",
		"secret-access-key",
		"session-token",
		"cloudiam-session",
		"aws_access_key_id",
		"aws_secret_access_key",
		"aws_session_token",
	} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("status message contains credential material %q: %s", forbidden, message)
		}
	}
}
