package auth

import "time"

type Hash [32]byte

type DeviceAuthorizationStatus string

const (
	DeviceAuthorizationStatusPending  DeviceAuthorizationStatus = "pending"
	DeviceAuthorizationStatusApproved DeviceAuthorizationStatus = "approved"
	DeviceAuthorizationStatusDenied   DeviceAuthorizationStatus = "denied"
	DeviceAuthorizationStatusExpired  DeviceAuthorizationStatus = "expired"
	DeviceAuthorizationStatusConsumed DeviceAuthorizationStatus = "consumed"
)

type DeviceAuthorization struct {
	ID              int64
	DeviceCodeHash  Hash
	UserCodeHash    Hash
	UserCodeDisplay string
	Workspace       string
	Principal       string
	Endpoint        string
	Status          DeviceAuthorizationStatus
	Interval        time.Duration
	ExpiresAt       time.Time
	ApprovedAt      *time.Time
	DeniedAt        *time.Time
	ConsumedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Session struct {
	ID                    int64
	SessionTokenHash      Hash
	DeviceAuthorizationID int64
	Workspace             string
	Principal             string
	Endpoint              string
	ExpiresAt             time.Time
	CreatedAt             time.Time
	RevokedAt             *time.Time
}
