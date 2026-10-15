package compliance

type FindingID string

const (
	FindingMissingAssumeRole             FindingID = "AWS_TRUST_MISSING_ASSUME_ROLE"
	FindingMissingSetSourceIdentity      FindingID = "AWS_TRUST_MISSING_SET_SOURCE_IDENTITY"
	FindingSourceIdentityCondition       FindingID = "AWS_TRUST_SOURCE_IDENTITY_CONDITION"
	FindingMissingTagSession             FindingID = "AWS_TRUST_MISSING_TAG_SESSION"
	FindingTagKeys                       FindingID = "AWS_TRUST_TAG_KEYS"
	FindingProviderTag                   FindingID = "AWS_TRUST_PROVIDER_TAG"
	FindingRequiredRequestTag            FindingID = "AWS_TRUST_REQUIRED_REQUEST_TAG"
	FindingTransitiveTags                FindingID = "AWS_TRUST_TRANSITIVE_TAGS"
	FindingCombinedSourceAndTagCondition FindingID = "AWS_TRUST_COMBINED_SOURCE_AND_TAG_CONDITIONS"
)

type TargetInput struct {
	TargetName         string
	RoleARN            string
	BrokerPrincipalARN string
	TrustPolicyJSON    []byte
}

type Result struct {
	TargetName string
	RoleARN    string
	Findings   []Finding
}

func (r Result) Compliant() bool {
	return len(r.Findings) == 0
}

type Finding struct {
	ID      FindingID
	Message string
}
