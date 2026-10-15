package audit

import "time"

type EventType string

const (
	EventTypeAccessRequested   EventType = "access_requested"
	EventTypeAccessDenied      EventType = "access_denied"
	EventTypeAccessIssued      EventType = "access_issued"
	EventTypeAccessFailed      EventType = "access_failed"
	EventTypeApprovalRequested EventType = "approval_requested"
	EventTypeApprovalApproved  EventType = "approval_approved"
	EventTypeApprovalDenied    EventType = "approval_denied"
	EventTypeApprovalExpired   EventType = "approval_expired"
)

type Decision string

const (
	DecisionAllowed Decision = "allowed"
	DecisionDenied  Decision = "denied"
	DecisionFailed  Decision = "failed"
	DecisionPending Decision = "pending"
)

type Event struct {
	ID                  int64
	OccurredAt          time.Time
	EventType           EventType
	Principal           string
	Target              string
	Decision            Decision
	Reason              string
	Provider            string
	Account             string
	Role                string
	AccessRequestID     *int64
	OperatorSubject     string
	IdempotencyKeyHash  string
	CredentialExpiresAt *time.Time
	CreatedAt           time.Time
}

type RecordEventInput struct {
	OccurredAt          time.Time
	EventType           EventType
	Principal           string
	Target              string
	Decision            Decision
	Reason              string
	Provider            string
	Account             string
	Role                string
	AccessRequestID     *int64
	OperatorSubject     string
	IdempotencyKeyHash  string
	CredentialExpiresAt *time.Time
}
