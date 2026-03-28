package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	"dcportal/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testBot(name, clientID string) *model.Bot {
	return &model.Bot{
		Name:         name,
		ClientID:     clientID,
		ClientSecret: "secret-" + clientID,
		Permissions:  "8",
		Scopes:       "bot",
		RedirectURI:  "http://localhost:8080/callback",
		Enabled:      true,
	}
}

func TestCreateAndListBots(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("TestBot", "123456")
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	if bot.ID == 0 {
		t.Fatal("expected bot ID to be set")
	}

	bots, err := s.ListBots()
	if err != nil {
		t.Fatalf("ListBots: %v", err)
	}
	if len(bots) != 1 {
		t.Fatalf("expected 1 bot, got %d", len(bots))
	}
	if bots[0].Name != "TestBot" {
		t.Errorf("Name = %q, want %q", bots[0].Name, "TestBot")
	}
	if bots[0].ClientSecret != "secret-123456" {
		t.Errorf("ClientSecret = %q, want %q", bots[0].ClientSecret, "secret-123456")
	}
	if bots[0].RedirectURI != "http://localhost:8080/callback" {
		t.Errorf("RedirectURI = %q", bots[0].RedirectURI)
	}

	links, err := s.ListInstallLinksByBot(bot.ID)
	if err != nil {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 default install link, got %d", len(links))
	}
	if links[0].Name != "Default" {
		t.Errorf("default link name = %q", links[0].Name)
	}
}

func TestGetBot(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("Bot1", "111")
	s.CreateBot(bot)

	got, err := s.GetBot(bot.ID)
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}
	if got == nil {
		t.Fatal("expected bot, got nil")
	}
	if got.Name != "Bot1" {
		t.Errorf("Name = %q, want %q", got.Name, "Bot1")
	}

	// Non-existent
	got, err = s.GetBot(999)
	if err != nil {
		t.Fatalf("GetBot(999): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent bot, got %+v", got)
	}
}

func TestToggleBot(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("Bot1", "111")
	s.CreateBot(bot)

	if err := s.ToggleBot(bot.ID); err != nil {
		t.Fatalf("ToggleBot: %v", err)
	}

	got, _ := s.GetBot(bot.ID)
	if got.Enabled {
		t.Error("expected bot to be disabled after toggle")
	}

	s.ToggleBot(bot.ID)
	got, _ = s.GetBot(bot.ID)
	if !got.Enabled {
		t.Error("expected bot to be enabled after second toggle")
	}
}

func TestUpdateBot(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("Bot1", "111")
	bot.BotToken = "bot-token-1"
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	bot.Name = "Bot1-Updated"
	bot.ClientID = "222"
	bot.ClientSecret = "secret-222"
	bot.BotToken = ""
	bot.Permissions = "16"
	bot.Scopes = "bot applications.commands"
	bot.RedirectURI = "https://example.com/callback"
	bot.Enabled = false

	if err := s.UpdateBot(bot); err != nil {
		t.Fatalf("UpdateBot: %v", err)
	}

	got, err := s.GetBot(bot.ID)
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}
	if got == nil {
		t.Fatal("expected bot, got nil")
	}
	if got.Name != "Bot1-Updated" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.ClientID != "222" {
		t.Errorf("ClientID = %q", got.ClientID)
	}
	if got.ClientSecret != "secret-222" {
		t.Errorf("ClientSecret = %q", got.ClientSecret)
	}
	if got.BotToken != "" {
		t.Errorf("BotToken = %q, want empty", got.BotToken)
	}
	if got.Permissions != "16" {
		t.Errorf("Permissions = %q", got.Permissions)
	}
	if got.Scopes != "bot applications.commands" {
		t.Errorf("Scopes = %q", got.Scopes)
	}
	if got.RedirectURI != "https://example.com/callback" {
		t.Errorf("RedirectURI = %q", got.RedirectURI)
	}
	if got.Enabled {
		t.Error("Enabled = true, want false")
	}
}

func TestUpdateBotNotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.UpdateBot(&model.Bot{ID: 99999, Name: "missing", ClientID: "1", ClientSecret: "2", Scopes: "bot"})
	if err == nil {
		t.Fatal("expected error for missing bot")
	}
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestToggleBotNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.ToggleBot(99999)
	if err == nil {
		t.Fatal("expected error for missing bot")
	}
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteBot(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("Bot1", "111")
	s.CreateBot(bot)

	// Add an install record to verify cascade delete
	s.RecordInstall(bot.ID, "guild-1", "Test Guild", 0, "", "")

	if err := s.DeleteBot(bot.ID); err != nil {
		t.Fatalf("DeleteBot: %v", err)
	}

	bots, _ := s.ListBots()
	if len(bots) != 0 {
		t.Errorf("expected 0 bots after delete, got %d", len(bots))
	}

	installs, _ := s.ListInstallsByBot(bot.ID)
	if len(installs) != 0 {
		t.Errorf("expected 0 installs after delete, got %d", len(installs))
	}
}

func TestDeleteBotNotFound(t *testing.T) {
	s := newTestStore(t)

	err := s.DeleteBot(99999)
	if err == nil {
		t.Fatal("expected error for missing bot")
	}
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListEnabledBots(t *testing.T) {
	s := newTestStore(t)

	s.CreateBot(testBot("Enabled1", "aaa"))

	disabled := testBot("Disabled1", "bbb")
	disabled.Enabled = false
	s.CreateBot(disabled)

	s.CreateBot(testBot("Enabled2", "ccc"))

	bots, err := s.ListEnabledBots()
	if err != nil {
		t.Fatalf("ListEnabledBots: %v", err)
	}
	if len(bots) != 2 {
		t.Errorf("expected 2 enabled bots, got %d", len(bots))
	}
}

func TestRecordAndListInstalls(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("Bot1", "111")
	s.CreateBot(bot)

	gi, err := s.RecordInstall(bot.ID, "guild-123", "My Server", 123, "access-1", "refresh-1")
	if err != nil {
		t.Fatalf("RecordInstall: %v", err)
	}
	if gi.GuildID != "guild-123" {
		t.Errorf("GuildID = %q", gi.GuildID)
	}
	if gi.MemberCount != 123 {
		t.Errorf("MemberCount = %d", gi.MemberCount)
	}

	// Add another install
	s.RecordInstall(bot.ID, "guild-456", "Other Server", 456, "", "")

	installs, err := s.ListInstalls()
	if err != nil {
		t.Fatalf("ListInstalls: %v", err)
	}
	if len(installs) != 2 {
		t.Errorf("expected 2 installs, got %d", len(installs))
	}
	// Installs sorted DESC by installed_at; both may have the same timestamp
	// so just verify both are present
	guildIDs := map[string]bool{}
	for _, gi := range installs {
		guildIDs[gi.GuildID] = true
		if gi.BotName != "Bot1" {
			t.Errorf("BotName = %q, expected Bot1", gi.BotName)
		}
	}
	if !guildIDs["guild-123"] || !guildIDs["guild-456"] {
		t.Errorf("expected both guilds, got %v", guildIDs)
	}

	// List by bot
	byBot, _ := s.ListInstallsByBot(bot.ID)
	if len(byBot) != 2 {
		t.Errorf("expected 2 installs for bot, got %d", len(byBot))
	}
}

func TestGuildBlacklist(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("Bot1", "111")
	s.CreateBot(bot)

	blocked, err := s.IsGuildBlacklisted(bot.ID, "guild-123")
	if err != nil {
		t.Fatalf("IsGuildBlacklisted: %v", err)
	}
	if blocked {
		t.Fatal("guild should not be blocked initially")
	}

	if err := s.AddGuildBlacklist(bot.ID, "guild-123", "My Server"); err != nil {
		t.Fatalf("AddGuildBlacklist: %v", err)
	}

	blocked, err = s.IsGuildBlacklisted(bot.ID, "guild-123")
	if err != nil {
		t.Fatalf("IsGuildBlacklisted after insert: %v", err)
	}
	if !blocked {
		t.Fatal("guild should be blocked")
	}

	list, err := s.ListGuildBlacklist()
	if err != nil {
		t.Fatalf("ListGuildBlacklist: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 blacklist record, got %d", len(list))
	}
	if list[0].GuildID != "guild-123" {
		t.Errorf("GuildID = %q", list[0].GuildID)
	}
}

func TestInstallLinkCRUD(t *testing.T) {
	s := newTestStore(t)

	bot := testBot("Bot1", "111")
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	link := &model.InstallLink{
		BotID:       bot.ID,
		Name:        "Limited",
		Permissions: "0",
		Scopes:      "bot",
		RedirectURI: "https://example.com/callback",
		Enabled:     true,
	}
	if err := s.CreateInstallLink(link); err != nil {
		t.Fatalf("CreateInstallLink: %v", err)
	}

	got, err := s.GetInstallLink(link.ID)
	if err != nil {
		t.Fatalf("GetInstallLink: %v", err)
	}
	if got == nil || got.Name != "Limited" {
		t.Fatalf("unexpected link: %+v", got)
	}

	got.Name = "Limited-Updated"
	got.Enabled = false
	if err := s.UpdateInstallLink(got); err != nil {
		t.Fatalf("UpdateInstallLink: %v", err)
	}

	if err := s.ToggleInstallLink(got.ID); err != nil {
		t.Fatalf("ToggleInstallLink: %v", err)
	}
	got, _ = s.GetInstallLink(got.ID)
	if !got.Enabled {
		t.Fatalf("expected link enabled after toggle")
	}

	links, err := s.ListInstallLinksByBot(bot.ID)
	if err != nil {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 links (default + custom), got %d", len(links))
	}
	allLinks, err := s.ListInstallLinks()
	if err != nil {
		t.Fatalf("ListInstallLinks: %v", err)
	}
	if len(allLinks) != 2 {
		t.Fatalf("expected 2 total install links, got %d", len(allLinks))
	}

	if err := s.DeleteInstallLink(got.ID); err != nil {
		t.Fatalf("DeleteInstallLink: %v", err)
	}
}

func TestCreateBotDefaultLinkDisabledWithoutRedirect(t *testing.T) {
	s := newTestStore(t)

	bot := &model.Bot{
		Name:         "BotNoRedirect",
		ClientID:     "no-redirect-client",
		ClientSecret: "secret",
		Scopes:       "bot",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	links, err := s.ListInstallLinksByBot(bot.ID)
	if err != nil {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 default link, got %d", len(links))
	}
	if links[0].Enabled {
		t.Fatalf("expected default link disabled when redirect URI is missing")
	}
}

func TestMigrateSeedsDisabledDefaultWhenRedirectMissing(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	rawDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	legacySetup := []string{
		`CREATE TABLE bots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			client_id TEXT NOT NULL UNIQUE,
			client_secret TEXT NOT NULL DEFAULT '',
			bot_token TEXT NOT NULL DEFAULT '',
			permissions TEXT NOT NULL DEFAULT '',
			scopes TEXT NOT NULL DEFAULT 'bot',
			redirect_uri TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE guild_installs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id INTEGER NOT NULL,
			guild_id TEXT NOT NULL,
			guild_name TEXT NOT NULL DEFAULT '',
			installed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE guild_blacklist (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			bot_id INTEGER NOT NULL,
			guild_id TEXT NOT NULL,
			guild_name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(bot_id, guild_id)
		)`,
		`INSERT INTO bots (name, client_id, client_secret, redirect_uri, enabled) VALUES ('LegacyBot', 'legacy-1', 'secret', '', 1)`,
	}
	for _, stmt := range legacySetup {
		if _, err := rawDB.Exec(stmt); err != nil {
			rawDB.Close()
			t.Fatalf("legacy setup exec failed: %v", err)
		}
	}
	if err := rawDB.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New store migrate: %v", err)
	}
	defer s.Close()

	bots, err := s.ListBots()
	if err != nil || len(bots) != 1 {
		t.Fatalf("ListBots: err=%v len=%d", err, len(bots))
	}
	links, err := s.ListInstallLinksByBot(bots[0].ID)
	if err != nil {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 seeded default link, got %d", len(links))
	}
	if links[0].Enabled {
		t.Fatalf("expected seeded default link disabled when redirect URI is missing")
	}
}

func TestMigrateDoesNotRecreateDefaultLinkAfterManualDelete(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "dcportal.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New store: %v", err)
	}

	bot := testBot("Bot1", "111")
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	links, err := s.ListInstallLinksByBot(bot.ID)
	if err != nil {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 default link, got %d", len(links))
	}
	if err := s.DeleteInstallLink(links[0].ID); err != nil {
		t.Fatalf("DeleteInstallLink: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := New(dbPath)
	if err != nil {
		t.Fatalf("New store reopen: %v", err)
	}
	defer s2.Close()

	links, err = s2.ListInstallLinksByBot(bot.ID)
	if err != nil {
		t.Fatalf("ListInstallLinksByBot after reopen: %v", err)
	}
	if len(links) != 0 {
		t.Fatalf("expected 0 links after reopen, got %d", len(links))
	}
}
