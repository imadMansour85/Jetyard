package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type contextKey struct{}

var claimsKey contextKey

func ClaimsFrom(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	return claims, ok
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Require rejects any request without a valid bearer token. SSE connections
// carry the token as a query parameter because EventSource cannot set headers.
func (s *Signer) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
			token = strings.TrimPrefix(header, "Bearer ")
		} else if q := r.URL.Query().Get("access_token"); q != "" {
			token = q
		}

		if token == "" {
			unauthorized(w, "missing bearer token")
			return
		}

		claims, err := s.Verify(token)
		if err != nil {
			unauthorized(w, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin gates mutations behind the admin role.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFrom(r.Context())
		if !ok {
			unauthorized(w, "unauthorized")
			return
		}
		if claims.Role != "admin" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "admin role required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
