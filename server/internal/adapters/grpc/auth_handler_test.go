package grpcadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	authv1 "github.com/ffakira/cloudiam/server/gen/auth/v1"
	"github.com/ffakira/cloudiam/server/internal/core/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeAuthService struct {
	startResult     auth.StartDeviceLoginResult
	sessionResult   auth.SessionResult
	startErr        error
	approveErr      error
	exchangeErr     error
	approvedCode    string
	approvedUser    string
	exchangedDevice string
}

func (f *fakeAuthService) StartDeviceLogin(ctx context.Context) (auth.StartDeviceLoginResult, error) {
	return f.startResult, f.startErr
}

func (f *fakeAuthService) ApproveDeviceLogin(ctx context.Context, userCode, principal string) error {
	f.approvedCode = userCode
	f.approvedUser = principal
	return f.approveErr
}

func (f *fakeAuthService) ExchangeDeviceCode(ctx context.Context, deviceCode string) (auth.SessionResult, error) {
	f.exchangedDevice = deviceCode
	return f.sessionResult, f.exchangeErr
}

func TestAuthHandler_StartDeviceLogin(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 10, 0, 0, time.UTC)
	handler := NewAuthHandler(&fakeAuthService{
		startResult: auth.StartDeviceLoginResult{
			DeviceCode:      "device-code",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://cloudiam.abacaxi.dev/device",
			ExpiresAt:       expiresAt,
			Interval:        5 * time.Second,
		},
	})

	response, err := handler.StartDeviceLogin(context.Background(), &authv1.StartDeviceLoginRequest{})
	if err != nil {
		t.Fatalf("start device login: %v", err)
	}
	if response.GetDeviceCode() != "device-code" {
		t.Fatalf("device code = %q", response.GetDeviceCode())
	}
	if response.GetUserCode() != "ABCD-1234" {
		t.Fatalf("user code = %q", response.GetUserCode())
	}
	if response.GetExpiresAt() != expiresAt.Format(time.RFC3339) {
		t.Fatalf("expires at = %q", response.GetExpiresAt())
	}
	if response.GetIntervalSeconds() != 5 {
		t.Fatalf("interval seconds = %d", response.GetIntervalSeconds())
	}
}

func TestAuthHandler_ApproveDeviceLogin(t *testing.T) {
	service := &fakeAuthService{}
	handler := NewAuthHandler(service)

	_, err := handler.ApproveDeviceLogin(context.Background(), &authv1.ApproveDeviceLoginRequest{
		UserCode:  "ABCD-1234",
		Principal: "akira",
	})
	if err != nil {
		t.Fatalf("approve device login: %v", err)
	}
	if service.approvedCode != "ABCD-1234" {
		t.Fatalf("approved code = %q", service.approvedCode)
	}
	if service.approvedUser != "akira" {
		t.Fatalf("approved user = %q", service.approvedUser)
	}
}

func TestAuthHandler_ExchangeDeviceCode(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 13, 0, 0, 0, time.UTC)
	service := &fakeAuthService{
		sessionResult: auth.SessionResult{
			AccessToken: "cloudiam-session",
			Workspace:   "sandbox",
			Principal:   "akira",
			Endpoint:    "cloudiam.internal.local",
			ExpiresAt:   expiresAt,
		},
	}
	handler := NewAuthHandler(service)

	response, err := handler.ExchangeDeviceCode(context.Background(), &authv1.ExchangeDeviceCodeRequest{
		DeviceCode: "device-code",
	})
	if err != nil {
		t.Fatalf("exchange device code: %v", err)
	}
	if service.exchangedDevice != "device-code" {
		t.Fatalf("exchanged device = %q", service.exchangedDevice)
	}
	if response.GetAccessToken() != "cloudiam-session" {
		t.Fatalf("access token = %q", response.GetAccessToken())
	}
	if response.GetExpiresAt() != expiresAt.Format(time.RFC3339) {
		t.Fatalf("expires at = %q", response.GetExpiresAt())
	}
}

