//go:build integration

package postgresadapter

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/approval"
)

func TestAccessRequestStoreIntegration_CreateAccessRequest(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "access-request-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	if request.ID == 0 {
		t.Fatal("expected access request id")
	}
	if request.Status != access.AccessRequestStatusPendingApproval {
		t.Fatalf("status = %q, want pending approval", request.Status)
	}
	if request.Principal != principal || request.Target != "aws-sandbox-readonly" {
		t.Fatalf("principal/target = %q/%q", request.Principal, request.Target)
	}
	if request.Provider != "aws" || request.Account != "sandbox" || request.Role != "readonly" {
		t.Fatalf("provider/account/role = %q/%q/%q", request.Provider, request.Account, request.Role)
	}
	if request.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("role arn = %q", request.RoleARN)
	}
	if request.RequestedDuration != 15*time.Minute {
		t.Fatalf("duration = %s, want 15m", request.RequestedDuration)
	}
	combined := strings.Join([]string{
		request.Principal,
		request.Target,
		request.Provider,
		request.Account,
		request.Role,
		request.RoleARN,
		request.PolicyReason,
	}, " ")
	for _, secret := range []string{"AWS_ACCESS_KEY_ID", "secret-access-key", "session-token"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("access request leaked credential material %q: %+v", secret, request)
		}
	}
}

func TestAccessRequestStoreIntegration_ClaimApprovedAccessRequest(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "approved-access-request-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE access_requests
		SET status = 'approved',
			approved_at = $2,
			updated_at = $2
		WHERE id = $1
	`, request.ID, now.Add(time.Second)); err != nil {
		t.Fatalf("approve access request fixture: %v", err)
	}

	claimed, ok, err := store.ClaimApprovedAccessRequest(ctx, access.ClaimApprovedAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		RequestedDuration: 15 * time.Minute,
		IssuedAt:          now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("claim approved access request: %v", err)
	}
	if !ok {
		t.Fatal("expected approved access request")
	}
	if claimed.ID != request.ID {
		t.Fatalf("claimed id = %d, want %d", claimed.ID, request.ID)
	}
	if claimed.Status != access.AccessRequestStatusIssued {
		t.Fatalf("status = %q, want issued", claimed.Status)
	}
	if claimed.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("claimed role arn = %q", claimed.RoleARN)
	}
	if claimed.IssuedAt == nil {
		t.Fatal("expected issued_at")
	}

	_, ok, err = store.ClaimApprovedAccessRequest(ctx, access.ClaimApprovedAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		RequestedDuration: 15 * time.Minute,
		IssuedAt:          now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("claim issued access request again: %v", err)
	}
	if ok {
		t.Fatal("issued access request should not be claimable twice")
	}
}

func TestAccessRequestStoreIntegration_ConcurrentClaimApprovedAccessRequestIsSingleUse(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "concurrent-claim-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE access_requests
		SET status = 'approved',
			approved_at = $2,
			updated_at = $2
		WHERE id = $1
	`, request.ID, now.Add(time.Second)); err != nil {
		t.Fatalf("approve access request fixture: %v", err)
	}

	const attempts = 8
	start := make(chan struct{})
	results := make(chan bool, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, ok, err := store.ClaimApprovedAccessRequest(context.Background(), access.ClaimApprovedAccessRequestInput{
				Principal:         principal,
				Target:            "aws-sandbox-readonly",
				RequestedDuration: 15 * time.Minute,
				IssuedAt:          now.Add(time.Duration(i+2) * time.Second),
			})
			results <- ok
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("claim approved access request: %v", err)
		}
	}
	claimed := 0
	for ok := range results {
		if ok {
			claimed++
		}
	}
	if claimed != 1 {
		t.Fatalf("claimed requests = %d, want 1", claimed)
	}
	assertAccessRequestStatus(ctx, t, db, request.ID, access.AccessRequestStatusIssued)
}

