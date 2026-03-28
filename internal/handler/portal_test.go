package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dcportal/internal/discord"
	"dcportal/internal/model"
	"dcportal/internal/store"
)

func newTestStore2(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newMockDiscord(t *testing.T) (*discord.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			resp := discord.TokenResponse{
				AccessToken:  "mock-token",
				TokenType:    "Bearer",
				ExpiresIn:    604800,
				RefreshToken: "mock-refresh",
				Scope:        "bot",
				Guild: &discord.GuildResponse{
					ID:   "guild-999",
					Name: "Mock Guild",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/oauth2/token/revoke" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "not found", 404)
	}))
	t.Cleanup(server.Close)
	return discord.NewClientWithBase(server.URL, server.Client()), server
}

func newMockDiscordNoGuild(t *testing.T) (*discord.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			resp := discord.TokenResponse{
				AccessToken:  "mock-token",
				TokenType:    "Bearer",
				ExpiresIn:    604800,
				RefreshToken: "mock-refresh",
				Scope:        "bot",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		if r.URL.Path == "/oauth2/token/revoke" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "not found", 404)
	}))
	t.Cleanup(server.Close)
	return discord.NewClientWithBase(server.URL, server.Client()), server
}

func testPortalTmpl() *template.Template {
	tmpl := template.Must(template.New("layout.html").Parse(
		`{{block "content" .}}{{end}}`))
	template.Must(tmpl.New("portal.html").Parse(
		`{{define "content"}}{{range .Links}}{{.BotName}}:{{.Name}},{{end}}{{end}}`))
	return tmpl
}

func testResultTmpl() *template.Template {
	tmpl := template.Must(template.New("layout.html").Parse(
		`{{block "content" .}}{{end}}`))
	template.Must(tmpl.New("result.html").Parse(
		`{{define "content"}}{{.Title}}|{{.Success}}|{{.Message}}{{end}}`))
	return tmpl
}

func TestPortalIndex(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	s.CreateBot(&model.Bot{Name: "Bot1", ClientID: "aaa", ClientSecret: "s1", Scopes: "bot", Enabled: true})
	s.CreateBot(&model.Bot{Name: "Bot2", ClientID: "bbb", ClientSecret: "s2", Scopes: "bot", Enabled: false})

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/portal", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "Bot1:Default," {
		t.Errorf("body = %q, want only enabled bot install link", body)
	}
}

func TestPortalInstallRedirect(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	s.CreateBot(&model.Bot{
		Name: "Bot1", ClientID: "123456", ClientSecret: "secret",
		Permissions: "8", Scopes: "bot", RedirectURI: "http://localhost:8080/callback", Enabled: true,
	})

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/install/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}

	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "client_id=123456") {
		t.Errorf("Location missing client_id: %q", loc)
	}
	if !strings.Contains(loc, "response_type=code") {
		t.Errorf("Location missing response_type=code: %q", loc)
	}
	if !strings.Contains(loc, "state=") {
		t.Errorf("Location missing state: %q", loc)
	}
	if !strings.Contains(loc, "redirect_uri=") {
		t.Errorf("Location missing redirect_uri: %q", loc)
	}
}

func TestPortalInstallDisabledBot(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	s.CreateBot(&model.Bot{Name: "Bot1", ClientID: "123456", ClientSecret: "s", Scopes: "bot", Enabled: false})

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/install/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d for disabled bot", w.Code, http.StatusNotFound)
	}
}

func TestPortalInstallMissingRedirectURI(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	bot := &model.Bot{
		Name:         "Bot1",
		ClientID:     "123456",
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
	links[0].RedirectURI = ""
	if err := s.UpdateInstallLink(&links[0]); err != nil {
		t.Fatalf("UpdateInstallLink: %v", err)
	}

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/install/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d for missing redirect URI", w.Code, http.StatusInternalServerError)
	}
}

