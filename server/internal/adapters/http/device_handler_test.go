package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/auth"
)

type fakeAuthService struct {
	userCode  string
	principal string
	err       error
}

func (f *fakeAuthService) ApproveDeviceLogin(ctx context.Context, userCode, principal string) error {
	f.userCode = userCode
	f.principal = principal
	return f.err
}

type fakeRateLimiter struct {
	allowed bool
	key     string
	limit   int
	window  time.Duration
}

func (f *fakeRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) bool {
	f.key = key
	f.limit = limit
	f.window = window
	return f.allowed
}

func TestDeviceHandlerApprove(t *testing.T) {
	service := &fakeAuthService{}
	mux := http.NewServeMux()
	NewDeviceHandler(service).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/device/approve", strings.NewReader(`{"user_code":"ABCD-1234","principal":"akira"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if service.userCode != "ABCD-1234" {
		t.Fatalf("user code = %q", service.userCode)
	}
	if service.principal != "akira" {
		t.Fatalf("principal = %q", service.principal)
	}
	if !strings.Contains(rec.Body.String(), `"status":"approved"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestDeviceHandlerApproveRateLimited(t *testing.T) {
	service := &fakeAuthService{}
	limiter := &fakeRateLimiter{allowed: false}
	mux := http.NewServeMux()
	NewDeviceHandler(service).WithRateLimiter(limiter).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/device/approve", strings.NewReader(`{"user_code":"ABCD-1234","principal":"akira"}`))
	req.RemoteAddr = "203.0.113.10:12345"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if service.userCode != "" || service.principal != "" {
		t.Fatalf("rate limited request called service with user_code=%q principal=%q", service.userCode, service.principal)
	}
	if limiter.key != "device_approve:203.0.113.10" {
		t.Fatalf("rate limit key = %q", limiter.key)
	}
	if limiter.limit != deviceApprovalRateLimit || limiter.window != deviceApprovalRateLimitWindow {
		t.Fatalf("rate limit config = %d/%s", limiter.limit, limiter.window)
	}
	assertNoSecretMaterial(t, rec.Body.String())
}

func TestDeviceHandlerApproveValidation(t *testing.T) {
	oversizedBody := `{"user_code":"ABCD-1234","principal":"` + strings.Repeat("a", 1<<20) + `"}`
	tests := []struct {
		name   string
		method string
		body   string
		status int
	}{
		{
			name:   "method not allowed",
			method: http.MethodGet,
			body:   "",
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid json",
			method: http.MethodPost,
			body:   `{`,
			status: http.StatusBadRequest,
		},
		{
			name:   "missing user code",
			method: http.MethodPost,
			body:   `{"principal":"akira"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "missing principal",
			method: http.MethodPost,
			body:   `{"user_code":"ABCD-1234"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "unknown field",
			method: http.MethodPost,
			body:   `{"user_code":"ABCD-1234","principal":"akira","device_code":"secret-device-code"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "trailing json",
			method: http.MethodPost,
			body:   `{"user_code":"ABCD-1234","principal":"akira"}{"principal":"admin"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "oversized body",
			method: http.MethodPost,
			body:   oversizedBody,
			status: http.StatusBadRequest,
		},
		{
			name:   "overlong user code",
			method: http.MethodPost,
			body:   `{"user_code":"` + strings.Repeat("A", maxDeviceApprovalUserCodeLength+1) + `","principal":"akira"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "overlong principal",
			method: http.MethodPost,
			body:   `{"user_code":"ABCD-1234","principal":"` + strings.Repeat("a", maxDeviceApprovalPrincipalLength+1) + `"}`,
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeAuthService{}
			mux := http.NewServeMux()
			NewDeviceHandler(service).Register(mux)

			req := httptest.NewRequest(tt.method, "/api/device/approve", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.status, rec.Body.String())
			}
			if service.userCode != "" || service.principal != "" {
				t.Fatalf("validation failure called service with user_code=%q principal=%q", service.userCode, service.principal)
			}
			assertNoSecretMaterial(t, rec.Body.String())
		})
	}
}

