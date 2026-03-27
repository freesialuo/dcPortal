package model

import "time"

// GuildInstall records a bot installation to a Discord guild.
type GuildInstall struct {
	ID          int64     `json:"id"`
	BotID       int64     `json:"bot_id"`
	BotName     string    `json:"bot_name"` // denormalized for display
	GuildID     string    `json:"guild_id"`
	GuildName   string    `json:"guild_name"`
	InstalledAt time.Time `json:"installed_at"`
}
