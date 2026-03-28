package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"dcportal/internal/model"

	_ "modernc.org/sqlite"
)

// Store provides data access to the SQLite database.
type Store struct {
	db *sql.DB
}

// ErrNotFound is returned when an expected record is missing.
var ErrNotFound = errors.New("record not found")

// New opens a SQLite database and ensures the schema exists.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS bots (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			name          TEXT NOT NULL,
			client_id     TEXT NOT NULL UNIQUE,
			client_secret TEXT NOT NULL DEFAULT '',
			bot_token     TEXT NOT NULL DEFAULT '',
			permissions   TEXT NOT NULL DEFAULT '',
			scopes        TEXT NOT NULL DEFAULT 'bot',
			redirect_uri  TEXT NOT NULL DEFAULT '',
			enabled       INTEGER NOT NULL DEFAULT 1,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS guild_installs (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id             INTEGER NOT NULL,
			guild_id           TEXT NOT NULL,
			guild_name         TEXT NOT NULL DEFAULT '',
			member_count       INTEGER NOT NULL DEFAULT 0,
			user_access_token  TEXT NOT NULL DEFAULT '',
			user_refresh_token TEXT NOT NULL DEFAULT '',
			installed_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_guild_installs_bot ON guild_installs(bot_id)`,
		`CREATE TABLE IF NOT EXISTS guild_blacklist (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id       INTEGER NOT NULL,
			guild_id     TEXT NOT NULL,
			guild_name   TEXT NOT NULL DEFAULT '',
			created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE,
			UNIQUE(bot_id, guild_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_guild_blacklist_bot ON guild_blacklist(bot_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}

	// Backward-compatible migrations.
	if _, err := db.Exec(`ALTER TABLE bots ADD COLUMN bot_token TEXT NOT NULL DEFAULT ''`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("add bots.bot_token: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE guild_installs ADD COLUMN member_count INTEGER NOT NULL DEFAULT 0`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("add guild_installs.member_count: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE guild_installs ADD COLUMN user_access_token TEXT NOT NULL DEFAULT ''`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("add guild_installs.user_access_token: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE guild_installs ADD COLUMN user_refresh_token TEXT NOT NULL DEFAULT ''`); err != nil && !isDuplicateColumnErr(err) {
		return fmt.Errorf("add guild_installs.user_refresh_token: %w", err)
	}
	return nil
}

func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate column name")
}

// ---- Bot CRUD ----

// CreateBot inserts a new bot record.
func (s *Store) CreateBot(b *model.Bot) error {
	result, err := s.db.Exec(
		`INSERT INTO bots (name, client_id, client_secret, bot_token, permissions, scopes, redirect_uri, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.Name, b.ClientID, b.ClientSecret, b.BotToken, b.Permissions, b.Scopes, b.RedirectURI, b.Enabled,
	)
	if err != nil {
		return fmt.Errorf("insert bot: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert id: %w", err)
	}
	b.ID = id
	b.CreatedAt = time.Now()
	return nil
}

const botColumns = `id, name, client_id, client_secret, bot_token, permissions, scopes, redirect_uri, enabled, created_at`

func scanBot(sc interface{ Scan(...any) error }) (*model.Bot, error) {
	var b model.Bot
	err := sc.Scan(&b.ID, &b.Name, &b.ClientID, &b.ClientSecret, &b.BotToken, &b.Permissions, &b.Scopes, &b.RedirectURI, &b.Enabled, &b.CreatedAt)
	return &b, err
}

// ListBots returns all bot records.
func (s *Store) ListBots() ([]model.Bot, error) {
	rows, err := s.db.Query("SELECT " + botColumns + " FROM bots ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("query bots: %w", err)
	}
	defer rows.Close()

	var bots []model.Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bot: %w", err)
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

// ListEnabledBots returns only enabled bot records.
func (s *Store) ListEnabledBots() ([]model.Bot, error) {
	rows, err := s.db.Query("SELECT " + botColumns + " FROM bots WHERE enabled = 1 ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("query enabled bots: %w", err)
	}
	defer rows.Close()

	var bots []model.Bot
	for rows.Next() {
		b, err := scanBot(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bot: %w", err)
		}
		bots = append(bots, *b)
	}
	return bots, rows.Err()
}

// GetBot returns a bot by ID.
func (s *Store) GetBot(id int64) (*model.Bot, error) {
	b, err := scanBot(s.db.QueryRow("SELECT "+botColumns+" FROM bots WHERE id = ?", id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bot: %w", err)
	}
	return b, nil
}

// UpdateBot updates an existing bot record.
func (s *Store) UpdateBot(b *model.Bot) error {
	result, err := s.db.Exec(
		`UPDATE bots
		 SET name = ?, client_id = ?, client_secret = ?, bot_token = ?, permissions = ?, scopes = ?, redirect_uri = ?, enabled = ?
		 WHERE id = ?`,
		b.Name, b.ClientID, b.ClientSecret, b.BotToken, b.Permissions, b.Scopes, b.RedirectURI, b.Enabled, b.ID,
	)
	if err != nil {
		return fmt.Errorf("update bot: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update bot rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ToggleBot flips the enabled status of a bot.
func (s *Store) ToggleBot(id int64) error {
	result, err := s.db.Exec("UPDATE bots SET enabled = NOT enabled WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("toggle bot: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("toggle bot rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBot removes a bot record and its install records.
func (s *Store) DeleteBot(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM guild_blacklist WHERE bot_id = ?", id); err != nil {
		return fmt.Errorf("delete blacklist: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM guild_installs WHERE bot_id = ?", id); err != nil {
		return fmt.Errorf("delete installs: %w", err)
	}
	result, err := tx.Exec("DELETE FROM bots WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete bot: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete bot rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// ---- Guild Installs ----

// RecordInstall records a bot installation to a guild.
func (s *Store) RecordInstall(botID int64, guildID, guildName string, memberCount int, userAccessToken, userRefreshToken string) (*model.GuildInstall, error) {
	result, err := s.db.Exec(
		"INSERT INTO guild_installs (bot_id, guild_id, guild_name, member_count, user_access_token, user_refresh_token) VALUES (?, ?, ?, ?, ?, ?)",
		botID, guildID, guildName, memberCount, userAccessToken, userRefreshToken,
	)
	if err != nil {
		return nil, fmt.Errorf("insert install: %w", err)
	}
	id, _ := result.LastInsertId()
	return &model.GuildInstall{
		ID:               id,
		BotID:            botID,
		GuildID:          guildID,
		GuildName:        guildName,
		MemberCount:      memberCount,
		UserAccessToken:  userAccessToken,
		UserRefreshToken: userRefreshToken,
		InstalledAt:      time.Now(),
	}, nil
}

// ListInstalls returns all installation records with bot name.
func (s *Store) ListInstalls() ([]model.GuildInstall, error) {
	rows, err := s.db.Query(`
		SELECT gi.id, gi.bot_id, b.name, gi.guild_id, gi.guild_name, gi.member_count, gi.user_access_token, gi.user_refresh_token, gi.installed_at
		FROM guild_installs gi
		JOIN bots b ON b.id = gi.bot_id
		ORDER BY gi.installed_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query installs: %w", err)
	}
	defer rows.Close()

	var installs []model.GuildInstall
	for rows.Next() {
		var gi model.GuildInstall
		if err := rows.Scan(&gi.ID, &gi.BotID, &gi.BotName, &gi.GuildID, &gi.GuildName, &gi.MemberCount, &gi.UserAccessToken, &gi.UserRefreshToken, &gi.InstalledAt); err != nil {
			return nil, fmt.Errorf("scan install: %w", err)
		}
		installs = append(installs, gi)
	}
	return installs, rows.Err()
}

// ListInstallsByBot returns installation records for a specific bot.
func (s *Store) ListInstallsByBot(botID int64) ([]model.GuildInstall, error) {
	rows, err := s.db.Query(`
		SELECT gi.id, gi.bot_id, b.name, gi.guild_id, gi.guild_name, gi.member_count, gi.user_access_token, gi.user_refresh_token, gi.installed_at
		FROM guild_installs gi
		JOIN bots b ON b.id = gi.bot_id
		WHERE gi.bot_id = ?
		ORDER BY gi.installed_at DESC`, botID)
	if err != nil {
		return nil, fmt.Errorf("query installs by bot: %w", err)
	}
	defer rows.Close()

	var installs []model.GuildInstall
	for rows.Next() {
		var gi model.GuildInstall
		if err := rows.Scan(&gi.ID, &gi.BotID, &gi.BotName, &gi.GuildID, &gi.GuildName, &gi.MemberCount, &gi.UserAccessToken, &gi.UserRefreshToken, &gi.InstalledAt); err != nil {
			return nil, fmt.Errorf("scan install: %w", err)
		}
		installs = append(installs, gi)
	}
	return installs, rows.Err()
}

// GetInstall returns an installation record by ID.
func (s *Store) GetInstall(id int64) (*model.GuildInstall, error) {
	row := s.db.QueryRow(`
		SELECT gi.id, gi.bot_id, b.name, gi.guild_id, gi.guild_name, gi.member_count, gi.user_access_token, gi.user_refresh_token, gi.installed_at
		FROM guild_installs gi
		JOIN bots b ON b.id = gi.bot_id
		WHERE gi.id = ?`, id)

	var gi model.GuildInstall
	if err := row.Scan(&gi.ID, &gi.BotID, &gi.BotName, &gi.GuildID, &gi.GuildName, &gi.MemberCount, &gi.UserAccessToken, &gi.UserRefreshToken, &gi.InstalledAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get install: %w", err)
	}
	return &gi, nil
}

// UpdateInstallGuildInfo updates guild display info for an install.
func (s *Store) UpdateInstallGuildInfo(id int64, guildName string, memberCount int) error {
	result, err := s.db.Exec(`UPDATE guild_installs SET guild_name = ?, member_count = ? WHERE id = ?`, guildName, memberCount, id)
	if err != nil {
		return fmt.Errorf("update install guild info: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update install guild info rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteInstall deletes an install record.
func (s *Store) DeleteInstall(id int64) error {
	result, err := s.db.Exec(`DELETE FROM guild_installs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete install: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete install rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// IsGuildBlacklisted reports whether the guild is blacklisted for the bot.
func (s *Store) IsGuildBlacklisted(botID int64, guildID string) (bool, error) {
	var exists int
	if err := s.db.QueryRow(`SELECT 1 FROM guild_blacklist WHERE bot_id = ? AND guild_id = ? LIMIT 1`, botID, guildID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check guild blacklist: %w", err)
	}
	return true, nil
}

// AddGuildBlacklist inserts a blacklist entry if it does not already exist.
func (s *Store) AddGuildBlacklist(botID int64, guildID, guildName string) error {
	if _, err := s.db.Exec(`
		INSERT INTO guild_blacklist (bot_id, guild_id, guild_name)
		VALUES (?, ?, ?)
		ON CONFLICT(bot_id, guild_id) DO UPDATE SET guild_name = excluded.guild_name`,
		botID, guildID, guildName,
	); err != nil {
		return fmt.Errorf("insert guild blacklist: %w", err)
	}
	return nil
}

// ListGuildBlacklist returns all blacklist records with bot name.
func (s *Store) ListGuildBlacklist() ([]model.GuildBlacklist, error) {
	rows, err := s.db.Query(`
		SELECT gb.id, gb.bot_id, b.name, gb.guild_id, gb.guild_name, gb.created_at
		FROM guild_blacklist gb
		JOIN bots b ON b.id = gb.bot_id
		ORDER BY gb.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query guild blacklist: %w", err)
	}
	defer rows.Close()

	var list []model.GuildBlacklist
	for rows.Next() {
		var item model.GuildBlacklist
		if err := rows.Scan(&item.ID, &item.BotID, &item.BotName, &item.GuildID, &item.GuildName, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan guild blacklist: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}
