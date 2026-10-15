package postgresadapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/approval"
)

type AccessRequestStore struct {
	db *sql.DB
}

func NewAccessRequestStore(db *sql.DB) *AccessRequestStore {
	return &AccessRequestStore{db: db}
}

func (s *AccessRequestStore) CreateAccessRequest(ctx context.Context, input access.CreateAccessRequestInput) (access.AccessRequest, error) {
	durationSeconds := int(input.RequestedDuration.Seconds())
	if durationSeconds <= 0 {
		return access.AccessRequest{}, fmt.Errorf("requested duration must be positive")
	}
	if strings.TrimSpace(input.RoleARN) == "" {
		return access.AccessRequest{}, fmt.Errorf("role arn is required")
	}

	row := s.db.QueryRowContext(ctx, `
			INSERT INTO access_requests (
				principal,
				target,
				provider,
				account,
				role,
				role_arn,
				requested_duration_seconds,
				policy_reason,
				status,
				expires_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending_approval', $9)
			RETURNING
				id, principal, target, provider, account, role, role_arn, requested_duration_seconds,
				policy_reason, status, expires_at, approved_by, denied_by, approved_at, denied_at, issued_at,
				created_at, updated_at
		`, input.Principal, input.Target, input.Provider, input.Account, input.Role, input.RoleARN, durationSeconds, input.PolicyReason, input.ExpiresAt)

	request, err := scanAccessRequest(row)
	if err != nil {
		return access.AccessRequest{}, fmt.Errorf("create access request: %w", err)
	}
	return request, nil
}

func (s *AccessRequestStore) ClaimApprovedAccessRequest(ctx context.Context, input access.ClaimApprovedAccessRequestInput) (access.AccessRequest, bool, error) {
	durationSeconds := int(input.RequestedDuration.Seconds())
	if durationSeconds <= 0 {
		return access.AccessRequest{}, false, fmt.Errorf("requested duration must be positive")
	}

	row := s.db.QueryRowContext(ctx, `
		UPDATE access_requests
		SET status = 'issued',
			issued_at = $4,
			updated_at = $4
		WHERE id = (
			SELECT id
			FROM access_requests
			WHERE principal = $1
				AND target = $2
				AND requested_duration_seconds = $3
				AND status = 'approved'
				AND expires_at > $4
			ORDER BY approved_at ASC, id ASC
			LIMIT 1
		)
			AND status = 'approved'
			AND expires_at > $4
			RETURNING
				id, principal, target, provider, account, role, role_arn, requested_duration_seconds,
				policy_reason, status, expires_at, approved_by, denied_by, approved_at, denied_at, issued_at,
				created_at, updated_at
	`, input.Principal, input.Target, durationSeconds, input.IssuedAt)

	request, err := scanAccessRequest(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return access.AccessRequest{}, false, nil
		}
		return access.AccessRequest{}, false, fmt.Errorf("claim approved access request: %w", err)
	}
	return request, true, nil
}

