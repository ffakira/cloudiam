package awsadapter

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type fakeGetRoleClient struct {
	input *iam.GetRoleInput
	doc   string
}

func (f *fakeGetRoleClient) GetRole(ctx context.Context, input *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
	f.input = input
	return &iam.GetRoleOutput{
		Role: &iamtypes.Role{
			AssumeRolePolicyDocument: aws.String(f.doc),
		},
	}, nil
}

func TestRolePolicyReader_ReadTrustPolicy(t *testing.T) {
	client := &fakeGetRoleClient{
		doc: `%7B%22Version%22%3A%222012-10-17%22%7D`,
	}
	reader := NewRolePolicyReader(client)

	got, err := reader.ReadTrustPolicy(context.Background(), "arn:aws:iam::000000000000:role/jit/readonly")
	if err != nil {
		t.Fatalf("read trust policy: %v", err)
	}

	if aws.ToString(client.input.RoleName) != "readonly" {
		t.Fatalf("role name = %q, want readonly", aws.ToString(client.input.RoleName))
	}
	if string(got) != `{"Version":"2012-10-17"}` {
		t.Fatalf("policy = %q", string(got))
	}
}

func TestRoleNameFromARN(t *testing.T) {
	tests := []struct {
		name    string
		roleARN string
		want    string
		wantErr bool
	}{
		{
			name:    "role without path",
			roleARN: "arn:aws:iam::000000000000:role/readonly",
			want:    "readonly",
		},
		{
			name:    "role with path",
			roleARN: "arn:aws:iam::000000000000:role/jit/readonly",
			want:    "readonly",
		},
		{
			name:    "invalid",
			roleARN: "arn:aws:iam::000000000000:user/cloud-iam-broker",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := roleNameFromARN(tt.roleARN)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("role name from arn: %v", err)
			}
			if got != tt.want {
				t.Fatalf("role name = %q, want %q", got, tt.want)
			}
		})
	}
}
