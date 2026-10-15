package postgresadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/consoleauth"
)

type ConsoleUserStore struct {
	db *sql.DB
}

func NewConsoleUserStore(db *sql.DB) *ConsoleUserStore {
	return &ConsoleUserStore{db: db}
}

type SeedConsoleUserInput struct {
	Username     string
	PasswordHash string
	Active       bool
	SeededAt     time.Time
}

func (s *ConsoleUserStore) SeedUser(ctx context.Context, input SeedConsoleUserInput) error {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if !strings.HasPrefix(input.PasswordHash, "argon2id:") {
		return consoleauth.ErrInvalidPasswordHash
	}
	seededAt := input.SeededAt
	if seededAt.IsZero() {
		seededAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO console_users (
			username,
			password_hash,
			active,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
			active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at
	`, username, input.PasswordHash, input.Active, seededAt)
	if err != nil {
		return fmt.Errorf("seed console user: %w", err)
	}
	return nil
}

func (s *ConsoleUserStore) GetUser(ctx context.Context, username string) (consoleauth.User, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT username, password_hash, active, created_at, updated_at
		FROM console_users
		WHERE username = $1
	`, strings.TrimSpace(username))

	var user consoleauth.User
	err := row.Scan(
		&user.Username,
		&user.PasswordHash,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return consoleauth.User{}, false, nil
		}
		return consoleauth.User{}, false, fmt.Errorf("get console user: %w", err)
	}
	return user, true, nil
}
