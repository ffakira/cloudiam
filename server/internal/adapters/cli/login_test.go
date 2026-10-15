package cliadapter

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	authv1 "github.com/ffakira/cloudiam/server/gen/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeAuthClient struct {
	exchanges int
}

func (f *fakeAuthClient) StartDeviceLogin(ctx context.Context, in *authv1.StartDeviceLoginRequest, opts ...grpc.CallOption) (*authv1.StartDeviceLoginResponse, error) {
	return &authv1.StartDeviceLoginResponse{
		DeviceCode:      "device-code",
		UserCode:        "ABCD-1234",
		VerificationUri: "https://cloudiam.abacaxi.dev/device",
		ExpiresAt:       time.Date(2026, 10, 7, 12, 10, 0, 0, time.UTC).Format(time.RFC3339),
		IntervalSeconds: 1,
	}, nil
}

func (f *fakeAuthClient) ExchangeDeviceCode(ctx context.Context, in *authv1.ExchangeDeviceCodeRequest, opts ...grpc.CallOption) (*authv1.ExchangeDeviceCodeResponse, error) {
	f.exchanges++
	if f.exchanges == 1 {
		return nil, status.Error(codes.FailedPrecondition, "device authorization is pending")
	}
	return &authv1.ExchangeDeviceCodeResponse{
		AccessToken: "cloudiam-session-token",
		Workspace:   "sandbox",
		Principal:   "akira",
		Endpoint:    "cloudiam.internal.local",
		ExpiresAt:   time.Date(2026, 10, 7, 13, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}, nil
}

type fakeSessionSaver struct {
	session Session
}

func (f *fakeSessionSaver) Save(session Session) error {
	f.session = session
	return nil
}

func TestLoginPollsAndSavesSession(t *testing.T) {
	client := &fakeAuthClient{}
	store := &fakeSessionSaver{}
	var out bytes.Buffer

	err := Login(context.Background(), client, store, LoginInput{PollTimeout: 3 * time.Second}, &out)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Open: https://cloudiam.abacaxi.dev/device") {
		t.Fatalf("output missing verification uri: %q", output)
	}
	if !strings.Contains(output, "Code: ABCD-1234") {
		t.Fatalf("output missing user code: %q", output)
	}
	if strings.Contains(output, "cloudiam-session-token") {
		t.Fatalf("output leaked session token: %q", output)
	}
	if store.session.AccessToken != "cloudiam-session-token" {
		t.Fatalf("stored token = %q", store.session.AccessToken)
	}
	if store.session.Principal != "akira" {
		t.Fatalf("stored principal = %q", store.session.Principal)
	}
}
