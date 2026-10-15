package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/approval"
	"github.com/ffakira/cloudiam/server/internal/core/audit"
	"github.com/ffakira/cloudiam/server/internal/core/consoleauth"
)

type fakeConsoleAuthService struct {
	session      consoleauth.Session
	principal    consoleauth.Principal
	loginErr     error
	refreshErr   error
	verifyErr    error
	loginCalls   int
	refreshCalls int
	verifyCalls  int
}

func (f *fakeConsoleAuthService) Login(ctx context.Context, username, password string) (consoleauth.Session, error) {
	f.loginCalls++
	return f.session, f.loginErr
}

func (f *fakeConsoleAuthService) Refresh(ctx context.Context, token string) (consoleauth.Session, error) {
	f.refreshCalls++
	return f.session, f.refreshErr
}

func (f *fakeConsoleAuthService) VerifyToken(token string) (consoleauth.Principal, error) {
	f.verifyCalls++
	return f.principal, f.verifyErr
}

type fakeStore struct {
	requests     []access.AccessRequest
	mutateInput  approval.MutationInput
	mutateResult approval.MutationResult
	listCalls    int
	mutateCalls  int
	err          error
}

func (f *fakeStore) ListAccessRequests(ctx context.Context, status access.AccessRequestStatus) ([]access.AccessRequest, error) {
	f.listCalls++
	return f.requests, f.err
}

func (f *fakeStore) MutateAccessRequest(ctx context.Context, input approval.MutationInput) (approval.MutationResult, error) {
	f.mutateCalls++
	f.mutateInput = input
	return f.mutateResult, f.err
}

type fakeConsoleAuditStore struct {
	fakeAuditStore
	recorded  []audit.RecordEventInput
	recordErr error
}

func (f *fakeConsoleAuditStore) RecordEvent(ctx context.Context, input audit.RecordEventInput) error {
	f.recorded = append(f.recorded, input)
	return f.recordErr
}

func TestConsoleHandlerLogin(t *testing.T) {
	expiresAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		session: consoleauth.Session{
			AccessToken: "jwt",
			TokenType:   "Bearer",
			ExpiresAt:   expiresAt,
		},
	}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "jwt") || strings.Contains(rec.Body.String(), "access_token") {
		t.Fatalf("login response leaked token: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"role":"console_operator"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != consoleSessionCookieName || cookies[0].Value != "jwt" || !cookies[0].HttpOnly {
		t.Fatalf("cookie = %+v", cookies[0])
	}
}

func TestConsoleHandlerLoginCanSetSecureCookie(t *testing.T) {
	expiresAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		session: consoleauth.Session{
			AccessToken: "jwt",
			TokenType:   "Bearer",
			ExpiresAt:   expiresAt,
		},
	}, &fakeStore{}).WithSecureCookies(true).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure {
		t.Fatalf("cookies = %+v", cookies)
	}
}

func TestConsoleHandlerLoginRejectsTrailingJSON(t *testing.T) {
	mux := http.NewServeMux()
	authService := &fakeConsoleAuthService{
		session: consoleauth.Session{
			AccessToken: "jwt",
			TokenType:   "Bearer",
			ExpiresAt:   time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
		},
	}
	NewConsoleHandler(authService, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(`{"username":"admin","password":"secret"}{"role":"console_operator"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(rec.Body.String(), "jwt") {
		t.Fatalf("login validation response leaked secret material: %s", rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatalf("trailing-json login set cookies: %+v", rec.Result().Cookies())
	}
	if authService.loginCalls != 0 {
		t.Fatalf("trailing-json login reached auth service: calls=%d", authService.loginCalls)
	}
}

func TestConsoleHandlerLoginRejectsMalformedBodiesBeforeAuth(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "unknown field",
			body: `{"username":"admin","password":"secret","access_token":"jwt"}`,
		},
		{
			name: "oversized body",
			body: `{"username":"admin","password":"` + strings.Repeat("a", 1<<20) + `"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authService := &fakeConsoleAuthService{
				session: consoleauth.Session{
					AccessToken: "jwt",
					TokenType:   "Bearer",
					ExpiresAt:   time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
				},
			}
			mux := http.NewServeMux()
			NewConsoleHandler(authService, &fakeStore{}).Register(mux)

			req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if authService.loginCalls != 0 {
				t.Fatalf("malformed login reached auth service: calls=%d", authService.loginCalls)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatalf("malformed login set cookies: %+v", rec.Result().Cookies())
			}
			for _, forbidden := range []string{"secret", "jwt", "access_token"} {
				if strings.Contains(rec.Body.String(), forbidden) {
					t.Fatalf("malformed login response leaked %q: %s", forbidden, rec.Body.String())
				}
			}
		})
	}
}

