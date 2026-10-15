//go:build integration

package postgresadapter

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/auth"
)

func TestAuthStoreIntegration_DeviceAuthorizationSessionFlow(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuthStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	workspace := "integration-" + now.Format("20060102150405.000000")
	deviceHash := testHash("device:" + workspace)
	userHash := testHash("user:" + workspace)
	sessionHash := testHash("session:" + workspace)

	t.Cleanup(func() {
		cleanupIntegrationRows(context.Background(), t, db, workspace)
	})

	authorization, err := store.CreateDeviceAuthorization(ctx, auth.CreateDeviceAuthorizationInput{
		DeviceCodeHash:  deviceHash,
		UserCodeHash:    userHash,
		UserCodeDisplay: "ABCD-1234",
		Workspace:       workspace,
		Endpoint:        "cloudiam.internal.local",
		Interval:        5 * time.Second,
		ExpiresAt:       now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create device authorization: %v", err)
	}
	if authorization.Status != auth.DeviceAuthorizationStatusPending {
		t.Fatalf("status = %q, want pending", authorization.Status)
	}

	found, ok, err := store.FindDeviceAuthorizationByDeviceCodeHash(ctx, deviceHash)
	if err != nil {
		t.Fatalf("find by device hash: %v", err)
	}
	if !ok {
		t.Fatal("expected device authorization by device hash")
	}
	if found.UserCodeDisplay != "ABCD-1234" {
		t.Fatalf("user code display = %q", found.UserCodeDisplay)
	}

	approved, err := store.ApproveDeviceAuthorization(ctx, auth.ApproveDeviceAuthorizationInput{
		UserCodeHash: userHash,
		Principal:    "akira",
		ApprovedAt:   now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("approve device authorization: %v", err)
	}
	if approved.Status != auth.DeviceAuthorizationStatusApproved {
		t.Fatalf("status = %q, want approved", approved.Status)
	}
	if approved.Principal != "akira" {
		t.Fatalf("principal = %q, want akira", approved.Principal)
	}

	session, err := store.ExchangeApprovedDeviceAuthorization(ctx, auth.ExchangeApprovedDeviceAuthorizationInput{
		DeviceCodeHash:   deviceHash,
		SessionTokenHash: sessionHash,
		SessionExpiresAt: now.Add(time.Hour),
		ConsumedAt:       now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("exchange approved device authorization: %v", err)
	}
	if session.Principal != "akira" {
		t.Fatalf("session principal = %q, want akira", session.Principal)
	}
	if session.DeviceAuthorizationID != authorization.ID {
		t.Fatalf("session device authorization id = %d, want %d", session.DeviceAuthorizationID, authorization.ID)
	}

	_, err = store.ExchangeApprovedDeviceAuthorization(ctx, auth.ExchangeApprovedDeviceAuthorizationInput{
		DeviceCodeHash:   deviceHash,
		SessionTokenHash: testHash("session-again:" + workspace),
		SessionExpiresAt: now.Add(time.Hour),
		ConsumedAt:       now.Add(3 * time.Second),
	})
	if !errors.Is(err, auth.ErrInvalidState) {
		t.Fatalf("second exchange error = %v, want ErrInvalidState", err)
	}

	activeSession, ok, err := store.FindSessionByTokenHash(ctx, sessionHash)
	if err != nil {
		t.Fatalf("find session by token hash: %v", err)
	}
	if !ok {
		t.Fatal("expected active session")
	}
	if activeSession.ID != session.ID {
		t.Fatalf("active session id = %d, want %d", activeSession.ID, session.ID)
	}

	revoked, err := store.RevokeSession(ctx, auth.RevokeSessionInput{
		SessionTokenHash: sessionHash,
		RevokedAt:        now.Add(4 * time.Second),
	})
	if err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("expected revoked_at")
	}

	_, ok, err = store.FindSessionByTokenHash(ctx, sessionHash)
	if err != nil {
		t.Fatalf("find revoked session by token hash: %v", err)
	}
	if ok {
		t.Fatal("revoked session should not be active")
	}

	_, err = store.RevokeSession(ctx, auth.RevokeSessionInput{
		SessionTokenHash: sessionHash,
		RevokedAt:        now.Add(5 * time.Second),
	})
	if !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("second revoke error = %v, want ErrNotFound", err)
	}
}

func TestAuthStoreIntegration_ExchangeExpiredApprovedDeviceAuthorizationFailsClosed(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuthStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	workspace := "expired-approved-exchange-" + now.Format("20060102150405.000000")
	deviceHash := testHash("device:" + workspace)
	userHash := testHash("user:" + workspace)

	t.Cleanup(func() {
		cleanupIntegrationRows(context.Background(), t, db, workspace)
	})

	_, err := store.CreateDeviceAuthorization(ctx, auth.CreateDeviceAuthorizationInput{
		DeviceCodeHash:  deviceHash,
		UserCodeHash:    userHash,
		UserCodeDisplay: "ABCD-1234",
		Workspace:       workspace,
		Endpoint:        "cloudiam.internal.local",
		Interval:        5 * time.Second,
		ExpiresAt:       now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("create device authorization: %v", err)
	}
	_, err = store.ApproveDeviceAuthorization(ctx, auth.ApproveDeviceAuthorizationInput{
		UserCodeHash: userHash,
		Principal:    "akira",
		ApprovedAt:   now,
	})
	if err != nil {
		t.Fatalf("approve device authorization: %v", err)
	}

	_, err = store.ExchangeApprovedDeviceAuthorization(ctx, auth.ExchangeApprovedDeviceAuthorizationInput{
		DeviceCodeHash:   deviceHash,
		SessionTokenHash: testHash("session:" + workspace),
		SessionExpiresAt: now.Add(time.Hour),
		ConsumedAt:       now.Add(2 * time.Second),
	})
	if !errors.Is(err, auth.ErrExpired) {
		t.Fatalf("exchange expired authorization error = %v, want ErrExpired", err)
	}

	authorization, ok, err := store.FindDeviceAuthorizationByDeviceCodeHash(ctx, deviceHash)
	if err != nil {
		t.Fatalf("find device authorization: %v", err)
	}
	if !ok {
		t.Fatal("expected device authorization")
	}
	if authorization.Status != auth.DeviceAuthorizationStatusApproved {
		t.Fatalf("status = %q, want approved", authorization.Status)
	}
	if authorization.ConsumedAt != nil {
		t.Fatalf("expired exchange consumed authorization: %+v", authorization.ConsumedAt)
	}
	var sessionCount int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM sessions
		WHERE workspace = $1
	`, workspace).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("sessions = %d, want 0", sessionCount)
	}
}

func TestAuthStoreIntegration_ConcurrentExchangeApprovedDeviceAuthorizationIsSingleUse(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuthStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	workspace := "concurrent-exchange-" + now.Format("20060102150405.000000")
	deviceHash := testHash("device:" + workspace)
	userHash := testHash("user:" + workspace)

	t.Cleanup(func() {
		cleanupIntegrationRows(context.Background(), t, db, workspace)
	})

	_, err := store.CreateDeviceAuthorization(ctx, auth.CreateDeviceAuthorizationInput{
		DeviceCodeHash:  deviceHash,
		UserCodeHash:    userHash,
		UserCodeDisplay: "ABCD-1234",
		Workspace:       workspace,
		Endpoint:        "cloudiam.internal.local",
		Interval:        5 * time.Second,
		ExpiresAt:       now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create device authorization: %v", err)
	}
	if _, err := store.ApproveDeviceAuthorization(ctx, auth.ApproveDeviceAuthorizationInput{
		UserCodeHash: userHash,
		Principal:    "akira",
		ApprovedAt:   now.Add(time.Second),
	}); err != nil {
		t.Fatalf("approve device authorization: %v", err)
	}

	const attempts = 8
	start := make(chan struct{})
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := store.ExchangeApprovedDeviceAuthorization(context.Background(), auth.ExchangeApprovedDeviceAuthorizationInput{
				DeviceCodeHash:   deviceHash,
				SessionTokenHash: testHash(workspace + ":session:" + time.Duration(i).String()),
				SessionExpiresAt: now.Add(time.Hour),
				ConsumedAt:       now.Add(2 * time.Second),
			})
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	invalidState := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, auth.ErrInvalidState) {
			invalidState++
			continue
		}
		t.Fatalf("unexpected exchange error: %v", err)
	}
	if successes != 1 {
		t.Fatalf("successful exchanges = %d, want 1", successes)
	}
	if invalidState != attempts-1 {
		t.Fatalf("invalid state exchanges = %d, want %d", invalidState, attempts-1)
	}

	var sessionCount int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM sessions
		WHERE workspace = $1
	`, workspace).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("sessions = %d, want 1", sessionCount)
	}
}

