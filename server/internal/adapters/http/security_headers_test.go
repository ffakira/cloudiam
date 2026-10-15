package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headers := rec.Result().Header
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", headers.Get("X-Content-Type-Options"))
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q", headers.Get("X-Frame-Options"))
	}
	if headers.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q", headers.Get("Referrer-Policy"))
	}
	if headers.Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy was not set")
	}
	if headers.Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", headers.Get("Cache-Control"))
	}
}

func TestSecurityHeadersDoNotDisableStaticCaching(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Result().Header.Get("Cache-Control") != "" {
		t.Fatalf("Cache-Control = %q", rec.Result().Header.Get("Cache-Control"))
	}
}
