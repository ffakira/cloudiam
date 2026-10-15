package postgresadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/auth"
)

type AuthStore struct {
	db *sql.DB
}

func NewAuthStore(db *sql.DB) *AuthStore {
	return &AuthStore{db: db}
}

func (s *AuthStore) CreateDeviceAuthorization(ctx context.Context, input auth.CreateDeviceAuthorizationInput) (auth.DeviceAuthorization, error) {
	intervalSeconds := int(input.Interval.Seconds())
	if intervalSeconds <= 0 {
		intervalSeconds = 5
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO device_authorizations (
			device_code_hash,
			user_code_hash,
			user_code_display,
			workspace,
			endpoint,
			interval_seconds,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING
			id, device_code_hash, user_code_hash, user_code_display, workspace, principal,
			endpoint, status, interval_seconds, expires_at, approved_at, denied_at,
			consumed_at, created_at, updated_at
	`, input.DeviceCodeHash[:], input.UserCodeHash[:], input.UserCodeDisplay, input.Workspace, input.Endpoint, intervalSeconds, input.ExpiresAt)

	authorization, err := scanDeviceAuthorization(row)
	if err != nil {
		return auth.DeviceAuthorization{}, fmt.Errorf("create device authorization: %w", err)
	}
	return authorization, nil
}

func (s *AuthStore) FindDeviceAuthorizationByDeviceCodeHash(ctx context.Context, hash auth.Hash) (auth.DeviceAuthorization, bool, error) {
	return s.findDeviceAuthorization(ctx, findDeviceAuthorizationByDeviceCodeHashQuery, hash, "device_code_hash")
}

func (s *AuthStore) FindDeviceAuthorizationByUserCodeHash(ctx context.Context, hash auth.Hash) (auth.DeviceAuthorization, bool, error) {
	return s.findDeviceAuthorization(ctx, findDeviceAuthorizationByUserCodeHashQuery, hash, "user_code_hash")
}

func (s *AuthStore) ApproveDeviceAuthorization(ctx context.Context, input auth.ApproveDeviceAuthorizationInput) (auth.DeviceAuthorization, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE device_authorizations
		SET status = 'approved',
			principal = $2,
			approved_at = $3,
			updated_at = $3
		WHERE user_code_hash = $1
			AND status = 'pending'
			AND expires_at > $3
		RETURNING
			id, device_code_hash, user_code_hash, user_code_display, workspace, principal,
			endpoint, status, interval_seconds, expires_at, approved_at, denied_at,
			consumed_at, created_at, updated_at
	`, input.UserCodeHash[:], input.Principal, input.ApprovedAt)

	authorization, err := scanDeviceAuthorization(row)
	if err != nil {
		return auth.DeviceAuthorization{}, mapMutationError(err, "approve device authorization")
	}
	return authorization, nil
}

func (s *AuthStore) DenyDeviceAuthorization(ctx context.Context, input auth.DenyDeviceAuthorizationInput) (auth.DeviceAuthorization, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE device_authorizations
		SET status = 'denied',
			denied_at = $2,
			updated_at = $2
		WHERE user_code_hash = $1
			AND status = 'pending'
			AND expires_at > $2
		RETURNING
			id, device_code_hash, user_code_hash, user_code_display, workspace, principal,
			endpoint, status, interval_seconds, expires_at, approved_at, denied_at,
			consumed_at, created_at, updated_at
	`, input.UserCodeHash[:], input.DeniedAt)

	authorization, err := scanDeviceAuthorization(row)
	if err != nil {
		return auth.DeviceAuthorization{}, mapMutationError(err, "deny device authorization")
	}
	return authorization, nil
}

func (s *AuthStore) ExpireDeviceAuthorization(ctx context.Context, input auth.ExpireDeviceAuthorizationInput) (auth.DeviceAuthorization, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE device_authorizations
		SET status = 'expired',
			updated_at = $2
		WHERE device_code_hash = $1
			AND status = 'pending'
			AND expires_at <= $2
		RETURNING
			id, device_code_hash, user_code_hash, user_code_display, workspace, principal,
			endpoint, status, interval_seconds, expires_at, approved_at, denied_at,
			consumed_at, created_at, updated_at
	`, input.DeviceCodeHash[:], input.ExpiredAt)

	authorization, err := scanDeviceAuthorization(row)
	if err != nil {
		return auth.DeviceAuthorization{}, mapMutationError(err, "expire device authorization")
	}
	return authorization, nil
}