func TestConsoleHandlerRequiresSession(t *testing.T) {
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/access-requests?status=pending", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestConsoleHandlerRejectsBearerSession(t *testing.T) {
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/access-requests?status=pending", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestConsoleHandlerRejectsWrongRoleOnProtectedRoutes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		setup  func(*http.Request)
	}{
		{
			name:   "me",
			method: http.MethodGet,
			path:   "/api/admin/me",
		},
		{
			name:   "list access requests",
			method: http.MethodGet,
			path:   "/api/admin/access-requests?status=pending",
		},
		{
			name:   "mutate access request",
			method: http.MethodPost,
			path:   "/api/admin/access-requests/42/approve",
			setup: func(r *http.Request) {
				r.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
			},
		},
		{
			name:   "audit events",
			method: http.MethodGet,
			path:   "/api/admin/audit/events?limit=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			auditStore := &fakeConsoleAuditStore{}
			mux := http.NewServeMux()
			NewConsoleHandler(&fakeConsoleAuthService{
				principal: consoleauth.Principal{Subject: "viewer", Role: "viewer"},
			}, store).WithAuditStore(auditStore).Register(mux)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
			if tt.setup != nil {
				tt.setup(req)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if store.listCalls != 0 || store.mutateCalls != 0 || len(auditStore.recorded) != 0 || auditStore.limit != 0 {
				t.Fatalf("wrong-role request reached protected stores: list=%d mutate=%d audit-record=%d audit-limit=%d", store.listCalls, store.mutateCalls, len(auditStore.recorded), auditStore.limit)
			}
			assertConsoleResponseDoesNotContainSecrets(t, rec.Body.String())
		})
	}
}

func TestConsoleHandlerRefreshRejectsWrongRoleBeforeSettingSessionCookie(t *testing.T) {
	expiresAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		session:   consoleauth.Session{AccessToken: "new-jwt", TokenType: "Bearer", ExpiresAt: expiresAt},
		principal: consoleauth.Principal{Subject: "viewer", Role: "viewer", ExpiresAt: expiresAt},
	}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/refresh", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "old-jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "new-jwt") || strings.Contains(body, "old-jwt") {
		t.Fatalf("refresh wrong-role response leaked token: %s", body)
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Value == "new-jwt" {
			t.Fatalf("wrong-role refresh set refreshed session cookie: %+v", cookie)
		}
	}
}

