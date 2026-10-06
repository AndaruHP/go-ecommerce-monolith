package cart

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

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	cart, err := h.svc.Get(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) SetItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	var body struct {
		ProductID int64 `json:"product_id"`
		Quantity  int64 `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	cart, err := h.svc.SetItem(r.Context(), userID, body.ProductID, body.Quantity)
	if err != nil {
		writeCartError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	productID, err := strconv.ParseInt(r.PathValue("product_id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	cart, err := h.svc.RemoveItem(r.Context(), userID, productID)
	if err != nil {
		writeCartError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func writeCartError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_id must be greater than 0 and quantity must be at least 0"})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
