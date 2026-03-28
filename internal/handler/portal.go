package handler

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	// Maps state → install context. Entries are single-use.
	stateMu sync.Mutex
	states  map[string]stateEntry
}

const stateTTL = 10 * time.Minute

type stateEntry struct {
	BotID     int64
	LinkID    int64
	Redirect  string
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
	links, err := h.store.ListEnabledInstallLinks()
	if err != nil {
		log.Printf("ERROR list enabled install links: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title": "DCPortal — Bot Install",
		"Links": links,
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

	installLink, err := h.store.GetInstallLinkWithBot(id)
	if err != nil {
		log.Printf("ERROR get install link %d: %v", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if installLink == nil || !installLink.Bot.Enabled || !installLink.Link.Enabled {
		http.Error(w, "Install link not found", http.StatusNotFound)
		return
	}
	redirectURI := strings.TrimSpace(installLink.Link.RedirectURI)
	if redirectURI == "" {
		redirectURI = strings.TrimSpace(installLink.Bot.RedirectURI)
	}
	if redirectURI == "" {
		log.Printf("ERROR install link %d missing redirect URI", installLink.Link.ID)
		http.Error(w, "Install link is misconfigured: missing redirect URI", http.StatusInternalServerError)
		return
	}

	// Generate CSRF state and store mapping to link+bot IDs.
	state, err := generateState()
	if err != nil {
		log.Printf("ERROR generate state: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.stateMu.Lock()
	h.pruneExpiredStatesLocked(time.Now())
	h.states[state] = stateEntry{
		BotID:     installLink.Bot.ID,
		LinkID:    installLink.Link.ID,
		Redirect:  redirectURI,
		ExpiresAt: time.Now().Add(stateTTL),
	}
	h.stateMu.Unlock()

	linkForOAuth := installLink.Link
	linkForOAuth.RedirectURI = redirectURI
	http.Redirect(w, r, linkForOAuth.OAuthURL(installLink.Bot.ClientID, state), http.StatusTemporaryRedirect)
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

	installLink, err := h.store.GetInstallLinkWithBot(entry.LinkID)
	if err != nil || installLink == nil {
		log.Printf("ERROR get install link %d: %v", entry.LinkID, err)
		http.Error(w, "Install link not found", http.StatusNotFound)
		return
	}
	bot := &installLink.Bot

	// Exchange the authorization code for an access token
	redirectURI := strings.TrimSpace(entry.Redirect)
	if redirectURI == "" {
		redirectURI = strings.TrimSpace(installLink.Link.RedirectURI)
	}
	if redirectURI == "" {
		redirectURI = strings.TrimSpace(bot.RedirectURI)
	}
	if redirectURI == "" {
		log.Printf("ERROR callback install link %d missing redirect URI", entry.LinkID)
		h.renderResult(w, false, "Install link is misconfigured: missing redirect URI.", bot.Name, "", "")
		return
	}
	tokenResp, err := h.discordClient.ExchangeCode(
		bot.ClientID, bot.ClientSecret, code, redirectURI,
	)
	if err != nil {
		log.Printf("ERROR exchange code for bot %d: %v", entry.BotID, err)
		h.renderResult(w, false, "Failed to complete authorization with Discord. Please try again.", "", "", "")
		return
	}

	// Determine guild info
	guildID := ""
	guildName := ""
	memberCount := 0
	if tokenResp.Guild != nil {
		guildID = tokenResp.Guild.ID
		guildName = tokenResp.Guild.Name
		if guildID != "" {
			isBlocked, err := h.store.IsGuildBlacklisted(entry.BotID, guildID)
			if err != nil {
				log.Printf("ERROR check blacklist: %v", err)
			}
			if isBlocked {
				log.Printf("WARN blocked guild install attempt bot=%d guild=%s", entry.BotID, guildID)
				if bot.BotToken != "" {
					go func() {
						if err := h.discordClient.LeaveGuild(bot.BotToken, guildID); err != nil {
							log.Printf("WARN leave blocked guild: %v", err)
						}
					}()
				}
				if tokenResp.AccessToken != "" {
					go func() {
						if err := h.discordClient.RevokeToken(bot.ClientID, bot.ClientSecret, tokenResp.AccessToken); err != nil {
							log.Printf("WARN revoke token: %v", err)
						}
					}()
				}
				h.renderResult(w, false, "This server has been blocked by admin and cannot install this bot.", bot.Name, guildName, guildID)
				return
			}
			if bot.BotToken != "" {
				guild, err := h.discordClient.GetGuild(bot.BotToken, guildID)
				if err != nil {
					log.Printf("WARN refresh guild details on callback: %v", err)
				} else {
					guildName = guild.Name
					memberCount = guild.ApproximateMemberCount
				}
			}
			if _, err := h.store.RecordInstallWithLink(entry.BotID, installLink.Link.ID, installLink.Link.Name, guildID, guildName, memberCount, tokenResp.AccessToken, tokenResp.RefreshToken); err != nil {
				log.Printf("ERROR record install: %v", err)
			}
		} else {
			log.Printf("WARN token response missing guild ID for bot %d", entry.BotID)
		}
	} else {
		log.Printf("WARN token response missing guild info for bot %d", entry.BotID)
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
