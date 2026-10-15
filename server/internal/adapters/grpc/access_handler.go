package grpcadapter

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"

	accessv1 "github.com/ffakira/cloudiam/server/gen/access/v1"
	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccessService interface {
	RequestAccess(ctx context.Context, request access.Request) (access.Result, error)
}

type AccessHandler struct {
	accessv1.UnimplementedAccessServiceServer
	service         AccessService
	sessionResolver SessionResolver
}

func NewAccessHandler(service AccessService) *AccessHandler {
	return &AccessHandler{service: service}
}

type SessionResolver interface {
	FindSessionByTokenHash(ctx context.Context, hash auth.Hash) (auth.Session, bool, error)
}

func NewAccessHandlerWithSessions(service AccessService, sessionResolver SessionResolver) *AccessHandler {
	return &AccessHandler{
		service:         service,
		sessionResolver: sessionResolver,
	}
}

func (h *AccessHandler) RequestAccess(ctx context.Context, request *accessv1.RequestAccessRequest) (*accessv1.RequestAccessResponse, error) {
	if request.GetDurationSeconds() < 0 {
		return nil, status.Error(codes.InvalidArgument, "duration_seconds must be greater than or equal to 0")
	}
	principal, err := h.resolvePrincipal(ctx, request)
	if err != nil {
		return nil, err
	}

	result, err := h.service.RequestAccess(ctx, access.Request{
		Principal: principal,
		Target:    request.GetTarget(),
		Duration:  time.Duration(request.GetDurationSeconds()) * time.Second,
	})
	if err != nil {
		return nil, mapError(err)
	}

	if result.Status == access.ResultStatusPendingApproval {
		return &accessv1.RequestAccessResponse{
			Status:            string(result.Status),
			AccessRequestId:   result.AccessRequest.ID,
			ApprovalExpiresAt: result.AccessRequest.ExpiresAt.Format(time.RFC3339),
		}, nil
	}

	return &accessv1.RequestAccessResponse{
		Status:          string(access.ResultStatusIssued),
		AccessKeyId:     result.Grant.AccessKeyID,
		SecretAccessKey: result.Grant.SecretAccessKey,
		SessionToken:    result.Grant.SessionToken,
		ExpiresAt:       result.Grant.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (h *AccessHandler) resolvePrincipal(ctx context.Context, request *accessv1.RequestAccessRequest) (string, error) {
	if h.sessionResolver == nil {
		return request.GetPrincipal(), nil
	}
	token := strings.TrimSpace(request.GetCloudiamSessionToken())
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "cloudiam session token is required")
	}
	hash := sha256.Sum256([]byte(token))
	session, ok, err := h.sessionResolver.FindSessionByTokenHash(ctx, auth.Hash(hash))
	if err != nil {
		return "", status.Error(codes.Internal, "resolve cloudiam session failed")
	}
	if !ok {
		return "", status.Error(codes.Unauthenticated, "cloudiam session is invalid or expired")
	}
	return session.Principal, nil
}

func mapError(err error) error {
	var validationErr access.ValidationError
	if errors.As(err, &validationErr) {
		return status.Error(codes.InvalidArgument, validationErr.Error())
	}
	if errors.Is(err, access.ErrAccessDenied) {
		return status.Error(codes.PermissionDenied, "access denied")
	}
	if errors.Is(err, access.ErrCredentialIssuanceDenied) {
		return status.Error(codes.FailedPrecondition, "credential issuance denied by cloud provider")
	}
	return status.Error(codes.Internal, "request access failed")
}
