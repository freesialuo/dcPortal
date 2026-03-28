package model

import "time"

// GuildInstall records a bot installation to a Discord guild.
type GuildInstall struct {
	ID               int64     `json:"id"`
	BotID            int64     `json:"bot_id"`
	BotName          string    `json:"bot_name"` // denormalized for display
	LinkID           int64     `json:"link_id"`
	LinkName         string    `json:"link_name"`
	GuildID          string    `json:"guild_id"`
	GuildName        string    `json:"guild_name"`
	MemberCount      int       `json:"member_count"`
	UserAccessToken  string    `json:"-"`
	UserRefreshToken string    `json:"-"`
	InstalledAt      time.Time `json:"installed_at"`
}

// GuildBlacklist records guilds that should reject re-installs for a bot.
type GuildBlacklist struct {
	ID        int64     `json:"id"`
	BotID     int64     `json:"bot_id"`
	BotName   string    `json:"bot_name"` // denormalized for display
	GuildID   string    `json:"guild_id"`
	GuildName string    `json:"guild_name"`
	CreatedAt time.Time `json:"created_at"`
}
