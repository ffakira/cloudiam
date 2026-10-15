package postgresadapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type schemaQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type schemaRequirement struct {
	table   string
	columns []string
}

var persistenceSchemaRequirements = []schemaRequirement{
	{
		table: "device_authorizations",
		columns: []string{
			"id",
			"device_code_hash",
			"user_code_hash",
			"user_code_display",
			"workspace",
			"principal",
			"endpoint",
			"status",
			"interval_seconds",
			"expires_at",
			"approved_at",
			"denied_at",
			"consumed_at",
			"created_at",
			"updated_at",
		},
	},
	{
		table: "sessions",
		columns: []string{
			"id",
			"session_token_hash",
			"device_authorization_id",
			"workspace",
			"principal",
			"endpoint",
			"expires_at",
			"created_at",
			"revoked_at",
		},
	},
	{
		table: "audit_events",
		columns: []string{
			"id",
			"occurred_at",
			"event_type",
			"principal",
			"target",
			"decision",
			"reason",
			"provider",
			"account",
			"role",
			"access_request_id",
			"operator_subject",
			"idempotency_key_hash",
			"credential_expires_at",
			"created_at",
		},
	},
	{
		table: "access_requests",
		columns: []string{
			"id",
			"principal",
			"target",
			"provider",
			"account",
			"role",
			"role_arn",
			"requested_duration_seconds",
			"policy_reason",
			"status",
			"expires_at",
			"approved_by",
			"denied_by",
			"approved_at",
			"denied_at",
			"issued_at",
			"created_at",
			"updated_at",
		},
	},
	{
		table: "idempotency_records",
		columns: []string{
			"id",
			"key",
			"admin_subject",
			"method",
			"route",
			"action",
			"access_request_id",
			"request_fingerprint",
			"response_status",
			"response_body",
			"expires_at",
			"created_at",
		},
	},
	{
		table: "console_users",
		columns: []string{
			"username",
			"password_hash",
			"active",
			"created_at",
			"updated_at",
		},
	},
}

func EnsurePersistenceSchema(ctx context.Context, db *sql.DB) error {
	return ensurePersistenceSchemaQuerier(ctx, db, "public")
}

func ensurePersistenceSchemaQuerier(ctx context.Context, db schemaQuerier, schema string) error {
	for _, requirement := range persistenceSchemaRequirements {
		missing, err := missingColumns(ctx, db, schema, requirement)
		if err != nil {
			return err
		}
		if len(missing) > 0 {
			return fmt.Errorf("postgres persistence schema is missing %s.%s; run migrations", requirement.table, strings.Join(missing, ","))
		}
	}
	return nil
}

func missingColumns(ctx context.Context, db schemaQuerier, schema string, requirement schemaRequirement) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = $2
	`, schema, requirement.table)
	if err != nil {
		return nil, fmt.Errorf("check postgres persistence schema table %s: %w", requirement.table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	found := map[string]struct{}{}
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, fmt.Errorf("scan postgres persistence schema table %s: %w", requirement.table, err)
		}
		found[column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres persistence schema table %s: %w", requirement.table, err)
	}

	var missing []string
	for _, column := range requirement.columns {
		if _, ok := found[column]; !ok {
			missing = append(missing, column)
		}
	}
	return missing, nil
}
