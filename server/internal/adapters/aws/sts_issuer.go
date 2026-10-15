package awsadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/smithy-go"
	"github.com/ffakira/cloudiam/server/internal/core/access"
)

const defaultAssumeRoleTimeout = 10 * time.Second

type AssumeRoleClient interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

type STSIssuer struct {
	client  AssumeRoleClient
	timeout time.Duration
}

func NewSTSIssuer(client AssumeRoleClient) *STSIssuer {
	return &STSIssuer{
		client:  client,
		timeout: defaultAssumeRoleTimeout,
	}
}

func (i *STSIssuer) Issue(ctx context.Context, request access.IssueRequest) (access.Grant, error) {
	if strings.TrimSpace(request.RoleARN) == "" {
		return access.Grant{}, fmt.Errorf("role arn is required")
	}
	if request.Duration <= 0 {
		return access.Grant{}, fmt.Errorf("duration must be positive")
	}

	ctx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	durationSeconds := int32(request.Duration.Seconds())
	out, err := i.client.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(request.RoleARN),
		RoleSessionName: aws.String(roleSessionName(request.Principal)),
		SourceIdentity:  aws.String(sourceIdentity(request.Principal)),
		Tags:            sessionTags(request),
		DurationSeconds: aws.Int32(durationSeconds),
	})
	if err != nil {
		if isAccessDenied(err) {
			return access.Grant{}, fmt.Errorf("%w: assume role: %w", access.ErrCredentialIssuanceDenied, err)
		}
		return access.Grant{}, fmt.Errorf("assume role: %w", err)
	}
	if out.Credentials == nil {
		return access.Grant{}, fmt.Errorf("assume role: missing credentials")
	}
	if strings.TrimSpace(aws.ToString(out.Credentials.AccessKeyId)) == "" ||
		strings.TrimSpace(aws.ToString(out.Credentials.SecretAccessKey)) == "" ||
		strings.TrimSpace(aws.ToString(out.Credentials.SessionToken)) == "" ||
		aws.ToTime(out.Credentials.Expiration).IsZero() {
		return access.Grant{}, fmt.Errorf("assume role: incomplete credentials")
	}

	return access.Grant{
		AccessKeyID:     aws.ToString(out.Credentials.AccessKeyId),
		SecretAccessKey: aws.ToString(out.Credentials.SecretAccessKey),
		SessionToken:    aws.ToString(out.Credentials.SessionToken),
		ExpiresAt:       aws.ToTime(out.Credentials.Expiration),
	}, nil
}

func isAccessDenied(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.EqualFold(apiErr.ErrorCode(), "AccessDenied") ||
		strings.EqualFold(apiErr.ErrorCode(), "AccessDeniedException")
}

func roleSessionName(principal string) string {
	return awsIdentityValue(principal, "cloudiam-session", 64)
}

func sourceIdentity(principal string) string {
	const prefix = "cloudiam-"
	cleaned := sanitizeAWSIdentityValue(principal, 64-len(prefix))
	if len(cleaned) < 2 {
		cleaned = "principal"
	}
	return prefix + cleaned
}

func awsIdentityValue(value, fallback string, maxLen int) string {
	cleaned := sanitizeAWSIdentityValue(value, maxLen)
	if len(cleaned) < 2 {
		return fallback
	}
	return cleaned
}

func sanitizeAWSIdentityValue(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || strings.ContainsRune("+=,.@-", r):
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
		if b.Len() >= maxLen {
			break
		}
	}
	return b.String()
}

func sessionTags(request access.IssueRequest) []ststypes.Tag {
	return []ststypes.Tag{
		sessionTag("cloudiam_principal", request.Principal),
		sessionTag("cloudiam_target", request.Target),
		sessionTag("cloudiam_provider", request.Provider),
		sessionTag("cloudiam_account", request.Account),
		sessionTag("cloudiam_role", request.Role),
	}
}

func sessionTag(key, value string) ststypes.Tag {
	return ststypes.Tag{
		Key:   aws.String(key),
		Value: aws.String(sessionTagValue(value)),
	}
}

func sessionTagValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 256 {
		value = value[:256]
	}

	var b strings.Builder
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			b.WriteRune('-')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
