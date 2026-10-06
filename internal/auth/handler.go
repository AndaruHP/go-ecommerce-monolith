package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	user, err := h.svc.Register(r.Context(), body.Name, body.Email, body.Password)
	if errors.Is(err, ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name, email, and a password of at least 8 characters are required",
		})
		return
	}
	if errors.Is(err, ErrEmailTaken) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	token, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if errors.Is(err, ErrUnauthorized) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"access_token": token})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := UserID(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"user_id": id})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