func (s *AuthStore) ExchangeApprovedDeviceAuthorization(ctx context.Context, input auth.ExchangeApprovedDeviceAuthorizationInput) (auth.Session, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return auth.Session{}, fmt.Errorf("begin exchange transaction: %w", err)
	}
	defer rollbackUnlessCommitted(tx)

	authorization, err := findDeviceAuthorizationForUpdate(ctx, tx, input.DeviceCodeHash)
	if err != nil {
		return auth.Session{}, err
	}
	if authorization.Status != auth.DeviceAuthorizationStatusApproved {
		return auth.Session{}, auth.ErrInvalidState
	}
	if !authorization.ExpiresAt.After(input.ConsumedAt) {
		return auth.Session{}, auth.ErrExpired
	}
	if authorization.Principal == "" {
		return auth.Session{}, auth.ErrInvalidState
	}

	session, err := insertSession(ctx, tx, authorization, input)
	if err != nil {
		return auth.Session{}, err
	}
	if err := markDeviceAuthorizationConsumed(ctx, tx, authorization.ID, input.ConsumedAt); err != nil {
		return auth.Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return auth.Session{}, fmt.Errorf("commit exchange transaction: %w", err)
	}

	return session, nil
}

func (s *AuthStore) FindSessionByTokenHash(ctx context.Context, hash auth.Hash) (auth.Session, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, session_token_hash, device_authorization_id, workspace, principal,
			endpoint, expires_at, created_at, revoked_at
		FROM sessions
		WHERE session_token_hash = $1
			AND revoked_at IS NULL
			AND expires_at > now()
	`, hash[:])

	session, err := scanSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.Session{}, false, nil
		}
		return auth.Session{}, false, fmt.Errorf("find session by token hash: %w", err)
	}
	return session, true, nil
}

func (s *AuthStore) RevokeSession(ctx context.Context, input auth.RevokeSessionInput) (auth.Session, error) {
	row := s.db.QueryRowContext(ctx, `
		UPDATE sessions
		SET revoked_at = $2
		WHERE session_token_hash = $1
			AND revoked_at IS NULL
		RETURNING
			id, session_token_hash, device_authorization_id, workspace, principal,
			endpoint, expires_at, created_at, revoked_at
	`, input.SessionTokenHash[:], input.RevokedAt)

	session, err := scanSession(row)
	if err != nil {
		return auth.Session{}, mapMutationError(err, "revoke session")
	}
	return session, nil
}

func (s *AuthStore) findDeviceAuthorization(ctx context.Context, query string, hash auth.Hash, lookup string) (auth.DeviceAuthorization, bool, error) {
	row := s.db.QueryRowContext(ctx, query, hash[:])
	authorization, err := scanDeviceAuthorization(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.DeviceAuthorization{}, false, nil
		}
		return auth.DeviceAuthorization{}, false, fmt.Errorf("find device authorization by %s: %w", lookup, err)
	}
	return authorization, true, nil
}

const findDeviceAuthorizationByDeviceCodeHashQuery = `
	SELECT
		id, device_code_hash, user_code_hash, user_code_display, workspace, principal,
		endpoint, status, interval_seconds, expires_at, approved_at, denied_at,
		consumed_at, created_at, updated_at
	FROM device_authorizations
	WHERE device_code_hash = $1
`

const findDeviceAuthorizationByUserCodeHashQuery = `
	SELECT
		id, device_code_hash, user_code_hash, user_code_display, workspace, principal,
		endpoint, status, interval_seconds, expires_at, approved_at, denied_at,
		consumed_at, created_at, updated_at
	FROM device_authorizations
	WHERE user_code_hash = $1
`

type scanner interface {
	Scan(dest ...any) error
}

func scanDeviceAuthorization(row scanner) (auth.DeviceAuthorization, error) {
	var authorization auth.DeviceAuthorization
	var deviceHash []byte
	var userHash []byte
	var principal sql.NullString
	var approvedAt sql.NullTime
	var deniedAt sql.NullTime
	var consumedAt sql.NullTime
	var status string
	var intervalSeconds int

	err := row.Scan(
		&authorization.ID,
		&deviceHash,
		&userHash,
		&authorization.UserCodeDisplay,
		&authorization.Workspace,
		&principal,
		&authorization.Endpoint,
		&status,
		&intervalSeconds,
		&authorization.ExpiresAt,
		&approvedAt,
		&deniedAt,
		&consumedAt,
		&authorization.CreatedAt,
		&authorization.UpdatedAt,
	)
	if err != nil {
		return auth.DeviceAuthorization{}, err
	}
	deviceCodeHash, err := hashFromBytes(deviceHash)
	if err != nil {
		return auth.DeviceAuthorization{}, fmt.Errorf("scan device code hash: %w", err)
	}
	userCodeHash, err := hashFromBytes(userHash)
	if err != nil {
		return auth.DeviceAuthorization{}, fmt.Errorf("scan user code hash: %w", err)
	}

	authorization.DeviceCodeHash = deviceCodeHash
	authorization.UserCodeHash = userCodeHash
	authorization.Status = auth.DeviceAuthorizationStatus(status)
	authorization.Interval = time.Duration(intervalSeconds) * time.Second
	if principal.Valid {
		authorization.Principal = principal.String
	}
	authorization.ApprovedAt = nullableTimePtr(approvedAt)
	authorization.DeniedAt = nullableTimePtr(deniedAt)
	authorization.ConsumedAt = nullableTimePtr(consumedAt)
	return authorization, nil
}

func scanSession(row scanner) (auth.Session, error) {
	var session auth.Session
	var tokenHash []byte
	var revokedAt sql.NullTime

	err := row.Scan(
		&session.ID,
		&tokenHash,
		&session.DeviceAuthorizationID,
		&session.Workspace,
		&session.Principal,
		&session.Endpoint,
		&session.ExpiresAt,
		&session.CreatedAt,
		&revokedAt,
	)
	if err != nil {
		return auth.Session{}, err
	}
	hash, err := hashFromBytes(tokenHash)
	if err != nil {
		return auth.Session{}, fmt.Errorf("scan session token hash: %w", err)
	}

	session.SessionTokenHash = hash
	session.RevokedAt = nullableTimePtr(revokedAt)
	return session, nil
}

func hashFromBytes(data []byte) (auth.Hash, error) {
	var hash auth.Hash
	if len(data) != len(hash) {
		return auth.Hash{}, fmt.Errorf("hash length = %d, want %d", len(data), len(hash))
	}
	copy(hash[:], data)
	return hash, nil
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func findDeviceAuthorizationForUpdate(ctx context.Context, tx *sql.Tx, hash auth.Hash) (auth.DeviceAuthorization, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT
			id, device_code_hash, user_code_hash, user_code_display, workspace, principal,
			endpoint, status, interval_seconds, expires_at, approved_at, denied_at,
			consumed_at, created_at, updated_at
		FROM device_authorizations
		WHERE device_code_hash = $1
		FOR UPDATE
	`, hash[:])

	authorization, err := scanDeviceAuthorization(row)
	if err != nil {
		return auth.DeviceAuthorization{}, mapMutationError(err, "find device authorization for exchange")
	}
	return authorization, nil
}