func TestConsoleHandlerRejectsBodiesOnNoBodyMutations(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		setup  func(*http.Request)
	}{
		{
			name:   "logout",
			method: http.MethodPost,
			path:   "/api/admin/logout",
		},
		{
			name:   "refresh",
			method: http.MethodPost,
			path:   "/api/admin/refresh",
		},
		{
			name:   "approve",
			method: http.MethodPost,
			path:   "/api/admin/access-requests/42/approve",
			setup: func(r *http.Request) {
				r.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
			},
		},
		{
			name:   "deny",
			method: http.MethodPost,
			path:   "/api/admin/access-requests/42/deny",
			setup: func(r *http.Request) {
				r.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			authService := &fakeConsoleAuthService{
				session: consoleauth.Session{
					AccessToken: "new-jwt",
					TokenType:   "Bearer",
					ExpiresAt:   time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
				},
				principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
			}
			mux := http.NewServeMux()
			NewConsoleHandler(authService, store).Register(mux)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{"principal":"spoofed","role_arn":"arn:aws:iam::000000000000:role/admin","user_code":"ABCD-1234"}`))
			req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
			if tt.setup != nil {
				tt.setup(req)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if store.mutateCalls != 0 || store.listCalls != 0 {
				t.Fatalf("no-body mutation reached store: list=%d mutate=%d", store.listCalls, store.mutateCalls)
			}
			assertConsoleResponseDoesNotContainSecrets(t, rec.Body.String())
		})
	}
}

func TestConsoleHandlerRejectsChunkedBodiesOnNoBodyMutationsBeforeAuth(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		setup  func(*http.Request)
	}{
		{
			name:   "logout",
			method: http.MethodPost,
			path:   "/api/admin/logout",
		},
		{
			name:   "refresh",
			method: http.MethodPost,
			path:   "/api/admin/refresh",
		},
		{
			name:   "approve",
			method: http.MethodPost,
			path:   "/api/admin/access-requests/42/approve",
			setup: func(r *http.Request) {
				r.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
			},
		},
		{
			name:   "deny",
			method: http.MethodPost,
			path:   "/api/admin/access-requests/42/deny",
			setup: func(r *http.Request) {
				r.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			authService := &fakeConsoleAuthService{
				session: consoleauth.Session{
					AccessToken: "new-jwt",
					TokenType:   "Bearer",
					ExpiresAt:   time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC),
				},
				principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
			}
			mux := http.NewServeMux()
			NewConsoleHandler(authService, store).Register(mux)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{"principal":"spoofed","aws_secret_access_key":"secret-access-key"}`))
			req.ContentLength = -1
			req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
			if tt.setup != nil {
				tt.setup(req)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			if authService.refreshCalls != 0 || authService.verifyCalls != 0 {
				t.Fatalf("chunked no-body mutation reached auth service: refresh=%d verify=%d", authService.refreshCalls, authService.verifyCalls)
			}
			if store.mutateCalls != 0 || store.listCalls != 0 {
				t.Fatalf("chunked no-body mutation reached store: list=%d mutate=%d", store.listCalls, store.mutateCalls)
			}
			assertConsoleResponseDoesNotContainSecrets(t, rec.Body.String())
		})
	}
}