func TestAccessRequestStoreIntegration_DoesNotClaimExpiredApprovedAccessRequest(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "expired-access-request-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE access_requests
		SET status = 'approved',
			approved_at = $2,
			updated_at = $2
		WHERE id = $1
	`, request.ID, now.Add(10*time.Second)); err != nil {
		t.Fatalf("approve access request fixture: %v", err)
	}

	_, ok, err := store.ClaimApprovedAccessRequest(ctx, access.ClaimApprovedAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		RequestedDuration: 15 * time.Minute,
		IssuedAt:          now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("claim expired approved access request: %v", err)
	}
	if ok {
		t.Fatal("expired approved access request should not be claimable")
	}
}

func TestAccessRequestStoreIntegration_MutateAccessRequestIsIdempotent(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "admin-approval-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	key := "a8098c1a-f86e-4c89-947d-0d3a2b316e11"
	fingerprint := approval.Fingerprint(approval.ActionApprove, request.ID, "admin", key)
	result, err := store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: request.ID,
		ApproverSubject: "admin",
		IdempotencyKey:  key,
		Fingerprint:     fingerprint,
		Action:          approval.ActionApprove,
		Now:             now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("approve access request: %v", err)
	}
	if result.Replayed {
		t.Fatal("first approval should not be replayed")
	}
	if result.AccessRequest.Status != access.AccessRequestStatusApproved {
		t.Fatalf("status = %q, want approved", result.AccessRequest.Status)
	}
	if result.AccessRequest.ApprovedBy != "admin" {
		t.Fatalf("approved by = %q, want admin", result.AccessRequest.ApprovedBy)
	}

	replayed, err := store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: request.ID,
		ApproverSubject: "admin",
		IdempotencyKey:  key,
		Fingerprint:     fingerprint,
		Action:          approval.ActionApprove,
		Now:             now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("replay approval: %v", err)
	}
	if !replayed.Replayed {
		t.Fatal("expected idempotent replay")
	}
	if replayed.AccessRequest.ID != request.ID {
		t.Fatalf("replayed id = %d, want %d", replayed.AccessRequest.ID, request.ID)
	}

	_, err = store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: request.ID,
		ApproverSubject: "admin",
		IdempotencyKey:  key,
		Fingerprint:     approval.Fingerprint(approval.ActionDeny, request.ID, "admin", key),
		Action:          approval.ActionDeny,
		Now:             now.Add(3 * time.Second),
	})
	if !errors.Is(err, approval.ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want idempotency conflict", err)
	}
}

func TestAccessRequestStoreIntegration_ConcurrentMutateAccessRequestIsIdempotent(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "concurrent-idempotency-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	const attempts = 8
	key := "00000000-0000-4000-8000-000000000005"
	fingerprint := approval.Fingerprint(approval.ActionApprove, request.ID, "admin", key)
	start := make(chan struct{})
	results := make(chan approval.MutationResult, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			result, err := store.MutateAccessRequest(context.Background(), approval.MutationInput{
				AccessRequestID: request.ID,
				ApproverSubject: "admin",
				IdempotencyKey:  key,
				Fingerprint:     fingerprint,
				Action:          approval.ActionApprove,
				Now:             now.Add(time.Duration(i+1) * time.Second),
			})
			results <- result
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("mutate access request: %v", err)
		}
	}
	firstMutations := 0
	replays := 0
	for result := range results {
		if result.AccessRequest.ID != request.ID {
			t.Fatalf("result access request id = %d, want %d", result.AccessRequest.ID, request.ID)
		}
		if result.Replayed {
			replays++
		} else {
			firstMutations++
		}
	}
	if firstMutations != 1 {
		t.Fatalf("first mutations = %d, want 1", firstMutations)
	}
	if replays != attempts-1 {
		t.Fatalf("replays = %d, want %d", replays, attempts-1)
	}
	assertAccessRequestStatus(ctx, t, db, request.ID, access.AccessRequestStatusApproved)
}

func TestAccessRequestStoreIntegration_ConcurrentMutateAccessRequestConflictsOnMismatchedFingerprint(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "concurrent-conflict-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	key := "00000000-0000-4000-8000-000000000006"
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, mutation := range []approval.MutationInput{
		{
			AccessRequestID: request.ID,
			ApproverSubject: "admin",
			IdempotencyKey:  key,
			Fingerprint:     approval.Fingerprint(approval.ActionApprove, request.ID, "admin", key),
			Action:          approval.ActionApprove,
			Now:             now.Add(time.Second),
		},
		{
			AccessRequestID: request.ID,
			ApproverSubject: "admin",
			IdempotencyKey:  key,
			Fingerprint:     approval.Fingerprint(approval.ActionDeny, request.ID, "admin", key),
			Action:          approval.ActionDeny,
			Now:             now.Add(2 * time.Second),
		},
	} {
		mutation := mutation
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.MutateAccessRequest(context.Background(), mutation)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	conflicts := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, approval.ErrIdempotencyConflict) {
			conflicts++
			continue
		}
		t.Fatalf("unexpected mutation error: %v", err)
	}
	if successes != 1 {
		t.Fatalf("successful mutations = %d, want 1", successes)
	}
	if conflicts != 1 {
		t.Fatalf("conflicts = %d, want 1", conflicts)
	}
}

func TestAccessRequestStoreIntegration_ConcurrentApproveAndDenyAccessRequestSettlesOneState(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "concurrent-approval-denial-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, mutation := range []approval.MutationInput{
		{
			AccessRequestID: request.ID,
			ApproverSubject: "admin",
			IdempotencyKey:  "00000000-0000-4000-8000-000000000007",
			Fingerprint:     approval.Fingerprint(approval.ActionApprove, request.ID, "admin", "00000000-0000-4000-8000-000000000007"),
			Action:          approval.ActionApprove,
			Now:             now.Add(time.Second),
		},
		{
			AccessRequestID: request.ID,
			ApproverSubject: "admin",
			IdempotencyKey:  "00000000-0000-4000-8000-000000000008",
			Fingerprint:     approval.Fingerprint(approval.ActionDeny, request.ID, "admin", "00000000-0000-4000-8000-000000000008"),
			Action:          approval.ActionDeny,
			Now:             now.Add(time.Second),
		},
	} {
		mutation := mutation
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.MutateAccessRequest(context.Background(), mutation)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	invalidState := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, approval.ErrAccessRequestInvalidState) {
			invalidState++
			continue
		}
		t.Fatalf("unexpected mutation error: %v", err)
	}
	if successes != 1 {
		t.Fatalf("successful mutations = %d, want 1", successes)
	}
	if invalidState != 1 {
		t.Fatalf("invalid state mutations = %d, want 1", invalidState)
	}

	var status string
	if err := db.QueryRowContext(ctx, `
		SELECT status
		FROM access_requests
		WHERE id = $1
	`, request.ID).Scan(&status); err != nil {
		t.Fatalf("query access request status: %v", err)
	}
	if access.AccessRequestStatus(status) != access.AccessRequestStatusApproved &&
		access.AccessRequestStatus(status) != access.AccessRequestStatusDenied {
		t.Fatalf("status = %q, want approved or denied", status)
	}
}

func TestAccessRequestStoreIntegration_ConcurrentApproveAndExpireAccessRequestSettlesOneState(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "concurrent-approval-expiry-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, mutation := range []approval.MutationInput{
		{
			AccessRequestID: request.ID,
			ApproverSubject: "admin",
			IdempotencyKey:  "00000000-0000-4000-8000-000000000009",
			Fingerprint:     approval.Fingerprint(approval.ActionApprove, request.ID, "admin", "00000000-0000-4000-8000-000000000009"),
			Action:          approval.ActionApprove,
			Now:             now,
		},
		{
			AccessRequestID: request.ID,
			ApproverSubject: "admin",
			IdempotencyKey:  "00000000-0000-4000-8000-000000000010",
			Fingerprint:     approval.Fingerprint(approval.ActionApprove, request.ID, "admin", "00000000-0000-4000-8000-000000000010"),
			Action:          approval.ActionApprove,
			Now:             now.Add(2 * time.Second),
		},
	} {
		mutation := mutation
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.MutateAccessRequest(context.Background(), mutation)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	expired := 0
	invalidState := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, approval.ErrAccessRequestExpired) {
			expired++
			continue
		}
		if errors.Is(err, approval.ErrAccessRequestInvalidState) {
			invalidState++
			continue
		}
		t.Fatalf("unexpected mutation error: %v", err)
	}
	if successes+expired != 1 {
		t.Fatalf("terminal successful/expired mutations = %d, want 1", successes+expired)
	}
	if invalidState != 1 {
		t.Fatalf("invalid state mutations = %d, want 1", invalidState)
	}

	var status string
	if err := db.QueryRowContext(ctx, `
		SELECT status
		FROM access_requests
		WHERE id = $1
	`, request.ID).Scan(&status); err != nil {
		t.Fatalf("query access request status: %v", err)
	}
	if access.AccessRequestStatus(status) != access.AccessRequestStatusApproved &&
		access.AccessRequestStatus(status) != access.AccessRequestStatusExpired {
		t.Fatalf("status = %q, want approved or expired", status)
	}
}

func TestAccessRequestStoreIntegration_ConcurrentClaimAtExpiryBoundaryDoesNotDoubleIssue(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "concurrent-claim-expiry-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE access_requests
		SET status = 'approved',
			approved_at = $2,
			updated_at = $2
		WHERE id = $1
	`, request.ID, now); err != nil {
		t.Fatalf("approve access request fixture: %v", err)
	}

	start := make(chan struct{})
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, issuedAt := range []time.Time{now, now.Add(2 * time.Second)} {
		issuedAt := issuedAt
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, ok, err := store.ClaimApprovedAccessRequest(context.Background(), access.ClaimApprovedAccessRequestInput{
				Principal:         principal,
				Target:            "aws-sandbox-readonly",
				RequestedDuration: 15 * time.Minute,
				IssuedAt:          issuedAt,
			})
			results <- ok
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("claim approved access request: %v", err)
		}
	}
	claims := 0
	for ok := range results {
		if ok {
			claims++
		}
	}
	if claims != 1 {
		t.Fatalf("claims = %d, want 1", claims)
	}
	assertAccessRequestStatus(ctx, t, db, request.ID, access.AccessRequestStatusIssued)
}

