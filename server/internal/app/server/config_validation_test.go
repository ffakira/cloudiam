package serverapp

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	t.Setenv("CLOUDIAM_TEST_AWS_ACCOUNT_ID", "123456789012")
	path := writeConfigFixture(t, strings.ReplaceAll(validConfigYAML, "000000000000", "${CLOUDIAM_TEST_AWS_ACCOUNT_ID}"))

	var out bytes.Buffer
	if err := ValidateConfig(path, &out); err != nil {
		t.Fatalf("validate config: %v", err)
	}
	if !strings.Contains(out.String(), "is valid") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestValidateConfigMissingEnvExpansionFailsClearly(t *testing.T) {
	path := writeConfigFixture(t, strings.ReplaceAll(validConfigYAML, "000000000000", "${CLOUDIAM_MISSING_AWS_ACCOUNT_ID}"))

	var out bytes.Buffer
	err := ValidateConfig(path, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "providers.aws.accounts.sandbox.id is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestValidateConfigInvalidYAMLFailsClearly(t *testing.T) {
	path := writeConfigFixture(t, "version: [")

	var out bytes.Buffer
	err := ValidateConfig(path, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse yaml") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestValidateConfigSecretFieldFailsClearly(t *testing.T) {
	path := writeConfigFixture(t, "aws_secret_access_key: nope\n"+validConfigYAML)

	var out bytes.Buffer
	err := ValidateConfig(path, &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "forbidden secret field") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestValidateConfigRequiresPath(t *testing.T) {
	var out bytes.Buffer
	err := ValidateConfig("", &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "config path is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func writeConfigFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "cloudiam.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return path
}

const validConfigYAML = `
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
