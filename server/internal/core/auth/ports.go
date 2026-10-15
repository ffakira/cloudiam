package auth

import (
	"context"
	"time"
)

type Store interface {
	CreateDeviceAuthorization(ctx context.Context, input CreateDeviceAuthorizationInput) (DeviceAuthorization, error)
	FindDeviceAuthorizationByDeviceCodeHash(ctx context.Context, hash Hash) (DeviceAuthorization, bool, error)
	FindDeviceAuthorizationByUserCodeHash(ctx context.Context, hash Hash) (DeviceAuthorization, bool, error)
	ApproveDeviceAuthorization(ctx context.Context, input ApproveDeviceAuthorizationInput) (DeviceAuthorization, error)
	DenyDeviceAuthorization(ctx context.Context, input DenyDeviceAuthorizationInput) (DeviceAuthorization, error)
	ExpireDeviceAuthorization(ctx context.Context, input ExpireDeviceAuthorizationInput) (DeviceAuthorization, error)
	ExchangeApprovedDeviceAuthorization(ctx context.Context, input ExchangeApprovedDeviceAuthorizationInput) (Session, error)
	FindSessionByTokenHash(ctx context.Context, hash Hash) (Session, bool, error)
	RevokeSession(ctx context.Context, input RevokeSessionInput) (Session, error)
}

type CreateDeviceAuthorizationInput struct {
	DeviceCodeHash  Hash
	UserCodeHash    Hash
	UserCodeDisplay string
	Workspace       string
	Endpoint        string
	Interval        time.Duration
	ExpiresAt       time.Time
}

type ApproveDeviceAuthorizationInput struct {
	UserCodeHash Hash
	Principal    string
	ApprovedAt   time.Time
}

type DenyDeviceAuthorizationInput struct {
	UserCodeHash Hash
	DeniedAt     time.Time
}

type ExpireDeviceAuthorizationInput struct {
	DeviceCodeHash Hash
	ExpiredAt      time.Time
}

type ExchangeApprovedDeviceAuthorizationInput struct {
	DeviceCodeHash   Hash
	SessionTokenHash Hash
	SessionExpiresAt time.Time
	ConsumedAt       time.Time
}

type RevokeSessionInput struct {
	SessionTokenHash Hash
	RevokedAt        time.Time
}
