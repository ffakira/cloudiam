package awsadapter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/smithy-go"
	"github.com/ffakira/cloudiam/server/internal/core/access"
)

type fakeAssumeRoleClient struct {
	calls  int
	input  *sts.AssumeRoleInput
	output *sts.AssumeRoleOutput
	err    error
}

func (f *fakeAssumeRoleClient) AssumeRole(ctx context.Context, input *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return nil, f.err
	}
	if f.output != nil {
		return f.output, nil
	}
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	return &sts.AssumeRoleOutput{
		Credentials: &ststypes.Credentials{
			AccessKeyId:     aws.String("access-key-id"),
			SecretAccessKey: aws.String("secret-access-key"),
			SessionToken:    aws.String("session-token"),
			Expiration:      aws.Time(expiresAt),
		},
	}, nil
}

func TestSTSIssuer_IssueRequiresResolvedRoleARN(t *testing.T) {
	client := &fakeAssumeRoleClient{}
	issuer := NewSTSIssuer(client)

	_, err := issuer.Issue(context.Background(), access.IssueRequest{
		Principal: "akira",
		Duration:  15 * time.Minute,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if client.calls != 0 {
		t.Fatalf("assume role calls = %d, want 0", client.calls)
	}
}

func TestSTSIssuer_IssueRequiresPositiveDuration(t *testing.T) {
	client := &fakeAssumeRoleClient{}
	issuer := NewSTSIssuer(client)

	_, err := issuer.Issue(context.Background(), access.IssueRequest{
		Principal: "akira",
		RoleARN:   "arn:aws:iam::000000000000:role/jit/readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "duration must be positive") {
		t.Fatalf("error = %q", err.Error())
	}
	if client.calls != 0 {
		t.Fatalf("assume role calls = %d, want 0", client.calls)
	}
}

func TestSTSIssuer_IssueUsesResolvedRoleARN(t *testing.T) {
	client := &fakeAssumeRoleClient{}
	issuer := NewSTSIssuer(client)

	_, err := issuer.Issue(context.Background(), access.IssueRequest{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
		Provider:  "aws",
		Account:   "sandbox",
		Role:      "readonly",
		RoleARN:   "arn:aws:iam::000000000000:role/jit/readonly",
		Duration:  15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("assume role calls = %d, want 1", client.calls)
	}
	if got := aws.ToString(client.input.RoleArn); got != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("role arn = %q", got)
	}
	if got := aws.ToString(client.input.RoleSessionName); got != "akira" {
		t.Fatalf("role session name = %q", got)
	}
	if got := aws.ToString(client.input.SourceIdentity); got != "cloudiam-akira" {
		t.Fatalf("source identity = %q", got)
	}
	if len(client.input.TransitiveTagKeys) != 0 {
		t.Fatalf("transitive tag keys = %v, want none", client.input.TransitiveTagKeys)
	}
	wantTags := map[string]string{
		"cloudiam_principal": "akira",
		"cloudiam_target":    "aws-sandbox-readonly",
		"cloudiam_provider":  "aws",
		"cloudiam_account":   "sandbox",
		"cloudiam_role":      "readonly",
	}
	gotTags := make(map[string]string, len(client.input.Tags))
	for _, tag := range client.input.Tags {
		gotTags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}
	for key, want := range wantTags {
		if got := gotTags[key]; got != want {
			t.Fatalf("tag %s = %q, want %q", key, got, want)
		}
	}
	for _, forbidden := range []string{"secret-access-key", "session-token", "device-code", "user-code"} {
		for key, value := range gotTags {
			if strings.Contains(value, forbidden) {
				t.Fatalf("tag %s contains forbidden value %q", key, forbidden)
			}
		}
	}
}

func TestSTSIssuer_IssueFailsClosedOnProviderErrors(t *testing.T) {
	providerErr := errors.New("sts unavailable with access-key-id secret-access-key session-token")
	client := &fakeAssumeRoleClient{err: providerErr}
	issuer := NewSTSIssuer(client)

	grant, err := issuer.Issue(context.Background(), access.IssueRequest{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
		Provider:  "aws",
		Account:   "sandbox",
		Role:      "readonly",
		RoleARN:   "arn:aws:iam::000000000000:role/jit/readonly",
		Duration:  15 * time.Minute,
	})
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v, want provider error", err)
	}
	if grant != (access.Grant{}) {
		t.Fatalf("grant = %+v, want zero grant", grant)
	}
}

func TestSTSIssuer_IssueClassifiesAccessDenied(t *testing.T) {
	providerErr := &smithy.GenericAPIError{
		Code:    "AccessDenied",
		Message: "not authorized to perform sts:SetSourceIdentity on arn:aws:iam::000000000000:role/jit/readonly",
	}
	client := &fakeAssumeRoleClient{err: providerErr}
	issuer := NewSTSIssuer(client)

	grant, err := issuer.Issue(context.Background(), access.IssueRequest{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
		Provider:  "aws",
		Account:   "sandbox",
		Role:      "readonly",
		RoleARN:   "arn:aws:iam::000000000000:role/jit/readonly",
		Duration:  15 * time.Minute,
	})
	if !errors.Is(err, access.ErrCredentialIssuanceDenied) {
		t.Fatalf("error = %v, want ErrCredentialIssuanceDenied", err)
	}
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v, want provider error in chain", err)
	}
	if grant != (access.Grant{}) {
		t.Fatalf("grant = %+v, want zero grant", grant)
	}
}

func TestSTSIssuer_IssueRejectsMissingOrIncompleteCredentials(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		output *sts.AssumeRoleOutput
	}{
		{
			name:   "missing credentials",
			output: &sts.AssumeRoleOutput{},
		},
		{
			name: "missing access key",
			output: &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
				SecretAccessKey: aws.String("secret-access-key"),
				SessionToken:    aws.String("session-token"),
				Expiration:      aws.Time(expiresAt),
			}},
		},
		{
			name: "missing secret key",
			output: &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
				AccessKeyId:  aws.String("access-key-id"),
				SessionToken: aws.String("session-token"),
				Expiration:   aws.Time(expiresAt),
			}},
		},
		{
			name: "missing session token",
			output: &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
				AccessKeyId:     aws.String("access-key-id"),
				SecretAccessKey: aws.String("secret-access-key"),
				Expiration:      aws.Time(expiresAt),
			}},
		},
		{
			name: "missing expiration",
			output: &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
				AccessKeyId:     aws.String("access-key-id"),
				SecretAccessKey: aws.String("secret-access-key"),
				SessionToken:    aws.String("session-token"),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeAssumeRoleClient{output: tt.output}
			issuer := NewSTSIssuer(client)

			grant, err := issuer.Issue(context.Background(), access.IssueRequest{
				Principal: "akira",
				Target:    "aws-sandbox-readonly",
				Provider:  "aws",
				Account:   "sandbox",
				Role:      "readonly",
				RoleARN:   "arn:aws:iam::000000000000:role/jit/readonly",
				Duration:  15 * time.Minute,
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if grant != (access.Grant{}) {
				t.Fatalf("grant = %+v, want zero grant", grant)
			}
		})
	}
}

