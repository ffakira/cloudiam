package approval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/access"
)

type Action string

const (
	ActionApprove Action = "approve"
	ActionDeny    Action = "deny"
)

type MutationInput struct {
	AccessRequestID int64
	ApproverSubject string
	IdempotencyKey  string
	Fingerprint     string
	Action          Action
	Now             time.Time
}

type MutationResult struct {
	AccessRequest access.AccessRequest
	Replayed      bool
}

type Store interface {
	ListAccessRequests(ctx context.Context, status access.AccessRequestStatus) ([]access.AccessRequest, error)
	MutateAccessRequest(ctx context.Context, input MutationInput) (MutationResult, error)
}

func Fingerprint(action Action, accessRequestID int64, approverSubject, idempotencyKey string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\n%d\n%s\n%s", action, accessRequestID, approverSubject, idempotencyKey)))
	return hex.EncodeToString(sum[:])
}