func TestAccessRequestStoreIntegration_MutateAccessRequestDeniesPendingRequest(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "admin-denial-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	key := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	result, err := store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: request.ID,
		ApproverSubject: "admin",
		IdempotencyKey:  key,
		Fingerprint:     approval.Fingerprint(approval.ActionDeny, request.ID, "admin", key),
		Action:          approval.ActionDeny,
		Now:             now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("deny access request: %v", err)
	}
	if result.AccessRequest.Status != access.AccessRequestStatusDenied {
		t.Fatalf("status = %q, want denied", result.AccessRequest.Status)
	}
	if result.AccessRequest.DeniedBy != "admin" {
		t.Fatalf("denied by = %q, want admin", result.AccessRequest.DeniedBy)
	}
	if result.AccessRequest.DeniedAt == nil {
		t.Fatal("expected denied_at")
	}
}

func TestAccessRequestStoreIntegration_MutateAccessRequestReturnsExpiredRequest(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "admin-expired-" + now.Format("20060102150405.000000")

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("create access request: %v", err)
	}

	key := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	result, err := store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: request.ID,
		ApproverSubject: "admin",
		IdempotencyKey:  key,
		Fingerprint:     approval.Fingerprint(approval.ActionApprove, request.ID, "admin", key),
		Action:          approval.ActionApprove,
		Now:             now.Add(2 * time.Second),
	})
	if !errors.Is(err, approval.ErrAccessRequestExpired) {
		t.Fatalf("error = %v, want expired", err)
	}
	if result.AccessRequest.ID != request.ID {
		t.Fatalf("expired id = %d, want %d", result.AccessRequest.ID, request.ID)
	}
	if result.AccessRequest.Status != access.AccessRequestStatusExpired {
		t.Fatalf("status = %q, want expired", result.AccessRequest.Status)
	}
}

