package access

import "time"

type Request struct {
	Principal string
	Target    string
	Duration  time.Duration
}

type ResultStatus string

const (
	ResultStatusIssued          ResultStatus = "issued"
	ResultStatusPendingApproval ResultStatus = "pending_approval"
)

type Result struct {
	Status        ResultStatus
	Grant         Grant
	AccessRequest AccessRequest
}

type IssueRequest struct {
	Principal string
	Target    string
	Provider  string
	Account   string
	Role      string
	RoleARN   string
	Duration  time.Duration
}

type Grant struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	ExpiresAt       time.Time
}

type AccessRequestStatus string

const (
	AccessRequestStatusPendingApproval AccessRequestStatus = "pending_approval"
	AccessRequestStatusApproved        AccessRequestStatus = "approved"
	AccessRequestStatusDenied          AccessRequestStatus = "denied"
	AccessRequestStatusExpired         AccessRequestStatus = "expired"
	AccessRequestStatusIssued          AccessRequestStatus = "issued"
)

type AccessRequest struct {
	ID                int64
	Principal         string
	Target            string
	Provider          string
	Account           string
	Role              string
	RoleARN           string
	RequestedDuration time.Duration
	PolicyReason      string
	Status            AccessRequestStatus
	ExpiresAt         time.Time
	ApprovedBy        string
	DeniedBy          string
	ApprovedAt        *time.Time
	DeniedAt          *time.Time
	IssuedAt          *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
