package consoleauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordParams struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

const (
	minPasswordMemoryKiB   uint32 = 19 * 1024
	maxPasswordMemoryKiB   uint32 = 256 * 1024
	minPasswordIterations  uint32 = 1
	maxPasswordIterations  uint32 = 10
	minPasswordParallelism uint8  = 1
	maxPasswordParallelism uint8  = 8
	minPasswordSaltLength  uint32 = 16
	maxPasswordSaltLength  uint32 = 64
	minPasswordKeyLength   uint32 = 16
	maxPasswordKeyLength   uint32 = 64
)

var DefaultPasswordParams = PasswordParams{
	MemoryKiB:   64 * 1024,
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

func HashPassword(password string, params PasswordParams) (string, error) {
	if params.MemoryKiB == 0 {
		params = DefaultPasswordParams
	}
	if !validPasswordParams(params) {
		return "", fmt.Errorf("argon2id parameters are invalid")
	}

	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKiB, params.Parallelism, params.KeyLength)
	return fmt.Sprintf(
		"argon2id:v=19:m=%d,t=%d,p=%d:%s:%s",
		params.MemoryKiB,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	params, salt, expected, err := parsePasswordHash(encodedHash)
	if err != nil {
		return false, err
	}
	keyLength, ok := boundedPasswordLength(len(expected), minPasswordKeyLength, maxPasswordKeyLength)
	if !ok {
		return false, ErrInvalidPasswordHash
	}
	actual := argon2.IDKey([]byte(password), salt, params.Iterations, params.MemoryKiB, params.Parallelism, keyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func parsePasswordHash(encodedHash string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, ":")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != "v=19" {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}

	var params PasswordParams
	for _, field := range strings.Split(parts[2], ",") {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
		}
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
		}
		switch key {
		case "m":
			params.MemoryKiB = uint32(parsed)
		case "t":
			params.Iterations = uint32(parsed)
		case "p":
			if parsed > 255 {
				return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
			}
			params.Parallelism = uint8(parsed)
		default:
			return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
		}
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}

	saltLength, ok := boundedPasswordLength(len(salt), minPasswordSaltLength, maxPasswordSaltLength)
	if !ok {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	keyLength, ok := boundedPasswordLength(len(expected), minPasswordKeyLength, maxPasswordKeyLength)
	if !ok {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}

	if !validPasswordParams(PasswordParams{
		MemoryKiB:   params.MemoryKiB,
		Iterations:  params.Iterations,
		Parallelism: params.Parallelism,
		SaltLength:  saltLength,
		KeyLength:   keyLength,
	}) {
		return PasswordParams{}, nil, nil, ErrInvalidPasswordHash
	}
	return params, salt, expected, nil
}

func boundedPasswordLength(length int, minValue, maxValue uint32) (uint32, bool) {
	for value := minValue; value <= maxValue; value++ {
		if int(value) == length {
			return value, true
		}
	}
	return 0, false
}

func validPasswordParams(params PasswordParams) bool {
	return params.MemoryKiB >= minPasswordMemoryKiB &&
		params.MemoryKiB <= maxPasswordMemoryKiB &&
		params.Iterations >= minPasswordIterations &&
		params.Iterations <= maxPasswordIterations &&
		params.Parallelism >= minPasswordParallelism &&
		params.Parallelism <= maxPasswordParallelism &&
		params.SaltLength >= minPasswordSaltLength &&
		params.SaltLength <= maxPasswordSaltLength &&
		params.KeyLength >= minPasswordKeyLength &&
		params.KeyLength <= maxPasswordKeyLength
}
