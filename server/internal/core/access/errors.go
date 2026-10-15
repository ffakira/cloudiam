package access

import (
	"errors"
	"fmt"
)

var (
	ErrAccessDenied             = errors.New("access denied")
	ErrCredentialIssuanceDenied = errors.New("credential issuance denied")
)

type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	if e.Reason == "" {
		return fmt.Sprintf("%s is invalid", e.Field)
	}
	return fmt.Sprintf("%s %s", e.Field, e.Reason)
}
