package serverapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	accessv1 "github.com/ffakira/cloudiam/server/gen/access/v1"
	authv1 "github.com/ffakira/cloudiam/server/gen/auth/v1"
	awsadapter "github.com/ffakira/cloudiam/server/internal/adapters/aws"
	yamlconfig "github.com/ffakira/cloudiam/server/internal/adapters/config/yaml"
	grpcadapter "github.com/ffakira/cloudiam/server/internal/adapters/grpc"
	httpadapter "github.com/ffakira/cloudiam/server/internal/adapters/http"
	postgresadapter "github.com/ffakira/cloudiam/server/internal/adapters/postgres"
	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/audit"
	"github.com/ffakira/cloudiam/server/internal/core/auth"
	coreconfig "github.com/ffakira/cloudiam/server/internal/core/config"
	"github.com/ffakira/cloudiam/server/internal/core/consoleauth"
	"google.golang.org/grpc"
)

type Config struct {
	Addr                              string
	ConfigPath                        string
	AuthDatabaseURL                   string
	AuthWorkspace                     string
	AuthEndpoint                      string
	AuthVerificationURI               string
	HTTPAddr                          string
	RedisURL                          string
	DeploymentEnv                     string
	ConsoleSeedUsername               string
	ConsoleSeedPassword               string
	ConsoleJWTSecret                  string
	ConsoleJWTTTL                     time.Duration
	ConsoleCookieSecure               bool
	AllowConsoleSeedInProduction      bool
	AllowInsecureDatabaseInProduction bool
	DashboardDistDir                  string
	ReloadStatusWriter                io.Writer
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:50051"
	}
	if strings.TrimSpace(cfg.ConfigPath) == "" {
		return fmt.Errorf("config path is required")
	}

	appCfg, err := yamlconfig.LoadFile(cfg.ConfigPath)
	if err != nil {
		return err
	}
	configStore := newConfigStore(cfg.ConfigPath, appCfg)

	if cfg.AuthDatabaseURL == "" {
		cfg.AuthDatabaseURL = os.Getenv("CLOUDIAM_DATABASE_URL")
	}
	if cfg.AuthDatabaseURL == "" {
		return fmt.Errorf("auth database url is required for authenticated access service")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(appCfg.Providers.AWS.Region))
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	stsClient := sts.NewFromConfig(awsCfg)
	issuer := awsadapter.NewSTSIssuer(stsClient)
	grpcServer := grpc.NewServer()
	errCh := make(chan error, 2)

	if cfg.AuthDatabaseURL == "" {
		cfg.AuthDatabaseURL = os.Getenv("CLOUDIAM_DATABASE_URL")
	}
	if cfg.AuthWorkspace == "" {
		cfg.AuthWorkspace = os.Getenv("CLOUDIAM_AUTH_WORKSPACE")
	}
	if cfg.AuthEndpoint == "" {
		cfg.AuthEndpoint = os.Getenv("CLOUDIAM_AUTH_ENDPOINT")
	}
	if cfg.AuthVerificationURI == "" {
		cfg.AuthVerificationURI = os.Getenv("CLOUDIAM_AUTH_VERIFICATION_URI")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = os.Getenv("CLOUDIAM_HTTP_ADDR")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = "127.0.0.1:8080"
	}
	if cfg.RedisURL == "" {
		cfg.RedisURL = os.Getenv("CLOUDIAM_REDIS_URL")
	}
	if cfg.ConsoleSeedUsername == "" {
		cfg.ConsoleSeedUsername = os.Getenv("CLOUDIAM_CONSOLE_SEED_USERNAME")
	}
	if cfg.ConsoleSeedPassword == "" {
		cfg.ConsoleSeedPassword = os.Getenv("CLOUDIAM_CONSOLE_SEED_PASSWORD")
	}
	if cfg.ConsoleJWTSecret == "" {
		cfg.ConsoleJWTSecret = os.Getenv("CLOUDIAM_CONSOLE_JWT_SECRET")
	}
	if cfg.DeploymentEnv == "" {
		cfg.DeploymentEnv = os.Getenv("CLOUDIAM_DEPLOYMENT_ENV")
	}
	if !cfg.AllowConsoleSeedInProduction {
		cfg.AllowConsoleSeedInProduction = boolFromEnv("CLOUDIAM_ALLOW_CONSOLE_SEED_IN_PRODUCTION")
	}
	if !cfg.AllowInsecureDatabaseInProduction {
		cfg.AllowInsecureDatabaseInProduction = boolFromEnv("CLOUDIAM_ALLOW_INSECURE_DATABASE_IN_PRODUCTION")
	}
	if !cfg.ConsoleCookieSecure {
		secureCookie, err := consoleCookieSecureFromEnv()
		if err != nil {
			return err
		}
		cfg.ConsoleCookieSecure = secureCookie
	}
	if err := validateProductionSecretConfig(cfg); err != nil {
		return err
	}
	rateLimiter, redisReadiness, closeRateLimiter, err := newHTTPRateLimiterFromEnv(cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = closeRateLimiter()
	}()

	var closeAuth func()
	var httpServer *http.Server
	var authService *auth.Service
	var sessionStore *postgresadapter.AuthStore
	var auditStore audit.Store
	var accessRequestStore *postgresadapter.AccessRequestStore
	var consoleUserStore *postgresadapter.ConsoleUserStore
	var consoleAuthService *consoleauth.Service
	var accessOptions []access.Option
	var authDB *sql.DB
	if cfg.AuthDatabaseURL != "" {
		var cleanup func()
		authDB, authService, sessionStore, auditStore, accessRequestStore, consoleUserStore, cleanup, err = newAuthServices(ctx, cfg, appCfg)
		if err != nil {
			return err
		}
		closeAuth = cleanup
		defer closeAuth()
		accessOptions = append(accessOptions, access.WithAuditRecorder(auditStore))
		accessOptions = append(accessOptions, access.WithAccessRequestStore(accessRequestStore))
		if err := seedConsoleUserFromEnv(ctx, cfg, consoleUserStore); err != nil {
			return err
		}
		consoleAuthService, err = newConsoleAuthServiceFromEnv(cfg, consoleUserStore)
		if err != nil && !errors.Is(err, consoleauth.ErrDisabled) {
			return err
		}
	}

	accessService := newAccessServiceFromStore(configStore, issuer, accessOptions...)
	authv1.RegisterAuthServiceServer(grpcServer, grpcadapter.NewAuthHandler(authService))
	accessHandler, err := newAccessHandlerForServer(accessService, sessionStore)
	if err != nil {
		return err
	}
	accessv1.RegisterAccessServiceServer(grpcServer, accessHandler)

	readinessDependencies := map[string]readinessDependency{
		"config": configReadinessDependency(configStore),
	}
	if authDB != nil {
		readinessDependencies["postgres"] = pingReadinessDependency{name: "postgres", pinger: authDB}
	}
	if redisReadiness != nil {
		readinessDependencies["redis"] = redisReadiness
	}
	server, err := startHTTPDevServer(cfg.HTTPAddr, authService, auditStore, consoleAuthService, accessRequestStore, dashboardDistDir(cfg), rateLimiter, cfg.ConsoleCookieSecure, newReadinessChecker(readinessDependencies), errCh)
	if err != nil {
		return err
	}
	httpServer = server
	stopReloadSignals := startReloadSignalHandler(ctx, configStore, cfg.ReloadStatusWriter)
	defer stopReloadSignals()
	defer shutdownHTTPServer(httpServer, 5*time.Second)
	defer stopGRPCServer(grpcServer, 5*time.Second)

	go func() {
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			reportServerError(errCh, fmt.Errorf("serve grpc: %w", err))
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func reportServerError(errCh chan<- error, err error) {
	select {
	case errCh <- err:
	default:
	}
}

func shutdownHTTPServer(server *http.Server, timeout time.Duration) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_ = server.Shutdown(ctx)
}

