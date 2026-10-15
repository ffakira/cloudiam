package consoleauth

import "errors"

var (
	ErrInvalidCredentials  = errors.New("invalid console credentials")
	ErrInvalidToken        = errors.New("invalid console token")
	ErrDisabled            = errors.New("console auth disabled")
	ErrInvalidPasswordHash = errors.New("invalid console password hash")
)