func TestConsoleHandlerMeValidatesCookie(t *testing.T) {
	expiresAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole, ExpiresAt: expiresAt},
	}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"authenticated":true`) || strings.Contains(rec.Body.String(), "jwt") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func assertConsoleResponseDoesNotContainSecrets(t *testing.T, body string) {
	t.Helper()

	for _, forbidden := range []string{
		"jwt",
		"cloudiam_session_token",
		"session_token",
		"secret-access-key",
		"aws_secret_access_key",
		"a8098c1a-f86e-4c89-947d-0d3a2b316e11",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response contains forbidden value %q: %s", forbidden, body)
		}
	}
}

func TestConsoleHandlerRefreshRotatesCookie(t *testing.T) {
	expiresAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		session:   consoleauth.Session{AccessToken: "new-jwt", TokenType: "Bearer", ExpiresAt: expiresAt},
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole, ExpiresAt: expiresAt},
	}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/refresh", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "old-jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != "new-jwt" || !cookies[0].HttpOnly {
		t.Fatalf("cookies = %+v", cookies)
	}
	if strings.Contains(rec.Body.String(), "new-jwt") {
		t.Fatalf("refresh response leaked token: %s", rec.Body.String())
	}
}

func TestConsoleHandlerRefreshSetsCookieLifetimeFromHandlerClock(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	handler := NewConsoleHandler(&fakeConsoleAuthService{
		session:   consoleauth.Session{AccessToken: "new-jwt", TokenType: "Bearer", ExpiresAt: expiresAt},
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole, ExpiresAt: expiresAt},
	}, &fakeStore{})
	handler.now = func() time.Time {
		return now
	}
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/refresh", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "old-jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %+v", cookies)
	}
	if cookies[0].MaxAge != int(time.Hour.Seconds()) {
		t.Fatalf("cookie max age = %d, want %d", cookies[0].MaxAge, int(time.Hour.Seconds()))
	}
}

func TestConsoleHandlerApprovesWithIdempotencyKey(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		mutateResult: approval.MutationResult{
			AccessRequest: access.AccessRequest{
				ID:                42,
				Principal:         "alice",
				Target:            "aws-sandbox-readonly",
				Provider:          "aws",
				Account:           "sandbox",
				Role:              "readonly",
				RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
				RequestedDuration: 15 * time.Minute,
				Status:            access.AccessRequestStatusApproved,
				ExpiresAt:         now.Add(10 * time.Minute),
				ApprovedBy:        "admin",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	auditStore := &fakeConsoleAuditStore{}
	handler := NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, store).WithAuditStore(auditStore)
	handler.now = func() time.Time {
		return now
	}
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/access-requests/42/approve", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	req.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.mutateInput.AccessRequestID != 42 {
		t.Fatalf("access request id = %d, want 42", store.mutateInput.AccessRequestID)
	}
	if store.mutateInput.Action != approval.ActionApprove {
		t.Fatalf("action = %q, want approve", store.mutateInput.Action)
	}
	if store.mutateInput.ApproverSubject != "admin" {
		t.Fatalf("approver subject = %q, want admin", store.mutateInput.ApproverSubject)
	}
	if store.mutateInput.Fingerprint == "" {
		t.Fatal("expected fingerprint")
	}
	if !strings.Contains(rec.Body.String(), `"status":"approved"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if len(auditStore.recorded) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.recorded))
	}
	event := auditStore.recorded[0]
	if event.EventType != audit.EventTypeApprovalApproved {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeApprovalApproved)
	}
	if event.OperatorSubject != "admin" {
		t.Fatalf("operator subject = %q, want admin", event.OperatorSubject)
	}
	if event.AccessRequestID == nil || *event.AccessRequestID != 42 {
		t.Fatalf("access request id = %v, want 42", event.AccessRequestID)
	}
	if event.IdempotencyKeyHash != hashID("a8098c1a-f86e-4c89-947d-0d3a2b316e11") {
		t.Fatalf("idempotency key hash = %q", event.IdempotencyKeyHash)
	}
	if strings.Contains(event.IdempotencyKeyHash, "a8098c1a") {
		t.Fatalf("idempotency key hash contains raw key material: %q", event.IdempotencyKeyHash)
	}
	if event.Principal != "alice" || event.Target != "aws-sandbox-readonly" {
		t.Fatalf("principal/target = %q/%q", event.Principal, event.Target)
	}
	for _, secret := range []string{"a8098c1a-f86e-4c89-947d-0d3a2b316e11", "jwt", "session-token", "secret-access-key"} {
		if strings.Contains(strings.Join([]string{event.Principal, event.Target, event.Reason, event.OperatorSubject}, " "), secret) {
			t.Fatalf("approval audit event contains secret %q: %+v", secret, event)
		}
	}
}

func TestConsoleHandlerReplayedApprovalSkipsDuplicateAuditEvent(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		mutateResult: approval.MutationResult{
			Replayed: true,
			AccessRequest: access.AccessRequest{
				ID:                42,
				Principal:         "alice",
				Target:            "aws-sandbox-readonly",
				Provider:          "aws",
				Account:           "sandbox",
				Role:              "readonly",
				RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
				RequestedDuration: 15 * time.Minute,
				Status:            access.AccessRequestStatusApproved,
				ExpiresAt:         now.Add(10 * time.Minute),
				ApprovedBy:        "admin",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	auditStore := &fakeConsoleAuditStore{}
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, store).WithAuditStore(auditStore).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/access-requests/42/approve", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	req.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Idempotency-Replayed"); got != "true" {
		t.Fatalf("Idempotency-Replayed = %q, want true", got)
	}
	if len(auditStore.recorded) != 0 {
		t.Fatalf("replayed approval recorded duplicate audit events: %+v", auditStore.recorded)
	}
}