func TestAuthStoreIntegration_ConcurrentApproveAndExpireDeviceAuthorizationSettlesOneState(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuthStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	workspace := "concurrent-approve-expire-" + now.Format("20060102150405.000000")
	deviceHash := testHash("device:" + workspace)
	userHash := testHash("user:" + workspace)

	t.Cleanup(func() {
		cleanupIntegrationRows(context.Background(), t, db, workspace)
	})

	_, err := store.CreateDeviceAuthorization(ctx, auth.CreateDeviceAuthorizationInput{
		DeviceCodeHash:  deviceHash,
		UserCodeHash:    userHash,
		UserCodeDisplay: "ABCD-1234",
		Workspace:       workspace,
		Endpoint:        "cloudiam.internal.local",
		Interval:        5 * time.Second,
		ExpiresAt:       now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("create device authorization: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := store.ApproveDeviceAuthorization(context.Background(), auth.ApproveDeviceAuthorizationInput{
			UserCodeHash: userHash,
			Principal:    "akira",
			ApprovedAt:   now,
		})
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := store.ExpireDeviceAuthorization(context.Background(), auth.ExpireDeviceAuthorizationInput{
			DeviceCodeHash: deviceHash,
			ExpiredAt:      now.Add(2 * time.Second),
		})
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	notFound := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, auth.ErrNotFound) {
			notFound++
			continue
		}
		t.Fatalf("unexpected transition error: %v", err)
	}
	if successes != 1 {
		t.Fatalf("successful transitions = %d, want 1", successes)
	}
	if notFound != 1 {
		t.Fatalf("not found transitions = %d, want 1", notFound)
	}

	authorization, ok, err := store.FindDeviceAuthorizationByDeviceCodeHash(ctx, deviceHash)
	if err != nil {
		t.Fatalf("find device authorization: %v", err)
	}
	if !ok {
		t.Fatal("expected device authorization")
	}
	if authorization.Status != auth.DeviceAuthorizationStatusApproved &&
		authorization.Status != auth.DeviceAuthorizationStatusExpired {
		t.Fatalf("status = %q, want approved or expired", authorization.Status)
	}
}