func TestDeviceHandlerApproveTrimsInput(t *testing.T) {
	service := &fakeAuthService{}
	mux := http.NewServeMux()
	NewDeviceHandler(service).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/device/approve", strings.NewReader(`{"user_code":" ABCD-1234 ","principal":" akira "}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if service.userCode != "ABCD-1234" {
		t.Fatalf("user code = %q", service.userCode)
	}
	if service.principal != "akira" {
		t.Fatalf("principal = %q", service.principal)
	}
}

func TestDeviceHandlerApproveStateErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{
			name:    "invalid code",
			err:     auth.ErrNotFound,
			status:  http.StatusNotFound,
			message: "device authorization not found",
		},
		{
			name:    "expired code",
			err:     auth.ErrExpired,
			status:  http.StatusGone,
			message: "device authorization expired",
		},
		{
			name:    "already used or non pending",
			err:     auth.ErrInvalidState,
			status:  http.StatusConflict,
			message: "device authorization cannot be approved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewDeviceHandler(&fakeAuthService{err: tt.err}).Register(mux)

			req := httptest.NewRequest(http.MethodPost, "/api/device/approve", strings.NewReader(`{"user_code":"ABCD-1234","principal":"akira"}`))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.status, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.message) {
				t.Fatalf("body = %s, want message %q", rec.Body.String(), tt.message)
			}
		})
	}
}

func TestDeviceHandlerApproveInternalError(t *testing.T) {
	mux := http.NewServeMux()
	NewDeviceHandler(&fakeAuthService{err: errors.New("database offline with ABCD-1234 akira device-code session-token")}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/device/approve", strings.NewReader(`{"user_code":"ABCD-1234","principal":"akira"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	assertNoSecretMaterial(t, rec.Body.String())
}

func TestDeviceHandlerApproveResponsesDoNotExposeSecrets(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		err    error
		status int
	}{
		{name: "success", path: "/api/device/approve", status: http.StatusOK},
		{name: "invalid code", path: "/api/device/approve", err: auth.ErrNotFound, status: http.StatusNotFound},
		{name: "expired code", path: "/api/device/approve", err: auth.ErrExpired, status: http.StatusGone},
		{name: "invalid state", path: "/api/device/approve", err: auth.ErrInvalidState, status: http.StatusConflict},
		{name: "deprecated raw endpoint", path: "/device/approve", status: http.StatusGone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewDeviceHandler(&fakeAuthService{err: tt.err}).Register(mux)

			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(`{"user_code":"ABCD-1234","principal":"akira"}`))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.status, rec.Body.String())
			}
			assertNoSecretMaterial(t, rec.Body.String())
		})
	}
}

func TestDeviceHandlerDeprecatedManualApproveEndpoint(t *testing.T) {
	service := &fakeAuthService{}
	mux := http.NewServeMux()
	NewDeviceHandler(service).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/device/approve", strings.NewReader(`{"user_code":"ABCD-1234","principal":"akira"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusGone, rec.Body.String())
	}
	if service.userCode != "" || service.principal != "" {
		t.Fatalf("deprecated endpoint called service with user_code=%q principal=%q", service.userCode, service.principal)
	}
	if !strings.Contains(rec.Body.String(), "open /device in a browser") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func assertNoSecretMaterial(t *testing.T, body string) {
	t.Helper()

	forbidden := []string{
		"device_code",
		"device_code_hash",
		"user_code_hash",
		"session_token",
		"cloudiam_session_token",
		"aws_access_key_id",
		"aws_secret_access_key",
		"aws_session_token",
		"console_jwt",
		"jwt-secret",
		"secret-access-key",
		"ABCD-1234",
		"akira",
		"device-code",
		"session-token",
	}
	for _, value := range forbidden {
		if strings.Contains(strings.ToLower(body), value) {
			t.Fatalf("response exposed forbidden value %q in body %s", value, body)
		}
	}
}
