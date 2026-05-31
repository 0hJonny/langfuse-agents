package http

import (
	"context"
	"net/http"
)

func (h *ChatHandler) FromGatewayHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			h.respondWithError(w, http.StatusUnauthorized, "missing X-User-ID header")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
