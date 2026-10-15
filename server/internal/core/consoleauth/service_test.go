package consoleauth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var testPasswordParams = PasswordParams{
	MemoryKiB:   19 * 1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

type fakeUserStore struct {
	user User
	ok   bool
	err  error
}

func (f fakeUserStore) GetUser(ctx context.Context, username string) (User, bool, error) {
	if f.err != nil {
		return User{}, false, f.err
	}
	if !f.ok || username != f.user.Username {
		return User{}, false, nil
	}
	return f.user, true, nil
}

func TestServiceLoginAndVerifyToken(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	passwordHash, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	service, err := NewService(ServiceConfig{
		UserStore: fakeUserStore{
			user: User{Username: "admin", PasswordHash: passwordHash, Active: true},
			ok:   true,
		},
		JWTSecret: "test-secret",
		TokenTTL:  time.Hour,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	session, err := service.Login(t.Context(), "admin", "correct-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if session.TokenType != "Bearer" {
		t.Fatalf("token type = %q, want Bearer", session.TokenType)
	}
	if !session.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expires at = %s, want %s", session.ExpiresAt, now.Add(time.Hour))
	}

	principal, err := service.VerifyToken(session.AccessToken)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if principal.Subject != "admin" || principal.Role != ConsoleOperatorRole {
		t.Fatalf("principal = %+v", principal)
	}
}

func TestServiceRejectsInvalidCredentials(t *testing.T) {
	passwordHash, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	service, err := NewService(ServiceConfig{
		UserStore: fakeUserStore{
			user: User{Username: "admin", PasswordHash: passwordHash, Active: true},
			ok:   true,
		},
		JWTSecret: "test-secret",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.Login(t.Context(), "admin", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want invalid credentials", err)
	}
}

func TestServiceRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	passwordHash, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	service, err := NewService(ServiceConfig{
		UserStore: fakeUserStore{
			user: User{Username: "admin", PasswordHash: passwordHash, Active: true},
			ok:   true,
		},
		JWTSecret: "test-secret",
		TokenTTL:  time.Hour,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	session, err := service.Login(t.Context(), "admin", "correct-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	service.now = func() time.Time {
		return now.Add(2 * time.Hour)
	}
	_, err = service.VerifyToken(session.AccessToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want invalid token", err)
	}
}

func TestServiceRefreshRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	passwordHash, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	service, err := NewService(ServiceConfig{
		UserStore: fakeUserStore{
			user: User{Username: "admin", PasswordHash: passwordHash, Active: true},
			ok:   true,
		},
		JWTSecret: "test-secret",
		TokenTTL:  time.Hour,
		Now: func() time.Time {
			return now
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	session, err := service.Login(t.Context(), "admin", "correct-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	service.now = func() time.Time {
		return now.Add(2 * time.Hour)
	}
	refreshed, err := service.Refresh(t.Context(), session.AccessToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want invalid token", err)
	}
	if refreshed.AccessToken != "" {
		t.Fatalf("refreshed access token = %q, want empty", refreshed.AccessToken)
	}
}

func TestServiceRejectsTamperedToken(t *testing.T) {
	passwordHash, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	service, err := NewService(ServiceConfig{
		UserStore: fakeUserStore{
			user: User{Username: "admin", PasswordHash: passwordHash, Active: true},
			ok:   true,
		},
		JWTSecret: "test-secret",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	session, err := service.Login(t.Context(), "admin", "correct-password")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	parts := strings.Split(session.AccessToken, ".")
	if len(parts) != 3 || len(parts[2]) == 0 {
		t.Fatalf("unexpected token shape: %q", session.AccessToken)
	}
	replacement := "a"
	if strings.HasPrefix(parts[2], replacement) {
		replacement = "b"
	}
	parts[2] = replacement + parts[2][1:]
	tampered := strings.Join(parts, ".")
	_, err = service.VerifyToken(tampered)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want invalid token", err)
	}
}
