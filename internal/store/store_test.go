package store

import (
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
