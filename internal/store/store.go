package store

import (
	"database/sql"
	"errors"
	"fmt"
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
			permissions   TEXT NOT NULL DEFAULT '',
			scopes        TEXT NOT NULL DEFAULT 'bot',
			redirect_uri  TEXT NOT NULL DEFAULT '',
			enabled       INTEGER NOT NULL DEFAULT 1,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS guild_installs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id       INTEGER NOT NULL,
			guild_id     TEXT NOT NULL,
			guild_name   TEXT NOT NULL DEFAULT '',
			installed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (bot_id) REFERENCES bots(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_guild_installs_bot ON guild_installs(bot_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:40], err)
		}
	}
	return nil
}

// ---- Bot CRUD ----

// CreateBot inserts a new bot record.
func (s *Store) CreateBot(b *model.Bot) error {
	result, err := s.db.Exec(
		`INSERT INTO bots (name, client_id, client_secret, permissions, scopes, redirect_uri, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.Name, b.ClientID, b.ClientSecret, b.Permissions, b.Scopes, b.RedirectURI, b.Enabled,
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

const botColumns = `id, name, client_id, client_secret, permissions, scopes, redirect_uri, enabled, created_at`

func scanBot(sc interface{ Scan(...any) error }) (*model.Bot, error) {
	var b model.Bot
	err := sc.Scan(&b.ID, &b.Name, &b.ClientID, &b.ClientSecret, &b.Permissions, &b.Scopes, &b.RedirectURI, &b.Enabled, &b.CreatedAt)
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
func (s *Store) RecordInstall(botID int64, guildID, guildName string) (*model.GuildInstall, error) {
	result, err := s.db.Exec(
		"INSERT INTO guild_installs (bot_id, guild_id, guild_name) VALUES (?, ?, ?)",
		botID, guildID, guildName,
	)
	if err != nil {
		return nil, fmt.Errorf("insert install: %w", err)
	}
	id, _ := result.LastInsertId()
	return &model.GuildInstall{
		ID:          id,
		BotID:       botID,
		GuildID:     guildID,
		GuildName:   guildName,
		InstalledAt: time.Now(),
	}, nil
}

// ListInstalls returns all installation records with bot name.
func (s *Store) ListInstalls() ([]model.GuildInstall, error) {
	rows, err := s.db.Query(`
		SELECT gi.id, gi.bot_id, b.name, gi.guild_id, gi.guild_name, gi.installed_at
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
		if err := rows.Scan(&gi.ID, &gi.BotID, &gi.BotName, &gi.GuildID, &gi.GuildName, &gi.InstalledAt); err != nil {
			return nil, fmt.Errorf("scan install: %w", err)
		}
		installs = append(installs, gi)
	}
	return installs, rows.Err()
}

// ListInstallsByBot returns installation records for a specific bot.
func (s *Store) ListInstallsByBot(botID int64) ([]model.GuildInstall, error) {
	rows, err := s.db.Query(`
		SELECT gi.id, gi.bot_id, b.name, gi.guild_id, gi.guild_name, gi.installed_at
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
		if err := rows.Scan(&gi.ID, &gi.BotID, &gi.BotName, &gi.GuildID, &gi.GuildName, &gi.InstalledAt); err != nil {
			return nil, fmt.Errorf("scan install: %w", err)
		}
		installs = append(installs, gi)
	}
	return installs, rows.Err()
}