func TestAuthStoreIntegration_ConcurrentApproveAndDenyDeviceAuthorizationSettlesOneState(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuthStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	workspace := "concurrent-approve-deny-" + now.Format("20060102150405.000000")
	deviceHash := testHash("device:" + workspace)
	userHash := testHash("user:" + workspace)

	t.Cleanup(func() {
		cleanupIntegrationRows(context.Background(), t, db, workspace)
	})

	_, err := store.CreateDeviceAuthorization(ctx, auth.CreateDeviceAuthorizationInput{
		DeviceCodeHash:  deviceHash,
		UserCodeHash:    userHash,
		UserCodeDisplay: "ABCD-1234",
		Workspace:       workspace,
		Endpoint:        "cloudiam.internal.local",
		Interval:        5 * time.Second,
		ExpiresAt:       now.Add(10 * time.Minute),
	})
	if err != nil {
		t.Fatalf("create device authorization: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := store.ApproveDeviceAuthorization(context.Background(), auth.ApproveDeviceAuthorizationInput{
			UserCodeHash: userHash,
			Principal:    "akira",
			ApprovedAt:   now,
		})
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := store.DenyDeviceAuthorization(context.Background(), auth.DenyDeviceAuthorizationInput{
			UserCodeHash: userHash,
			DeniedAt:     now,
		})
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	notFound := 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, auth.ErrNotFound) {
			notFound++
			continue
		}
		t.Fatalf("unexpected transition error: %v", err)
	}
	if successes != 1 {
		t.Fatalf("successful transitions = %d, want 1", successes)
	}
	if notFound != 1 {
		t.Fatalf("not found transitions = %d, want 1", notFound)
	}

	authorization, ok, err := store.FindDeviceAuthorizationByDeviceCodeHash(ctx, deviceHash)
	if err != nil {
		t.Fatalf("find device authorization: %v", err)
	}
	if !ok {
		t.Fatal("expected device authorization")
	}
	if authorization.Status != auth.DeviceAuthorizationStatusApproved &&
		authorization.Status != auth.DeviceAuthorizationStatusDenied {
		t.Fatalf("status = %q, want approved or denied", authorization.Status)
	}
}