func TestSourceIdentity(t *testing.T) {
	tests := []struct {
		name      string
		principal string
		want      string
	}{
		{
			name:      "plain principal",
			principal: "akira",
			want:      "cloudiam-akira",
		},
		{
			name:      "spaces and invalid characters become dashes",
			principal: "akira dev/ops",
			want:      "cloudiam-akira-dev-ops",
		},
		{
			name:      "too short falls back",
			principal: "a",
			want:      "cloudiam-principal",
		},
		{
			name:      "empty falls back",
			principal: " ",
			want:      "cloudiam-principal",
		},
		{
			name:      "reserved aws prefix colon is sanitized",
			principal: "aws:akira",
			want:      "cloudiam-aws-akira",
		},
		{
			name:      "long value is truncated",
			principal: strings.Repeat("a", 70),
			want:      "cloudiam-" + strings.Repeat("a", 55),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sourceIdentity(tt.principal); got != tt.want {
				t.Fatalf("source identity = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionTagValue(t *testing.T) {
	t.Run("control characters are replaced", func(t *testing.T) {
		if got := sessionTagValue("akira\nadmin\tdev"); got != "akira-admin-dev" {
			t.Fatalf("tag value = %q", got)
		}
	})

	t.Run("long values are bounded", func(t *testing.T) {
		got := sessionTagValue(strings.Repeat("a", 300))
		if len(got) != 256 {
			t.Fatalf("tag value length = %d, want 256", len(got))
		}
	})
}

func TestSessionTagsSanitizeBoundaryValues(t *testing.T) {
	tags := sessionTags(access.IssueRequest{
		Principal: "akira\nadmin",
		Target:    "aws-sandbox-readonly\tprod",
		Provider:  "aws",
		Account:   strings.Repeat("a", 300),
		Role:      "readonly\x7fadmin",
	})
	got := make(map[string]string, len(tags))
	for _, tag := range tags {
		got[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	if got["cloudiam_principal"] != "akira-admin" {
		t.Fatalf("principal tag = %q", got["cloudiam_principal"])
	}
	if got["cloudiam_target"] != "aws-sandbox-readonly-prod" {
		t.Fatalf("target tag = %q", got["cloudiam_target"])
	}
	if len(got["cloudiam_account"]) != 256 {
		t.Fatalf("account tag length = %d, want 256", len(got["cloudiam_account"]))
	}
	if got["cloudiam_role"] != "readonly-admin" {
		t.Fatalf("role tag = %q", got["cloudiam_role"])
	}
}
