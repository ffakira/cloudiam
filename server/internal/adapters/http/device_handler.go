package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ffakira/cloudiam/server/internal/core/auth"
)

type AuthService interface {
	ApproveDeviceLogin(ctx context.Context, userCode, principal string) error
}

type DeviceHandler struct {
	service     AuthService
	rateLimiter RateLimiter
}

const (
	maxDeviceApprovalUserCodeLength  = 64
	maxDeviceApprovalPrincipalLength = 256
	deviceApprovalRateLimit          = 10
	deviceApprovalRateLimitWindow    = time.Minute
)

func NewDeviceHandler(service AuthService) *DeviceHandler {
	return &DeviceHandler{service: service}
}

func (h *DeviceHandler) WithRateLimiter(limiter RateLimiter) *DeviceHandler {
	h.rateLimiter = limiter
	return h
}

func (h *DeviceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/device/approve", h.approve)
	mux.HandleFunc("/device/approve", h.deprecatedApprove)
}

type approveRequest struct {
	UserCode  string `json:"user_code"`
	Principal string `json:"principal"`
}

type approveResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *DeviceHandler) approve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if !h.allowDeviceApproval(r) {
		writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: "rate limit exceeded"})
		return
	}

	var request approveRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}

	request.UserCode = strings.TrimSpace(request.UserCode)
	request.Principal = strings.TrimSpace(request.Principal)
	if request.UserCode == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_code is required"})
		return
	}
	if len(request.UserCode) > maxDeviceApprovalUserCodeLength {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_code is invalid"})
		return
	}
	if request.Principal == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "principal is required"})
		return
	}
	if len(request.Principal) > maxDeviceApprovalPrincipalLength {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "principal is invalid"})
		return
	}

	if err := h.service.ApproveDeviceLogin(r.Context(), request.UserCode, request.Principal); err != nil {
		writeApprovalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, approveResponse{Status: "approved"})
}

func (h *DeviceHandler) allowDeviceApproval(r *http.Request) bool {
	if h.rateLimiter == nil {
		return true
	}
	return h.rateLimiter.Allow(r.Context(), "device_approve:"+remoteAddrHost(r.RemoteAddr), deviceApprovalRateLimit, deviceApprovalRateLimitWindow)
}

func remoteAddrHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(remoteAddr)
}

func (h *DeviceHandler) deprecatedApprove(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, errorResponse{Error: "manual device approval endpoint is deprecated; open /device in a browser"})
}

func writeApprovalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "device authorization not found"})
	case errors.Is(err, auth.ErrExpired):
		writeJSON(w, http.StatusGone, errorResponse{Error: "device authorization expired"})
	case errors.Is(err, auth.ErrInvalidState):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "device authorization cannot be approved"})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "approve device login failed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
