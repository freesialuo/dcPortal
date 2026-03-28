package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"dcportal/internal/discord"
	"dcportal/internal/model"
	"dcportal/internal/store"
)

func setupAdminTest(t *testing.T) (*AdminHandler, *store.Store) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	tmpl := template.Must(template.New("layout.html").Parse(`{{block "content" .}}{{end}}`))
	tmpl = template.Must(tmpl.New("admin.html").Parse(
		`{{define "content"}}{{range .Bots}}{{.Name}},{{end}}{{end}}`))

	dc := discord.NewClientWithBase("http://127.0.0.1", http.DefaultClient)
	h := NewAdminHandler(s, tmpl, dc)
	return h, s
}

func TestAdminCreateBot(t *testing.T) {
	h, s := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	form := url.Values{
		"name":          {"MyBot"},
		"client_id":     {"12345"},
		"client_secret": {"my-secret"},
		"permissions":   {"8"},
		"scopes":        {"bot"},
		"redirect_uri":  {"http://localhost:8080/callback"},
	}

	req := httptest.NewRequest("POST", "/admin/bots", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	bots, _ := s.ListBots()
	if len(bots) != 1 || bots[0].Name != "MyBot" {
		t.Errorf("expected 1 bot named MyBot, got %v", bots)
	}
	if bots[0].ClientSecret != "my-secret" {
		t.Errorf("ClientSecret = %q", bots[0].ClientSecret)
	}
}

func TestAdminCreateBotRequiresFields(t *testing.T) {
	h, _ := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Missing client_secret
	form := url.Values{"name": {"Bot"}, "client_id": {"12345"}}
	req := httptest.NewRequest("POST", "/admin/bots", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAdminUpdateBot(t *testing.T) {
	h, s := setupAdminTest(t)

	bot := &model.Bot{
		Name:         "OldBot",
		ClientID:     "12345",
		ClientSecret: "old-secret",
		BotToken:     "old-token",
		Permissions:  "8",
		Scopes:       "bot",
		RedirectURI:  "http://localhost:8080/callback",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	form := url.Values{
		"name":         {"NewBot"},
		"client_id":    {"67890"},
		"permissions":  {"16"},
		"scopes":       {"bot applications.commands"},
		"redirect_uri": {"https://example.com/callback"},
	}

	req := httptest.NewRequest("POST", "/admin/bots/"+strconv.FormatInt(bot.ID, 10)+"/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	got, err := s.GetBot(bot.ID)
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}
	if got.Name != "NewBot" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.ClientID != "67890" {
		t.Errorf("ClientID = %q", got.ClientID)
	}
	if got.ClientSecret != "old-secret" {
		t.Errorf("ClientSecret should be preserved, got %q", got.ClientSecret)
	}
	if got.BotToken != "old-token" {
		t.Errorf("BotToken should be preserved, got %q", got.BotToken)
	}
	if got.Scopes != "bot applications.commands" {
		t.Errorf("Scopes = %q", got.Scopes)
	}
}

func TestAdminUpdateBotNotFound(t *testing.T) {
	h, _ := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	form := url.Values{
		"name":          {"Bot"},
		"client_id":     {"123"},
		"client_secret": {"secret"},
	}
	req := httptest.NewRequest("POST", "/admin/bots/999/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAdminUpdateBotClearToken(t *testing.T) {
	h, s := setupAdminTest(t)

	bot := &model.Bot{
		Name:         "OldBot",
		ClientID:     "12345",
		ClientSecret: "old-secret",
		BotToken:     "old-token",
		Scopes:       "bot",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	form := url.Values{
		"name":            {"OldBot"},
		"client_id":       {"12345"},
		"client_secret":   {"old-secret"},
		"clear_bot_token": {"1"},
	}
	req := httptest.NewRequest("POST", "/admin/bots/"+strconv.FormatInt(bot.ID, 10)+"/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	got, err := s.GetBot(bot.ID)
	if err != nil {
		t.Fatalf("GetBot: %v", err)
	}
	if got.BotToken != "" {
		t.Errorf("BotToken = %q, want empty", got.BotToken)
	}
}

func TestAdminCreateInstallLink(t *testing.T) {
	h, s := setupAdminTest(t)

	bot := &model.Bot{
		Name:         "Bot1",
		ClientID:     "12345",
		ClientSecret: "secret",
		RedirectURI:  "http://localhost:8080/callback",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	form := url.Values{
		"link_name":    {"High Perm"},
		"permissions":  {"8"},
		"scopes":       {"bot applications.commands"},
		"redirect_uri": {"http://localhost:8080/callback"},
	}
	req := httptest.NewRequest("POST", "/admin/bots/"+strconv.FormatInt(bot.ID, 10)+"/links", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	links, err := s.ListInstallLinksByBot(bot.ID)
	if err != nil {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected default link + new link, got %d", len(links))
	}
}

func TestAdminUpdateInstallLink(t *testing.T) {
	h, s := setupAdminTest(t)

	bot := &model.Bot{
		Name:         "Bot1",
		ClientID:     "12345",
		ClientSecret: "secret",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	links, err := s.ListInstallLinksByBot(bot.ID)
	if err != nil || len(links) == 0 {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	linkID := links[0].ID

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	form := url.Values{
		"link_name":    {"Updated Default"},
		"permissions":  {"16"},
		"scopes":       {"bot"},
		"redirect_uri": {"https://example.com/callback"},
	}
	req := httptest.NewRequest("POST", "/admin/links/"+strconv.FormatInt(linkID, 10)+"/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	updated, err := s.GetInstallLink(linkID)
	if err != nil {
		t.Fatalf("GetInstallLink: %v", err)
	}
	if updated.Name != "Updated Default" {
		t.Errorf("Name = %q", updated.Name)
	}
	if updated.Permissions != "16" {
		t.Errorf("Permissions = %q", updated.Permissions)
	}
}

func TestAdminToggleInstallLink(t *testing.T) {
	h, s := setupAdminTest(t)

	bot := &model.Bot{
		Name:         "Bot1",
		ClientID:     "12345",
		ClientSecret: "secret",
		RedirectURI:  "http://localhost:8080/callback",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	links, err := s.ListInstallLinksByBot(bot.ID)
	if err != nil || len(links) == 0 {
		t.Fatalf("ListInstallLinksByBot: %v", err)
	}
	linkID := links[0].ID

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/admin/links/"+strconv.FormatInt(linkID, 10)+"/toggle", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	got, err := s.GetInstallLink(linkID)
	if err != nil {
		t.Fatalf("GetInstallLink: %v", err)
	}
	if got.Enabled {
		t.Errorf("expected link disabled after toggle")
	}
}

func TestAdminToggleInstallLinkNotFound(t *testing.T) {
	h, _ := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/admin/links/999/toggle", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAdminDeleteInstallLink(t *testing.T) {
	h, s := setupAdminTest(t)

	bot := &model.Bot{
		Name:         "Bot1",
		ClientID:     "12345",
		ClientSecret: "secret",
		RedirectURI:  "http://localhost:8080/callback",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	link := &model.InstallLink{
		BotID:       bot.ID,
		Name:        "Extra",
		Permissions: "8",
		Scopes:      "bot",
		RedirectURI: "http://localhost:8080/callback",
		Enabled:     true,
	}
	if err := s.CreateInstallLink(link); err != nil {
		t.Fatalf("CreateInstallLink: %v", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	req := httptest.NewRequest("POST", "/admin/links/"+strconv.FormatInt(link.ID, 10)+"/delete", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	got, err := s.GetInstallLink(link.ID)
	if err != nil {
		t.Fatalf("GetInstallLink: %v", err)
	}
	if got != nil {
		t.Fatalf("expected link deleted, got %+v", got)
	}
}

func TestAdminDeleteInstallLinkNotFound(t *testing.T) {
	h, _ := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/admin/links/999/delete", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAdminIndex(t *testing.T) {
	h, _ := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminToggleBotNotFound(t *testing.T) {
	h, _ := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/admin/bots/999/toggle", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestAdminDeleteBotNotFound(t *testing.T) {
	h, _ := setupAdminTest(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("POST", "/admin/bots/999/delete", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
