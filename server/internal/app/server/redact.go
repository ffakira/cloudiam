package serverapp

import (
	"errors"
	"regexp"
)

var (
	secretURLUserinfoPattern = regexp.MustCompile(`(?i)\b(postgres|postgresql|redis)://([^:/@\s]+):([^@\s]+)@`)
	secretKeyValuePattern    = regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|token|jwt|device[_-]?code|user[_-]?code|idempotency[_-]?key|request[_-]?fingerprint|response[_-]?body|aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret[_-]?access[_-]?key|aws[_-]?session[_-]?token)(["']?\s*[:=]\s*["']?)[^"'\s,;]+`)
	awsAccessKeyPattern      = regexp.MustCompile(`\b(AKIA|ASIA)[A-Z0-9]{16}\b`)
)

func safeErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	message = secretURLUserinfoPattern.ReplaceAllString(message, `${1}://${2}:<redacted>@`)
	message = secretKeyValuePattern.ReplaceAllString(message, `${1}${2}<redacted>`)
	message = awsAccessKeyPattern.ReplaceAllString(message, `<redacted>`)
	return message
}

func redactedError(prefix string, err error) error {
	if err == nil {
		return errors.New(prefix)
	}
	return errors.New(prefix + ": " + safeErrorMessage(err))
}
