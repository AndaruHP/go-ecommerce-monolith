package order

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-monolith/internal/auth"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	order, err := h.svc.Checkout(r.Context(), userID)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	orders, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	order, err := h.svc.Get(r.Context(), userID, id)
	if err != nil {
		writeOrderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func writeOrderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmpty):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cart is empty"})
	case errors.Is(err, ErrInsufficient):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "not enough stock"})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order not found"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