func TestAuthHandler_ValidationErrors(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{})

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "approve missing user code",
			call: func() error {
				_, err := handler.ApproveDeviceLogin(context.Background(), &authv1.ApproveDeviceLoginRequest{Principal: "akira"})
				return err
			},
		},
		{
			name: "approve missing principal",
			call: func() error {
				_, err := handler.ApproveDeviceLogin(context.Background(), &authv1.ApproveDeviceLoginRequest{UserCode: "ABCD-1234"})
				return err
			},
		},
		{
			name: "exchange missing device code",
			call: func() error {
				_, err := handler.ExchangeDeviceCode(context.Background(), &authv1.ExchangeDeviceCodeRequest{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error")
			}
			if code := status.Code(err); code != codes.InvalidArgument {
				t.Fatalf("code = %s, want %s", code, codes.InvalidArgument)
			}
		})
	}
}

func TestAuthHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "not found", err: fmt.Errorf("%w: device-code user-code cloudiam-session", auth.ErrNotFound), code: codes.NotFound},
		{name: "pending", err: fmt.Errorf("%w: device-code user-code cloudiam-session", auth.ErrAuthorizationPending), code: codes.FailedPrecondition},
		{name: "denied", err: fmt.Errorf("%w: device-code user-code cloudiam-session", auth.ErrAuthorizationDenied), code: codes.PermissionDenied},
		{name: "expired", err: fmt.Errorf("%w: device-code user-code cloudiam-session", auth.ErrExpired), code: codes.DeadlineExceeded},
		{name: "invalid state", err: fmt.Errorf("%w: device-code user-code cloudiam-session", auth.ErrInvalidState), code: codes.FailedPrecondition},
		{name: "internal", err: errors.New("database offline with device-code user-code cloudiam-session"), code: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(&fakeAuthService{exchangeErr: tt.err})
			_, err := handler.ExchangeDeviceCode(context.Background(), &authv1.ExchangeDeviceCodeRequest{
				DeviceCode: "device-code",
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if code := status.Code(err); code != tt.code {
				t.Fatalf("code = %s, want %s", code, tt.code)
			}
			assertAuthStatusMessageDoesNotContainSecretMaterial(t, err)
		})
	}
}

func TestAuthHandler_StartAndApproveInternalErrorsDoNotLeakSecrets(t *testing.T) {
	tests := []struct {
		name string
		call func(handler *AuthHandler) error
	}{
		{
			name: "start",
			call: func(handler *AuthHandler) error {
				_, err := handler.StartDeviceLogin(context.Background(), &authv1.StartDeviceLoginRequest{})
				return err
			},
		},
		{
			name: "approve",
			call: func(handler *AuthHandler) error {
				_, err := handler.ApproveDeviceLogin(context.Background(), &authv1.ApproveDeviceLoginRequest{
					UserCode:  "ABCD-1234",
					Principal: "akira",
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(&fakeAuthService{
				startErr:   errors.New("device-code user-code cloudiam-session"),
				approveErr: errors.New("device-code user-code cloudiam-session"),
			})

			err := tt.call(handler)
			if err == nil {
				t.Fatal("expected error")
			}
			if code := status.Code(err); code != codes.Internal {
				t.Fatalf("code = %s, want %s", code, codes.Internal)
			}
			assertAuthStatusMessageDoesNotContainSecretMaterial(t, err)
		})
	}
}

func assertAuthStatusMessageDoesNotContainSecretMaterial(t *testing.T, err error) {
	t.Helper()

	message := status.Convert(err).Message()
	for _, forbidden := range []string{
		"device-code",
		"user-code",
		"cloudiam-session",
		"raw-device-code",
		"raw-user-code",
		"raw-session-token",
		"ABCD-1234",
	} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("status message contains secret material %q: %s", forbidden, message)
		}
	}
}
