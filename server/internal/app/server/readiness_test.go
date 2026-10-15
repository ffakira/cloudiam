package serverapp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadinessCheckerReportsReady(t *testing.T) {
	checker := newReadinessChecker(map[string]readinessDependency{
		"config": readinessDependencyFunc(func(ctx context.Context) error { return nil }),
	})

	response := checker.Check(context.Background())
	if response.Status != "ready" {
		t.Fatalf("status = %q, want ready", response.Status)
	}
	if len(response.Checks) != 1 || response.Checks[0].Name != "config" || response.Checks[0].Status != "ok" {
		t.Fatalf("checks = %+v", response.Checks)
	}
}

func TestReadinessCheckerReportsNotReadyWithRedactedErrors(t *testing.T) {
	checker := newReadinessChecker(map[string]readinessDependency{
		"postgres": readinessDependencyFunc(func(ctx context.Context) error {
			return errors.New("connect postgres://cloudiam:db-password@db/cloudiam jwt=console-jwt AWS_SECRET_ACCESS_KEY=secret-access-key")
		}),
	})

	response := checker.Check(context.Background())
	if response.Status != "not_ready" {
		t.Fatalf("status = %q, want not_ready", response.Status)
	}
	if len(response.Checks) != 1 {
		t.Fatalf("checks = %+v", response.Checks)
	}
	check := response.Checks[0]
	if check.Name != "postgres" || check.Status != "failed" {
		t.Fatalf("check = %+v", check)
	}
	for _, secret := range []string{"db-password", "console-jwt", "secret-access-key"} {
		if strings.Contains(check.Error, secret) {
			t.Fatalf("readiness error leaked %q: %q", secret, check.Error)
		}
	}
}

func TestReadinessHandlerUsesServiceUnavailableWhenNotReady(t *testing.T) {
	checker := newReadinessChecker(map[string]readinessDependency{
		"redis": readinessDependencyFunc(func(ctx context.Context) error {
			return errors.New("redis unavailable")
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	checker.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	var response readinessResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "not_ready" {
		t.Fatalf("response = %+v", response)
	}
}

func TestReadinessHandlerRejectsNonGET(t *testing.T) {
	checker := newReadinessChecker(nil)

	req := httptest.NewRequest(http.MethodPost, "/readyz", nil)
	rec := httptest.NewRecorder()
	checker.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status code = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}
