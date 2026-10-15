package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
)

const userCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type TokenGenerator interface {
	DeviceCode() (string, error)
	UserCode() (string, error)
	SessionToken() (string, error)
}

type SecureTokenGenerator struct{}

func (SecureTokenGenerator) DeviceCode() (string, error) {
	return randomBase64URL(32)
}

func (SecureTokenGenerator) UserCode() (string, error) {
	first, err := randomAlphabetString(4)
	if err != nil {
		return "", err
	}
	second, err := randomAlphabetString(4)
	if err != nil {
		return "", err
	}
	return first + "-" + second, nil
}

func (SecureTokenGenerator) SessionToken() (string, error) {
	return randomBase64URL(32)
}

func HashToken(token string) Hash {
	return sha256.Sum256([]byte(token))
}

func NormalizeUserCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ToUpper(code)
	code = strings.ReplaceAll(code, " ", "")
	return code
}

func randomBase64URL(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func randomAlphabetString(size int) (string, error) {
	var builder strings.Builder
	builder.Grow(size)

	max := big.NewInt(int64(len(userCodeAlphabet)))
	for range size {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("read random index: %w", err)
		}
		builder.WriteByte(userCodeAlphabet[n.Int64()])
	}
	return builder.String(), nil
}
