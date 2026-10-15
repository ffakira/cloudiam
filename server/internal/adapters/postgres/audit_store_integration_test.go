//go:build integration

package postgresadapter

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/audit"
)

func TestAuditStoreIntegration_RecordAndListRecentEvents(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuditStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "audit-" + now.Format("20060102150405.000000")
	expiresAt := now.Add(15 * time.Minute)
	accessRequestID := int64(42)

	t.Cleanup(func() {
		cleanupAuditEvents(context.Background(), t, db, principal)
	})

	if err := store.RecordEvent(ctx, audit.RecordEventInput{
		OccurredAt:          now,
		EventType:           audit.EventTypeAccessIssued,
		Principal:           principal,
		Target:              "aws-sandbox-readonly",
		Decision:            audit.DecisionAllowed,
		Reason:              "policy matched",
		Provider:            "aws",
		Account:             "sandbox",
		Role:                "readonly",
		AccessRequestID:     &accessRequestID,
		OperatorSubject:     "admin",
		IdempotencyKeyHash:  "6dbd925562509bc30f5277ed4b5a35d2869c07b16c258f33fae541435d9dd619",
		CredentialExpiresAt: &expiresAt,
	}); err != nil {
		t.Fatalf("record audit event: %v", err)
	}

	events, err := store.ListRecentEvents(ctx, 100)
	if err != nil {
		t.Fatalf("list recent events: %v", err)
	}

	event, ok := findAuditEvent(events, principal)
	if !ok {
		t.Fatalf("expected audit event for principal %q in %+v", principal, events)
	}
	if event.EventType != audit.EventTypeAccessIssued {
		t.Fatalf("event type = %q, want %q", event.EventType, audit.EventTypeAccessIssued)
	}
	if event.Decision != audit.DecisionAllowed {
		t.Fatalf("decision = %q, want %q", event.Decision, audit.DecisionAllowed)
	}
	if event.Provider != "aws" || event.Account != "sandbox" || event.Role != "readonly" {
		t.Fatalf("provider/account/role = %q/%q/%q", event.Provider, event.Account, event.Role)
	}
	if event.AccessRequestID == nil || *event.AccessRequestID != accessRequestID {
		t.Fatalf("access request id = %v, want %d", event.AccessRequestID, accessRequestID)
	}
	if event.OperatorSubject != "admin" {
		t.Fatalf("operator subject = %q, want admin", event.OperatorSubject)
	}
	if event.IdempotencyKeyHash != "6dbd925562509bc30f5277ed4b5a35d2869c07b16c258f33fae541435d9dd619" {
		t.Fatalf("idempotency key hash = %q", event.IdempotencyKeyHash)
	}
	if event.CredentialExpiresAt == nil || !event.CredentialExpiresAt.Equal(expiresAt) {
		t.Fatalf("credential expiration = %v, want %v", event.CredentialExpiresAt, expiresAt)
	}
}

func TestAuditStoreIntegration_SchemaHasNoSecretColumns(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := t.Context()

	rows, err := db.QueryContext(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'audit_events'
		  AND column_name IN (
			'access_key_id',
			'secret_access_key',
			'session_token',
			'cloudiam_session_token',
			'device_code',
			'user_code'
		  )
	`)
	if err != nil {
		t.Fatalf("query audit event columns: %v", err)
	}
	defer rows.Close()

	var forbidden []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		forbidden = append(forbidden, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns: %v", err)
	}
	if len(forbidden) > 0 {
		t.Fatalf("audit_events has forbidden secret columns: %v", forbidden)
	}
}

func cleanupAuditEvents(ctx context.Context, t *testing.T, db *sql.DB, principal string) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		DELETE FROM audit_events
		WHERE principal = $1
	`, principal); err != nil {
		t.Fatalf("cleanup audit events: %v", err)
	}
}

func findAuditEvent(events []audit.Event, principal string) (audit.Event, bool) {
	for _, event := range events {
		if event.Principal == principal {
			return event, true
		}
	}
	return audit.Event{}, false
}

func TestAuditStoreIntegration_SchemaRejectsInvalidDecision(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := t.Context()
	principal := fmt.Sprintf("audit-invalid-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		cleanupAuditEvents(context.Background(), t, db, principal)
	})

	_, err := db.ExecContext(ctx, `
		INSERT INTO audit_events (
			event_type,
			principal,
			target,
			decision
		)
		VALUES ($1, $2, $3, $4)
	`, "access_issued", principal, "aws-sandbox-readonly", "maybe")
	if err == nil {
		t.Fatal("expected invalid decision error")
	}
}

func TestAuditStoreIntegration_AcceptsApprovalLifecycleEventTypes(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuditStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	principal := "approval-audit-" + now.Format("20060102150405.000000")
	accessRequestID := int64(43)

	t.Cleanup(func() {
		cleanupAuditEvents(context.Background(), t, db, principal)
	})

	for _, eventType := range []audit.EventType{
		audit.EventTypeApprovalRequested,
		audit.EventTypeApprovalApproved,
		audit.EventTypeApprovalDenied,
		audit.EventTypeApprovalExpired,
	} {
		if err := store.RecordEvent(ctx, audit.RecordEventInput{
			OccurredAt:         now,
			EventType:          eventType,
			Principal:          principal,
			Target:             "aws-sandbox-readonly",
			Decision:           audit.DecisionPending,
			Reason:             "policy matched; approval required",
			Provider:           "aws",
			Account:            "sandbox",
			Role:               "readonly",
			AccessRequestID:    &accessRequestID,
			OperatorSubject:    "admin",
			IdempotencyKeyHash: "6dbd925562509bc30f5277ed4b5a35d2869c07b16c258f33fae541435d9dd619",
		}); err != nil {
			t.Fatalf("record %s audit event: %v", eventType, err)
		}
	}
}
