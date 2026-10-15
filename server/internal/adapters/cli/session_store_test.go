package cliadapter

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionStoreSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cloudiam", "session.json")
	store := NewSessionStore(path)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	want := Session{
		Version:     "v1",
		Workspace:   "sandbox",
		Principal:   "akira",
		Endpoint:    "cloudiam.internal.local",
		AccessToken: "cloudiam-token",
		ExpiresAt:   now.Add(time.Hour),
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("save session: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat session file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("session mode = %o, want 600", mode)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if got != want {
		t.Fatalf("session = %+v, want %+v", got, want)
	}
}

func TestSessionStoreLoadRejectsExpired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	store := NewSessionStore(path)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	err := store.Save(Session{
		Version:     "v1",
		Workspace:   "sandbox",
		Principal:   "akira",
		Endpoint:    "cloudiam.internal.local",
		AccessToken: "cloudiam-token",
		ExpiresAt:   now.Add(-time.Second),
	})
	if err != nil {
		t.Fatalf("save session: %v", err)
	}

	_, err = store.Load()
	if err != ErrSessionExpired {
		t.Fatalf("error = %v, want ErrSessionExpired", err)
	}
}

func TestSessionStoreLoadRejectsMissingAndMalformedFiles(t *testing.T) {
	tests := []struct {
		name    string
		content *string
		wantErr error
	}{
		{
			name:    "missing file",
			wantErr: ErrSessionNotFound,
		},
		{
			name:    "malformed json",
			content: ptrString(`{"version":"v1"`),
		},
		{
			name: "missing access token",
			content: ptrString(`{
				"version": "v1",
				"workspace": "sandbox",
				"principal": "akira",
				"endpoint": "cloudiam.internal.local",
				"expires_at": "2026-10-07T13:00:00Z"
			}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session.json")
			if tt.content != nil {
				if err := os.WriteFile(path, []byte(*tt.content), 0o600); err != nil {
					t.Fatalf("write session fixture: %v", err)
				}
			}
			store := NewSessionStore(path)
			store.now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }

			_, err := store.Load()
			if err == nil {
				t.Fatal("expected error")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if strings.Contains(err.Error(), "cloudiam-token") {
				t.Fatalf("session load error leaked token: %v", err)
			}
		})
	}
}

func TestSessionStoreRejectsForbiddenCredentialFields(t *testing.T) {
	tests := []struct {
		name  string
		field string
	}{
		{name: "aws secret access key", field: "aws_secret_access_key"},
		{name: "aws session token", field: "aws-session-token"},
		{name: "generic session token", field: "session_token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session.json")
			body := `{
				"version": "v1",
				"workspace": "sandbox",
				"principal": "akira",
				"endpoint": "cloudiam.internal.local",
				"access_token": "cloudiam-token",
				"expires_at": "2026-10-07T13:00:00Z",
				"` + tt.field + `": "nope"
			}`
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write session fixture: %v", err)
			}

			store := NewSessionStore(path)
			store.now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }
			_, err := store.Load()
			if err == nil {
				t.Fatal("expected forbidden field error")
			}
		})
	}
}

func ptrString(value string) *string {
	return &value
}
