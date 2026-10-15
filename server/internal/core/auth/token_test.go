package auth

import (
	"strings"
	"testing"
)

func TestSecureTokenGenerator(t *testing.T) {
	generator := SecureTokenGenerator{}

	deviceCode, err := generator.DeviceCode()
	if err != nil {
		t.Fatalf("device code: %v", err)
	}
	if len(deviceCode) < 40 {
		t.Fatalf("device code length = %d, want at least 40", len(deviceCode))
	}

	sessionToken, err := generator.SessionToken()
	if err != nil {
		t.Fatalf("session token: %v", err)
	}
	if len(sessionToken) < 40 {
		t.Fatalf("session token length = %d, want at least 40", len(sessionToken))
	}
	if deviceCode == sessionToken {
		t.Fatal("device code and session token should differ")
	}

	userCode, err := generator.UserCode()
	if err != nil {
		t.Fatalf("user code: %v", err)
	}
	if len(userCode) != len("ABCD-1234") {
		t.Fatalf("user code length = %d", len(userCode))
	}
	if userCode[4] != '-' {
		t.Fatalf("user code = %q, want hyphen separator", userCode)
	}
}

func TestNormalizeUserCode(t *testing.T) {
	got := NormalizeUserCode(" abcd  -1234 ")
	if got != "ABCD-1234" {
		t.Fatalf("normalized code = %q, want ABCD-1234", got)
	}
}

func TestHashToken(t *testing.T) {
	hash := HashToken("raw-token")
	if len(hash) != 32 {
		t.Fatalf("hash length = %d, want 32", len(hash))
	}
	if strings.Contains(string(hash[:]), "raw-token") {
		t.Fatal("hash contains raw token")
	}
}
