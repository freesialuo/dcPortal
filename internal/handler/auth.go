package handler

import (
	"crypto/subtle"
	"html/template"
	"log"
	"net/http"
	"strings"

	"dcportal/internal/middleware"
)

// AuthHandler handles the install access login page.
type AuthHandler struct {
	tmpl         *template.Template
	installToken string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(tmpl *template.Template, installToken string) *AuthHandler {
	return &AuthHandler{
		tmpl:         tmpl,
		installToken: installToken,
	}
}

// RegisterRoutes registers login/logout routes.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.index)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /logout", h.logout)
}

func (h *AuthHandler) index(w http.ResponseWriter, r *http.Request) {
	if middleware.HasValidInstallToken(r, h.installToken) {
		http.Redirect(w, r, "/portal", http.StatusSeeOther)
		return
	}
	h.renderLogin(w, "", r.URL.Query().Get("next"), map[string]any{
		"Title":      "DCPortal Install Access",
		"Subtitle":   "Enter install token to continue",
		"Heading":    "Install Portal Login",
		"TokenLabel": "Install Token",
		"FormAction": "/login",
	})
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	provided := strings.TrimSpace(r.FormValue("token"))
	nextPath := r.FormValue("next")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.installToken)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		h.renderLogin(w, "Invalid install token", nextPath, map[string]any{
			"Title":      "DCPortal Install Access",
			"Subtitle":   "Enter install token to continue",
			"Heading":    "Install Portal Login",
			"TokenLabel": "Install Token",
			"FormAction": "/login",
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "install_token",
		Value:    provided,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, sanitizeNextPath(r.FormValue("next"), "/portal"), http.StatusSeeOther)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "install_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) renderLogin(w http.ResponseWriter, errMsg, nextRaw string, base map[string]any) {
	data := map[string]any{}
	for k, v := range base {
		data[k] = v
	}
	data["Error"] = errMsg
	data["Next"] = sanitizeNextPath(nextRaw, "/portal")
	if err := h.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ERROR render login: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func sanitizeNextPath(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	// Only allow local absolute path.
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return fallback
	}
	return raw
}
