package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestService_StartDeviceLoginStoresOnlyHashes(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	service := newTestService(t, store, fakeTokenGenerator{
		deviceCode:   "raw-device-code",
		userCode:     "abcd-1234",
		sessionToken: "raw-session-token",
	}, now)

	result, err := service.StartDeviceLogin(context.Background())
	if err != nil {
		t.Fatalf("start device login: %v", err)
	}

	if result.DeviceCode != "raw-device-code" {
		t.Fatalf("device code = %q", result.DeviceCode)
	}
	if result.UserCode != "ABCD-1234" {
		t.Fatalf("user code = %q, want ABCD-1234", result.UserCode)
	}
	if result.ExpiresAt != now.Add(10*time.Minute) {
		t.Fatalf("expires at = %v", result.ExpiresAt)
	}
	if store.created.DeviceCodeHash != HashToken("raw-device-code") {
		t.Fatal("device code hash was not stored")
	}
	if store.created.UserCodeHash != HashToken("ABCD-1234") {
		t.Fatal("normalized user code hash was not stored")
	}
	if string(store.created.DeviceCodeHash[:]) == "raw-device-code" {
		t.Fatal("raw device code leaked into hash field")
	}
}

func TestService_ApproveDeviceLoginNormalizesCode(t *testing.T) {
	store := newFakeStore()
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	service := newTestService(t, store, fakeTokenGenerator{}, now)

	err := service.ApproveDeviceLogin(context.Background(), " abcd-1234 ", "akira")
	if err != nil {
		t.Fatalf("approve device login: %v", err)
	}
	if store.approved.UserCodeHash != HashToken("ABCD-1234") {
		t.Fatal("approved hash did not use normalized user code")
	}
	if store.approved.Principal != "akira" {
		t.Fatalf("principal = %q, want akira", store.approved.Principal)
	}
}

func TestService_ExchangeDeviceCode(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		status      DeviceAuthorizationStatus
		expiresAt   time.Time
		wantErr     error
		wantSession bool
	}{
		{
			name:      "pending returns pending",
			status:    DeviceAuthorizationStatusPending,
			expiresAt: now.Add(time.Minute),
			wantErr:   ErrAuthorizationPending,
		},
		{
			name:      "expired pending marks expired",
			status:    DeviceAuthorizationStatusPending,
			expiresAt: now.Add(-time.Second),
			wantErr:   ErrExpired,
		},
		{
			name:      "denied returns denied",
			status:    DeviceAuthorizationStatusDenied,
			expiresAt: now.Add(time.Minute),
			wantErr:   ErrAuthorizationDenied,
		},
		{
			name:      "consumed returns invalid state",
			status:    DeviceAuthorizationStatusConsumed,
			expiresAt: now.Add(time.Minute),
			wantErr:   ErrInvalidState,
		},
		{
			name:      "unknown state returns invalid state",
			status:    DeviceAuthorizationStatus("mystery"),
			expiresAt: now.Add(time.Minute),
			wantErr:   ErrInvalidState,
		},
		{
			name:        "approved creates one hour session",
			status:      DeviceAuthorizationStatusApproved,
			expiresAt:   now.Add(time.Minute),
			wantSession: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			store.authorization = DeviceAuthorization{
				ID:        1,
				Status:    tt.status,
				Workspace: "sandbox",
				Principal: "akira",
				Endpoint:  "cloudiam.internal.local",
				ExpiresAt: tt.expiresAt,
			}
			service := newTestService(t, store, fakeTokenGenerator{
				sessionToken: "raw-session-token",
			}, now)

			session, err := service.ExchangeDeviceCode(context.Background(), "raw-device-code")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				if store.exchanged.SessionTokenHash != (Hash{}) {
					t.Fatal("device authorization was exchanged for a failed state")
				}
				return
			}
			if err != nil {
				t.Fatalf("exchange device code: %v", err)
			}
			if !tt.wantSession {
				t.Fatal("test case should expect a session")
			}
			if session.AccessToken != "raw-session-token" {
				t.Fatalf("access token = %q", session.AccessToken)
			}
			if store.exchanged.SessionTokenHash != HashToken("raw-session-token") {
				t.Fatal("session token hash was not stored")
			}
			if !store.exchanged.SessionExpiresAt.Equal(now.Add(time.Hour)) {
				t.Fatalf("session expires at = %v", store.exchanged.SessionExpiresAt)
			}
		})
	}
}