func TestAccessRequestStoreIntegration_CleanupExpiredIdempotencyRecords(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "idempotency-cleanup-" + now.Format("20060102150405.000000")
	cutoff := now.Add(24 * time.Hour)

	t.Cleanup(func() {
		cleanupAuditEvents(context.Background(), t, db, principal)
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	expiredBefore := createApprovedIdempotencyFixture(t, ctx, store, principal, "before-admin", "00000000-0000-4000-8000-000000000001", now)
	expiredEqual := createApprovedIdempotencyFixture(t, ctx, store, principal, "equal-admin", "00000000-0000-4000-8000-000000000002", now)
	activeAfter := createApprovedIdempotencyFixture(t, ctx, store, principal, "after-admin", "00000000-0000-4000-8000-000000000003", now)

	if _, err := db.ExecContext(ctx, `
		UPDATE idempotency_records
		SET expires_at = CASE access_request_id
			WHEN $1 THEN $4
			WHEN $2 THEN $5
			WHEN $3 THEN $6
			ELSE expires_at
		END
		WHERE access_request_id IN ($1, $2, $3)
	`, expiredBefore.ID, expiredEqual.ID, activeAfter.ID, cutoff.Add(-time.Microsecond), cutoff, cutoff.Add(time.Microsecond)); err != nil {
		t.Fatalf("adjust idempotency expiry fixtures: %v", err)
	}

	if err := recordCleanupAuditFixture(ctx, db, principal, expiredBefore.ID, now); err != nil {
		t.Fatalf("record audit fixture: %v", err)
	}

	deleted, err := store.CleanupExpiredIdempotencyRecords(ctx, cutoff)
	if err != nil {
		t.Fatalf("cleanup expired idempotency records: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}

	assertIdempotencyRecordExists(ctx, t, db, expiredBefore.ID, false)
	assertIdempotencyRecordExists(ctx, t, db, expiredEqual.ID, false)
	assertIdempotencyRecordExists(ctx, t, db, activeAfter.ID, true)
	assertAccessRequestStatus(ctx, t, db, expiredBefore.ID, access.AccessRequestStatusApproved)
	assertAccessRequestStatus(ctx, t, db, expiredEqual.ID, access.AccessRequestStatusApproved)
	assertAccessRequestStatus(ctx, t, db, activeAfter.ID, access.AccessRequestStatusApproved)
	assertAuditEventExists(ctx, t, db, principal, true)

	deletedAgain, err := store.CleanupExpiredIdempotencyRecords(ctx, cutoff)
	if err != nil {
		t.Fatalf("cleanup expired idempotency records again: %v", err)
	}
	if deletedAgain != 0 {
		t.Fatalf("second deleted = %d, want 0", deletedAgain)
	}
}

func TestAccessRequestStoreIntegration_ReusesKeyAfterExpiredCleanup(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAccessRequestStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "idempotency-reuse-" + now.Format("20060102150405.000000")
	key := "00000000-0000-4000-8000-000000000004"

	t.Cleanup(func() {
		cleanupAccessRequests(context.Background(), t, db, principal)
	})

	first := createApprovedIdempotencyFixture(t, ctx, store, principal, "admin", key, now)
	second, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create second access request: %v", err)
	}

	_, err = store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: second.ID,
		ApproverSubject: "admin",
		IdempotencyKey:  key,
		Fingerprint:     approval.Fingerprint(approval.ActionApprove, second.ID, "admin", key),
		Action:          approval.ActionApprove,
		Now:             now.Add(2 * time.Second),
	})
	if !errors.Is(err, approval.ErrIdempotencyConflict) {
		t.Fatalf("error before cleanup = %v, want conflict", err)
	}

	if _, err := db.ExecContext(ctx, `
		UPDATE idempotency_records
		SET expires_at = $2
		WHERE access_request_id = $1
	`, first.ID, now.Add(-time.Second)); err != nil {
		t.Fatalf("expire idempotency record: %v", err)
	}
	deleted, err := store.CleanupExpiredIdempotencyRecords(ctx, now)
	if err != nil {
		t.Fatalf("cleanup expired idempotency records: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}

	result, err := store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: second.ID,
		ApproverSubject: "admin",
		IdempotencyKey:  key,
		Fingerprint:     approval.Fingerprint(approval.ActionApprove, second.ID, "admin", key),
		Action:          approval.ActionApprove,
		Now:             now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("reuse key after cleanup: %v", err)
	}
	if result.Replayed {
		t.Fatal("reuse after cleanup should be a new mutation, not replay")
	}
	if result.AccessRequest.ID != second.ID {
		t.Fatalf("approved id = %d, want %d", result.AccessRequest.ID, second.ID)
	}
}

func cleanupAccessRequests(ctx context.Context, t *testing.T, db *sql.DB, principal string) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		DELETE FROM idempotency_records
		WHERE access_request_id IN (
			SELECT id
			FROM access_requests
			WHERE principal = $1
		)
	`, principal); err != nil {
		t.Fatalf("cleanup idempotency records: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		DELETE FROM access_requests
		WHERE principal = $1
	`, principal); err != nil {
		t.Fatalf("cleanup access requests: %v", err)
	}
}