func TestAuthStoreIntegration_DenyExpiredDeviceAuthorizationFailsClosed(t *testing.T) {
	db := openIntegrationDB(t)
	store := NewAuthStore(db)
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)
	workspace := "deny-expired-" + now.Format("20060102150405.000000")
	deviceHash := testHash("device:" + workspace)
	userHash := testHash("user:" + workspace)

	t.Cleanup(func() {
		cleanupIntegrationRows(context.Background(), t, db, workspace)
	})

	_, err := store.CreateDeviceAuthorization(ctx, auth.CreateDeviceAuthorizationInput{
		DeviceCodeHash:  deviceHash,
		UserCodeHash:    userHash,
		UserCodeDisplay: "ABCD-1234",
		Workspace:       workspace,
		Endpoint:        "cloudiam.internal.local",
		Interval:        5 * time.Second,
		ExpiresAt:       now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("create device authorization: %v", err)
	}

	_, err = store.DenyDeviceAuthorization(ctx, auth.DenyDeviceAuthorizationInput{
		UserCodeHash: userHash,
		DeniedAt:     now.Add(2 * time.Second),
	})
	if !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("deny expired authorization error = %v, want ErrNotFound", err)
	}
}

func TestAuthStoreIntegration_SchemaRejectsInvalidHashLength(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := t.Context()
	validUserHash := testHash("valid-user")

	_, err := db.ExecContext(ctx, `
		INSERT INTO device_authorizations (
			device_code_hash,
			user_code_hash,
			user_code_display,
			workspace,
			endpoint,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, []byte("short"), validUserHash[:], "ABCD-1234", "invalid-hash-test", "cloudiam.internal.local", time.Now().Add(10*time.Minute))
	if err == nil {
		t.Fatal("expected invalid hash length error")
	}
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("CLOUDIAM_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://cloudiam:cloudiam@127.0.0.1:6000/cloudiam?sslmode=disable"
	}
	db, err := Open(t.Context(), databaseURL)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close integration database: %v", err)
		}
	})
	return db
}

func cleanupIntegrationRows(ctx context.Context, t *testing.T, db *sql.DB, workspace string) {
	t.Helper()

	if _, err := db.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE workspace = $1
	`, workspace); err != nil {
		t.Fatalf("cleanup sessions: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		DELETE FROM device_authorizations
		WHERE workspace = $1
	`, workspace); err != nil {
		t.Fatalf("cleanup device authorizations: %v", err)
	}
}

func testHash(value string) auth.Hash {
	return sha256.Sum256([]byte(value))
}
