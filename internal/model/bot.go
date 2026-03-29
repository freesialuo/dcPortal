package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"
)

// Bot represents a Discord bot configuration.
type Bot struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	BotToken     string    `json:"bot_token"`
	Permissions  string    `json:"permissions"`
	Scopes       string    `json:"scopes"`
	RedirectURI  string    `json:"redirect_uri"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

// BotAdminView is a redacted bot shape for admin pages.
// Secret values are never exposed to templates.
type BotAdminView struct {
	ID              int64
	Name            string
	ClientID        string
	Permissions     string
	Scopes          string
	RedirectURI     string
	Enabled         bool
	CreatedAt       time.Time
	HasClientSecret bool
	HasBotToken     bool
}

// BotUpdatePatch updates bot fields without requiring secret read-back.
// A nil secret/token pointer means "do not change that field".
type BotUpdatePatch struct {
	ID            int64
	Name          string
	ClientID      string
	Permissions   string
	Scopes        string
	RedirectURI   string
	ClientSecret  *string
	BotToken      *string
	ClearBotToken bool
}

// OAuthURL builds the Discord OAuth2 authorization URL for this bot.
// It uses the authorization code grant flow with state for CSRF protection.
func (b *Bot) OAuthURL(state string) string {
	params := url.Values{}
	params.Set("client_id", b.ClientID)
	params.Set("response_type", "code")
	if b.Permissions != "" {
		params.Set("permissions", b.Permissions)
	}
	if b.Scopes != "" {
		params.Set("scope", b.Scopes)
	} else {
		params.Set("scope", "bot")
	}
	if b.RedirectURI != "" {
		params.Set("redirect_uri", b.RedirectURI)
	}
	if state != "" {
		params.Set("state", state)
	}
	return "https://discord.com/oauth2/authorize?" + params.Encode()
}

// GenerateState creates a cryptographically random state string for CSRF protection.
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
