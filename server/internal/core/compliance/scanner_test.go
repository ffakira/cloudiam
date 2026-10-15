package compliance

import (
	"slices"
	"strings"
	"testing"
)

const (
	testTarget = "aws-sandbox-readonly"
	testRole   = "arn:aws:iam::000000000000:role/jit/readonly"
	testBroker = "arn:aws:iam::000000000000:user/cloud-iam-broker"
)

func TestScanTrustPolicyCompliant(t *testing.T) {
	result, err := ScanTrustPolicy(inputWithPolicy(compliantPolicy))
	if err != nil {
		t.Fatalf("scan trust policy: %v", err)
	}
	if !result.Compliant() {
		t.Fatalf("findings = %+v, want compliant", result.Findings)
	}
	if result.TargetName != testTarget {
		t.Fatalf("target = %q, want %q", result.TargetName, testTarget)
	}
	if result.RoleARN != testRole {
		t.Fatalf("role arn = %q, want %q", result.RoleARN, testRole)
	}
}

func TestScanTrustPolicyFindings(t *testing.T) {
	tests := []struct {
		name       string
		policyJSON string
		want       []FindingID
	}{
		{
			name:       "missing set source identity",
			policyJSON: strings.Replace(compliantPolicy, `"sts:SetSourceIdentity"`, `"sts:GetCallerIdentity"`, 1),
			want: []FindingID{
				FindingMissingSetSourceIdentity,
				FindingSourceIdentityCondition,
			},
		},
		{
			name:       "missing tag session",
			policyJSON: strings.Replace(compliantPolicy, `"sts:TagSession"`, `"sts:GetCallerIdentity"`, 1),
			want: []FindingID{
				FindingMissingTagSession,
				FindingTagKeys,
				FindingProviderTag,
				FindingRequiredRequestTag,
				FindingTransitiveTags,
			},
		},
		{
			name:       "wrong source identity condition",
			policyJSON: strings.Replace(compliantPolicy, `"cloudiam-*"`, `"*"`, 1),
			want:       []FindingID{FindingSourceIdentityCondition},
		},
		{
			name:       "missing tag key",
			policyJSON: strings.Replace(compliantPolicy, `"cloudiam_role"`, `"unexpected_tag"`, 1),
			want:       []FindingID{FindingTagKeys},
		},
		{
			name:       "missing request tag condition",
			policyJSON: strings.Replace(compliantPolicy, `"aws:RequestTag/cloudiam_role"`, `"aws:RequestTag/unexpected_tag"`, 1),
			want:       []FindingID{FindingRequiredRequestTag},
		},
		{
			name:       "wrong provider tag",
			policyJSON: strings.Replace(compliantPolicy, `"aws:RequestTag/cloudiam_provider": "aws"`, `"aws:RequestTag/cloudiam_provider": "gcp"`, 1),
			want: []FindingID{
				FindingProviderTag,
				FindingRequiredRequestTag,
			},
		},
		{
			name:       "transitive tags not rejected",
			policyJSON: strings.Replace(compliantPolicy, `"sts:TransitiveTagKeys": "true"`, `"sts:TransitiveTagKeys": "false"`, 1),
			want:       []FindingID{FindingTransitiveTags},
		},
		{
			name:       "combined source and tag conditions",
			policyJSON: combinedPolicy,
			want:       []FindingID{FindingCombinedSourceAndTagCondition},
		},
		{
			name:       "wrong broker principal",
			policyJSON: strings.ReplaceAll(compliantPolicy, testBroker, "arn:aws:iam::000000000000:user/other"),
			want: []FindingID{
				FindingMissingAssumeRole,
				FindingMissingSetSourceIdentity,
				FindingSourceIdentityCondition,
				FindingMissingTagSession,
				FindingTagKeys,
				FindingProviderTag,
				FindingRequiredRequestTag,
				FindingTransitiveTags,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ScanTrustPolicy(inputWithPolicy(tt.policyJSON))
			if err != nil {
				t.Fatalf("scan trust policy: %v", err)
			}
			got := findingIDs(result.Findings)
			for _, id := range tt.want {
				if !slices.Contains(got, id) {
					t.Fatalf("findings = %v, want %s", got, id)
				}
			}
			if len(got) != len(tt.want) {
				t.Fatalf("findings = %v, want exactly %v", got, tt.want)
			}
		})
	}
}

func TestScanTrustPolicyValidatesInput(t *testing.T) {
	tests := []struct {
		name  string
		input TargetInput
	}{
		{
			name:  "target name required",
			input: TargetInput{RoleARN: testRole, BrokerPrincipalARN: testBroker, TrustPolicyJSON: []byte(compliantPolicy)},
		},
		{
			name:  "role arn required",
			input: TargetInput{TargetName: testTarget, BrokerPrincipalARN: testBroker, TrustPolicyJSON: []byte(compliantPolicy)},
		},
		{
			name:  "broker principal required",
			input: TargetInput{TargetName: testTarget, RoleARN: testRole, TrustPolicyJSON: []byte(compliantPolicy)},
		},
		{
			name:  "policy required",
			input: TargetInput{TargetName: testTarget, RoleARN: testRole, BrokerPrincipalARN: testBroker},
		},
		{
			name:  "policy must be json",
			input: TargetInput{TargetName: testTarget, RoleARN: testRole, BrokerPrincipalARN: testBroker, TrustPolicyJSON: []byte(`{`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ScanTrustPolicy(tt.input)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func inputWithPolicy(policyJSON string) TargetInput {
	return TargetInput{
		TargetName:         testTarget,
		RoleARN:            testRole,
		BrokerPrincipalARN: testBroker,
		TrustPolicyJSON:    []byte(policyJSON),
	}
}

func findingIDs(findings []Finding) []FindingID {
	ids := make([]FindingID, 0, len(findings))
	for _, finding := range findings {
		ids = append(ids, finding.ID)
	}
	return ids
}

const compliantPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowCloudIAMAssumeRoleWithSourceIdentity",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::000000000000:user/cloud-iam-broker"
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
      "Sid": "AllowCloudIAMSessionTags",
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::000000000000:user/cloud-iam-broker"
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
}`

const combinedPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::000000000000:user/cloud-iam-broker"
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
}`
