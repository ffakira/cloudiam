package consoleauth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashPasswordUsesArgon2idAndRandomSalt(t *testing.T) {
	first, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash first password: %v", err)
	}
	second, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash second password: %v", err)
	}
	if !strings.HasPrefix(first, "argon2id:v=19:") {
		t.Fatalf("hash = %q, want argon2id prefix", first)
	}
	if first == second {
		t.Fatal("expected random salt to produce different hashes")
	}
}

func TestVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	ok, err := VerifyPassword("correct-password", hash)
	if err != nil {
		t.Fatalf("verify password: %v", err)
	}
	if !ok {
		t.Fatal("expected password to match")
	}

	ok, err = VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}
	if ok {
		t.Fatal("wrong password should not match")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	_, err := VerifyPassword("password", "sha256:not-allowed")
	if !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("error = %v, want invalid password hash", err)
	}
}

func TestVerifyPasswordRejectsOutOfBoundsArgon2idCost(t *testing.T) {
	hash, err := HashPassword("correct-password", testPasswordParams)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	tooExpensive := strings.Replace(hash, "m=19456,", "m=999999999,", 1)

	_, err = VerifyPassword("correct-password", tooExpensive)
	if !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("error = %v, want invalid password hash", err)
	}
}
