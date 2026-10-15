package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/audit"
)

type fakeAuditStore struct {
	events []audit.Event
	limit  int
	err    error
}

func (f *fakeAuditStore) ListRecentEvents(ctx context.Context, limit int) ([]audit.Event, error) {
	f.limit = limit
	return f.events, f.err
}

func TestAuditHandlerListEvents(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 15, 0, 0, time.UTC)
	store := &fakeAuditStore{events: []audit.Event{
		{
			ID:                  42,
			OccurredAt:          time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
			EventType:           audit.EventTypeAccessIssued,
			Principal:           "akira",
			Target:              "aws-sandbox-readonly",
			Decision:            audit.DecisionAllowed,
			Reason:              "policy matched",
			Provider:            "aws",
			Account:             "sandbox",
			Role:                "readonly",
			AccessRequestID:     int64Ptr(42),
			OperatorSubject:     "admin",
			IdempotencyKeyHash:  "6dbd925562509bc30f5277ed4b5a35d2869c07b16c258f33fae541435d9dd619",
			CredentialExpiresAt: &expiresAt,
			CreatedAt:           time.Date(2026, 10, 7, 12, 0, 1, 0, time.UTC),
		},
	}}
	mux := http.NewServeMux()
	NewAuditHandler(store).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/audit/events?limit=10", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.limit != 10 {
		t.Fatalf("limit = %d, want 10", store.limit)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`"event_type":"access_issued"`,
		`"principal":"akira"`,
		`"target":"aws-sandbox-readonly"`,
		`"decision":"allowed"`,
		`"access_request_id":42`,
		`"operator_subject":"admin"`,
		`"idempotency_key_hash":"6dbd925562509bc30f5277ed4b5a35d2869c07b16c258f33fae541435d9dd619"`,
		`"credential_expires_at":"2026-10-07T12:15:00Z"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %s: %s", want, body)
		}
	}
	for _, secretField := range []string{"secret_access_key", "session_token", "device_code", "user_code"} {
		if strings.Contains(body, secretField) {
			t.Fatalf("body contains secret field %q: %s", secretField, body)
		}
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}

func TestAuditHandlerListEventsValidation(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{
			name:   "method not allowed",
			method: http.MethodPost,
			path:   "/audit/events",
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "invalid limit",
			method: http.MethodGet,
			path:   "/audit/events?limit=abc",
			status: http.StatusBadRequest,
		},
		{
			name:   "limit too high",
			method: http.MethodGet,
			path:   "/audit/events?limit=101",
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			NewAuditHandler(&fakeAuditStore{}).Register(mux)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.status, rec.Body.String())
			}
		})
	}
}

func TestAuditHandlerListEventsInternalError(t *testing.T) {
	mux := http.NewServeMux()
	NewAuditHandler(&fakeAuditStore{err: errors.New("database offline")}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/audit/events", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
