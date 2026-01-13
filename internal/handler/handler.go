package handler

import (

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

// Create godoc
// @Summary Create subscription
// @Description Create a new subscription
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param subscription body model.CreateSubscriptionRequest true "Subscription data"
// @Success 201 {object} model.SubscriptionResponse
// @Failure 400 {object} handler.errorResponse
// @Failure 500 {object} handler.errorResponse
// @Router /subscriptions [post]
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

// Get godoc
// @Summary Get subscription
// @Description Get subscription by ID
// @Tags subscriptions
// @Produce json
// @Param id path string true "Subscription ID" format(uuid) example("550e8400-e29b-41d4-a716-446655440000")
// @Success 200 {object} model.SubscriptionResponse
// @Failure 400 {object} handler.errorResponse
// @Failure 404 {object} handler.errorResponse
// @Router /subscriptions/{id} [get]
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

// Update godoc
// @Summary Update subscription
// @Description Update subscription by ID
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param id path string true "Subscription ID" format(uuid) example("550e8400-e29b-41d4-a716-446655440000")
// @Param subscription body model.CreateSubscriptionRequest true "Updated subscription data"
// @Success 200 {object} model.SubscriptionResponse
// @Failure 400 {object} handler.errorResponse
// @Failure 500 {object} handler.errorResponse
// @Router /subscriptions/{id} [put]
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

// Delete godoc
// @Summary Delete subscription
// @Description Delete subscription by ID
// @Tags subscriptions
// @Param id path string true "Subscription ID" format(uuid) example("550e8400-e29b-41d4-a716-446655440000")
// @Success 204
// @Failure 400 {object} handler.errorResponse
// @Failure 500 {object} handler.errorResponse
// @Router /subscriptions/{id} [delete]
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

// List godoc
// @Summary List subscriptions
// @Description List subscriptions with optional filters
// @Tags subscriptions
// @Produce json
// @Param user_id query string false "User ID" format(uuid)
// @Param service_name query string false "Service name"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} model.SubscriptionResponse
// @Failure 400 {object} handler.errorResponse
// @Failure 500 {object} handler.errorResponse
// @Router /subscriptions [get]
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
		r.Context(),
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

// @Summary Aggregate subscriptions cost
// @Description Calculate total cost of subscriptions for a period
// @Tags subscriptions
// @Produce json
// @Param from query string true "Start period" example("01-2026")
// @Param to query string true "End period" example("12-2026")
// @Param user_id query string false "User ID" format(uuid)
// @Param service_name query string false "Service name" example("Netflix")
// @Success 200 {object} map[string]int "Example: {"total": 1500}"
// @Failure 400 {object} handler.errorResponse
// @Router /subscriptions/summary [get]
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
