package model

import (
	"net/url"
	"time"
)

// InstallLink represents an install entry bound to a bot.
// One bot can have multiple links with different permissions/scopes/redirect settings.
type InstallLink struct {
	ID          int64     `json:"id"`
	BotID       int64     `json:"bot_id"`
	BotName     string    `json:"bot_name,omitempty"`
	Name        string    `json:"name"`
	Permissions string    `json:"permissions"`
	Scopes      string    `json:"scopes"`
	RedirectURI string    `json:"redirect_uri"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

// OAuthURL builds the Discord OAuth2 authorization URL for this install link.
func (l *InstallLink) OAuthURL(clientID, state string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("response_type", "code")
	if l.Permissions != "" {
		params.Set("permissions", l.Permissions)
	}
	if l.Scopes != "" {
		params.Set("scope", l.Scopes)
	} else {
		params.Set("scope", "bot")
	}
	if l.RedirectURI != "" {
		params.Set("redirect_uri", l.RedirectURI)
	}
	if state != "" {
		params.Set("state", state)
	}
	return "https://discord.com/oauth2/authorize?" + params.Encode()
}

// InstallLinkWithBot is the install-ready view that includes bot credentials.
type InstallLinkWithBot struct {
	Link InstallLink
	Bot  Bot
}