func insertSession(ctx context.Context, tx *sql.Tx, authorization auth.DeviceAuthorization, input auth.ExchangeApprovedDeviceAuthorizationInput) (auth.Session, error) {
	row := tx.QueryRowContext(ctx, `
		INSERT INTO sessions (
			session_token_hash,
			device_authorization_id,
			workspace,
			principal,
			endpoint,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id, session_token_hash, device_authorization_id, workspace, principal,
			endpoint, expires_at, created_at, revoked_at
	`, input.SessionTokenHash[:], authorization.ID, authorization.Workspace, authorization.Principal, authorization.Endpoint, input.SessionExpiresAt)

	session, err := scanSession(row)
	if err != nil {
		return auth.Session{}, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

func markDeviceAuthorizationConsumed(ctx context.Context, tx *sql.Tx, id int64, consumedAt time.Time) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE device_authorizations
		SET status = 'consumed',
			consumed_at = $2,
			updated_at = $2
		WHERE id = $1
			AND status = 'approved'
	`, id, consumedAt)
	if err != nil {
		return fmt.Errorf("mark device authorization consumed: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read consumed rows affected: %w", err)
	}
	if rows != 1 {
		return auth.ErrInvalidState
	}
	return nil
}

func rollbackUnlessCommitted(tx *sql.Tx) {
	_ = tx.Rollback()
}

func mapMutationError(err error, operation string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return auth.ErrNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}