type grpcStopper interface {
	GracefulStop()
	Stop()
}

func stopGRPCServer(server grpcStopper, timeout time.Duration) {
	if server == nil {
		return
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.GracefulStop()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		server.Stop()
		<-done
	}
}

func startReloadSignalHandler(ctx context.Context, store *configStore, out io.Writer) func() {
	if out == nil {
		out = os.Stderr
	}
	hupCh := make(chan os.Signal, 1)
	stopCh := make(chan struct{})
	done := make(chan struct{})
	signal.Notify(hupCh, syscall.SIGHUP)
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case <-stopCh:
				return
			case <-hupCh:
				status := store.Reload(ctx)
				writeReloadStatus(out, status)
			}
		}
	}()

	return func() {
		signal.Stop(hupCh)
		close(stopCh)
		<-done
	}
}

func writeReloadStatus(out io.Writer, status reloadStatus) {
	if status.Success {
		_, _ = fmt.Fprintf(out, "config reload succeeded path=%q generation=%d\n", status.ConfigPath, status.Generation)
		return
	}
	_, _ = fmt.Fprintf(out, "config reload failed path=%q generation=%d error=%s\n", status.ConfigPath, status.Generation, safeErrorMessage(status.Err))
}

func startHTTPDevServer(addr string, authService *auth.Service, auditStore audit.Store, consoleAuthService *consoleauth.Service, accessRequestStore *postgresadapter.AccessRequestStore, dashboardDir string, rateLimiter httpadapter.RateLimiter, secureConsoleCookie bool, readiness readinessChecker, errCh chan<- error) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/readyz", readiness.Handler())
	httpadapter.NewDeviceHandler(authService).WithRateLimiter(rateLimiter).Register(mux)
	httpadapter.NewAuditHandler(auditStore).Register(mux)
	if consoleAuthService != nil && accessRequestStore != nil {
		httpadapter.NewConsoleHandler(consoleAuthService, accessRequestStore).WithAuditStore(auditStore).WithSecureCookies(secureConsoleCookie).Register(mux)
	}
	if dashboardDir != "" {
		httpadapter.NewDashboardHandler(os.DirFS(dashboardDir)).Register(mux)
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           httpadapter.SecurityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen http: %w", err)
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve http: %w", err)
		}
	}()

	return server, nil
}

