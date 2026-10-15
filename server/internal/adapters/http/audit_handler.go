package httpadapter

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ffakira/cloudiam/server/internal/core/audit"
)

type AuditStore interface {
	ListRecentEvents(ctx context.Context, limit int) ([]audit.Event, error)
}

type AuditHandler struct {
	store AuditStore
}

func NewAuditHandler(store AuditStore) *AuditHandler {
	return &AuditHandler{store: store}
}

func (h *AuditHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/audit/events", h.listEvents)
}

type auditEventsResponse struct {
	Events []auditEventResponse `json:"events"`
}

type auditEventResponse struct {
	ID                  int64   `json:"id"`
	OccurredAt          string  `json:"occurred_at"`
	EventType           string  `json:"event_type"`
	Principal           string  `json:"principal"`
	Target              string  `json:"target"`
	Decision            string  `json:"decision"`
	Reason              string  `json:"reason"`
	Provider            string  `json:"provider"`
	Account             string  `json:"account"`
	Role                string  `json:"role"`
	AccessRequestID     *int64  `json:"access_request_id,omitempty"`
	OperatorSubject     string  `json:"operator_subject,omitempty"`
	IdempotencyKeyHash  string  `json:"idempotency_key_hash,omitempty"`
	CredentialExpiresAt *string `json:"credential_expires_at,omitempty"`
	CreatedAt           string  `json:"created_at"`
}

func (h *AuditHandler) listEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	listAuditEvents(w, r, h.store)
}

func listAuditEvents(w http.ResponseWriter, r *http.Request, store AuditStore) {
	limit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > 100 {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "limit must be between 1 and 100"})
			return
		}
		limit = parsed
	}

	events, err := store.ListRecentEvents(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "list audit events failed"})
		return
	}

	response := auditEventsResponse{
		Events: make([]auditEventResponse, 0, len(events)),
	}
	for _, event := range events {
		response.Events = append(response.Events, auditEventResponseFromEvent(event))
	}
	writeJSON(w, http.StatusOK, response)
}

func auditEventResponseFromEvent(event audit.Event) auditEventResponse {
	var credentialExpiresAt *string
	if event.CredentialExpiresAt != nil {
		value := event.CredentialExpiresAt.UTC().Format(timeFormatRFC3339Nano)
		credentialExpiresAt = &value
	}

	return auditEventResponse{
		ID:                  event.ID,
		OccurredAt:          event.OccurredAt.UTC().Format(timeFormatRFC3339Nano),
		EventType:           string(event.EventType),
		Principal:           event.Principal,
		Target:              event.Target,
		Decision:            string(event.Decision),
		Reason:              event.Reason,
		Provider:            event.Provider,
		Account:             event.Account,
		Role:                event.Role,
		AccessRequestID:     event.AccessRequestID,
		OperatorSubject:     event.OperatorSubject,
		IdempotencyKeyHash:  event.IdempotencyKeyHash,
		CredentialExpiresAt: credentialExpiresAt,
		CreatedAt:           event.CreatedAt.UTC().Format(timeFormatRFC3339Nano),
	}
}

const timeFormatRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