func createApprovedIdempotencyFixture(t *testing.T, ctx context.Context, store *AccessRequestStore, principal, adminSubject, key string, now time.Time) access.AccessRequest {
	t.Helper()

	request, err := store.CreateAccessRequest(ctx, access.CreateAccessRequestInput{
		Principal:         principal,
		Target:            "aws-sandbox-readonly",
		Provider:          "aws",
		Account:           "sandbox",
		Role:              "readonly",
		RoleARN:           "arn:aws:iam::000000000000:role/jit/readonly",
		RequestedDuration: 15 * time.Minute,
		PolicyReason:      "policy matched; approval required",
		ExpiresAt:         now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create access request fixture: %v", err)
	}

	result, err := store.MutateAccessRequest(ctx, approval.MutationInput{
		AccessRequestID: request.ID,
		ApproverSubject: adminSubject,
		IdempotencyKey:  key,
		Fingerprint:     approval.Fingerprint(approval.ActionApprove, request.ID, adminSubject, key),
		Action:          approval.ActionApprove,
		Now:             now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("approve access request fixture: %v", err)
	}
	return result.AccessRequest
}

func assertIdempotencyRecordExists(ctx context.Context, t *testing.T, db *sql.DB, accessRequestID int64, want bool) {
	t.Helper()

	var exists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM idempotency_records
			WHERE access_request_id = $1
		)
	`, accessRequestID).Scan(&exists); err != nil {
		t.Fatalf("query idempotency record existence: %v", err)
	}
	if exists != want {
		t.Fatalf("idempotency record exists for access request %d = %t, want %t", accessRequestID, exists, want)
	}
}

func assertAccessRequestStatus(ctx context.Context, t *testing.T, db *sql.DB, accessRequestID int64, want access.AccessRequestStatus) {
	t.Helper()

	var status string
	if err := db.QueryRowContext(ctx, `
		SELECT status
		FROM access_requests
		WHERE id = $1
	`, accessRequestID).Scan(&status); err != nil {
		t.Fatalf("query access request status: %v", err)
	}
	if access.AccessRequestStatus(status) != want {
		t.Fatalf("access request %d status = %q, want %q", accessRequestID, status, want)
	}
}

func recordCleanupAuditFixture(ctx context.Context, db *sql.DB, principal string, accessRequestID int64, occurredAt time.Time) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO audit_events (
			occurred_at,
			event_type,
			principal,
			target,
			decision,
			reason,
			provider,
			account,
			role,
			access_request_id,
			operator_subject
		)
		VALUES ($1, 'approval_approved', $2, 'aws-sandbox-readonly', 'allowed', 'policy matched; approval required', 'aws', 'sandbox', 'readonly', $3, 'admin')
	`, occurredAt, principal, accessRequestID)
	return err
}

func assertAuditEventExists(ctx context.Context, t *testing.T, db *sql.DB, principal string, want bool) {
	t.Helper()

	var exists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM audit_events
			WHERE principal = $1
		)
	`, principal).Scan(&exists); err != nil {
		t.Fatalf("query audit event existence: %v", err)
	}
	if exists != want {
		t.Fatalf("audit event exists for principal %q = %t, want %t", principal, exists, want)
	}
}
