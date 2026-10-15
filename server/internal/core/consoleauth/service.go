package consoleauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	jwtIssuer = "cloudiam"

	ConsoleAudience     = "cloudiam-console"
	ConsoleOperatorRole = "console_operator"
)

type ServiceConfig struct {
	UserStore UserStore
	JWTSecret string
	TokenTTL  time.Duration
	Now       func() time.Time
}

type Service struct {
	userStore UserStore
	jwtSecret []byte
	tokenTTL  time.Duration
	now       func() time.Time
}

type Session struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

type Principal struct {
	Subject   string
	Role      string
	ExpiresAt time.Time
}

type jwtClaims struct {
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	Subject   string `json:"sub"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	ID        string `json:"jti"`
}

func NewService(cfg ServiceConfig) (*Service, error) {
	jwtSecret := strings.TrimSpace(cfg.JWTSecret)
	if cfg.UserStore == nil || jwtSecret == "" {
		return nil, ErrDisabled
	}
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = 8 * time.Hour
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		userStore: cfg.UserStore,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  cfg.TokenTTL,
		now:       now,
	}, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (Session, error) {
	select {
	case <-ctx.Done():
		return Session{}, ctx.Err()
	default:
	}

	username = strings.TrimSpace(username)
	if username == "" {
		return Session{}, ErrInvalidCredentials
	}
	user, ok, err := s.userStore.GetUser(ctx, username)
	if err != nil {
		return Session{}, fmt.Errorf("get console user: %w", err)
	}
	if !ok || !user.Active {
		return Session{}, ErrInvalidCredentials
	}
	matched, err := VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return Session{}, fmt.Errorf("verify console password hash: %w", err)
	}
	if !matched {
		return Session{}, ErrInvalidCredentials
	}

	now := s.now().UTC()
	expiresAt := now.Add(s.tokenTTL)
	tokenID, err := randomTokenID()
	if err != nil {
		return Session{}, fmt.Errorf("generate admin token id: %w", err)
	}
	return s.signSession(user.Username, now, expiresAt, tokenID)
}

func (s *Service) Refresh(ctx context.Context, token string) (Session, error) {
	select {
	case <-ctx.Done():
		return Session{}, ctx.Err()
	default:
	}

	claims, err := s.verifyClaims(token)
	if err != nil {
		return Session{}, err
	}

	now := s.now().UTC()
	expiresAt := now.Add(s.tokenTTL)
	tokenID, err := randomTokenID()
	if err != nil {
		return Session{}, fmt.Errorf("generate admin token id: %w", err)
	}
	return s.signSession(claims.Subject, now, expiresAt, tokenID)
}

func (s *Service) signSession(subject string, now, expiresAt time.Time, tokenID string) (Session, error) {
	token, err := s.signClaims(jwtClaims{
		Issuer:    jwtIssuer,
		Audience:  ConsoleAudience,
		Subject:   subject,
		Role:      ConsoleOperatorRole,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
		ID:        tokenID,
	})
	if err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *Service) VerifyToken(token string) (Principal, error) {
	claims, err := s.verifyClaims(token)
	if err != nil {
		return Principal{}, err
	}
	return Principal{
		Subject:   claims.Subject,
		Role:      claims.Role,
		ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(),
	}, nil
}

func (s *Service) signClaims(claims jwtClaims) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := encodedHeader + "." + encodedClaims
	signature := signHS256([]byte(signingInput), s.jwtSecret)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *Service) verifyClaims(token string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, ErrInvalidToken
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtClaims{}, ErrInvalidToken
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return jwtClaims{}, ErrInvalidToken
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return jwtClaims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := signHS256([]byte(signingInput), s.jwtSecret)
	actualSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtClaims{}, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare(actualSignature, expectedSignature) != 1 {
		return jwtClaims{}, ErrInvalidToken
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, ErrInvalidToken
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return jwtClaims{}, ErrInvalidToken
	}
	if claims.Issuer != jwtIssuer ||
		claims.Audience != ConsoleAudience ||
		claims.Subject == "" ||
		claims.Role != ConsoleOperatorRole ||
		claims.ExpiresAt <= s.now().UTC().Unix() {
		return jwtClaims{}, ErrInvalidToken
	}
	return claims, nil
}

func signHS256(input, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(input)
	return mac.Sum(nil)
}

func randomTokenID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}
