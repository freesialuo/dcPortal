package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// AdminAuth returns middleware that checks for a valid admin token.
// It accepts "Authorization: Bearer <token>" header or "admin_token" cookie.
func AdminAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if checkToken(r, token) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}

func checkToken(r *http.Request, expected string) bool {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		provided := strings.TrimPrefix(auth, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1 {
			return true
		}
	}

	// Check cookie
	cookie, err := r.Cookie("admin_token")
	if err == nil && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(expected)) == 1 {
		return true
	}

	return false
}
