package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultDeviceCodeTTL = 10 * time.Minute
	defaultPollInterval  = 5 * time.Second
	defaultSessionTTL    = time.Hour
)

type Clock func() time.Time

type Service struct {
	store           Store
	tokens          TokenGenerator
	now             Clock
	workspace       string
	endpoint        string
	verificationURI string
	deviceCodeTTL   time.Duration
	pollInterval    time.Duration
	sessionTTL      time.Duration
}

type ServiceConfig struct {
	Workspace       string
	Endpoint        string
	VerificationURI string
	DeviceCodeTTL   time.Duration
	PollInterval    time.Duration
	SessionTTL      time.Duration
	Now             Clock
	TokenGenerator  TokenGenerator
}

type StartDeviceLoginResult struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresAt       time.Time
	Interval        time.Duration
}

type SessionResult struct {
	AccessToken string
	Workspace   string
	Principal   string
	Endpoint    string
	ExpiresAt   time.Time
}

func NewService(store Store, cfg ServiceConfig) (*Service, error) {
	if store == nil {
		return nil, errors.New("auth store is required")
	}
	cfg.Workspace = strings.TrimSpace(cfg.Workspace)
	if cfg.Workspace == "" {
		return nil, errors.New("workspace is required")
	}
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	cfg.VerificationURI = strings.TrimSpace(cfg.VerificationURI)
	if cfg.VerificationURI == "" {
		return nil, errors.New("verification uri is required")
	}
	if cfg.DeviceCodeTTL <= 0 {
		cfg.DeviceCodeTTL = defaultDeviceCodeTTL
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = defaultSessionTTL
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.TokenGenerator == nil {
		cfg.TokenGenerator = SecureTokenGenerator{}
	}

	return &Service{
		store:           store,
		tokens:          cfg.TokenGenerator,
		now:             cfg.Now,
		workspace:       cfg.Workspace,
		endpoint:        cfg.Endpoint,
		verificationURI: cfg.VerificationURI,
		deviceCodeTTL:   cfg.DeviceCodeTTL,
		pollInterval:    cfg.PollInterval,
		sessionTTL:      cfg.SessionTTL,
	}, nil
}

func (s *Service) StartDeviceLogin(ctx context.Context) (StartDeviceLoginResult, error) {
	if err := contextError(ctx); err != nil {
		return StartDeviceLoginResult{}, err
	}
	deviceCode, err := s.tokens.DeviceCode()
	if err != nil {
		return StartDeviceLoginResult{}, fmt.Errorf("generate device code: %w", err)
	}
	userCode, err := s.tokens.UserCode()
	if err != nil {
		return StartDeviceLoginResult{}, fmt.Errorf("generate user code: %w", err)
	}

	now := s.now()
	expiresAt := now.Add(s.deviceCodeTTL)
	normalizedUserCode := NormalizeUserCode(userCode)

	_, err = s.store.CreateDeviceAuthorization(ctx, CreateDeviceAuthorizationInput{
		DeviceCodeHash:  HashToken(deviceCode),
		UserCodeHash:    HashToken(normalizedUserCode),
		UserCodeDisplay: normalizedUserCode,
		Workspace:       s.workspace,
		Endpoint:        s.endpoint,
		Interval:        s.pollInterval,
		ExpiresAt:       expiresAt,
	})
	if err != nil {
		return StartDeviceLoginResult{}, fmt.Errorf("create device authorization: %w", err)
	}

	return StartDeviceLoginResult{
		DeviceCode:      deviceCode,
		UserCode:        normalizedUserCode,
		VerificationURI: s.verificationURI,
		ExpiresAt:       expiresAt,
		Interval:        s.pollInterval,
	}, nil
}

func (s *Service) ApproveDeviceLogin(ctx context.Context, userCode, principal string) error {
	userCode = NormalizeUserCode(userCode)
	if userCode == "" {
		return errors.New("user code is required")
	}
	principal = strings.TrimSpace(principal)
	if principal == "" {
		return errors.New("principal is required")
	}
	if err := contextError(ctx); err != nil {
		return err
	}

	_, err := s.store.ApproveDeviceAuthorization(ctx, ApproveDeviceAuthorizationInput{
		UserCodeHash: HashToken(userCode),
		Principal:    principal,
		ApprovedAt:   s.now(),
	})
	if err != nil {
		return fmt.Errorf("approve device authorization: %w", err)
	}
	return nil
}

func (s *Service) ExchangeDeviceCode(ctx context.Context, deviceCode string) (SessionResult, error) {
	deviceCode = strings.TrimSpace(deviceCode)
	if deviceCode == "" {
		return SessionResult{}, errors.New("device code is required")
	}
	if err := contextError(ctx); err != nil {
		return SessionResult{}, err
	}

	authorization, ok, err := s.store.FindDeviceAuthorizationByDeviceCodeHash(ctx, HashToken(deviceCode))
	if err != nil {
		return SessionResult{}, fmt.Errorf("find device authorization: %w", err)
	}
	if !ok {
		return SessionResult{}, ErrNotFound
	}

	now := s.now()
	switch authorization.Status {
	case DeviceAuthorizationStatusPending:
		if !authorization.ExpiresAt.After(now) {
			_, expireErr := s.store.ExpireDeviceAuthorization(ctx, ExpireDeviceAuthorizationInput{
				DeviceCodeHash: HashToken(deviceCode),
				ExpiredAt:      now,
			})
			if expireErr != nil && !errors.Is(expireErr, ErrNotFound) {
				return SessionResult{}, fmt.Errorf("expire device authorization: %w", expireErr)
			}
			return SessionResult{}, ErrExpired
		}
		return SessionResult{}, ErrAuthorizationPending
	case DeviceAuthorizationStatusDenied:
		return SessionResult{}, ErrAuthorizationDenied
	case DeviceAuthorizationStatusExpired:
		return SessionResult{}, ErrExpired
	case DeviceAuthorizationStatusConsumed:
		return SessionResult{}, ErrInvalidState
	case DeviceAuthorizationStatusApproved:
	default:
		return SessionResult{}, ErrInvalidState
	}
	if err := contextError(ctx); err != nil {
		return SessionResult{}, err
	}

	sessionToken, err := s.tokens.SessionToken()
	if err != nil {
		return SessionResult{}, fmt.Errorf("generate session token: %w", err)
	}
	session, err := s.store.ExchangeApprovedDeviceAuthorization(ctx, ExchangeApprovedDeviceAuthorizationInput{
		DeviceCodeHash:   HashToken(deviceCode),
		SessionTokenHash: HashToken(sessionToken),
		SessionExpiresAt: now.Add(s.sessionTTL),
		ConsumedAt:       now,
	})
	if err != nil {
		return SessionResult{}, fmt.Errorf("exchange device authorization: %w", err)
	}

	return SessionResult{
		AccessToken: sessionToken,
		Workspace:   session.Workspace,
		Principal:   session.Principal,
		Endpoint:    session.Endpoint,
		ExpiresAt:   session.ExpiresAt,
	}, nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
