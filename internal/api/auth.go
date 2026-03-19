package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/a-tunnels/a-tunnels/internal/config"
)

type AuthMiddleware struct {
	apiKeys []string
	config  *config.AuthConfig
}

func NewAuthMiddleware(cfg config.AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		apiKeys: cfg.APIKeys,
		config:  &cfg,
	}
}

func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token != "" {
			token = token[7:] // Remove "Bearer "
		}

		if token != "" && a.isValidToken(token) {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (a *AuthMiddleware) isValidToken(token string) bool {
	for _, key := range a.apiKeys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

func (a *AuthMiddleware) ValidateToken(token string) bool {
	return a.isValidToken(token)
}
