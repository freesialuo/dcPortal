package handler

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dcportal/internal/discord"
	"dcportal/internal/model"
	"dcportal/internal/store"
)

// AdminHandler handles the admin management pages.
type AdminHandler struct {
	store         *store.Store
	tmpl          *template.Template
	discordClient *discord.Client
}

type adminInstallView struct {
	ID          int64
	BotName     string
	LinkName    string
	GuildID     string
	GuildName   string
	MemberCount int
	InstalledAt time.Time
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(s *store.Store, tmpl *template.Template, dc *discord.Client) *AdminHandler {
	return &AdminHandler{store: s, tmpl: tmpl, discordClient: dc}
}

// RegisterRoutes registers admin routes on the given mux.
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin", h.index)
	mux.HandleFunc("POST /admin/bots", h.createBot)
	mux.HandleFunc("POST /admin/bots/{id}/update", h.updateBot)
	mux.HandleFunc("POST /admin/bots/{id}/toggle", h.toggleBot)
	mux.HandleFunc("POST /admin/bots/{id}/delete", h.deleteBot)
	mux.HandleFunc("POST /admin/bots/{id}/links", h.createInstallLink)
	mux.HandleFunc("POST /admin/links/{id}/update", h.updateInstallLink)
	mux.HandleFunc("POST /admin/links/{id}/toggle", h.toggleInstallLink)
	mux.HandleFunc("POST /admin/links/{id}/delete", h.deleteInstallLink)
	mux.HandleFunc("POST /admin/installs/refresh", h.refreshAllInstalls)
	mux.HandleFunc("POST /admin/installs/{id}/refresh", h.refreshInstall)
	mux.HandleFunc("POST /admin/installs/{id}/revoke", h.revokeInstallAuth)
	mux.HandleFunc("POST /admin/installs/{id}/disconnect", h.disconnectInstall)
	mux.HandleFunc("POST /admin/installs/{id}/disconnect-blacklist", h.disconnectAndBlacklistInstall)
}

