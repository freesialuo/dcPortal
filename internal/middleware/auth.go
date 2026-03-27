package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// AdminAuth returns middleware that checks for a valid admin token.
// It accepts "Authorization: Bearer <token>" header, "admin_token" cookie,
// or "token" query parameter (which also sets the cookie for convenience).
func AdminAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if checkToken(w, r, token) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}

func checkToken(w http.ResponseWriter, r *http.Request, expected string) bool {
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

	// Check query parameter (for simple browser access)
	if q := r.URL.Query().Get("token"); q != "" {
		if subtle.ConstantTimeCompare([]byte(q), []byte(expected)) == 1 {
			// Set cookie so subsequent requests don't need the query param
			http.SetCookie(w, &http.Cookie{
				Name:     "admin_token",
				Value:    expected,
				Path:     "/admin",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   86400 * 7, // 7 days
			})
			return true
		}
	}

	return false
}
