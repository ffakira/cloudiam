package consoleauth

import (
	"context"
	"time"
)

type User struct {
	Username     string
	PasswordHash string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserStore interface {
	GetUser(ctx context.Context, username string) (User, bool, error)
}
