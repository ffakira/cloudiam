//go:build integration

package postgresadapter

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/consoleauth"
)

func TestConsoleUserStoreIntegration_SeedAndGetUser(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewConsoleUserStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	username := "console-user-" + now.Format("20060102150405.000000")
	t.Cleanup(func() {
		cleanupConsoleUser(context.Background(), t, db, username)
	})

	passwordHash, err := consoleauth.HashPassword("correct-password", integrationPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if err := store.SeedUser(ctx, SeedConsoleUserInput{
		Username:     username,
		PasswordHash: passwordHash,
		Active:       true,
		SeededAt:     now,
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	user, ok, err := store.GetUser(ctx, username)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if !ok {
		t.Fatal("expected seeded user")
	}
	if user.Username != username || !user.Active {
		t.Fatalf("user = %+v", user)
	}
	matched, err := consoleauth.VerifyPassword("correct-password", user.PasswordHash)
	if err != nil {
		t.Fatalf("verify stored password: %v", err)
	}
	if !matched {
		t.Fatal("stored password hash did not match seed password")
	}

	reseededHash, err := consoleauth.HashPassword("new-password", integrationPasswordParams)
	if err != nil {
		t.Fatalf("hash reseed password: %v", err)
	}
	if err := store.SeedUser(ctx, SeedConsoleUserInput{
		Username:     username,
		PasswordHash: reseededHash,
		Active:       false,
		SeededAt:     now.Add(time.Second),
	}); err != nil {
		t.Fatalf("reseed user: %v", err)
	}

	user, ok, err = store.GetUser(ctx, " "+username+" ")
	if err != nil {
		t.Fatalf("get reseeded user: %v", err)
	}
	if !ok {
		t.Fatal("expected reseeded user")
	}
	if user.Active {
		t.Fatal("expected reseeded user to be inactive")
	}
	matched, err = consoleauth.VerifyPassword("new-password", user.PasswordHash)
	if err != nil {
		t.Fatalf("verify reseeded password: %v", err)
	}
	if !matched {
		t.Fatal("stored password hash did not match reseed password")
	}
}

func TestConsoleUserStoreIntegration_RejectsNonArgon2idSeedHash(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewConsoleUserStore(db)

	err := store.SeedUser(t.Context(), SeedConsoleUserInput{
		Username:     "invalid-console-seed",
		PasswordHash: "sha256:not-accepted",
		Active:       true,
	})
	if !errors.Is(err, consoleauth.ErrInvalidPasswordHash) {
		t.Fatalf("error = %v, want invalid password hash", err)
	}
}

var integrationPasswordParams = consoleauth.PasswordParams{
	MemoryKiB:   19 * 1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func cleanupConsoleUser(ctx context.Context, t *testing.T, db *sql.DB, username string) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		DELETE FROM console_users
		WHERE username = $1
	`, username); err != nil {
		t.Fatalf("cleanup console user: %v", err)
	}
}