func TestConsoleHandlerDeniesWithIdempotencyKey(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		mutateResult: approval.MutationResult{
			AccessRequest: access.AccessRequest{
				ID:                42,
				Principal:         "alice",
				Target:            "aws-sandbox-readonly",
				Provider:          "aws",
				Account:           "sandbox",
				Role:              "readonly",
				RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
				RequestedDuration: 15 * time.Minute,
				Status:            access.AccessRequestStatusDenied,
				ExpiresAt:         now.Add(10 * time.Minute),
				DeniedBy:          "admin",
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}
	auditStore := &fakeConsoleAuditStore{}
	handler := NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, store).WithAuditStore(auditStore)
	handler.now = func() time.Time {
		return now
	}
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/access-requests/42/deny", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	req.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.mutateInput.Action != approval.ActionDeny {
		t.Fatalf("action = %q, want deny", store.mutateInput.Action)
	}
	if store.mutateInput.ApproverSubject != "admin" {
		t.Fatalf("approver subject = %q, want admin", store.mutateInput.ApproverSubject)
	}
	if !strings.Contains(rec.Body.String(), `"status":"denied"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if len(auditStore.recorded) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.recorded))
	}
	event := auditStore.recorded[0]
	if event.EventType != audit.EventTypeApprovalDenied {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeApprovalDenied)
	}
	if event.Decision != audit.DecisionDenied {
		t.Fatalf("decision = %q, want %q", event.Decision, audit.DecisionDenied)
	}
	if event.OperatorSubject != "admin" {
		t.Fatalf("operator subject = %q, want admin", event.OperatorSubject)
	}
	if event.IdempotencyKeyHash != hashID("a8098c1a-f86e-4c89-947d-0d3a2b316e11") {
		t.Fatalf("idempotency key hash = %q", event.IdempotencyKeyHash)
	}
}

func TestConsoleHandlerRecordsExpiredApprovalAuditEvent(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{
		mutateResult: approval.MutationResult{
			AccessRequest: access.AccessRequest{
				ID:                42,
				Principal:         "alice",
				Target:            "aws-sandbox-readonly",
				Provider:          "aws",
				Account:           "sandbox",
				Role:              "readonly",
				RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
				RequestedDuration: 15 * time.Minute,
				PolicyReason:      "policy matched; approval required",
				Status:            access.AccessRequestStatusExpired,
				ExpiresAt:         now.Add(-time.Second),
				CreatedAt:         now.Add(-time.Hour),
				UpdatedAt:         now,
			},
		},
		err: approval.ErrAccessRequestExpired,
	}
	auditStore := &fakeConsoleAuditStore{}
	handler := NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, store).WithAuditStore(auditStore)
	handler.now = func() time.Time {
		return now
	}
	mux := http.NewServeMux()
	handler.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/access-requests/42/approve", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	req.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(auditStore.recorded) != 1 {
		t.Fatalf("audit events = %d, want 1", len(auditStore.recorded))
	}
	event := auditStore.recorded[0]
	if event.EventType != audit.EventTypeApprovalExpired {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeApprovalExpired)
	}
	if event.Decision != audit.DecisionDenied {
		t.Fatalf("decision = %q, want %q", event.Decision, audit.DecisionDenied)
	}
	if event.Reason != "access request expired" {
		t.Fatalf("reason = %q", event.Reason)
	}
	if event.IdempotencyKeyHash != hashID("a8098c1a-f86e-4c89-947d-0d3a2b316e11") {
		t.Fatalf("idempotency key hash = %q", event.IdempotencyKeyHash)
	}
}

func TestConsoleHandlerRejectsMissingIdempotencyKey(t *testing.T) {
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/access-requests/42/approve", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestConsoleHandlerRejectsMalformedIdempotencyKey(t *testing.T) {
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/access-requests/42/approve", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	req.Header.Set("Idempotency-Key", "not-a-uuid")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestConsoleHandlerRejectsMalformedMutationPathBeforeStore(t *testing.T) {
	tests := []string{
		"/api/admin/access-requests/not-a-number/approve",
		"/api/admin/access-requests/0/approve",
		"/api/admin/access-requests/42/archive",
		"/api/admin/access-requests/42/approve/extra",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			store := &fakeStore{}
			auditStore := &fakeConsoleAuditStore{}
			mux := http.NewServeMux()
			NewConsoleHandler(&fakeConsoleAuthService{
				principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
			}, store).WithAuditStore(auditStore).Register(mux)

			req := httptest.NewRequest(http.MethodPost, path, nil)
			req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
			req.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
			}
			if store.mutateCalls != 0 || store.listCalls != 0 || len(auditStore.recorded) != 0 {
				t.Fatalf("malformed mutation path reached stores: list=%d mutate=%d audit=%d", store.listCalls, store.mutateCalls, len(auditStore.recorded))
			}
			assertConsoleResponseDoesNotContainSecrets(t, rec.Body.String())
		})
	}
}

func TestConsoleHandlerMapsIdempotencyConflict(t *testing.T) {
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, &fakeStore{err: approval.ErrIdempotencyConflict}).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/access-requests/42/deny", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	req.Header.Set("Idempotency-Key", "a8098c1a-f86e-4c89-947d-0d3a2b316e11")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestConsoleHandlerMapsInvalidToken(t *testing.T) {
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{verifyErr: errors.New("bad token")}, &fakeStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/access-requests?status=pending", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "invalid-token"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestConsoleHandlerListsPendingAccessRequestsWithoutSecrets(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, &fakeStore{
		requests: []access.AccessRequest{
			{
				ID:                42,
				Principal:         "alice",
				Target:            "aws-sandbox-readonly",
				Provider:          "aws",
				Account:           "sandbox",
				Role:              "readonly",
				RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
				RequestedDuration: 15 * time.Minute,
				PolicyReason:      "policy matched",
				Status:            access.AccessRequestStatusPendingApproval,
				ExpiresAt:         now.Add(10 * time.Minute),
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/access-requests?status=pending", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"access_requests":[`) || !strings.Contains(body, `"principal":"alice"`) {
		t.Fatalf("body = %s", body)
	}
	for _, forbidden := range []string{
		"cloudiam_session_token",
		"session_token",
		"session_token_hash",
		"device_code",
		"user_code",
		"aws_access_key_id",
		"aws_secret_access_key",
		"aws_session_token",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body contains secret field %q: %s", forbidden, body)
		}
	}
}

func TestConsoleHandlerRejectsInvalidListStatusBeforeStore(t *testing.T) {
	store := &fakeStore{}
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, store).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/access-requests?status=issued%00", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if store.listCalls != 0 || store.mutateCalls != 0 {
		t.Fatalf("invalid status reached store: list=%d mutate=%d", store.listCalls, store.mutateCalls)
	}
	assertConsoleResponseDoesNotContainSecrets(t, rec.Body.String())
}

func TestConsoleHandlerSecuresAuditEvents(t *testing.T) {
	mux := http.NewServeMux()
	NewConsoleHandler(&fakeConsoleAuthService{
		principal: consoleauth.Principal{Subject: "admin", Role: consoleauth.ConsoleOperatorRole},
	}, &fakeStore{}).WithAuditStore(&fakeConsoleAuditStore{}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/audit/events?limit=10", nil)
	req.AddCookie(&http.Cookie{Name: consoleSessionCookieName, Value: "jwt"})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"events":[]`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
