package httpadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/access"
	"github.com/ffakira/cloudiam/server/internal/core/approval"
	"github.com/ffakira/cloudiam/server/internal/core/audit"
	"github.com/ffakira/cloudiam/server/internal/core/consoleauth"
)

type ConsoleAuthService interface {
	Login(ctx context.Context, username, password string) (consoleauth.Session, error)
	Refresh(ctx context.Context, token string) (consoleauth.Session, error)
	VerifyToken(token string) (consoleauth.Principal, error)
}

type ConsoleHandler struct {
	auth         ConsoleAuthService
	store        approval.Store
	auditStore   audit.Store
	now          func() time.Time
	secureCookie bool
}

func NewConsoleHandler(auth ConsoleAuthService, store approval.Store) *ConsoleHandler {
	return &ConsoleHandler{
		auth:  auth,
		store: store,
		now:   time.Now,
	}
}

func (h *ConsoleHandler) WithAuditStore(store audit.Store) *ConsoleHandler {
	h.auditStore = store
	return h
}

func (h *ConsoleHandler) WithSecureCookies(secure bool) *ConsoleHandler {
	h.secureCookie = secure
	return h
}

func (h *ConsoleHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/admin/login", h.login)
	mux.HandleFunc("/api/admin/logout", h.logout)
	mux.HandleFunc("/api/admin/me", h.me)
	mux.HandleFunc("/api/admin/refresh", h.refresh)
	mux.HandleFunc("/api/admin/access-requests", h.listAccessRequests)
	mux.HandleFunc("/api/admin/access-requests/", h.mutateAccessRequest)
	mux.HandleFunc("/api/admin/audit/events", h.listAuditEvents)
}

type consoleLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type consoleLoginResponse struct {
	Authenticated bool      `json:"authenticated"`
	Subject       string    `json:"subject"`
	Role          string    `json:"role"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type consoleSessionResponse struct {
	Authenticated bool      `json:"authenticated"`
	Subject       string    `json:"subject"`
	Role          string    `json:"role"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type accessRequestResponse struct {
	ID                       int64      `json:"id"`
	Principal                string     `json:"principal"`
	Target                   string     `json:"target"`
	Provider                 string     `json:"provider"`
	Account                  string     `json:"account"`
	Role                     string     `json:"role"`
	RoleARN                  string     `json:"role_arn"`
	RequestedDurationSeconds int64      `json:"requested_duration_seconds"`
	PolicyReason             string     `json:"policy_reason"`
	Status                   string     `json:"status"`
	ExpiresAt                time.Time  `json:"expires_at"`
	ApprovedBy               string     `json:"approved_by,omitempty"`
	DeniedBy                 string     `json:"denied_by,omitempty"`
	ApprovedAt               *time.Time `json:"approved_at,omitempty"`
	DeniedAt                 *time.Time `json:"denied_at,omitempty"`
	IssuedAt                 *time.Time `json:"issued_at,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type accessRequestsResponse struct {
	AccessRequests []accessRequestResponse `json:"access_requests"`
}

func (h *ConsoleHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	var request consoleLoginRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}

	session, err := h.auth.Login(r.Context(), strings.TrimSpace(request.Username), request.Password)
	if err != nil {
		if errors.Is(err, consoleauth.ErrInvalidCredentials) || errors.Is(err, consoleauth.ErrDisabled) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid console credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "console login failed"})
		return
	}
	h.setConsoleSessionCookie(w, session)
	writeJSON(w, http.StatusOK, consoleLoginResponse{
		Authenticated: true,
		Subject:       strings.TrimSpace(request.Username),
		Role:          consoleauth.ConsoleOperatorRole,
		ExpiresAt:     session.ExpiresAt,
	})
}

func (h *ConsoleHandler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if rejectRequestBody(w, r) {
		return
	}
	h.clearConsoleSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed_out"})
}

func (h *ConsoleHandler) me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	principal, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, consoleSessionResponse{
		Authenticated: true,
		Subject:       principal.Subject,
		Role:          principal.Role,
		ExpiresAt:     principal.ExpiresAt,
	})
}

func (h *ConsoleHandler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if rejectRequestBody(w, r) {
		return
	}
	token, ok := consoleTokenFromCookie(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "console session cookie is required"})
		return
	}
	session, err := h.auth.Refresh(r.Context(), token)
	if err != nil {
		h.clearConsoleSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "console session cookie is invalid"})
		return
	}
	principal, err := h.auth.VerifyToken(session.AccessToken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "console refresh failed"})
		return
	}
	if principal.Role != consoleauth.ConsoleOperatorRole {
		h.clearConsoleSessionCookie(w)
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "console operator role is required"})
		return
	}
	h.setConsoleSessionCookie(w, session)
	writeJSON(w, http.StatusOK, consoleSessionResponse{
		Authenticated: true,
		Subject:       principal.Subject,
		Role:          principal.Role,
		ExpiresAt:     principal.ExpiresAt,
	})
}

func (h *ConsoleHandler) listAccessRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if _, ok := h.authenticate(w, r); !ok {
		return
	}

	status, ok := parseAccessRequestStatus(r.URL.Query().Get("status"))
	if !ok {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "status is invalid"})
		return
	}
	requests, err := h.store.ListAccessRequests(r.Context(), status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list access requests failed"})
		return
	}

	response := accessRequestsResponse{AccessRequests: make([]accessRequestResponse, 0, len(requests))}
	for _, request := range requests {
		response.AccessRequests = append(response.AccessRequests, newAccessRequestResponse(request))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *ConsoleHandler) mutateAccessRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if rejectRequestBody(w, r) {
		return
	}
	principal, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	id, action, ok := parseAccessRequestMutationPath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "console route not found"})
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if !isUUIDv4(idempotencyKey) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "valid Idempotency-Key is required"})
		return
	}

	result, err := h.store.MutateAccessRequest(r.Context(), approval.MutationInput{
		AccessRequestID: id,
		ApproverSubject: principal.Subject,
		IdempotencyKey:  idempotencyKey,
		Fingerprint:     approval.Fingerprint(action, id, principal.Subject, idempotencyKey),
		Action:          action,
		Now:             h.now().UTC(),
	})
	if err != nil {
		if !result.Replayed {
			if auditErr := h.recordApprovalAuditEvent(r.Context(), action, principal.Subject, idempotencyKey, result.AccessRequest); auditErr != nil {
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "record approval audit event failed"})
				return
			}
		}
		writeConsoleApprovalError(w, err)
		return
	}
	if !result.Replayed {
		if err := h.recordApprovalAuditEvent(r.Context(), action, principal.Subject, idempotencyKey, result.AccessRequest); err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "record approval audit event failed"})
			return
		}
	}
	w.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replayed))
	writeJSON(w, http.StatusOK, newAccessRequestResponse(result.AccessRequest))
}

func (h *ConsoleHandler) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if _, ok := h.authenticate(w, r); !ok {
		return
	}
	if h.auditStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "audit store is unavailable"})
		return
	}
	listAuditEvents(w, r, h.auditStore)
}

func (h *ConsoleHandler) authenticate(w http.ResponseWriter, r *http.Request) (consoleauth.Principal, bool) {
	if token, ok := consoleTokenFromCookie(r); ok {
		principal, err := h.auth.VerifyToken(token)
		if err != nil {
			h.clearConsoleSessionCookie(w)
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "console session cookie is invalid"})
			return consoleauth.Principal{}, false
		}
		if principal.Role != consoleauth.ConsoleOperatorRole {
			h.clearConsoleSessionCookie(w)
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "console operator role is required"})
			return consoleauth.Principal{}, false
		}
		return principal, true
	}

	writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "console session cookie is required"})
	return consoleauth.Principal{}, false
}

func (h *ConsoleHandler) recordApprovalAuditEvent(ctx context.Context, action approval.Action, operatorSubject, idempotencyKey string, request access.AccessRequest) error {
	if h.auditStore == nil || request.ID == 0 {
		return nil
	}

	eventType := audit.EventTypeApprovalApproved
	decision := audit.DecisionAllowed
	reason := request.PolicyReason
	switch {
	case request.Status == access.AccessRequestStatusExpired:
		eventType = audit.EventTypeApprovalExpired
		decision = audit.DecisionDenied
		reason = "access request expired"
	case action == approval.ActionDeny:
		eventType = audit.EventTypeApprovalDenied
		decision = audit.DecisionDenied
	}

	if err := h.auditStore.RecordEvent(ctx, audit.RecordEventInput{
		OccurredAt:         h.now().UTC(),
		EventType:          eventType,
		Principal:          request.Principal,
		Target:             request.Target,
		Decision:           decision,
		Reason:             reason,
		Provider:           request.Provider,
		Account:            request.Account,
		Role:               request.Role,
		AccessRequestID:    &request.ID,
		OperatorSubject:    operatorSubject,
		IdempotencyKeyHash: hashID(idempotencyKey),
	}); err != nil {
		return err
	}
	return nil
}

func hashID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

const consoleSessionCookieName = "cloudiam_console_session"

func (h *ConsoleHandler) setConsoleSessionCookie(w http.ResponseWriter, session consoleauth.Session) {
	maxAge := int(session.ExpiresAt.Sub(h.now()).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	// #nosec G124 -- Secure is configurable so local HTTP development works; production sets CLOUDIAM_CONSOLE_COOKIE_SECURE=true.
	http.SetCookie(w, &http.Cookie{
		Name:     consoleSessionCookieName,
		Value:    session.AccessToken,
		Path:     "/api/admin",
		Expires:  session.ExpiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *ConsoleHandler) clearConsoleSessionCookie(w http.ResponseWriter) {
	// #nosec G124 -- Secure is configurable so local HTTP development works; production sets CLOUDIAM_CONSOLE_COOKIE_SECURE=true.
	http.SetCookie(w, &http.Cookie{
		Name:     consoleSessionCookieName,
		Value:    "",
		Path:     "/api/admin",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func consoleTokenFromCookie(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(consoleSessionCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", false
	}
	return cookie.Value, true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra struct{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("json body must contain exactly one object")
	}
	return nil
}

func rejectRequestBody(w http.ResponseWriter, r *http.Request) bool {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 {
		return false
	}
	writeJSON(w, http.StatusBadRequest, errorResponse{Error: "request body is not allowed"})
	return true
}

func parseAccessRequestStatus(raw string) (access.AccessRequestStatus, bool) {
	switch strings.TrimSpace(raw) {
	case "", "pending", string(access.AccessRequestStatusPendingApproval):
		return access.AccessRequestStatusPendingApproval, true
	case string(access.AccessRequestStatusApproved):
		return access.AccessRequestStatusApproved, true
	case string(access.AccessRequestStatusDenied):
		return access.AccessRequestStatusDenied, true
	case string(access.AccessRequestStatusExpired):
		return access.AccessRequestStatusExpired, true
	case string(access.AccessRequestStatusIssued):
		return access.AccessRequestStatusIssued, true
	default:
		return "", false
	}
}

func parseAccessRequestMutationPath(path string) (int64, approval.Action, bool) {
	rest := strings.TrimPrefix(path, "/api/admin/access-requests/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	switch parts[1] {
	case string(approval.ActionApprove):
		return id, approval.ActionApprove, true
	case string(approval.ActionDeny):
		return id, approval.ActionDeny, true
	default:
		return 0, "", false
	}
}

func isUUIDv4(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, c := range value {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		case 14:
			if c != '4' {
				return false
			}
		case 19:
			if !strings.ContainsRune("89abAB", c) {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
				return false
			}
		}
	}
	return true
}

func writeConsoleApprovalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, approval.ErrIdempotencyConflict):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "idempotency key was already used for a different request"})
	case errors.Is(err, approval.ErrAccessRequestNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "access request not found"})
	case errors.Is(err, approval.ErrAccessRequestExpired):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "access request expired"})
	case errors.Is(err, approval.ErrAccessRequestInvalidState):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "access request cannot be changed"})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "access request approval failed"})
	}
}

func newAccessRequestResponse(request access.AccessRequest) accessRequestResponse {
	return accessRequestResponse{
		ID:                       request.ID,
		Principal:                request.Principal,
		Target:                   request.Target,
		Provider:                 request.Provider,
		Account:                  request.Account,
		Role:                     request.Role,
		RoleARN:                  request.RoleARN,
		RequestedDurationSeconds: int64(request.RequestedDuration.Seconds()),
		PolicyReason:             request.PolicyReason,
		Status:                   string(request.Status),
		ExpiresAt:                request.ExpiresAt,
		ApprovedBy:               request.ApprovedBy,
		DeniedBy:                 request.DeniedBy,
		ApprovedAt:               request.ApprovedAt,
		DeniedAt:                 request.DeniedAt,
		IssuedAt:                 request.IssuedAt,
		CreatedAt:                request.CreatedAt,
		UpdatedAt:                request.UpdatedAt,
	}
}
