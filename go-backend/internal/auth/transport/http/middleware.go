package http

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const UserIDKey ctxKey = "userID"

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.respondWithError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			h.respondWithError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		userID, err := h.service.ValidateToken(r.Context(), parts[1])
		if err != nil {
			h.log.Warn("invalid token attempt", "error", err)
			h.respondWithError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
