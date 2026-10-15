package serverapp

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeTrustPolicyReader struct {
	policies map[string]string
	requests []string
}

func (f *fakeTrustPolicyReader) ReadTrustPolicy(ctx context.Context, roleARN string) ([]byte, error) {
	f.requests = append(f.requests, roleARN)
	return []byte(f.policies[roleARN]), nil
}

func TestScanAWSComplianceReportsCompliantTarget(t *testing.T) {
	path := writeConfigFixture(t, strings.ReplaceAll(validConfigYAML, "000000000000", "123456789012"))
	reader := &fakeTrustPolicyReader{
		policies: map[string]string{
			"arn:aws:iam::123456789012:role/jit/readonly": compliantTrustPolicy("123456789012"),
		},
	}
	var out bytes.Buffer

	err := ScanAWSCompliance(context.Background(), reader, ScanAWSComplianceInput{
		ConfigPath:     path,
		BrokerUserName: "cloud-iam-broker",
	}, &out)
	if err != nil {
		t.Fatalf("scan aws compliance: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "target: aws-sandbox-readonly") {
		t.Fatalf("output missing target: %q", output)
	}
	if !strings.Contains(output, "role: arn:aws:iam::<AWS_ACCOUNT_ID>:role/jit/readonly") {
		t.Fatalf("output did not mask account id: %q", output)
	}
	if !strings.Contains(output, "status: compliant") {
		t.Fatalf("output missing compliant status: %q", output)
	}
	if strings.Contains(output, "123456789012") {
		t.Fatalf("output leaked account id: %q", output)
	}
	if len(reader.requests) != 1 || reader.requests[0] != "arn:aws:iam::123456789012:role/jit/readonly" {
		t.Fatalf("requests = %v", reader.requests)
	}
}

func TestScanAWSComplianceReturnsFailureForFindings(t *testing.T) {
	path := writeConfigFixture(t, strings.ReplaceAll(validConfigYAML, "000000000000", "123456789012"))
	reader := &fakeTrustPolicyReader{
		policies: map[string]string{
			"arn:aws:iam::123456789012:role/jit/readonly": combinedTrustPolicy("123456789012"),
		},
	}
	var out bytes.Buffer

	err := ScanAWSCompliance(context.Background(), reader, ScanAWSComplianceInput{
		ConfigPath: path,
	}, &out)
	if !errors.Is(err, ErrComplianceFailed) {
		t.Fatalf("error = %v, want ErrComplianceFailed", err)
	}

	output := out.String()
	if !strings.Contains(output, "status: non-compliant") {
		t.Fatalf("output missing non-compliant status: %q", output)
	}
	if !strings.Contains(output, "AWS_TRUST_COMBINED_SOURCE_AND_TAG_CONDITIONS") {
		t.Fatalf("output missing combined-condition finding: %q", output)
	}
	if strings.Contains(output, "ASIA") || strings.Contains(output, "session-token") || strings.Contains(output, "secret-access-key") {
		t.Fatalf("output leaked credential-looking value: %q", output)
	}
}

func TestScanAWSComplianceRequiresConfigPath(t *testing.T) {
	err := ScanAWSCompliance(context.Background(), &fakeTrustPolicyReader{}, ScanAWSComplianceInput{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "config path is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func compliantTrustPolicy(accountID string) string {
	return strings.ReplaceAll(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<AWS_ACCOUNT_ID>:user/cloud-iam-broker"
      },
      "Action": [
        "sts:AssumeRole",
        "sts:SetSourceIdentity"
      ],
      "Condition": {
        "StringLike": {
          "sts:SourceIdentity": "cloudiam-*"
        }
      }
    },
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<AWS_ACCOUNT_ID>:user/cloud-iam-broker"
      },
      "Action": [
        "sts:TagSession"
      ],
      "Condition": {
        "StringLike": {
          "aws:RequestTag/cloudiam_principal": "*",
          "aws:RequestTag/cloudiam_target": "*",
          "aws:RequestTag/cloudiam_account": "*",
          "aws:RequestTag/cloudiam_role": "*"
        },
        "StringEquals": {
          "aws:RequestTag/cloudiam_provider": "aws"
        },
        "ForAllValues:StringEquals": {
          "aws:TagKeys": [
            "cloudiam_principal",
            "cloudiam_target",
            "cloudiam_provider",
            "cloudiam_account",
            "cloudiam_role"
          ]
        },
        "Null": {
          "sts:TransitiveTagKeys": "true"
        }
      }
    }
  ]
}`, "<AWS_ACCOUNT_ID>", accountID)
}

func combinedTrustPolicy(accountID string) string {
	return strings.ReplaceAll(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<AWS_ACCOUNT_ID>:user/cloud-iam-broker"
      },
      "Action": [
        "sts:AssumeRole",
        "sts:SetSourceIdentity",
        "sts:TagSession"
      ],
      "Condition": {
        "StringLike": {
          "sts:SourceIdentity": "cloudiam-*",
          "aws:RequestTag/cloudiam_principal": "*",
          "aws:RequestTag/cloudiam_target": "*",
          "aws:RequestTag/cloudiam_account": "*",
          "aws:RequestTag/cloudiam_role": "*"
        },
        "StringEquals": {
          "aws:RequestTag/cloudiam_provider": "aws"
        },
        "ForAllValues:StringEquals": {
          "aws:TagKeys": [
            "cloudiam_principal",
            "cloudiam_target",
            "cloudiam_provider",
            "cloudiam_account",
            "cloudiam_role"
          ]
        },
        "Null": {
          "sts:TransitiveTagKeys": "true"
        }
      }
    }
  ]
}`, "<AWS_ACCOUNT_ID>", accountID)
}
