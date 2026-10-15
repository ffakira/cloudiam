package compliance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

var cloudIAMTagKeys = []string{
	"cloudiam_principal",
	"cloudiam_target",
	"cloudiam_provider",
	"cloudiam_account",
	"cloudiam_role",
}

type trustPolicy struct {
	Statement statements `json:"Statement"`
}

type statements []statement

func (s *statements) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		*s = nil
		return nil
	}
	if data[0] == '[' {
		var out []statement
		if err := json.Unmarshal(data, &out); err != nil {
			return err
		}
		*s = out
		return nil
	}

	var out statement
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*s = []statement{out}
	return nil
}

type statement struct {
	Sid       string              `json:"Sid"`
	Effect    string              `json:"Effect"`
	Principal principal           `json:"Principal"`
	Action    stringList          `json:"Action"`
	Condition map[string]keyValue `json:"Condition"`
}

type principal struct {
	AWS stringList `json:"AWS"`
}

func (p *principal) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte(`"*"`)) {
		p.AWS = stringList{"*"}
		return nil
	}

	var raw struct {
		AWS stringList `json:"AWS"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.AWS = raw.AWS
	return nil
}

type stringList []string

func (s *stringList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*s = nil
		return nil
	}
	if data[0] == '[' {
		var out []string
		if err := json.Unmarshal(data, &out); err != nil {
			return err
		}
		*s = out
		return nil
	}

	var out string
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*s = []string{out}
	return nil
}

type keyValue map[string]stringList

func ScanTrustPolicy(input TargetInput) (Result, error) {
	if strings.TrimSpace(input.TargetName) == "" {
		return Result{}, fmt.Errorf("target name is required")
	}
	if strings.TrimSpace(input.RoleARN) == "" {
		return Result{}, fmt.Errorf("role arn is required")
	}
	if strings.TrimSpace(input.BrokerPrincipalARN) == "" {
		return Result{}, fmt.Errorf("broker principal arn is required")
	}
	if len(bytes.TrimSpace(input.TrustPolicyJSON)) == 0 {
		return Result{}, fmt.Errorf("trust policy json is required")
	}

	var policy trustPolicy
	if err := json.Unmarshal(input.TrustPolicyJSON, &policy); err != nil {
		return Result{}, fmt.Errorf("decode trust policy: %w", err)
	}

	scanner := policyScanner{
		brokerPrincipalARN: input.BrokerPrincipalARN,
	}
	findings := scanner.scan(policy)
	return Result{
		TargetName: input.TargetName,
		RoleARN:    input.RoleARN,
		Findings:   findings,
	}, nil
}

type policyScanner struct {
	brokerPrincipalARN string
}

func (s policyScanner) scan(policy trustPolicy) []Finding {
	var findings []Finding

	assumeOK := false
	setSourceOK := false
	sourceConditionOK := false
	tagSessionOK := false
	tagKeysOK := false
	providerTagOK := false
	requiredRequestTagsOK := false
	transitiveTagsOK := false
	combinedSourceAndTag := false

	for _, stmt := range policy.Statement {
		if !stmt.allowsCloudIAMPrincipal(s.brokerPrincipalARN) {
			continue
		}

		hasAssume := stmt.hasAction("sts:AssumeRole")
		hasSource := stmt.hasAction("sts:SetSourceIdentity")
		hasTagSession := stmt.hasAction("sts:TagSession")

		if hasAssume {
			assumeOK = true
		}
		if hasSource {
			setSourceOK = true
		}
		if hasSource && stmt.hasSourceIdentityCondition() {
			sourceConditionOK = true
		}
		if hasTagSession {
			tagSessionOK = true
		}
		if hasTagSession && stmt.hasExactTagKeys() {
			tagKeysOK = true
		}
		if hasTagSession && stmt.hasProviderTag() {
			providerTagOK = true
		}
		if hasTagSession && stmt.hasRequiredRequestTags() {
			requiredRequestTagsOK = true
		}
		if hasTagSession && stmt.rejectsTransitiveTags() {
			transitiveTagsOK = true
		}
		if hasSource && hasTagSession && stmt.hasSourceIdentityCondition() && stmt.hasAnyRequestTagCondition() {
			combinedSourceAndTag = true
		}
	}

	if !assumeOK {
		findings = append(findings, finding(FindingMissingAssumeRole, "trust policy does not allow sts:AssumeRole for the CloudIAM broker principal"))
	}
	if !setSourceOK {
		findings = append(findings, finding(FindingMissingSetSourceIdentity, "trust policy does not allow sts:SetSourceIdentity for the CloudIAM broker principal"))
	}
	if !sourceConditionOK {
		findings = append(findings, finding(FindingSourceIdentityCondition, "trust policy must require sts:SourceIdentity to match cloudiam-*"))
	}
	if !tagSessionOK {
		findings = append(findings, finding(FindingMissingTagSession, "trust policy does not allow sts:TagSession for the CloudIAM broker principal"))
	}
	if !tagKeysOK {
		findings = append(findings, finding(FindingTagKeys, "trust policy must restrict aws:TagKeys to the CloudIAM tag set"))
	}
	if !providerTagOK {
		findings = append(findings, finding(FindingProviderTag, "trust policy must require aws:RequestTag/cloudiam_provider to equal aws"))
	}
	if !requiredRequestTagsOK {
		findings = append(findings, finding(FindingRequiredRequestTag, "trust policy must require all CloudIAM request tag keys"))
	}
	if !transitiveTagsOK {
		findings = append(findings, finding(FindingTransitiveTags, "trust policy must reject transitive session tags"))
	}
	if combinedSourceAndTag {
		findings = append(findings, finding(FindingCombinedSourceAndTagCondition, "sts:SetSourceIdentity and sts:TagSession conditions must be in separate trust-policy statements"))
	}

	return findings
}

func finding(id FindingID, message string) Finding {
	return Finding{ID: id, Message: message}
}

func (s statement) allowsCloudIAMPrincipal(brokerPrincipalARN string) bool {
	if !strings.EqualFold(s.Effect, "Allow") {
		return false
	}
	return slices.Contains(s.Principal.AWS, brokerPrincipalARN)
}

func (s statement) hasAction(action string) bool {
	for _, candidate := range s.Action {
		if strings.EqualFold(candidate, action) {
			return true
		}
	}
	return false
}

func (s statement) hasSourceIdentityCondition() bool {
	return conditionContains(s.Condition, "StringLike", "sts:SourceIdentity", "cloudiam-*")
}

func (s statement) hasAnyRequestTagCondition() bool {
	for _, values := range s.Condition {
		for key := range values {
			if strings.HasPrefix(key, "aws:RequestTag/") {
				return true
			}
		}
	}
	return false
}

func (s statement) hasExactTagKeys() bool {
	values := s.Condition["ForAllValues:StringEquals"]["aws:TagKeys"]
	return sameStringSet(values, cloudIAMTagKeys)
}

func (s statement) hasProviderTag() bool {
	return conditionContains(s.Condition, "StringEquals", "aws:RequestTag/cloudiam_provider", "aws")
}

func (s statement) hasRequiredRequestTags() bool {
	for _, key := range cloudIAMTagKeys {
		conditionKey := "aws:RequestTag/" + key
		if key == "cloudiam_provider" {
			if !s.hasProviderTag() {
				return false
			}
			continue
		}
		if !conditionContains(s.Condition, "StringLike", conditionKey, "*") {
			return false
		}
	}
	return true
}

func (s statement) rejectsTransitiveTags() bool {
	return conditionContains(s.Condition, "Null", "sts:TransitiveTagKeys", "true")
}

func conditionContains(conditions map[string]keyValue, operator, key, value string) bool {
	values, ok := conditions[operator][key]
	if !ok {
		return false
	}
	return slices.Contains(values, value)
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for _, value := range want {
		if !slices.Contains(got, value) {
			return false
		}
	}
	return true
}
