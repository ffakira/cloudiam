package yamlconfig

import (
	"os"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "valid config loads",
			input: validYAML,
		},
		{
			name: "valid approval required target loads",
			input: strings.Replace(
				validYAML,
				"      max_duration: 1h\n",
				"      max_duration: 1h\n      approval_required: true\n",
				1,
			),
		},
		{
			name: "non boolean approval required fails",
			input: strings.Replace(
				validYAML,
				"      max_duration: 1h\n",
				"      max_duration: 1h\n      approval_required: sometimes\n",
				1,
			),
			wantErr: true,
		},
		{
			name: "unknown field fails",
			input: strings.Replace(
				validYAML,
				"version: v1\n",
				"version: v1\nunknown: value\n",
				1,
			),
			wantErr: true,
		},
		{
			name: "secret-looking field fails",
			input: strings.Replace(
				validYAML,
				"version: v1\n",
				"version: v1\naws_secret_access_key: nope\n",
				1,
			),
			wantErr: true,
		},
		{
			name: "unknown target reference fails",
			input: strings.Replace(
				validYAML,
				"        - aws-sandbox-readonly\n",
				"        - missing\n",
				1,
			),
			wantErr: true,
		},
		{
			name: "invalid duration fails",
			input: strings.Replace(
				validYAML,
				"default_duration: 15m",
				"default_duration: soon",
				1,
			),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(strings.NewReader(tt.input))
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLoadExpandsEnvironmentVariables(t *testing.T) {
	t.Setenv("CLOUDIAM_TEST_AWS_ACCOUNT_ID", "123456789012")

	input := strings.ReplaceAll(validYAML, "000000000000", "${CLOUDIAM_TEST_AWS_ACCOUNT_ID}")
	cfg, err := Load(strings.NewReader(input))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	account := cfg.Providers.AWS.Accounts["sandbox"]
	if account.ID != "123456789012" {
		t.Fatalf("account id = %q, want 123456789012", account.ID)
	}
	if account.Roles["readonly"].ARN != "arn:aws:iam::123456789012:role/jit/readonly" {
		t.Fatalf("role arn = %q", account.Roles["readonly"].ARN)
	}
}

func TestLoadDefaultsApprovalRequiredToFalse(t *testing.T) {
	cfg, err := Load(strings.NewReader(validYAML))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Access.Targets["aws-sandbox-readonly"].ApprovalRequired {
		t.Fatal("approval_required defaulted to true, want false")
	}
}

func TestLoadFailsWhenEnvironmentVariableIsMissing(t *testing.T) {
	if err := os.Unsetenv("CLOUDIAM_MISSING_AWS_ACCOUNT_ID"); err != nil {
		t.Fatalf("unset env: %v", err)
	}

	input := strings.ReplaceAll(validYAML, "000000000000", "${CLOUDIAM_MISSING_AWS_ACCOUNT_ID}")
	_, err := Load(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error")
	}
}

const validYAML = `
version: v1
tenant: "<COMPANY_UUID>"

serve:
  grpc:
    addr: 127.0.0.1:50051

providers:
  aws:
    region: us-east-1
    accounts:
      sandbox:
        id: "000000000000"
        roles:
          readonly:
            arn: "arn:aws:iam::000000000000:role/jit/readonly"

principals:
  users:
    akira:
      groups:
        - developers

access:
  targets:
    aws-sandbox-readonly:
      provider: aws
      account: sandbox
      role: readonly
      default_duration: 15m
      max_duration: 1h

  policies:
    - name: developers-sandbox-readonly
      groups:
        - developers
      targets:
        - aws-sandbox-readonly
`
