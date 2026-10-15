package awsadapter

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

const defaultGetRoleTimeout = 10 * time.Second

var roleNameFromARNPattern = regexp.MustCompile(`^arn:aws[a-zA-Z-]*:iam::[0-9]{12}:role/(?:.*/)?([^/]+)$`)

type GetRoleClient interface {
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
}

type RolePolicyReader struct {
	client  GetRoleClient
	timeout time.Duration
}

func NewRolePolicyReader(client GetRoleClient) *RolePolicyReader {
	return &RolePolicyReader{
		client:  client,
		timeout: defaultGetRoleTimeout,
	}
}

func (r *RolePolicyReader) ReadTrustPolicy(ctx context.Context, roleARN string) ([]byte, error) {
	roleName, err := roleNameFromARN(roleARN)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	out, err := r.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return nil, fmt.Errorf("get role %q: %w", roleName, err)
	}
	if out.Role == nil || strings.TrimSpace(aws.ToString(out.Role.AssumeRolePolicyDocument)) == "" {
		return nil, fmt.Errorf("get role %q: missing assume role policy document", roleName)
	}

	policy := aws.ToString(out.Role.AssumeRolePolicyDocument)
	decoded, err := url.QueryUnescape(policy)
	if err != nil {
		return nil, fmt.Errorf("decode assume role policy document for %q: %w", roleName, err)
	}
	return []byte(decoded), nil
}

func roleNameFromARN(roleARN string) (string, error) {
	matches := roleNameFromARNPattern.FindStringSubmatch(strings.TrimSpace(roleARN))
	if len(matches) != 2 {
		return "", fmt.Errorf("role arn %q is not a valid AWS IAM role ARN", roleARN)
	}
	return matches[1], nil
}
