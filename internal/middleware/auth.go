package middleware

import (
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
)

// AdminAuth returns middleware that checks for a valid admin token.
// It accepts "Authorization: Bearer <token>" header or "admin_token" cookie.
func AdminAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if HasValidAdminToken(r, token) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}

// AdminAuthWithRedirect checks for a valid admin token and redirects to loginPath when missing.
// It is intended for browser-based pages.
func AdminAuthWithRedirect(token, loginPath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if HasValidAdminToken(r, token) {
				next.ServeHTTP(w, r)
				return
			}
			target := loginPath
			if target == "" {
				target = "/"
			}
			nextParam := r.URL.RequestURI()
			if nextParam != "" && nextParam != "/" {
				target = target + "?next=" + url.QueryEscape(nextParam)
			}
			http.Redirect(w, r, target, http.StatusSeeOther)
		})
	}
}

// HasValidAdminToken checks Authorization Bearer token or admin_token cookie.
func HasValidAdminToken(r *http.Request, expected string) bool {
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
