package cliadapter

import (
	"bytes"
	"context"
	"strings"
	"testing"

	accessv1 "github.com/ffakira/cloudiam/server/gen/access/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeAccessClient struct {
	request  *accessv1.RequestAccessRequest
	response *accessv1.RequestAccessResponse
	err      error
}

func (f *fakeAccessClient) RequestAccess(ctx context.Context, request *accessv1.RequestAccessRequest, opts ...grpc.CallOption) (*accessv1.RequestAccessResponse, error) {
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &accessv1.RequestAccessResponse{
		Status:          "issued",
		AccessKeyId:     "access-key-id",
		SecretAccessKey: "secret-access-key",
		SessionToken:    "session-token",
		ExpiresAt:       "2026-10-07T12:00:00Z",
	}, nil
}

func TestRequestAccessPendingApprovalSummary(t *testing.T) {
	client := &fakeAccessClient{response: &accessv1.RequestAccessResponse{
		Status:            "pending_approval",
		AccessRequestId:   42,
		ApprovalExpiresAt: "2026-10-07T12:15:00Z",
	}}
	var out bytes.Buffer

	err := RequestAccess(context.Background(), client, RequestAccessInput{
		CloudIAMSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
		DurationSeconds:      900,
	}, &out)
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"access pending approval",
		"access_request_id: 42",
		"approval_expires_at: 2026-10-07T12:15:00Z",
		"credentials: not issued",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
	assertDoesNotContainCredentialMaterial(t, output)
}

func TestRequestAccessPendingApprovalRejectsEnvFormat(t *testing.T) {
	client := &fakeAccessClient{response: &accessv1.RequestAccessResponse{
		Status:          "pending_approval",
		AccessRequestId: 42,
	}}
	var out bytes.Buffer

	err := RequestAccess(context.Background(), client, RequestAccessInput{
		CloudIAMSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
		OutputFormat:         "env",
	}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pending approval") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRequestAccessSendsTargetRequest(t *testing.T) {
	client := &fakeAccessClient{}
	var out bytes.Buffer

	err := RequestAccess(context.Background(), client, RequestAccessInput{
		CloudIAMSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
		DurationSeconds:      900,
	}, &out)
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if client.request.GetCloudiamSessionToken() != "cloudiam-session" {
		t.Fatalf("session token = %q, want cloudiam-session", client.request.GetCloudiamSessionToken())
	}
	if client.request.GetTarget() != "aws-sandbox-readonly" {
		t.Fatalf("target = %q, want aws-sandbox-readonly", client.request.GetTarget())
	}
	if client.request.GetDurationSeconds() != 900 {
		t.Fatalf("duration seconds = %d, want 900", client.request.GetDurationSeconds())
	}
	output := out.String()
	if !strings.Contains(output, "access granted") {
		t.Fatalf("output = %q, want access granted summary", output)
	}
	if !strings.Contains(output, "credentials: hidden") {
		t.Fatalf("output = %q, want hidden credential note", output)
	}
	assertDoesNotContainCredentialMaterial(t, output)
}

func TestRequestAccessEnvFormatPrintsShellExports(t *testing.T) {
	client := &fakeAccessClient{}
	var out bytes.Buffer

	err := RequestAccess(context.Background(), client, RequestAccessInput{
		CloudIAMSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
		DurationSeconds:      900,
		OutputFormat:         "env",
	}, &out)
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		`export AWS_ACCESS_KEY_ID="access-key-id"`,
		`export AWS_SECRET_ACCESS_KEY="secret-access-key"`,
		`export AWS_SESSION_TOKEN="session-token"`,
		`export AWS_SESSION_EXPIRATION="2026-10-07T12:00:00Z"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
}

func TestRequestAccessRejectsUnsupportedOutputFormat(t *testing.T) {
	client := &fakeAccessClient{}
	var out bytes.Buffer

	err := RequestAccess(context.Background(), client, RequestAccessInput{
		CloudIAMSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
		OutputFormat:         "json",
	}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `unsupported output format "json"`) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRequestAccessFormatsCredentialIssuanceDenied(t *testing.T) {
	client := &fakeAccessClient{
		err: status.Error(codes.FailedPrecondition, "credential issuance denied by cloud provider"),
	}
	var out bytes.Buffer

	err := RequestAccess(context.Background(), client, RequestAccessInput{
		CloudIAMSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
	}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	for _, want := range []string{
		`access request failed for target "aws-sandbox-readonly"`,
		"AWS denied credential issuance",
		"sts:AssumeRole",
		"sts:SetSourceIdentity",
		"sts:TagSession",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want %q", message, want)
		}
	}
	for _, raw := range []string{"rpc error", "arn:aws:iam", "AccessDenied"} {
		if strings.Contains(message, raw) {
			t.Fatalf("error leaked raw provider detail %q: %q", raw, message)
		}
	}
}

func TestRequestAccessFormatsUnauthenticated(t *testing.T) {
	client := &fakeAccessClient{
		err: status.Error(codes.Unauthenticated, "cloudiam session is invalid or expired"),
	}
	var out bytes.Buffer

	err := RequestAccess(context.Background(), client, RequestAccessInput{
		CloudIAMSessionToken: "cloudiam-session",
		Target:               "aws-sandbox-readonly",
	}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "cloudiam session is invalid or expired; run login again" {
		t.Fatalf("error = %q", err.Error())
	}
}

func assertDoesNotContainCredentialMaterial(t *testing.T, output string) {
	t.Helper()
	for _, secret := range []string{
		"access-key-id",
		"secret-access-key",
		"session-token",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_SESSION_TOKEN",
	} {
		if strings.Contains(output, secret) {
			t.Fatalf("output leaked credential material %q: %q", secret, output)
		}
	}
}
