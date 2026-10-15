package approval

import "errors"

var (
	ErrInvalidIdempotency        = errors.New("invalid idempotency key")
	ErrIdempotencyConflict       = errors.New("idempotency key reused with different request")
	ErrAccessRequestNotFound     = errors.New("access request not found")
	ErrAccessRequestExpired      = errors.New("access request expired")
	ErrAccessRequestInvalidState = errors.New("access request cannot be changed")
)