func TestService_ExchangeDeviceCodeNotFound(t *testing.T) {
	store := newFakeStore()
	store.exists = false
	service := newTestService(t, store, fakeTokenGenerator{}, time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))

	_, err := service.ExchangeDeviceCode(context.Background(), "missing-device-code")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestService_CanceledContextStopsDeviceLoginMutations(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

	t.Run("start device login", func(t *testing.T) {
		store := newFakeStore()
		service := newTestService(t, store, fakeTokenGenerator{}, now)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := service.StartDeviceLogin(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if store.created.DeviceCodeHash != (Hash{}) {
			t.Fatalf("created authorization after cancellation: %+v", store.created)
		}
	})

	t.Run("approve device login", func(t *testing.T) {
		store := newFakeStore()
		service := newTestService(t, store, fakeTokenGenerator{}, now)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := service.ApproveDeviceLogin(ctx, "ABCD-1234", "akira")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if store.approved.UserCodeHash != (Hash{}) {
			t.Fatalf("approved authorization after cancellation: %+v", store.approved)
		}
	})

	t.Run("exchange device code", func(t *testing.T) {
		store := newFakeStore()
		service := newTestService(t, store, fakeTokenGenerator{sessionToken: "raw-session-token"}, now)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := service.ExchangeDeviceCode(ctx, "raw-device-code")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if store.exchanged.SessionTokenHash != (Hash{}) {
			t.Fatalf("exchanged authorization after cancellation: %+v", store.exchanged)
		}
	})
}

func TestNewServiceValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServiceConfig
	}{
		{
			name: "missing workspace",
			cfg: ServiceConfig{
				Endpoint:        "cloudiam.internal.local",
				VerificationURI: "https://cloudiam.abacaxi.dev/device",
			},
		},
		{
			name: "missing endpoint",
			cfg: ServiceConfig{
				Workspace:       "sandbox",
				VerificationURI: "https://cloudiam.abacaxi.dev/device",
			},
		},
		{
			name: "missing verification uri",
			cfg: ServiceConfig{
				Workspace: "sandbox",
				Endpoint:  "cloudiam.internal.local",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewService(newFakeStore(), tt.cfg)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func newTestService(t *testing.T, store *fakeStore, tokens fakeTokenGenerator, now time.Time) *Service {
	t.Helper()

	service, err := NewService(store, ServiceConfig{
		Workspace:       "sandbox",
		Endpoint:        "cloudiam.internal.local",
		VerificationURI: "https://cloudiam.abacaxi.dev/device",
		Now:             func() time.Time { return now },
		TokenGenerator:  tokens,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return service
}

type fakeTokenGenerator struct {
	deviceCode   string
	userCode     string
	sessionToken string
}

func (f fakeTokenGenerator) DeviceCode() (string, error) {
	if f.deviceCode == "" {
		return "device-code", nil
	}
	return f.deviceCode, nil
}

func (f fakeTokenGenerator) UserCode() (string, error) {
	if f.userCode == "" {
		return "USER-1234", nil
	}
	return f.userCode, nil
}

func (f fakeTokenGenerator) SessionToken() (string, error) {
	if f.sessionToken == "" {
		return "session-token", nil
	}
	return f.sessionToken, nil
}

type fakeStore struct {
	exists        bool
	created       CreateDeviceAuthorizationInput
	approved      ApproveDeviceAuthorizationInput
	expired       ExpireDeviceAuthorizationInput
	exchanged     ExchangeApprovedDeviceAuthorizationInput
	authorization DeviceAuthorization
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		exists: true,
		authorization: DeviceAuthorization{
			ID:        1,
			Status:    DeviceAuthorizationStatusApproved,
			Workspace: "sandbox",
			Principal: "akira",
			Endpoint:  "cloudiam.internal.local",
			ExpiresAt: time.Date(2026, 10, 7, 12, 10, 0, 0, time.UTC),
		},
	}
}

func (f *fakeStore) CreateDeviceAuthorization(ctx context.Context, input CreateDeviceAuthorizationInput) (DeviceAuthorization, error) {
	f.created = input
	return DeviceAuthorization{
		ID:              1,
		DeviceCodeHash:  input.DeviceCodeHash,
		UserCodeHash:    input.UserCodeHash,
		UserCodeDisplay: input.UserCodeDisplay,
		Workspace:       input.Workspace,
		Endpoint:        input.Endpoint,
		Status:          DeviceAuthorizationStatusPending,
		Interval:        input.Interval,
		ExpiresAt:       input.ExpiresAt,
	}, nil
}

func (f *fakeStore) FindDeviceAuthorizationByDeviceCodeHash(ctx context.Context, hash Hash) (DeviceAuthorization, bool, error) {
	if !f.exists {
		return DeviceAuthorization{}, false, nil
	}
	authorization := f.authorization
	authorization.DeviceCodeHash = hash
	return authorization, true, nil
}

func (f *fakeStore) FindDeviceAuthorizationByUserCodeHash(ctx context.Context, hash Hash) (DeviceAuthorization, bool, error) {
	return DeviceAuthorization{}, false, nil
}

func (f *fakeStore) ApproveDeviceAuthorization(ctx context.Context, input ApproveDeviceAuthorizationInput) (DeviceAuthorization, error) {
	f.approved = input
	return DeviceAuthorization{
		ID:        1,
		Status:    DeviceAuthorizationStatusApproved,
		Principal: input.Principal,
	}, nil
}

func (f *fakeStore) DenyDeviceAuthorization(ctx context.Context, input DenyDeviceAuthorizationInput) (DeviceAuthorization, error) {
	return DeviceAuthorization{}, nil
}

func (f *fakeStore) ExpireDeviceAuthorization(ctx context.Context, input ExpireDeviceAuthorizationInput) (DeviceAuthorization, error) {
	f.expired = input
	return DeviceAuthorization{
		ID:     1,
		Status: DeviceAuthorizationStatusExpired,
	}, nil
}

func (f *fakeStore) ExchangeApprovedDeviceAuthorization(ctx context.Context, input ExchangeApprovedDeviceAuthorizationInput) (Session, error) {
	f.exchanged = input
	return Session{
		ID:                    1,
		SessionTokenHash:      input.SessionTokenHash,
		DeviceAuthorizationID: 1,
		Workspace:             f.authorization.Workspace,
		Principal:             f.authorization.Principal,
		Endpoint:              f.authorization.Endpoint,
		ExpiresAt:             input.SessionExpiresAt,
	}, nil
}

func (f *fakeStore) FindSessionByTokenHash(ctx context.Context, hash Hash) (Session, bool, error) {
	return Session{}, false, nil
}

func (f *fakeStore) RevokeSession(ctx context.Context, input RevokeSessionInput) (Session, error) {
	return Session{}, nil
}