func (s *AccessRequestStore) ListAccessRequests(ctx context.Context, status access.AccessRequestStatus) ([]access.AccessRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, principal, target, provider, account, role, role_arn, requested_duration_seconds,
			policy_reason, status, expires_at, approved_by, denied_by, approved_at, denied_at, issued_at,
			created_at, updated_at
		FROM access_requests
		WHERE status = $1
		ORDER BY created_at ASC, id ASC
		LIMIT 100
	`, status)
	if err != nil {
		return nil, fmt.Errorf("list access requests: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var requests []access.AccessRequest
	for rows.Next() {
		request, err := scanAccessRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate access requests: %w", err)
	}
	return requests, nil
}

func (s *AccessRequestStore) CleanupExpiredIdempotencyRecords(ctx context.Context, cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		cutoff = time.Now().UTC()
	}

	result, err := s.db.ExecContext(ctx, `
		DELETE FROM idempotency_records
		WHERE expires_at <= $1
	`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired idempotency records: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count expired idempotency records cleanup: %w", err)
	}
	return deleted, nil
}

func (s *AccessRequestStore) MutateAccessRequest(ctx context.Context, input approval.MutationInput) (approval.MutationResult, error) {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return approval.MutationResult{}, fmt.Errorf("begin approval transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM idempotency_records
		WHERE key = $1
			AND admin_subject = $2
			AND expires_at <= $3
	`, input.IdempotencyKey, input.ApproverSubject, now); err != nil {
		return approval.MutationResult{}, fmt.Errorf("delete expired idempotency record: %w", err)
	}

	var existingFingerprint string
	var existingBody []byte
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint, response_body
		FROM idempotency_records
		WHERE key = $1
			AND admin_subject = $2
		FOR UPDATE
	`, input.IdempotencyKey, input.ApproverSubject).Scan(&existingFingerprint, &existingBody)
	if err == nil {
		if existingFingerprint != input.Fingerprint {
			return approval.MutationResult{}, approval.ErrIdempotencyConflict
		}
		var request access.AccessRequest
		if err := json.Unmarshal(existingBody, &request); err != nil {
			return approval.MutationResult{}, fmt.Errorf("decode idempotency response: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return approval.MutationResult{}, fmt.Errorf("commit approval replay transaction: %w", err)
		}
		return approval.MutationResult{AccessRequest: request, Replayed: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return approval.MutationResult{}, fmt.Errorf("read idempotency record: %w", err)
	}

	request, err := selectAccessRequestForUpdate(ctx, tx, input.AccessRequestID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return approval.MutationResult{}, approval.ErrAccessRequestNotFound
		}
		return approval.MutationResult{}, err
	}
	replayed, ok, err := readIdempotencyRecord(ctx, tx, input)
	if err != nil {
		return approval.MutationResult{}, err
	}
	if ok {
		if err := tx.Commit(); err != nil {
			return approval.MutationResult{}, fmt.Errorf("commit approval replay transaction: %w", err)
		}
		return replayed, nil
	}
	if request.Status != access.AccessRequestStatusPendingApproval {
		return approval.MutationResult{}, approval.ErrAccessRequestInvalidState
	}
	if !request.ExpiresAt.After(now) {
		expired, err := expireAccessRequest(ctx, tx, request.ID, now)
		if err != nil {
			return approval.MutationResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return approval.MutationResult{}, fmt.Errorf("commit expired access request transaction: %w", err)
		}
		return approval.MutationResult{AccessRequest: expired}, approval.ErrAccessRequestExpired
	}

	mutated, err := updateAccessRequestApproval(ctx, tx, input, now)
	if err != nil {
		return approval.MutationResult{}, err
	}
	responseBody, err := json.Marshal(mutated)
	if err != nil {
		return approval.MutationResult{}, fmt.Errorf("encode approval response: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO idempotency_records (
			key,
			admin_subject,
			method,
			route,
			action,
			access_request_id,
			request_fingerprint,
			response_status,
			response_body,
			expires_at,
			created_at
		)
		VALUES ($1, $2, 'POST', '/api/admin/access-requests/{id}/' || $3, $3, $4, $5, 200, $6::jsonb, $7, $8)
	`, input.IdempotencyKey, input.ApproverSubject, input.Action, input.AccessRequestID, input.Fingerprint, string(responseBody), now.Add(24*time.Hour), now); err != nil {
		return approval.MutationResult{}, fmt.Errorf("insert idempotency record: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return approval.MutationResult{}, fmt.Errorf("commit approval transaction: %w", err)
	}
	return approval.MutationResult{AccessRequest: mutated}, nil
}

