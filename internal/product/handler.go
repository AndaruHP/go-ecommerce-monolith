package product

import (
	"encoding/json"
	"errors"
	"go-monolith/internal/auth"
	"net/http"
	"strconv"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	sellerID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	in, err := decodeInput(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	product, err := h.svc.Create(r.Context(), sellerID, in)
	if err != nil {
		writeProductError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := int32(queryInt(r, "limit"))
	offset := int32(queryInt(r, "offset"))
	products, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	product, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeProductError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	in, err := decodeInput(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	product, err := h.svc.Update(r.Context(), userID, id, in)
	if err != nil {
		writeProductError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, product)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing or invalid token"})
		return
	}
	id, err := pathID(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(r.Context(), userId, id); err != nil {
		writeProductError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeInput(r *http.Request) (Input, error) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		PriceCents  int64  `json:"price_cents"`
		Stock       int64  `json:"stock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return Input{}, err
	}
	return Input{
		Name:        body.Name,
		Description: body.Description,
		PriceCents:  body.PriceCents,
		Stock:       body.Stock,
	}, nil
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func queryInt(r *http.Request, key string) int {
	n, _ := strconv.Atoi(r.URL.Query().Get(key))
	return n
}

func writeProductError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required, price_cents must be greater than 0, and stock must be at least 0"})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "product not found"})
	case errors.Is(err, ErrForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "you can only change your own product"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
