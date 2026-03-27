package handler

import (
	"crypto/subtle"
	"html/template"
	"log"
	"net/http"
	"strings"

	"dcportal/internal/middleware"
)

// AuthHandler handles the home login page using ADMIN token.
type AuthHandler struct {
	tmpl       *template.Template
	adminToken string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(tmpl *template.Template, adminToken string) *AuthHandler {
	return &AuthHandler{
		tmpl:       tmpl,
		adminToken: adminToken,
	}
}

// RegisterRoutes registers login/logout routes.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.index)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /logout", h.logout)
}

func (h *AuthHandler) index(w http.ResponseWriter, r *http.Request) {
	if middleware.HasValidAdminToken(r, h.adminToken) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	h.renderLogin(w, r, "")
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	provided := strings.TrimSpace(r.FormValue("token"))
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.adminToken)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		h.renderLogin(w, r, "Invalid ADMIN token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    provided,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, sanitizeNextPath(r.FormValue("next")), http.StatusSeeOther)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) renderLogin(w http.ResponseWriter, r *http.Request, errMsg string) {
	data := map[string]any{
		"Title": "DCPortal Login",
		"Error": errMsg,
		"Next":  sanitizeNextPath(r.URL.Query().Get("next")),
	}
	if err := h.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ERROR render login: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func sanitizeNextPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/admin"
	}
	// Only allow local absolute path.
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/admin"
	}
	return raw
}
