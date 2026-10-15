package serverapp

import (
	"context"
	"crypto/sha256"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	accessv1 "github.com/ffakira/cloudiam/server/gen/access/v1"
	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/auth"
	coreconfig "github.com/ffakira/cloudiam/server/internal/core/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRunRequiresConfigPath(t *testing.T) {
	err := Run(context.Background(), Config{Addr: "127.0.0.1:0"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "config path is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRunRequiresAuthDatabaseURLForAccessService(t *testing.T) {
	t.Setenv("CLOUDIAM_DATABASE_URL", "")
	path := writeConfigFixture(t, validConfigYAML)

	err := Run(context.Background(), Config{
		Addr:       "127.0.0.1:0",
		ConfigPath: path,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "auth database url is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

type fakeIssuer struct {
	calls   int
	request access.IssueRequest
}

func (f *fakeIssuer) Issue(ctx context.Context, request access.IssueRequest) (access.Grant, error) {
	f.calls++
	f.request = request
	return access.Grant{
		AccessKeyID:     "access-key-id",
		SecretAccessKey: "secret-access-key",
		SessionToken:    "session-token",
		ExpiresAt:       time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
	}, nil
}

type fakeServerAccessService struct {
	request access.Request
	result  access.Result
}

func (f *fakeServerAccessService) RequestAccess(ctx context.Context, request access.Request) (access.Result, error) {
	f.request = request
	return f.result, nil
}

type fakeServerSessionResolver struct {
	hash    auth.Hash
	session auth.Session
	ok      bool
}

func (f *fakeServerSessionResolver) FindSessionByTokenHash(ctx context.Context, hash auth.Hash) (auth.Session, bool, error) {
	f.hash = hash
	return f.session, f.ok, nil
}

func TestNewAccessHandlerForServerRequiresSessionResolver(t *testing.T) {
	_, err := newAccessHandlerForServer(&fakeServerAccessService{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "session resolver is required") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestNewAccessHandlerForServerUsesSessionPrincipal(t *testing.T) {
	expiresAt := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	service := &fakeServerAccessService{
		result: access.Result{
			Status: access.ResultStatusIssued,
			Grant: access.Grant{
				AccessKeyID:     "access-key-id",
				SecretAccessKey: "secret-access-key",
				SessionToken:    "session-token",
				ExpiresAt:       expiresAt,
			},
		},
	}
	resolver := &fakeServerSessionResolver{
		ok:      true,
		session: auth.Session{Principal: "akira"},
	}
	handler, err := newAccessHandlerForServer(service, resolver)
	if err != nil {
		t.Fatalf("new access handler: %v", err)
	}

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "cloudiam-session",
		Principal:            "spoofed-principal",
		Target:               "aws-sandbox-readonly",
		DurationSeconds:      900,
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if service.request.Principal != "akira" {
		t.Fatalf("principal = %q, want akira", service.request.Principal)
	}
	if resolver.hash != sha256.Sum256([]byte("cloudiam-session")) {
		t.Fatal("session token hash was not used for lookup")
	}
	if response.GetAccessKeyId() != "access-key-id" || response.GetExpiresAt() != expiresAt.Format(time.RFC3339) {
		t.Fatalf("response = %+v", response)
	}
}

func TestNewAccessHandlerForServerRejectsUnknownSession(t *testing.T) {
	service := &fakeServerAccessService{}
	handler, err := newAccessHandlerForServer(service, &fakeServerSessionResolver{})
	if err != nil {
		t.Fatalf("new access handler: %v", err)
	}

	response, err := handler.RequestAccess(context.Background(), &accessv1.RequestAccessRequest{
		CloudiamSessionToken: "unknown-session",
		Principal:            "spoofed-principal",
		Target:               "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("code = %s, want %s", code, codes.Unauthenticated)
	}
	if response != nil {
		t.Fatalf("response = %+v, want nil", response)
	}
	if service.request.Principal != "" {
		t.Fatalf("service was called with principal %q", service.request.Principal)
	}
}

func TestNewHTTPRateLimiterFromEnvUsesLocalFallbackWithoutRedis(t *testing.T) {
	limiter, readiness, closeLimiter, err := newHTTPRateLimiterFromEnv(Config{})
	if err != nil {
		t.Fatalf("new rate limiter: %v", err)
	}
	if readiness != nil {
		t.Fatalf("readiness = %T, want nil without Redis", readiness)
	}
	t.Cleanup(func() {
		if err := closeLimiter(); err != nil {
			t.Fatalf("close rate limiter: %v", err)
		}
	})

	if !limiter.Allow(context.Background(), "device:127.0.0.1", 1, time.Minute) {
		t.Fatal("first fallback request was rate limited")
	}
	if limiter.Allow(context.Background(), "device:127.0.0.1", 1, time.Minute) {
		t.Fatal("second fallback request should be rate limited")
	}
}

func TestNewHTTPRateLimiterFromEnvRejectsInvalidRedisURL(t *testing.T) {
	_, _, _, err := newHTTPRateLimiterFromEnv(Config{RedisURL: "://bad-url"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse redis rate limiter url") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestConsoleCookieSecureFromEnv(t *testing.T) {
	t.Setenv("CLOUDIAM_CONSOLE_COOKIE_SECURE", "true")

	secure, err := consoleCookieSecureFromEnv()
	if err != nil {
		t.Fatalf("console cookie secure: %v", err)
	}
	if !secure {
		t.Fatal("secure = false, want true")
	}
}

func TestConsoleCookieSecureFromEnvRejectsInvalidBool(t *testing.T) {
	t.Setenv("CLOUDIAM_CONSOLE_COOKIE_SECURE", "sometimes")

	_, err := consoleCookieSecureFromEnv()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "CLOUDIAM_CONSOLE_COOKIE_SECURE") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestValidateProductionSecretConfigAcceptsHardenedConfig(t *testing.T) {
	err := validateProductionSecretConfig(Config{
		DeploymentEnv:       "production",
		AuthDatabaseURL:     "postgres://cloudiam@db:5432/cloudiam?sslmode=require",
		AuthVerificationURI: "https://cloudiam.example.com/device",
		RedisURL:            "redis://redis:6379/0",
		ConsoleCookieSecure: true,
		ConsoleJWTSecret:    "0123456789abcdef0123456789abcdef",
	})
	if err != nil {
		t.Fatalf("validate production config: %v", err)
	}
}

func TestValidateProductionSecretConfigRequiresSecureCookie(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.ConsoleCookieSecure = false
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_CONSOLE_COOKIE_SECURE") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateProductionSecretConfigRequiresRedis(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.RedisURL = ""
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_REDIS_URL") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateProductionSecretConfigRequiresHTTPSVerificationURI(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.AuthVerificationURI = "http://cloudiam.example.com/device"
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_AUTH_VERIFICATION_URI") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateProductionSecretConfigRequiresExplicitVerificationURI(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.AuthVerificationURI = ""
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_AUTH_VERIFICATION_URI") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateProductionSecretConfigRejectsDisabledDatabaseTLS(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.AuthDatabaseURL = "postgres://cloudiam@db:5432/cloudiam?sslmode=disable"
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_DATABASE_URL") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateProductionSecretConfigAllowsExplicitDatabaseTLSOverride(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.AuthDatabaseURL = "postgres://cloudiam@db:5432/cloudiam?sslmode=disable"
	cfg.AllowInsecureDatabaseInProduction = true
	err := validateProductionSecretConfig(cfg)
	if err != nil {
		t.Fatalf("validate production config: %v", err)
	}
}

func TestValidateProductionSecretConfigRejectsWeakJWTSecret(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.ConsoleJWTSecret = "test-secret"
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_CONSOLE_JWT_SECRET") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateProductionSecretConfigRejectsConsoleSeedByDefault(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.ConsoleSeedPassword = "long-enough-bootstrap-password"
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_CONSOLE_SEED_PASSWORD") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateProductionSecretConfigAllowsStrongConsoleSeedWithOverride(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.ConsoleSeedPassword = "long-enough-bootstrap-password"
	cfg.AllowConsoleSeedInProduction = true
	err := validateProductionSecretConfig(cfg)
	if err != nil {
		t.Fatalf("validate production config: %v", err)
	}
}

func TestValidateProductionSecretConfigRejectsWeakConsoleSeedWithOverride(t *testing.T) {
	cfg := validProductionSecretConfig()
	cfg.ConsoleSeedPassword = "password"
	cfg.AllowConsoleSeedInProduction = true
	err := validateProductionSecretConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "CLOUDIAM_CONSOLE_SEED_PASSWORD") {
		t.Fatalf("error = %v", err)
	}
}

func validProductionSecretConfig() Config {
	return Config{
		DeploymentEnv:       "production",
		AuthDatabaseURL:     "postgres://cloudiam@db:5432/cloudiam?sslmode=require",
		AuthVerificationURI: "https://cloudiam.example.com/device",
		RedisURL:            "redis://redis:6379/0",
		ConsoleCookieSecure: true,
		ConsoleJWTSecret:    "0123456789abcdef0123456789abcdef",
	}
}

func TestNewAccessServiceUsesConfigPolicyAndTargetResolution(t *testing.T) {
	issuer := &fakeIssuer{}
	service := newAccessService(validConfig(), issuer)

	_, err := service.RequestAccess(context.Background(), access.Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
		Duration:  15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}

	if issuer.calls != 1 {
		t.Fatalf("issuer calls = %d, want 1", issuer.calls)
	}
	if issuer.request.Principal != "akira" {
		t.Fatalf("principal = %q, want akira", issuer.request.Principal)
	}
	if issuer.request.Target != "aws-sandbox-readonly" {
		t.Fatalf("target = %q, want aws-sandbox-readonly", issuer.request.Target)
	}
	if issuer.request.Provider != "aws" {
		t.Fatalf("provider = %q, want aws", issuer.request.Provider)
	}
	if issuer.request.Account != "sandbox" {
		t.Fatalf("account = %q, want sandbox", issuer.request.Account)
	}
	if issuer.request.Role != "readonly" {
		t.Fatalf("role = %q, want readonly", issuer.request.Role)
	}
	if issuer.request.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("role arn = %q", issuer.request.RoleARN)
	}
}

func TestNewAccessServiceDeniesBeforeIssuer(t *testing.T) {
	issuer := &fakeIssuer{}
	service := newAccessService(validConfig(), issuer)

	_, err := service.RequestAccess(context.Background(), access.Request{
		Principal: "mika",
		Target:    "aws-sandbox-readonly",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if issuer.calls != 0 {
		t.Fatalf("issuer calls = %d, want 0", issuer.calls)
	}
}

func TestSnapshotRequestEvaluatorReturnsApprovalRequired(t *testing.T) {
	cfg := validConfig()
	target := cfg.Access.Targets["aws-sandbox-readonly"]
	target.ApprovalRequired = true
	cfg.Access.Targets["aws-sandbox-readonly"] = target
	store := newConfigStore("", cfg)

	decision, resolved, err := (snapshotRequestEvaluator{store: store}).EvaluateRequest("akira", "aws-sandbox-readonly", 15*time.Minute)
	if err != nil {
		t.Fatalf("evaluate request: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("allowed = false, reason = %q", decision.Reason)
	}
	if !decision.ApprovalRequired {
		t.Fatal("approval required = false, want true")
	}
	if resolved.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("role arn = %q", resolved.RoleARN)
	}
}

func TestSnapshotRequestEvaluatorDeniedRequestDoesNotRequireApproval(t *testing.T) {
	cfg := validConfig()
	target := cfg.Access.Targets["aws-sandbox-readonly"]
	target.ApprovalRequired = true
	cfg.Access.Targets["aws-sandbox-readonly"] = target
	store := newConfigStore("", cfg)

	decision, _, err := (snapshotRequestEvaluator{store: store}).EvaluateRequest("mika", "aws-sandbox-readonly", 15*time.Minute)
	if err != nil {
		t.Fatalf("evaluate request: %v", err)
	}
	if decision.Allowed {
		t.Fatal("allowed = true, want false")
	}
	if decision.ApprovalRequired {
		t.Fatal("approval required = true for denied request")
	}
}

func TestConfigStoreClonesInitialConfigSnapshot(t *testing.T) {
	cfg := validConfig()
	store := newConfigStore("", cfg)

	account := cfg.Providers.AWS.Accounts["sandbox"]
	account.Roles["readonly"] = coreconfig.RoleConfig{ARN: "arn:aws:iam::000000000000:role/jit/mutated"}
	cfg.Providers.AWS.Accounts["sandbox"] = account
	user := cfg.Principals.Users["akira"]
	user.Groups[0] = "support"
	cfg.Principals.Users["akira"] = user
	target := cfg.Access.Targets["aws-sandbox-readonly"]
	target.ApprovalRequired = true
	cfg.Access.Targets["aws-sandbox-readonly"] = target
	cfg.Access.Policies[0].Targets[0] = "mutated-target"

	decision, resolved, err := (snapshotRequestEvaluator{store: store}).EvaluateRequest("akira", "aws-sandbox-readonly", 15*time.Minute)
	if err != nil {
		t.Fatalf("evaluate request: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("allowed = false, reason = %q", decision.Reason)
	}
	if decision.ApprovalRequired {
		t.Fatal("approval required changed through original config mutation")
	}
	if resolved.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("role arn = %q, want original readonly role", resolved.RoleARN)
	}
}

func validConfig() coreconfig.Config {
	return coreconfig.Config{
		Version: "v1",
		Tenant:  "<COMPANY_UUID>",
		Serve: coreconfig.ServeConfig{
			GRPC: coreconfig.GRPCConfig{
				Addr: "127.0.0.1:50051",
			},
		},
		Providers: coreconfig.ProvidersConfig{
			AWS: coreconfig.AWSProviderConfig{
				Region: "us-east-1",
				Accounts: map[string]coreconfig.AccountConfig{
					"sandbox": {
						ID: "000000000000",
						Roles: map[string]coreconfig.RoleConfig{
							"readonly": {
								ARN: "arn:aws:iam::000000000000:role/jit/readonly",
							},
						},
					},
				},
			},
		},
		Principals: coreconfig.PrincipalsConfig{
			Users: map[string]coreconfig.UserConfig{
				"akira": {
					Groups: []string{"developers"},
				},
				"mika": {
					Groups: []string{"support"},
				},
			},
		},
		Access: coreconfig.AccessConfig{
			Targets: map[string]coreconfig.TargetConfig{
				"aws-sandbox-readonly": {
					Provider:        "aws",
					Account:         "sandbox",
					Role:            "readonly",
					DefaultDuration: "15m",
					MaxDuration:     "1h",
				},
			},
			Policies: []coreconfig.PolicyConfig{
				{
					Name:    "developers-readonly",
					Groups:  []string{"developers"},
					Targets: []string{"aws-sandbox-readonly"},
				},
			},
		},
	}
}

type blockingIssuer struct {
	started chan access.IssueRequest
	release chan struct{}

	mu      sync.Mutex
	request access.IssueRequest
}

func newBlockingIssuer() *blockingIssuer {
	return &blockingIssuer{
		started: make(chan access.IssueRequest, 1),
		release: make(chan struct{}),
	}
}

func (i *blockingIssuer) Issue(ctx context.Context, request access.IssueRequest) (access.Grant, error) {
	i.mu.Lock()
	i.request = request
	i.mu.Unlock()
	i.started <- request

	select {
	case <-ctx.Done():
		return access.Grant{}, ctx.Err()
	case <-i.release:
		return access.Grant{
			AccessKeyID:     "access-key-id",
			SecretAccessKey: "secret-access-key",
			SessionToken:    "session-token",
			ExpiresAt:       time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC),
		}, nil
	}
}

type lockedBuffer struct {
	mu      sync.Mutex
	builder strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.builder.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.builder.String()
}

func TestConfigStoreReloadValidRoleARNChange(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	issuer := &fakeIssuer{}
	service := newAccessServiceFromStore(store, issuer)

	reloadedYAML := strings.ReplaceAll(validConfigYAML, "arn:aws:iam::000000000000:role/jit/readonly", "arn:aws:iam::000000000000:role/jit/reloaded")
	writeFile(t, path, reloadedYAML)

	status := store.Reload(context.Background())
	if !status.Success {
		t.Fatalf("reload failed: %v", status.Err)
	}
	if status.Generation != 2 {
		t.Fatalf("generation = %d, want 2", status.Generation)
	}

	_, err := service.RequestAccess(context.Background(), access.Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err != nil {
		t.Fatalf("request access: %v", err)
	}
	if issuer.request.RoleARN != "arn:aws:iam::000000000000:role/jit/reloaded" {
		t.Fatalf("role arn = %q", issuer.request.RoleARN)
	}
}

func TestConfigStoreReloadApprovalRequiredChange(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())

	decision, _, err := (snapshotRequestEvaluator{store: store}).EvaluateRequest("akira", "aws-sandbox-readonly", 15*time.Minute)
	if err != nil {
		t.Fatalf("evaluate request: %v", err)
	}
	if decision.ApprovalRequired {
		t.Fatal("approval required before reload = true, want false")
	}

	reloadedYAML := strings.Replace(
		validConfigYAML,
		"      max_duration: 1h\n",
		"      max_duration: 1h\n      approval_required: true\n",
		1,
	)
	writeFile(t, path, reloadedYAML)

	status := store.Reload(context.Background())
	if !status.Success {
		t.Fatalf("reload failed: %v", status.Err)
	}

	decision, _, err = (snapshotRequestEvaluator{store: store}).EvaluateRequest("akira", "aws-sandbox-readonly", 15*time.Minute)
	if err != nil {
		t.Fatalf("evaluate request after reload: %v", err)
	}
	if !decision.ApprovalRequired {
		t.Fatal("approval required after reload = false, want true")
	}
}

func TestConfigStoreReloadInvalidYAMLKeepsOldSnapshot(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	writeFile(t, path, "version: [")

	status := store.Reload(context.Background())
	if status.Success {
		t.Fatal("expected reload failure")
	}
	if status.Generation != 1 {
		t.Fatalf("generation = %d, want 1", status.Generation)
	}
	if store.Current().generation != 1 {
		t.Fatalf("active generation = %d, want 1", store.Current().generation)
	}
}

func TestConfigStoreReloadInvalidSemanticConfigKeepsOldSnapshot(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	writeFile(t, path, strings.ReplaceAll(validConfigYAML, "id: \"000000000000\"", "id: \"not-an-account\""))

	status := store.Reload(context.Background())
	if status.Success {
		t.Fatal("expected reload failure")
	}
	if store.Current().generation != 1 {
		t.Fatalf("active generation = %d, want 1", store.Current().generation)
	}
}

func TestConfigStoreReloadRejectsRestartRequiredFieldChange(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	writeFile(t, path, strings.ReplaceAll(validConfigYAML, "addr: 127.0.0.1:50051", "addr: 127.0.0.1:60051"))

	status := store.Reload(context.Background())
	if status.Success {
		t.Fatal("expected reload failure")
	}
	if !strings.Contains(status.Err.Error(), "requires server restart") {
		t.Fatalf("error = %q", status.Err.Error())
	}
	if store.Current().generation != 1 {
		t.Fatalf("active generation = %d, want 1", store.Current().generation)
	}
}

func TestConfigStoreReloadRejectsAWSRegionChange(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	writeFile(t, path, strings.ReplaceAll(validConfigYAML, "region: us-east-1", "region: us-west-2"))

	status := store.Reload(context.Background())
	if status.Success {
		t.Fatal("expected reload failure")
	}
	if !strings.Contains(status.Err.Error(), "requires server restart") {
		t.Fatalf("error = %q", status.Err.Error())
	}
	if store.Current().generation != 1 {
		t.Fatalf("active generation = %d, want 1", store.Current().generation)
	}
}

func TestConfigStoreReloadCanceledContextKeepsOldSnapshot(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	writeFile(t, path, "version: [")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status := store.Reload(ctx)
	if status.Success {
		t.Fatal("expected reload failure")
	}
	if !errors.Is(status.Err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", status.Err)
	}
	if status.Generation != 1 {
		t.Fatalf("generation = %d, want 1", status.Generation)
	}
	if store.Current().generation != 1 {
		t.Fatalf("active generation = %d, want 1", store.Current().generation)
	}

	decision, resolved, err := (snapshotRequestEvaluator{store: store}).EvaluateRequest("akira", "aws-sandbox-readonly", 15*time.Minute)
	if err != nil {
		t.Fatalf("evaluate request: %v", err)
	}
	if !decision.Allowed {
		t.Fatalf("allowed = false, reason = %q", decision.Reason)
	}
	if resolved.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("role arn = %q, want old snapshot", resolved.RoleARN)
	}
}

func TestConfigStoreReloadKeepsInFlightRequestOnOldSnapshot(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	issuer := newBlockingIssuer()
	service := newAccessServiceFromStore(store, issuer)
	errCh := make(chan error, 1)

	go func() {
		_, err := service.RequestAccess(context.Background(), access.Request{
			Principal: "akira",
			Target:    "aws-sandbox-readonly",
		})
		errCh <- err
	}()

	request := <-issuer.started
	if request.RoleARN != "arn:aws:iam::000000000000:role/jit/readonly" {
		t.Fatalf("in-flight role arn = %q", request.RoleARN)
	}

	reloadedYAML := strings.ReplaceAll(validConfigYAML, "arn:aws:iam::000000000000:role/jit/readonly", "arn:aws:iam::000000000000:role/jit/reloaded")
	writeFile(t, path, reloadedYAML)
	status := store.Reload(context.Background())
	if !status.Success {
		t.Fatalf("reload failed: %v", status.Err)
	}
	close(issuer.release)
	if err := <-errCh; err != nil {
		t.Fatalf("request access: %v", err)
	}

	nextIssuer := &fakeIssuer{}
	nextService := newAccessServiceFromStore(store, nextIssuer)
	_, err := nextService.RequestAccess(context.Background(), access.Request{
		Principal: "akira",
		Target:    "aws-sandbox-readonly",
	})
	if err != nil {
		t.Fatalf("next request access: %v", err)
	}
	if nextIssuer.request.RoleARN != "arn:aws:iam::000000000000:role/jit/reloaded" {
		t.Fatalf("new request role arn = %q", nextIssuer.request.RoleARN)
	}
}

func TestConfigStoreConcurrentReloadsKeepCompleteGeneration(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	writeFile(t, path, strings.ReplaceAll(validConfigYAML, "arn:aws:iam::000000000000:role/jit/readonly", "arn:aws:iam::000000000000:role/jit/reloaded"))

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := store.Reload(context.Background())
			if !status.Success {
				t.Errorf("reload failed: %v", status.Err)
			}
		}()
	}
	wg.Wait()

	snapshot := store.Current()
	resolved, err := snapshot.config.ResolveTarget("aws-sandbox-readonly", 0)
	if err != nil {
		t.Fatalf("resolve target: %v", err)
	}
	if resolved.RoleARN != "arn:aws:iam::000000000000:role/jit/reloaded" {
		t.Fatalf("role arn = %q", resolved.RoleARN)
	}
	if snapshot.generation != 11 {
		t.Fatalf("generation = %d, want 11", snapshot.generation)
	}
}

func TestConfigStoreConcurrentReadersAndReloadsSeeCompleteSnapshots(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	approvalRequiredYAML := strings.Replace(
		strings.ReplaceAll(validConfigYAML, "arn:aws:iam::000000000000:role/jit/readonly", "arn:aws:iam::000000000000:role/jit/reloaded"),
		"      max_duration: 1h\n",
		"      max_duration: 1h\n      approval_required: true\n",
		1,
	)

	var stop atomic.Bool
	invalid := make(chan string, 1)
	var reloadWG sync.WaitGroup
	for i := 0; i < 4; i++ {
		reloadWG.Add(1)
		go func(i int) {
			defer reloadWG.Done()
			for j := 0; j < 40; j++ {
				var err error
				if (i+j)%2 == 0 {
					err = os.WriteFile(filepath.Clean(path), []byte(validConfigYAML), 0o600)
				} else {
					err = os.WriteFile(filepath.Clean(path), []byte(approvalRequiredYAML), 0o600)
				}
				if err != nil {
					select {
					case invalid <- err.Error():
					default:
					}
					return
				}
				_ = store.Reload(context.Background())
			}
		}(i)
	}

	var readerWG sync.WaitGroup
	for i := 0; i < 8; i++ {
		readerWG.Add(1)
		go func() {
			defer readerWG.Done()
			evaluator := snapshotRequestEvaluator{store: store}
			for !stop.Load() {
				decision, resolved, err := evaluator.EvaluateRequest("akira", "aws-sandbox-readonly", 15*time.Minute)
				if err != nil {
					select {
					case invalid <- err.Error():
					default:
					}
					return
				}
				switch {
				case resolved.RoleARN == "arn:aws:iam::000000000000:role/jit/readonly" && !decision.ApprovalRequired:
				case resolved.RoleARN == "arn:aws:iam::000000000000:role/jit/reloaded" && decision.ApprovalRequired:
				default:
					select {
					case invalid <- resolved.RoleARN + " approval_required mismatch":
					default:
					}
					return
				}
			}
		}()
	}
	reloadWG.Wait()
	stop.Store(true)
	readerWG.Wait()

	select {
	case got := <-invalid:
		t.Fatalf("observed mixed config snapshot: %s", got)
	default:
	}
}

func TestReloadSignalHandlerReloadsOnSIGHUP(t *testing.T) {
	path := writeConfigFixture(t, validConfigYAML)
	store := newConfigStore(path, validConfig())
	var out lockedBuffer
	ctx, cancel := context.WithCancel(context.Background())
	stop := startReloadSignalHandler(ctx, store, &out)
	t.Cleanup(func() {
		cancel()
		stop()
	})

	reloadedYAML := strings.ReplaceAll(validConfigYAML, "arn:aws:iam::000000000000:role/jit/readonly", "arn:aws:iam::000000000000:role/jit/reloaded")
	writeFile(t, path, reloadedYAML)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("send sighup: %v", err)
	}

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for store.Current().generation != 2 {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for reload; output = %q", out.String())
		case <-ticker.C:
		}
	}

	if !strings.Contains(out.String(), "config reload succeeded") {
		t.Fatalf("reload output = %q", out.String())
	}
}

func TestStopGRPCServerStopsAfterTimeout(t *testing.T) {
	stopper := newFakeGRPCStopper(true)

	stopGRPCServer(stopper, time.Nanosecond)

	if !stopper.stopped.Load() {
		t.Fatal("Stop was not called after graceful stop timeout")
	}
	if !stopper.gracefulReturned.Load() {
		t.Fatal("GracefulStop did not return after Stop")
	}
}

func TestStopGRPCServerDoesNotForceStopWhenGracefulCompletes(t *testing.T) {
	stopper := newFakeGRPCStopper(false)

	stopGRPCServer(stopper, time.Second)

	if stopper.stopped.Load() {
		t.Fatal("Stop was called after graceful stop completed")
	}
	if !stopper.gracefulReturned.Load() {
		t.Fatal("GracefulStop did not run")
	}
}

func TestShutdownHTTPServerReturnsAfterTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	})
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second,
	}
	serverConn, clientConn := net.Pipe()
	listener := &pipeListener{conn: serverConn}
	defer func() {
		close(release)
		_ = clientConn.Close()
		_ = server.Close()
	}()

	serveErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	go func() {
		_, _ = clientConn.Write([]byte("GET / HTTP/1.1\r\nHost: cloudiam.local\r\n\r\n"))
	}()

	select {
	case <-started:
	case err := <-serveErr:
		t.Fatalf("server exited before request started: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for blocked request")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		shutdownHTTPServer(server, time.Millisecond)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdownHTTPServer did not return after timeout")
	}
}

