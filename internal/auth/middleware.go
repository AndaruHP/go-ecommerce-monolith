package auth

import (
	"context"
	"net/http"
	"strings"
)

type userIDKey struct {
}

func UserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey{}).(int64)
	return id, ok
}

func (h *Handler) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := bearerToken(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
			return
		}

		userID, err := h.svc.ParseToken(raw)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid token"})
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", ErrUnauthorized
	}

	return strings.TrimSpace(header[len(prefix):]), nil
}
