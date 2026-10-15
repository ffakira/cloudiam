package postgresadapter

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/audit"
)

type AuditStore struct {
	db *sql.DB
}

func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

func (s *AuditStore) RecordEvent(ctx context.Context, input audit.RecordEventInput) error {
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
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
			operator_subject,
			idempotency_key_hash,
			credential_expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		input.OccurredAt,
		input.EventType,
		input.Principal,
		input.Target,
		input.Decision,
		input.Reason,
		input.Provider,
		input.Account,
		input.Role,
		input.AccessRequestID,
		input.OperatorSubject,
		input.IdempotencyKeyHash,
		input.CredentialExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}
	return nil
}

func (s *AuditStore) ListRecentEvents(ctx context.Context, limit int) ([]audit.Event, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
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
			operator_subject,
			idempotency_key_hash,
			credential_expires_at,
			created_at
		FROM audit_events
		ORDER BY occurred_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent audit events: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var events []audit.Event
	for rows.Next() {
		event, err := scanAuditEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return events, nil
}

func scanAuditEvent(row scanner) (audit.Event, error) {
	var event audit.Event
	var eventType string
	var decision string
	var credentialExpiresAt sql.NullTime
	var accessRequestID sql.NullInt64

	err := row.Scan(
		&event.ID,
		&event.OccurredAt,
		&eventType,
		&event.Principal,
		&event.Target,
		&decision,
		&event.Reason,
		&event.Provider,
		&event.Account,
		&event.Role,
		&accessRequestID,
		&event.OperatorSubject,
		&event.IdempotencyKeyHash,
		&credentialExpiresAt,
		&event.CreatedAt,
	)
	if err != nil {
		return audit.Event{}, fmt.Errorf("scan audit event: %w", err)
	}

	event.EventType = audit.EventType(eventType)
	event.Decision = audit.Decision(decision)
	if accessRequestID.Valid {
		event.AccessRequestID = &accessRequestID.Int64
	}
	event.CredentialExpiresAt = nullableTimePtr(credentialExpiresAt)
	return event, nil
}