func readIdempotencyRecord(ctx context.Context, tx *sql.Tx, input approval.MutationInput) (approval.MutationResult, bool, error) {
	var existingFingerprint string
	var existingBody []byte
	err := tx.QueryRowContext(ctx, `
		SELECT request_fingerprint, response_body
		FROM idempotency_records
		WHERE key = $1
			AND admin_subject = $2
		FOR UPDATE
	`, input.IdempotencyKey, input.ApproverSubject).Scan(&existingFingerprint, &existingBody)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return approval.MutationResult{}, false, nil
		}
		return approval.MutationResult{}, false, fmt.Errorf("read idempotency record: %w", err)
	}
	if existingFingerprint != input.Fingerprint {
		return approval.MutationResult{}, false, approval.ErrIdempotencyConflict
	}
	var request access.AccessRequest
	if err := json.Unmarshal(existingBody, &request); err != nil {
		return approval.MutationResult{}, false, fmt.Errorf("decode idempotency response: %w", err)
	}
	return approval.MutationResult{AccessRequest: request, Replayed: true}, true, nil
}

func expireAccessRequest(ctx context.Context, tx *sql.Tx, id int64, now time.Time) (access.AccessRequest, error) {
	row := tx.QueryRowContext(ctx, `
			UPDATE access_requests
			SET status = 'expired',
				updated_at = $2
			WHERE id = $1
			RETURNING
				id, principal, target, provider, account, role, role_arn, requested_duration_seconds,
				policy_reason, status, expires_at, approved_by, denied_by, approved_at, denied_at, issued_at,
				created_at, updated_at
	`, id, now)
	request, err := scanAccessRequest(row)
	if err != nil {
		return access.AccessRequest{}, fmt.Errorf("expire access request: %w", err)
	}
	return request, nil
}

func selectAccessRequestForUpdate(ctx context.Context, tx *sql.Tx, id int64) (access.AccessRequest, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT
			id, principal, target, provider, account, role, role_arn, requested_duration_seconds,
			policy_reason, status, expires_at, approved_by, denied_by, approved_at, denied_at, issued_at,
			created_at, updated_at
		FROM access_requests
		WHERE id = $1
		FOR UPDATE
	`, id)
	return scanAccessRequest(row)
}

func updateAccessRequestApproval(ctx context.Context, tx *sql.Tx, input approval.MutationInput, now time.Time) (access.AccessRequest, error) {
	status := access.AccessRequestStatusApproved
	actorColumn := "approved_by"
	timeColumn := "approved_at"
	if input.Action == approval.ActionDeny {
		status = access.AccessRequestStatusDenied
		actorColumn = "denied_by"
		timeColumn = "denied_at"
	}

	row := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE access_requests
		SET status = $1,
			%s = $2,
			%s = $3,
			updated_at = $3
		WHERE id = $4
		RETURNING
			id, principal, target, provider, account, role, role_arn, requested_duration_seconds,
			policy_reason, status, expires_at, approved_by, denied_by, approved_at, denied_at, issued_at,
			created_at, updated_at
	`, actorColumn, timeColumn), status, input.ApproverSubject, now, input.AccessRequestID)
	request, err := scanAccessRequest(row)
	if err != nil {
		return access.AccessRequest{}, fmt.Errorf("update access request approval: %w", err)
	}
	return request, nil
}

func scanAccessRequest(row scanner) (access.AccessRequest, error) {
	var request access.AccessRequest
	var status string
	var durationSeconds int
	var approvedAt sql.NullTime
	var deniedAt sql.NullTime
	var issuedAt sql.NullTime

	err := row.Scan(
		&request.ID,
		&request.Principal,
		&request.Target,
		&request.Provider,
		&request.Account,
		&request.Role,
		&request.RoleARN,
		&durationSeconds,
		&request.PolicyReason,
		&status,
		&request.ExpiresAt,
		&request.ApprovedBy,
		&request.DeniedBy,
		&approvedAt,
		&deniedAt,
		&issuedAt,
		&request.CreatedAt,
		&request.UpdatedAt,
	)
	if err != nil {
		return access.AccessRequest{}, fmt.Errorf("scan access request: %w", err)
	}

	request.Status = access.AccessRequestStatus(status)
	request.RequestedDuration = time.Duration(durationSeconds) * time.Second
	request.ApprovedAt = nullableTimePtr(approvedAt)
	request.DeniedAt = nullableTimePtr(deniedAt)
	request.IssuedAt = nullableTimePtr(issuedAt)
	return request, nil
}
