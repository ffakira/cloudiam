package cliadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Session struct {
	Version     string    `json:"version"`
	Workspace   string    `json:"workspace"`
	Principal   string    `json:"principal"`
	Endpoint    string    `json:"endpoint"`
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type SessionStore struct {
	path string
	now  func() time.Time
}

func NewSessionStore(path string) *SessionStore {
	return &SessionStore{
		path: path,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func DefaultSessionPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "cloudiam", "session.json"), nil
}

func (s *SessionStore) Save(session Session) error {
	if err := validateSession(session); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	// #nosec G117 -- CLI intentionally persists the CloudIAM session token in a user-owned 0600 session file.
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}
	data = append(data, '\n')

	file, err := os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open session file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("write session file: %w; close session file: %v", err, closeErr)
		}
		return fmt.Errorf("write session file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("chmod session file: %w; close session file: %v", err, closeErr)
		}
		return fmt.Errorf("chmod session file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close session file: %w", err)
	}
	return nil
}

func (s *SessionStore) Load() (Session, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("read session file: %w", err)
	}
	if containsForbiddenSessionField(data) {
		return Session{}, errors.New("session file contains forbidden credential field")
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("decode session file: %w", err)
	}
	if err := validateSession(session); err != nil {
		return Session{}, err
	}
	if !session.ExpiresAt.After(s.now()) {
		return Session{}, ErrSessionExpired
	}
	return session, nil
}

var (
	ErrSessionNotFound = errors.New("cloudiam session not found")
	ErrSessionExpired  = errors.New("cloudiam session expired")
)

func validateSession(session Session) error {
	if strings.TrimSpace(session.Version) == "" {
		return errors.New("session version is required")
	}
	if strings.TrimSpace(session.Workspace) == "" {
		return errors.New("session workspace is required")
	}
	if strings.TrimSpace(session.Principal) == "" {
		return errors.New("session principal is required")
	}
	if strings.TrimSpace(session.Endpoint) == "" {
		return errors.New("session endpoint is required")
	}
	if strings.TrimSpace(session.AccessToken) == "" {
		return errors.New("session access token is required")
	}
	if session.ExpiresAt.IsZero() {
		return errors.New("session expiry is required")
	}
	return nil
}

func containsForbiddenSessionField(data []byte) bool {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	for key := range raw {
		normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(key)), "-", "_")
		switch normalized {
		case "aws_access_key_id", "aws_secret_access_key", "aws_session_token", "access_key_id", "secret_access_key", "session_token":
			return true
		}
	}
	return false
}
