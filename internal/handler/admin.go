package handler

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"dcportal/internal/model"
	"dcportal/internal/store"
)

// AdminHandler handles the admin management pages.
type AdminHandler struct {
	store *store.Store
	tmpl  *template.Template
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(s *store.Store, tmpl *template.Template) *AdminHandler {
	return &AdminHandler{store: s, tmpl: tmpl}
}

// RegisterRoutes registers admin routes on the given mux.
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin", h.index)
	mux.HandleFunc("POST /admin/bots", h.createBot)
	mux.HandleFunc("POST /admin/bots/{id}/toggle", h.toggleBot)
	mux.HandleFunc("POST /admin/bots/{id}/delete", h.deleteBot)
}

func (h *AdminHandler) index(w http.ResponseWriter, r *http.Request) {
	bots, err := h.store.ListBots()
	if err != nil {
		log.Printf("ERROR list bots: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	installs, err := h.store.ListInstalls()
	if err != nil {
		log.Printf("ERROR list installs: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":    "Admin — DCPortal",
		"Bots":     bots,
		"Installs": installs,
	}
	if err := h.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ERROR render admin: %v", err)
	}
}

func (h *AdminHandler) createBot(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	bot := &model.Bot{
		Name:         strings.TrimSpace(r.FormValue("name")),
		ClientID:     strings.TrimSpace(r.FormValue("client_id")),
		ClientSecret: strings.TrimSpace(r.FormValue("client_secret")),
		Permissions:  strings.TrimSpace(r.FormValue("permissions")),
		Scopes:       strings.TrimSpace(r.FormValue("scopes")),
		RedirectURI:  strings.TrimSpace(r.FormValue("redirect_uri")),
		Enabled:      true,
	}

	if bot.Name == "" || bot.ClientID == "" || bot.ClientSecret == "" {
		http.Error(w, "Name, Client ID, and Client Secret are required", http.StatusBadRequest)
		return
	}
	if bot.Scopes == "" {
		bot.Scopes = "bot"
	}

	if err := h.store.CreateBot(bot); err != nil {
		log.Printf("ERROR create bot: %v", err)
		http.Error(w, "Failed to create bot", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminHandler) toggleBot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.store.ToggleBot(id); err != nil {
		log.Printf("ERROR toggle bot %d: %v", id, err)
		http.Error(w, "Failed to toggle bot", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminHandler) deleteBot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteBot(id); err != nil {
		log.Printf("ERROR delete bot %d: %v", id, err)
		http.Error(w, "Failed to delete bot", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