type pipeListener struct {
	conn net.Conn
	once sync.Once
	done chan struct{}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	if l.done == nil {
		l.done = make(chan struct{})
	}
	var conn net.Conn
	l.once.Do(func() {
		conn = l.conn
	})
	if conn != nil {
		return conn, nil
	}
	<-l.done
	return nil, net.ErrClosed
}

func (l *pipeListener) Close() error {
	if l.done == nil {
		l.done = make(chan struct{})
	}
	select {
	case <-l.done:
	default:
		close(l.done)
	}
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return pipeAddr("pipe")
}

type pipeAddr string

func (a pipeAddr) Network() string { return string(a) }
func (a pipeAddr) String() string  { return string(a) }

func TestReportServerErrorDoesNotBlockWhenChannelFull(t *testing.T) {
	errCh := make(chan error, 1)
	errCh <- errors.New("first error")

	done := make(chan struct{})
	go func() {
		defer close(done)
		reportServerError(errCh, errors.New("second error"))
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reportServerError blocked on a full channel")
	}
}

func TestWriteReloadStatusDoesNotIncludeCredentialMaterial(t *testing.T) {
	var out strings.Builder
	writeReloadStatus(&out, reloadStatus{
		Success:    false,
		ConfigPath: "/etc/cloudiam/cloudiam.yaml",
		Generation: 3,
		Err:        errors.New("parse config token=cloudiam-session password=db-password redis://default:redis-pass@127.0.0.1:6379/0 AWS_SECRET_ACCESS_KEY=secret-access-key AKIA1234567890ABCDEF"),
	})

	text := out.String()
	for _, secret := range []string{"cloudiam-session", "db-password", "redis-pass", "secret-access-key", "AKIA1234567890ABCDEF"} {
		if strings.Contains(text, secret) {
			t.Fatalf("reload status contains secret material %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, "config reload failed") || !strings.Contains(text, "generation=3") {
		t.Fatalf("reload status = %q", text)
	}
}

type fakeIdempotencyRecordCleaner struct {
	cutoff time.Time
	count  int64
	err    error
}

func (f *fakeIdempotencyRecordCleaner) CleanupExpiredIdempotencyRecords(ctx context.Context, cutoff time.Time) (int64, error) {
	f.cutoff = cutoff
	return f.count, f.err
}

func TestCleanupExpiredIdempotencyRecordsOnStartup(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 123, time.UTC)
	cleaner := &fakeIdempotencyRecordCleaner{count: 3}

	deleted, err := cleanupExpiredIdempotencyRecordsOnStartup(context.Background(), cleaner, func() time.Time {
		return now
	})
	if err != nil {
		t.Fatalf("cleanup expired idempotency records: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted = %d, want 3", deleted)
	}
	if !cleaner.cutoff.Equal(now) {
		t.Fatalf("cutoff = %s, want %s", cleaner.cutoff, now)
	}
}

func TestCleanupExpiredIdempotencyRecordsOnStartupPropagatesRedactedError(t *testing.T) {
	cleaner := &fakeIdempotencyRecordCleaner{err: errors.New("database unavailable idempotency_key=replay-key request_fingerprint=fp-value response_body=body-value jwt=console-jwt AWS_SECRET_ACCESS_KEY=secret-access-key")}

	_, err := cleanupExpiredIdempotencyRecordsOnStartup(context.Background(), cleaner, func() time.Time {
		return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cleanup expired idempotency records on startup") {
		t.Fatalf("error = %q", err.Error())
	}
	for _, secret := range []string{"replay-key", "fp-value", "body-value", "console-jwt", "secret-access-key"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error contains secret material %q: %v", secret, err)
		}
	}
}

type fakeGRPCStopper struct {
	blockGraceful    bool
	release          chan struct{}
	releaseOnce      sync.Once
	stopped          atomic.Bool
	gracefulReturned atomic.Bool
}

func newFakeGRPCStopper(blockGraceful bool) *fakeGRPCStopper {
	return &fakeGRPCStopper{
		blockGraceful: blockGraceful,
		release:       make(chan struct{}),
	}
}

func (s *fakeGRPCStopper) GracefulStop() {
	if s.blockGraceful {
		<-s.release
	}
	s.gracefulReturned.Store(true)
}

func (s *fakeGRPCStopper) Stop() {
	s.stopped.Store(true)
	s.releaseOnce.Do(func() {
		close(s.release)
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Clean(path), []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
