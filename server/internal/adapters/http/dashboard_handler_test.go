package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDashboardHandlerServesIndexAndAssets(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<div>dashboard</div>"), 0o600); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0o700); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('ok')"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	mux := http.NewServeMux()
	NewDashboardHandler(os.DirFS(dir)).Register(mux)

	tests := []struct {
		path string
		want string
	}{
		{path: "/", want: "dashboard"},
		{path: "/device", want: "dashboard"},
		{path: "/admin/approvals", want: "dashboard"},
		{path: "/assets/app.js", want: "console.log('ok')"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Fatalf("body = %s, want %q", rec.Body.String(), tt.want)
			}
		})
	}
}

func TestDashboardHandlerDoesNotServeUnknownAPIRoutes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<div>dashboard</div>"), 0o600); err != nil {
		t.Fatalf("write index: %v", err)
	}

	mux := http.NewServeMux()
	NewDashboardHandler(os.DirFS(dir)).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
