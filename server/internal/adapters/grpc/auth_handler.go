package grpcadapter

import (
	"context"
	"errors"
	"strings"
	"time"

	authv1 "github.com/ffakira/cloudiam/server/gen/auth/v1"
	"github.com/ffakira/cloudiam/server/internal/core/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService interface {
	StartDeviceLogin(ctx context.Context) (auth.StartDeviceLoginResult, error)
	ApproveDeviceLogin(ctx context.Context, userCode, principal string) error
	ExchangeDeviceCode(ctx context.Context, deviceCode string) (auth.SessionResult, error)
}

type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) StartDeviceLogin(ctx context.Context, request *authv1.StartDeviceLoginRequest) (*authv1.StartDeviceLoginResponse, error) {
	result, err := h.service.StartDeviceLogin(ctx)
	if err != nil {
		return nil, mapAuthError("start device login", err)
	}
	return &authv1.StartDeviceLoginResponse{
		DeviceCode:      result.DeviceCode,
		UserCode:        result.UserCode,
		VerificationUri: result.VerificationURI,
		ExpiresAt:       result.ExpiresAt.Format(time.RFC3339),
		IntervalSeconds: int32(result.Interval.Seconds()),
	}, nil
}

func (h *AuthHandler) ApproveDeviceLogin(ctx context.Context, request *authv1.ApproveDeviceLoginRequest) (*authv1.ApproveDeviceLoginResponse, error) {
	if strings.TrimSpace(request.GetUserCode()) == "" {
		return nil, status.Error(codes.InvalidArgument, "user code is required")
	}
	if strings.TrimSpace(request.GetPrincipal()) == "" {
		return nil, status.Error(codes.InvalidArgument, "principal is required")
	}
	if err := h.service.ApproveDeviceLogin(ctx, request.GetUserCode(), request.GetPrincipal()); err != nil {
		return nil, mapAuthError("approve device login", err)
	}
	return &authv1.ApproveDeviceLoginResponse{}, nil
}

func (h *AuthHandler) ExchangeDeviceCode(ctx context.Context, request *authv1.ExchangeDeviceCodeRequest) (*authv1.ExchangeDeviceCodeResponse, error) {
	if strings.TrimSpace(request.GetDeviceCode()) == "" {
		return nil, status.Error(codes.InvalidArgument, "device code is required")
	}
	result, err := h.service.ExchangeDeviceCode(ctx, request.GetDeviceCode())
	if err != nil {
		return nil, mapAuthError("exchange device code", err)
	}
	return &authv1.ExchangeDeviceCodeResponse{
		AccessToken: result.AccessToken,
		Workspace:   result.Workspace,
		Principal:   result.Principal,
		Endpoint:    result.Endpoint,
		ExpiresAt:   result.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func mapAuthError(operation string, err error) error {
	switch {
	case errors.Is(err, auth.ErrNotFound):
		return status.Error(codes.NotFound, "device authorization not found")
	case errors.Is(err, auth.ErrAuthorizationPending):
		return status.Error(codes.FailedPrecondition, "device authorization is pending")
	case errors.Is(err, auth.ErrAuthorizationDenied):
		return status.Error(codes.PermissionDenied, "device authorization was denied")
	case errors.Is(err, auth.ErrExpired):
		return status.Error(codes.DeadlineExceeded, "device authorization expired")
	case errors.Is(err, auth.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "device authorization cannot be exchanged")
	default:
		return status.Error(codes.Internal, operation+" failed")
	}
}
