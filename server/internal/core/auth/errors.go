package auth

import "errors"

var (
	ErrNotFound             = errors.New("auth record not found")
	ErrInvalidState         = errors.New("auth record is in an invalid state")
	ErrExpired              = errors.New("auth record is expired")
	ErrAuthorizationPending = errors.New("device authorization is pending")
	ErrAuthorizationDenied  = errors.New("device authorization was denied")
)
