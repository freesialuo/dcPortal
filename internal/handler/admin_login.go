package handler

import (
	"crypto/subtle"
	"html/template"
	"log"
	"net/http"
	"strings"

	"dcportal/internal/middleware"
)

// AdminLoginHandler handles login/logout for admin pages.
type AdminLoginHandler struct {
	tmpl       *template.Template
	adminToken string
}

// NewAdminLoginHandler creates a new AdminLoginHandler.
func NewAdminLoginHandler(tmpl *template.Template, adminToken string) *AdminLoginHandler {
	return &AdminLoginHandler{
		tmpl:       tmpl,
		adminToken: adminToken,
	}
}

// RegisterRoutes registers admin login/logout routes.
func (h *AdminLoginHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/login", h.index)
	mux.HandleFunc("POST /admin/login", h.login)
	mux.HandleFunc("POST /admin/logout", h.logout)
}

func (h *AdminLoginHandler) index(w http.ResponseWriter, r *http.Request) {
	if middleware.HasValidAdminToken(r, h.adminToken) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	h.renderLogin(w, "", r.URL.Query().Get("next"))
}

func (h *AdminLoginHandler) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	provided := strings.TrimSpace(r.FormValue("token"))
	nextPath := r.FormValue("next")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.adminToken)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		h.renderLogin(w, "Invalid ADMIN token", nextPath)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    provided,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, sanitizeNextPath(nextPath, "/admin"), http.StatusSeeOther)
}

func (h *AdminLoginHandler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "admin_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (h *AdminLoginHandler) renderLogin(w http.ResponseWriter, errMsg, nextRaw string) {
	data := map[string]any{
		"Title":      "DCPortal Admin Login",
		"Subtitle":   "Enter ADMIN token to continue",
		"Heading":    "Admin Login",
		"TokenLabel": "ADMIN Token",
		"FormAction": "/admin/login",
		"Error":      errMsg,
		"Next":       sanitizeNextPath(nextRaw, "/admin"),
	}
	if err := h.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ERROR render admin login: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