func consoleCookieSecureFromEnv() (bool, error) {
	value := strings.TrimSpace(os.Getenv("CLOUDIAM_CONSOLE_COOKIE_SECURE"))
	if value == "" {
		return false, nil
	}
	secure, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse CLOUDIAM_CONSOLE_COOKIE_SECURE: %w", err)
	}
	return secure, nil
}

func boolFromEnv(name string) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func validateProductionSecretConfig(cfg Config) error {
	if !isProductionEnv(cfg.DeploymentEnv) {
		return nil
	}
	if !cfg.ConsoleCookieSecure {
		return fmt.Errorf("CLOUDIAM_CONSOLE_COOKIE_SECURE must be true when CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	if strings.TrimSpace(cfg.RedisURL) == "" {
		return fmt.Errorf("CLOUDIAM_REDIS_URL is required when CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	verificationURI := strings.TrimSpace(cfg.AuthVerificationURI)
	if verificationURI == "" {
		return fmt.Errorf("CLOUDIAM_AUTH_VERIFICATION_URI is required when CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	if strings.HasPrefix(verificationURI, "http://") {
		return fmt.Errorf("CLOUDIAM_AUTH_VERIFICATION_URI must use https when CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	if databaseUsesDisabledTLS(cfg.AuthDatabaseURL) && !cfg.AllowInsecureDatabaseInProduction {
		return fmt.Errorf("CLOUDIAM_DATABASE_URL must not use sslmode=disable when CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	if isWeakConsoleJWTSecret(cfg.ConsoleJWTSecret) {
		return fmt.Errorf("CLOUDIAM_CONSOLE_JWT_SECRET is too weak for CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	if strings.TrimSpace(cfg.ConsoleSeedPassword) != "" && !cfg.AllowConsoleSeedInProduction {
		return fmt.Errorf("CLOUDIAM_CONSOLE_SEED_PASSWORD is disabled when CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	if cfg.AllowConsoleSeedInProduction && isWeakConsoleSeedPassword(cfg.ConsoleSeedPassword) {
		return fmt.Errorf("CLOUDIAM_CONSOLE_SEED_PASSWORD is too weak for CLOUDIAM_DEPLOYMENT_ENV=production")
	}
	return nil
}

func isProductionEnv(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "prod" || normalized == "production"
}

func databaseUsesDisabledTLS(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Query().Get("sslmode"), "disable")
}

func isWeakConsoleJWTSecret(secret string) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return false
	}
	if len(secret) < 32 {
		return true
	}
	return isPlaceholderSecret(secret)
}

func isWeakConsoleSeedPassword(password string) bool {
	password = strings.TrimSpace(password)
	if password == "" {
		return true
	}
	if len(password) < 16 {
		return true
	}
	return isPlaceholderSecret(password)
}

func isPlaceholderSecret(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(normalized, "<") || strings.Contains(normalized, ">") {
		return true
	}
	placeholders := []string{
		"local-demo-password",
		"local-demo-secret",
		"test-secret",
		"change-me",
		"changeme",
		"password",
		"secret",
	}
	for _, placeholder := range placeholders {
		if normalized == placeholder {
			return true
		}
	}
	return false
}

func newHTTPRateLimiterFromEnv(cfg Config) (httpadapter.RateLimiter, readinessDependency, func() error, error) {
	fallback := httpadapter.NewLocalRateLimiter(nil)
	if strings.TrimSpace(cfg.RedisURL) == "" {
		return fallback, nil, func() error { return nil }, nil
	}
	limiter, closeLimiter, err := httpadapter.NewRedisRateLimiterFromURL(strings.TrimSpace(cfg.RedisURL), fallback, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	return limiter, limiter, closeLimiter, nil
}

func dashboardDistDir(cfg Config) string {
	if strings.TrimSpace(cfg.DashboardDistDir) != "" {
		return cfg.DashboardDistDir
	}
	if value := strings.TrimSpace(os.Getenv("CLOUDIAM_DASHBOARD_DIST_DIR")); value != "" {
		return value
	}
	path := filepath.Clean("../client/dist")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return ""
}

func seedConsoleUserFromEnv(ctx context.Context, cfg Config, store *postgresadapter.ConsoleUserStore) error {
	username := strings.TrimSpace(cfg.ConsoleSeedUsername)
	if username == "" {
		username = os.Getenv("CLOUDIAM_CONSOLE_SEED_USERNAME")
	}
	password := strings.TrimSpace(cfg.ConsoleSeedPassword)
	if password == "" {
		password = os.Getenv("CLOUDIAM_CONSOLE_SEED_PASSWORD")
	}
	if username == "" && password == "" {
		return nil
	}
	if username == "" || password == "" {
		return fmt.Errorf("both CLOUDIAM_CONSOLE_SEED_USERNAME and CLOUDIAM_CONSOLE_SEED_PASSWORD are required to seed a console user")
	}
	passwordHash, err := consoleauth.HashPassword(password, consoleauth.DefaultPasswordParams)
	if err != nil {
		return err
	}
	return store.SeedUser(ctx, postgresadapter.SeedConsoleUserInput{
		Username:     username,
		PasswordHash: passwordHash,
		Active:       true,
	})
}

func newConsoleAuthServiceFromEnv(cfg Config, store consoleauth.UserStore) (*consoleauth.Service, error) {
	jwtSecret := strings.TrimSpace(cfg.ConsoleJWTSecret)
	if jwtSecret == "" {
		jwtSecret = strings.TrimSpace(os.Getenv("CLOUDIAM_CONSOLE_JWT_SECRET"))
	}

	ttl, err := resolveConsoleJWTTTL(cfg.ConsoleJWTTTL)
	if err != nil {
		return nil, err
	}

	return consoleauth.NewService(consoleauth.ServiceConfig{
		UserStore: store,
		JWTSecret: jwtSecret,
		TokenTTL:  ttl,
	})
}

func resolveConsoleJWTTTL(configured time.Duration) (time.Duration, error) {
	if configured != 0 {
		return configured, nil
	}

	raw := strings.TrimSpace(os.Getenv("CLOUDIAM_CONSOLE_JWT_TTL"))
	if raw == "" {
		return 0, nil
	}

	ttl, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse console jwt ttl: %w", err)
	}

	return ttl, nil
}

func newAccessService(cfg coreconfig.Config, issuer access.CredentialIssuer, opts ...access.Option) *access.Service {
	return newAccessServiceFromStore(newConfigStore("", cfg), issuer, opts...)
}

func newAccessServiceFromStore(store *configStore, issuer access.CredentialIssuer, opts ...access.Option) *access.Service {
	options := []access.Option{
		access.WithRequestEvaluator(snapshotRequestEvaluator{store: store}),
	}
	options = append(options, opts...)
	return access.NewService(
		issuer,
		options...,
	)
}

func newAccessHandlerForServer(service grpcadapter.AccessService, sessionResolver grpcadapter.SessionResolver) (accessv1.AccessServiceServer, error) {
	if sessionResolver == nil {
		return nil, fmt.Errorf("cloudiam session resolver is required for access service")
	}
	return grpcadapter.NewAccessHandlerWithSessions(service, sessionResolver), nil
}

func newAuthServices(ctx context.Context, cfg Config, appCfg coreconfig.Config) (*sql.DB, *auth.Service, *postgresadapter.AuthStore, *postgresadapter.AuditStore, *postgresadapter.AccessRequestStore, *postgresadapter.ConsoleUserStore, func(), error) {
	db, err := postgresadapter.Open(ctx, cfg.AuthDatabaseURL)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	closeAuth := func() {
		_ = db.Close()
	}
	if err := postgresadapter.EnsurePersistenceSchema(ctx, db); err != nil {
		closeAuth()
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	accessRequestStore := postgresadapter.NewAccessRequestStore(db)
	if _, err := cleanupExpiredIdempotencyRecordsOnStartup(ctx, accessRequestStore, time.Now); err != nil {
		closeAuth()
		return nil, nil, nil, nil, nil, nil, nil, err
	}

	workspace := strings.TrimSpace(cfg.AuthWorkspace)
	if workspace == "" {
		workspace = strings.TrimSpace(appCfg.Tenant)
	}
	if workspace == "" {
		workspace = "local"
	}

	endpoint := strings.TrimSpace(cfg.AuthEndpoint)
	if endpoint == "" {
		endpoint = "cloudiam.internal.local"
	}

	verificationURI := strings.TrimSpace(cfg.AuthVerificationURI)
	if verificationURI == "" {
		verificationURI = "http://127.0.0.1:8080/device"
	}

	store := postgresadapter.NewAuthStore(db)
	service, err := auth.NewService(store, auth.ServiceConfig{
		Workspace:       workspace,
		Endpoint:        endpoint,
		VerificationURI: verificationURI,
	})
	if err != nil {
		closeAuth()
		return nil, nil, nil, nil, nil, nil, nil, err
	}
	return db,
		service,
		store,
		postgresadapter.NewAuditStore(db),
		accessRequestStore,
		postgresadapter.NewConsoleUserStore(db),
		closeAuth,
		nil
}

type idempotencyRecordCleaner interface {
	CleanupExpiredIdempotencyRecords(ctx context.Context, cutoff time.Time) (int64, error)
}

func cleanupExpiredIdempotencyRecordsOnStartup(ctx context.Context, cleaner idempotencyRecordCleaner, now func() time.Time) (int64, error) {
	if now == nil {
		now = time.Now
	}
	deleted, err := cleaner.CleanupExpiredIdempotencyRecords(ctx, now().UTC())
	if err != nil {
		return 0, redactedError("cleanup expired idempotency records on startup", err)
	}
	return deleted, nil
}
