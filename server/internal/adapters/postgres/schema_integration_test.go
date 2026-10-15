//go:build integration

package postgresadapter

import (
	"database/sql"
	"strings"
	"testing"
)

func TestEnsurePersistenceSchemaIntegration(t *testing.T) {
	db := openIntegrationDB(t)

	if err := EnsurePersistenceSchema(t.Context(), db); err != nil {
		t.Fatalf("ensure persistence schema: %v", err)
	}
}

func TestEnsurePersistenceSchemaIntegrationReportsMissingMigration(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := t.Context()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer rollbackIntegrationTx(t, tx)

	if _, err := tx.ExecContext(ctx, `
		DROP SCHEMA IF EXISTS task8_missing_schema_test CASCADE;
		CREATE SCHEMA task8_missing_schema_test;
		CREATE TABLE task8_missing_schema_test.device_authorizations (id BIGINT);
		CREATE TABLE task8_missing_schema_test.sessions (id BIGINT);
		CREATE TABLE task8_missing_schema_test.audit_events (id BIGINT);
	`); err != nil {
		t.Fatalf("create temporary incomplete schema: %v", err)
	}

	err = ensurePersistenceSchemaQuerier(ctx, tx, "task8_missing_schema_test")
	if err == nil {
		t.Fatal("expected missing schema error")
	}
	if !strings.Contains(err.Error(), "run migrations") {
		t.Fatalf("error = %q", err.Error())
	}
	if !strings.Contains(err.Error(), "device_authorizations.device_code_hash") {
		t.Fatalf("error = %q", err.Error())
	}
}

func rollbackIntegrationTx(t *testing.T, tx *sql.Tx) {
	t.Helper()

	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		t.Fatalf("rollback transaction: %v", err)
	}
}
