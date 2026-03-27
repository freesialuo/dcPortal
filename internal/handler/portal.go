package handler

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"dcportal/internal/discord"
	"dcportal/internal/model"
	"dcportal/internal/store"
)

// PortalHandler handles the public-facing portal pages.
type PortalHandler struct {
	store         *store.Store
	portalTmpl    *template.Template // for the index page
	resultTmpl    *template.Template // for the callback result page
	discordClient *discord.Client

	// In-memory state store for CSRF protection.
	// Maps state → botID. Entries are single-use.
	stateMu sync.Mutex
	states  map[string]stateEntry
}

const stateTTL = 10 * time.Minute

type stateEntry struct {
	BotID     int64
	ExpiresAt time.Time
}

// NewPortalHandler creates a new PortalHandler.
func NewPortalHandler(s *store.Store, portalTmpl, resultTmpl *template.Template, dc *discord.Client) *PortalHandler {
	return &PortalHandler{
		store:         s,
		portalTmpl:    portalTmpl,
		resultTmpl:    resultTmpl,
		discordClient: dc,
		states:        make(map[string]stateEntry),
	}
}

// RegisterRoutes registers public portal routes on the given mux.
func (h *PortalHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /portal", h.index)
	mux.HandleFunc("GET /install/{id}", h.install)
	mux.HandleFunc("GET /callback", h.callback)
}

func (h *PortalHandler) index(w http.ResponseWriter, r *http.Request) {
	bots, err := h.store.ListEnabledBots()
	if err != nil {
		log.Printf("ERROR list enabled bots: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title": "DCPortal — Bot Install",
		"Bots":  bots,
	}
	if err := h.portalTmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ERROR render portal: %v", err)
	}
}

func (h *PortalHandler) install(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	bot, err := h.store.GetBot(id)
	if err != nil {
		log.Printf("ERROR get bot %d: %v", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if bot == nil || !bot.Enabled {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// Generate CSRF state and store mapping to bot ID
	state, err := generateState()
	if err != nil {
		log.Printf("ERROR generate state: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.stateMu.Lock()
	h.pruneExpiredStatesLocked(time.Now())
	h.states[state] = stateEntry{
		BotID:     bot.ID,
		ExpiresAt: time.Now().Add(stateTTL),
	}
	h.stateMu.Unlock()

	http.Redirect(w, r, bot.OAuthURL(state), http.StatusTemporaryRedirect)
}

func (h *PortalHandler) callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	// Discord may return an error (e.g. user denied)
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		log.Printf("OAuth2 error from Discord: %s - %s", errMsg, r.URL.Query().Get("error_description"))
		h.renderResult(w, false, "Authorization was denied or an error occurred: "+errMsg, "", "", "")
		return
	}

	if code == "" || state == "" {
		http.Error(w, "Missing code or state parameter", http.StatusBadRequest)
		return
	}

	// Validate and consume state (single-use)
	h.stateMu.Lock()
	h.pruneExpiredStatesLocked(time.Now())
	entry, ok := h.states[state]
	if ok && time.Now().Before(entry.ExpiresAt) {
		delete(h.states, state)
	} else {
		ok = false
	}
	h.stateMu.Unlock()

	if !ok {
		http.Error(w, "Invalid or expired state parameter", http.StatusBadRequest)
		return
	}

	// Look up the bot
	bot, err := h.store.GetBot(entry.BotID)
	if err != nil || bot == nil {
		log.Printf("ERROR get bot %d: %v", entry.BotID, err)
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	// Exchange the authorization code for an access token
	tokenResp, err := h.discordClient.ExchangeCode(
		bot.ClientID, bot.ClientSecret, code, bot.RedirectURI,
	)
	if err != nil {
		log.Printf("ERROR exchange code for bot %d: %v", entry.BotID, err)
		h.renderResult(w, false, "Failed to complete authorization with Discord. Please try again.", "", "", "")
		return
	}

	// Determine guild info
	guildID := ""
	guildName := ""
	if tokenResp.Guild != nil {
		guildID = tokenResp.Guild.ID
		guildName = tokenResp.Guild.Name
		if guildID != "" {
			if _, err := h.store.RecordInstall(entry.BotID, guildID, guildName); err != nil {
				log.Printf("ERROR record install: %v", err)
			}
		} else {
			log.Printf("WARN token response missing guild ID for bot %d", entry.BotID)
		}
	} else {
		log.Printf("WARN token response missing guild info for bot %d", entry.BotID)
	}

	// Revoke the access token (we don't need it — we only needed the bot authorization)
	if tokenResp.AccessToken != "" {
		go func() {
			if err := h.discordClient.RevokeToken(bot.ClientID, bot.ClientSecret, tokenResp.AccessToken); err != nil {
				log.Printf("WARN revoke token: %v", err)
			}
		}()
	}

	h.renderResult(w, true, "Bot has been authorized successfully!", bot.Name, guildName, guildID)
}

func (h *PortalHandler) renderResult(w http.ResponseWriter, success bool, message, botName, guildName, guildID string) {
	title := "Authorization Successful"
	if !success {
		title = "Authorization Failed"
	}
	data := map[string]any{
		"Title":     title,
		"Success":   success,
		"Message":   message,
		"BotName":   botName,
		"GuildName": guildName,
		"GuildID":   guildID,
	}
	if err := h.resultTmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("ERROR render result: %v", err)
		if !success {
			http.Error(w, message, http.StatusBadRequest)
		}
	}
}

// generateState wraps model.GenerateState for easy testing override.
var generateState = func() (string, error) {
	return model.GenerateState()
}

func (h *PortalHandler) pruneExpiredStatesLocked(now time.Time) {
	for key, value := range h.states {
		if !now.Before(value.ExpiresAt) {
			delete(h.states, key)
		}
	}
}