func TestPortalCallback(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	s.CreateBot(&model.Bot{
		Name: "Bot1", ClientID: "123456", ClientSecret: "secret",
		Scopes: "bot", RedirectURI: "http://localhost:8080/callback", Enabled: true,
	})

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	// Use a fixed state for testing
	origGenState := generateState
	generateState = func() (string, error) { return "test-state-123", nil }
	t.Cleanup(func() { generateState = origGenState })

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Step 1: Trigger install to register the state
	req := httptest.NewRequest("GET", "/install/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("install redirect status = %d", w.Code)
	}

	// Step 2: Simulate Discord callback
	req = httptest.NewRequest("GET", "/callback?code=auth-code-123&state=test-state-123&guild_id=guild-999", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("callback status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "true") {
		t.Errorf("expected success in body: %q", body)
	}

	// Verify install was recorded
	installs, _ := s.ListInstalls()
	if len(installs) != 1 {
		t.Fatalf("expected 1 install, got %d", len(installs))
	}
	if installs[0].GuildID != "guild-999" {
		t.Errorf("GuildID = %q", installs[0].GuildID)
	}
	if installs[0].GuildName != "Mock Guild" {
		t.Errorf("GuildName = %q", installs[0].GuildName)
	}
	if installs[0].LinkID == 0 || installs[0].LinkName == "" {
		t.Errorf("install should record link info, got LinkID=%d LinkName=%q", installs[0].LinkID, installs[0].LinkName)
	}
}

func TestPortalCallbackInvalidState(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	s.CreateBot(&model.Bot{
		Name: "Bot1", ClientID: "123456", ClientSecret: "secret",
		Scopes: "bot", Enabled: true,
	})

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/callback?code=auth-code&state=unknown-state", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for invalid state", w.Code)
	}
}

func TestPortalCallbackStateIsSingleUse(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	s.CreateBot(&model.Bot{
		Name: "Bot1", ClientID: "123456", ClientSecret: "secret",
		Scopes: "bot", RedirectURI: "http://localhost:8080/callback", Enabled: true,
	})

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	origGenState := generateState
	generateState = func() (string, error) { return "one-time-state", nil }
	t.Cleanup(func() { generateState = origGenState })

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Trigger install
	req := httptest.NewRequest("GET", "/install/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// First callback — should succeed
	req = httptest.NewRequest("GET", "/callback?code=code1&state=one-time-state", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("first callback status = %d", w.Code)
	}

	// Second callback with same state — should fail (single use)
	req = httptest.NewRequest("GET", "/callback?code=code2&state=one-time-state", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("replay attack should be rejected, got status %d", w.Code)
	}
}

func TestPortalCallbackIgnoresUnverifiedGuildID(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscordNoGuild(t)

	s.CreateBot(&model.Bot{
		Name: "Bot1", ClientID: "123456", ClientSecret: "secret",
		Scopes: "bot", RedirectURI: "http://localhost:8080/callback", Enabled: true,
	})

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	origGenState := generateState
	generateState = func() (string, error) { return "test-state-no-guild", nil }
	t.Cleanup(func() { generateState = origGenState })

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/install/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("install redirect status = %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/callback?code=auth-code-123&state=test-state-no-guild&guild_id=forged-guild", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("callback status = %d, want 200", w.Code)
	}

	installs, err := s.ListInstalls()
	if err != nil {
		t.Fatalf("ListInstalls: %v", err)
	}
	if len(installs) != 0 {
		t.Fatalf("expected 0 installs when guild is not returned by Discord, got %d", len(installs))
	}
}

func TestPortalCallbackBlockedGuild(t *testing.T) {
	s := newTestStore2(t)
	dc, _ := newMockDiscord(t)

	bot := &model.Bot{
		Name:         "Bot1",
		ClientID:     "123456",
		ClientSecret: "secret",
		Scopes:       "bot",
		RedirectURI:  "http://localhost:8080/callback",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	if err := s.AddGuildBlacklist(bot.ID, "guild-999", "Mock Guild"); err != nil {
		t.Fatalf("AddGuildBlacklist: %v", err)
	}

	h := NewPortalHandler(s, testPortalTmpl(), testResultTmpl(), dc)

	origGenState := generateState
	generateState = func() (string, error) { return "blocked-state", nil }
	t.Cleanup(func() { generateState = origGenState })

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/install/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("install redirect status = %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/callback?code=auth-code-123&state=blocked-state", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("callback status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "false") {
		t.Fatalf("expected failed result page body, got: %s", w.Body.String())
	}

	installs, err := s.ListInstalls()
	if err != nil {
		t.Fatalf("ListInstalls: %v", err)
	}
	if len(installs) != 0 {
		t.Fatalf("expected 0 installs for blocked guild, got %d", len(installs))
	}
}
