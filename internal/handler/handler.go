package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"subscription-service/internal/model"
	"subscription-service/internal/service"
)

// SubscriptionHandler manages HTTP communication for subscription-related endpoints.
type SubscriptionHandler struct {
	service service.SubscriptionService
}

// NewSubscriptionHandler initializes a new handler with the provided subscription service.
func NewSubscriptionHandler(s service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: s}
}

// Create decodes the request body, validates it, and triggers the creation of a new subscription.
// It returns 201 Created on success or 400 Bad Request on validation failure.
func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	log.Printf("INFO: handler create subscription")

	var req model.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := model.Validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sub, err := model.ToDomain(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format")
		return
	}

	if err := h.service.Create(r.Context(), sub); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, model.ToResponse(sub))
}

// Get extracts the subscription ID from the URL and returns the matching record.
// Returns 404 Not Found if the subscription does not exist.
func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	sub, err := h.service.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.ToResponse(sub))
}

// Update replaces an existing subscription's data after validating the input and the ID.
func (h *SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req model.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := model.Validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sub, err := model.ToDomain(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format")
		return
	}
	sub.ID = id

	if err := h.service.Update(r.Context(), sub); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, model.ToResponse(sub))
}

// Delete removes a subscription based on the ID provided in the URL path.
// Returns 204 No Content upon successful deletion.
func (h *SubscriptionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List retrieves a filtered and paginated list of subscriptions based on query parameters.
// Parameters include user_id, service_name, limit, and offset.
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	var (
		userID      *uuid.UUID
		serviceName *string
	)

	if uid := r.URL.Query().Get("user_id"); uid != "" {
		parsed, err := uuid.Parse(uid)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		userID = &parsed
	}

	if sn := r.URL.Query().Get("service_name"); sn != "" {
		serviceName = &sn
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	subs, err := h.service.List(
		context.Background(),
		userID,
		serviceName,
		limit,
		offset,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]model.SubscriptionResponse, 0, len(subs))
	for _, s := range subs {
		resp = append(resp, model.ToResponse(s))
	}

	writeJSON(w, http.StatusOK, resp)
}

// Summary calculates the total cost of subscriptions for a user within a specific time range.
// Requires 'from' and 'to' query parameters in "MM-YYYY" format.
func (h *SubscriptionHandler) Summary(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		writeError(w, http.StatusBadRequest, "from and to are required")
		return
	}

	from, err := time.Parse("01-2006", fromStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from date")
		return
	}

	to, err := time.Parse("01-2006", toStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to date")
		return
	}

	var userID *uuid.UUID
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		parsed, err := uuid.Parse(uid)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		userID = &parsed
	}

	var serviceName *string
	if sn := r.URL.Query().Get("service_name"); sn != "" {
		serviceName = &sn
	}

	total, err := h.service.Aggregate(
		r.Context(),
		userID,
		serviceName,
		from,
		to,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"total": total,
	})
}