func (h *AdminHandler) index(w http.ResponseWriter, r *http.Request) {
	bots, err := h.store.ListBotsForAdmin()
	if err != nil {
		log.Printf("ERROR list bots: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	allLinks, err := h.store.ListInstallLinks()
	if err != nil {
		log.Printf("ERROR list install links: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	linksByBot := make(map[int64][]model.InstallLink, len(bots))
	for _, link := range allLinks {
		linksByBot[link.BotID] = append(linksByBot[link.BotID], link)
	}

	installsRaw, err := h.store.ListInstalls()
	if err != nil {
		log.Printf("ERROR list installs: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	installs := make([]adminInstallView, 0, len(installsRaw))
	for _, install := range installsRaw {
		installs = append(installs, adminInstallView{
			ID:          install.ID,
			BotName:     install.BotName,
			LinkName:    install.LinkName,
			GuildID:     install.GuildID,
			GuildName:   install.GuildName,
			MemberCount: install.MemberCount,
			InstalledAt: install.InstalledAt,
		})
	}

	blacklist, err := h.store.ListGuildBlacklist()
	if err != nil {
		log.Printf("ERROR list blacklist: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":      "Admin — DCPortal",
		"Bots":       bots,
		"LinksByBot": linksByBot,
		"Installs":   installs,
		"Blacklist":  blacklist,
		"Notice":     strings.TrimSpace(r.URL.Query().Get("notice")),
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
		BotToken:     strings.TrimSpace(r.FormValue("bot_token")),
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

	h.redirectWithNotice(w, r, "Bot created")
}

func (h *AdminHandler) toggleBot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.store.ToggleBot(id); err != nil {
		log.Printf("ERROR toggle bot %d: %v", id, err)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Bot not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to toggle bot", http.StatusInternalServerError)
		return
	}

	h.redirectWithNotice(w, r, "Bot status updated")
}

func (h *AdminHandler) deleteBot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteBot(id); err != nil {
		log.Printf("ERROR delete bot %d: %v", id, err)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Bot not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete bot", http.StatusInternalServerError)
		return
	}

	h.redirectWithNotice(w, r, "Bot deleted")
}

func (h *AdminHandler) updateBot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	clientSecret := strings.TrimSpace(r.FormValue("client_secret"))
	botToken := strings.TrimSpace(r.FormValue("bot_token"))
	clearBotToken := strings.TrimSpace(r.FormValue("clear_bot_token")) == "1"
	patch := &model.BotUpdatePatch{
		ID:            id,
		Name:          strings.TrimSpace(r.FormValue("name")),
		ClientID:      strings.TrimSpace(r.FormValue("client_id")),
		Permissions:   strings.TrimSpace(r.FormValue("permissions")),
		Scopes:        strings.TrimSpace(r.FormValue("scopes")),
		RedirectURI:   strings.TrimSpace(r.FormValue("redirect_uri")),
		ClearBotToken: clearBotToken,
	}
	if clientSecret != "" {
		patch.ClientSecret = &clientSecret
	}
	if botToken != "" && !clearBotToken {
		patch.BotToken = &botToken
	}

	if patch.Name == "" || patch.ClientID == "" {
		http.Error(w, "Name and Client ID are required", http.StatusBadRequest)
		return
	}
	if patch.Scopes == "" {
		patch.Scopes = "bot"
	}

	if err := h.store.UpdateBotPatch(patch); err != nil {
		log.Printf("ERROR update bot %d: %v", id, err)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Bot not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update bot", http.StatusInternalServerError)
		return
	}

	h.redirectWithNotice(w, r, "Bot updated")
}

func (h *AdminHandler) createInstallLink(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid bot ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	bot, err := h.store.GetBot(botID)
	if err != nil {
		log.Printf("ERROR get bot %d for create install link: %v", botID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if bot == nil {
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	link := &model.InstallLink{
		BotID:       botID,
		Name:        strings.TrimSpace(r.FormValue("link_name")),
		Permissions: strings.TrimSpace(r.FormValue("permissions")),
		Scopes:      strings.TrimSpace(r.FormValue("scopes")),
		RedirectURI: strings.TrimSpace(r.FormValue("redirect_uri")),
		Enabled:     true,
	}
	if link.Name == "" {
		http.Error(w, "Link name is required", http.StatusBadRequest)
		return
	}
	if link.Scopes == "" {
		link.Scopes = "bot"
	}
	if link.RedirectURI == "" {
		link.RedirectURI = bot.RedirectURI
	}

	if err := h.store.CreateInstallLink(link); err != nil {
		log.Printf("ERROR create install link bot=%d: %v", botID, err)
		http.Error(w, "Failed to create install link", http.StatusInternalServerError)
		return
	}

	h.redirectWithNotice(w, r, "Install link created")
}

func (h *AdminHandler) updateInstallLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid link ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	existing, err := h.store.GetInstallLink(id)
	if err != nil {
		log.Printf("ERROR get install link %d for update: %v", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Install link not found", http.StatusNotFound)
		return
	}

	updated := &model.InstallLink{
		ID:          existing.ID,
		BotID:       existing.BotID,
		Name:        strings.TrimSpace(r.FormValue("link_name")),
		Permissions: strings.TrimSpace(r.FormValue("permissions")),
		Scopes:      strings.TrimSpace(r.FormValue("scopes")),
		RedirectURI: strings.TrimSpace(r.FormValue("redirect_uri")),
		Enabled:     existing.Enabled,
	}
	if updated.Name == "" {
		http.Error(w, "Link name is required", http.StatusBadRequest)
		return
	}
	if updated.Scopes == "" {
		updated.Scopes = "bot"
	}

	if err := h.store.UpdateInstallLink(updated); err != nil {
		log.Printf("ERROR update install link %d: %v", id, err)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Install link not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update install link", http.StatusInternalServerError)
		return
	}

	h.redirectWithNotice(w, r, "Install link updated")
}

func (h *AdminHandler) toggleInstallLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid link ID", http.StatusBadRequest)
		return
	}
	if err := h.store.ToggleInstallLink(id); err != nil {
		log.Printf("ERROR toggle install link %d: %v", id, err)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Install link not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to toggle install link", http.StatusInternalServerError)
		return
	}
	h.redirectWithNotice(w, r, "Install link status updated")
}

func (h *AdminHandler) deleteInstallLink(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid link ID", http.StatusBadRequest)
		return
	}
	if err := h.store.DeleteInstallLink(id); err != nil {
		log.Printf("ERROR delete install link %d: %v", id, err)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "Install link not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete install link", http.StatusInternalServerError)
		return
	}
	h.redirectWithNotice(w, r, "Install link deleted")
}

func (h *AdminHandler) refreshAllInstalls(w http.ResponseWriter, r *http.Request) {
	installs, err := h.store.ListInstalls()
	if err != nil {
		log.Printf("ERROR list installs for refresh all: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	updated := 0
	for _, install := range installs {
		if h.refreshInstallByRecord(&install) {
			updated++
		}
	}
	h.redirectWithNotice(w, r, fmt.Sprintf("Refreshed %d/%d install records", updated, len(installs)))
}

func (h *AdminHandler) refreshInstall(w http.ResponseWriter, r *http.Request) {
	install, err := h.mustGetInstallFromPath(w, r)
	if err != nil {
		return
	}
	if ok := h.refreshInstallByRecord(install); !ok {
		h.redirectWithNotice(w, r, "Refresh failed: check Bot Token and guild access")
		return
	}
	h.redirectWithNotice(w, r, "Install info refreshed")
}

func (h *AdminHandler) refreshInstallByRecord(install *model.GuildInstall) bool {
	bot, err := h.store.GetBot(install.BotID)
	if err != nil || bot == nil {
		log.Printf("ERROR get bot %d for refresh: %v", install.BotID, err)
		return false
	}
	if bot.BotToken == "" {
		log.Printf("WARN missing bot token for bot %d, skip refresh", bot.ID)
		return false
	}

	guild, err := h.discordClient.GetGuild(bot.BotToken, install.GuildID)
	if err != nil {
		log.Printf("ERROR refresh guild info install=%d guild=%s: %v", install.ID, install.GuildID, err)
		return false
	}
	if err := h.store.UpdateInstallGuildInfo(install.ID, guild.Name, guild.ApproximateMemberCount); err != nil {
		log.Printf("ERROR update install guild info install=%d: %v", install.ID, err)
		return false
	}
	return true
}

func (h *AdminHandler) revokeInstallAuth(w http.ResponseWriter, r *http.Request) {
	install, err := h.mustGetInstallFromPath(w, r)
	if err != nil {
		return
	}

	bot, err := h.store.GetBot(install.BotID)
	if err != nil || bot == nil {
		log.Printf("ERROR get bot %d for revoke: %v", install.BotID, err)
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	revoked := false
	if install.UserAccessToken != "" {
		if err := h.discordClient.RevokeToken(bot.ClientID, bot.ClientSecret, install.UserAccessToken); err != nil {
			log.Printf("WARN revoke access token install=%d: %v", install.ID, err)
		} else {
			revoked = true
		}
	}
	if install.UserRefreshToken != "" {
		if err := h.discordClient.RevokeToken(bot.ClientID, bot.ClientSecret, install.UserRefreshToken); err != nil {
			log.Printf("WARN revoke refresh token install=%d: %v", install.ID, err)
		} else {
			revoked = true
		}
	}

	if revoked {
		h.redirectWithNotice(w, r, "OAuth2 user authorization revoked")
		return
	}
	h.redirectWithNotice(w, r, "No saved OAuth2 token found for this install")
}

func (h *AdminHandler) disconnectInstall(w http.ResponseWriter, r *http.Request) {
	h.disconnectInstallInternal(w, r, false)
}

func (h *AdminHandler) disconnectAndBlacklistInstall(w http.ResponseWriter, r *http.Request) {
	h.disconnectInstallInternal(w, r, true)
}

func (h *AdminHandler) disconnectInstallInternal(w http.ResponseWriter, r *http.Request, blacklist bool) {
	install, err := h.mustGetInstallFromPath(w, r)
	if err != nil {
		return
	}

	bot, err := h.store.GetBot(install.BotID)
	if err != nil || bot == nil {
		log.Printf("ERROR get bot %d for disconnect: %v", install.BotID, err)
		http.Error(w, "Bot not found", http.StatusNotFound)
		return
	}

	leftGuild := false
	if bot.BotToken != "" {
		if err := h.discordClient.LeaveGuild(bot.BotToken, install.GuildID); err != nil {
			log.Printf("WARN leave guild install=%d guild=%s: %v", install.ID, install.GuildID, err)
		} else {
			leftGuild = true
		}
	} else {
		log.Printf("WARN bot %d missing bot token, skip leave guild", bot.ID)
	}

	if blacklist {
		if err := h.store.AddGuildBlacklist(install.BotID, install.GuildID, install.GuildName); err != nil {
			log.Printf("ERROR add blacklist install=%d: %v", install.ID, err)
			http.Error(w, "Failed to blacklist guild", http.StatusInternalServerError)
			return
		}
	}

	if err := h.store.DeleteInstall(install.ID); err != nil {
		log.Printf("ERROR delete install %d: %v", install.ID, err)
		http.Error(w, "Failed to disconnect install", http.StatusInternalServerError)
		return
	}

	notice := "Disconnected."
	if blacklist {
		notice = "Disconnected and blacklisted."
	}
	if !leftGuild {
		notice += " Bot leave skipped/failed (check Bot Token)."
	}
	h.redirectWithNotice(w, r, notice)
}

func (h *AdminHandler) mustGetInstallFromPath(w http.ResponseWriter, r *http.Request) (*model.GuildInstall, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid install ID", http.StatusBadRequest)
		return nil, err
	}

	install, err := h.store.GetInstall(id)
	if err != nil {
		log.Printf("ERROR get install %d: %v", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return nil, err
	}
	if install == nil {
		http.Error(w, "Install not found", http.StatusNotFound)
		return nil, store.ErrNotFound
	}
	return install, nil
}

func (h *AdminHandler) redirectWithNotice(w http.ResponseWriter, r *http.Request, notice string) {
	target := "/admin"
	if strings.TrimSpace(notice) != "" {
		target = target + "?notice=" + url.QueryEscape(notice)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
