package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"dcportal/internal/discord"
	"dcportal/internal/model"
	"dcportal/internal/store"
)

func repoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func parseTemplates(t *testing.T, names ...string) *template.Template {
	t.Helper()
	root := repoRootFromThisFile(t)
	files := []string{filepath.Join(root, "web", "templates", "layout.html")}
	for _, name := range names {
		files = append(files, filepath.Join(root, "web", "templates", name))
	}
	tmpl, err := template.ParseFiles(files...)
	if err != nil {
		t.Fatalf("parse templates %v: %v", names, err)
	}
	return tmpl
}

func newSmokeStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAdminPageSmoke(t *testing.T) {
	s := newSmokeStore(t)
	tmpl := parseTemplates(t, "admin.html")
	dc := discord.NewClientWithBase("http://127.0.0.1", http.DefaultClient)
	h := NewAdminHandler(s, tmpl, dc)

	bot := &model.Bot{
		Name:         "SmokeBot",
		ClientID:     "123456789012345678",
		ClientSecret: "secret-1",
		BotToken:     "bot-token-1",
		Permissions:  "8",
		Scopes:       "bot applications.commands",
		RedirectURI:  "https://example.com/callback",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	customLink := &model.InstallLink{
		BotID:       bot.ID,
		Name:        "Full Access",
		Permissions: "8",
		Scopes:      "bot applications.commands",
		RedirectURI: "https://example.com/callback",
		Enabled:     true,
	}
	if err := s.CreateInstallLink(customLink); err != nil {
		t.Fatalf("CreateInstallLink: %v", err)
	}
	if _, err := s.RecordInstallWithLink(bot.ID, customLink.ID, customLink.Name, "guild-123", "Guild One", 42, "user-access", "user-refresh"); err != nil {
		t.Fatalf("RecordInstallWithLink: %v", err)
	}
	if err := s.AddGuildBlacklist(bot.ID, "guild-999", "Guild Blocked"); err != nil {
		t.Fatalf("AddGuildBlacklist: %v", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	want := []string{
		"Discord Control Portal",
		"Security model: Client Secret and Bot Token are write-only",
		"Client Secret: Configured",
		"Bot Token: Configured",
		"Install Links",
		"Copy Link URL",
		"Installed Guilds",
		"Guild Blacklist",
	}
	for _, token := range want {
		if !strings.Contains(body, token) {
			t.Fatalf("admin page missing %q", token)
		}
	}
}

func TestPortalPageSmoke(t *testing.T) {
	s := newSmokeStore(t)
	portalTmpl := parseTemplates(t, "portal.html")
	resultTmpl := parseTemplates(t, "result.html")
	dc := discord.NewClientWithBase("http://127.0.0.1", http.DefaultClient)
	h := NewPortalHandler(s, portalTmpl, resultTmpl, dc)

	bot := &model.Bot{
		Name:         "PortalBot",
		ClientID:     "987654321012345678",
		ClientSecret: "secret-2",
		Permissions:  "8",
		Scopes:       "bot applications.commands",
		RedirectURI:  "https://example.com/callback",
		Enabled:      true,
	}
	if err := s.CreateBot(bot); err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/portal", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	want := []string{
		"Install Portal",
		"Install Bot",
		"PortalBot",
		"Scopes:",
		"Permissions:",
	}
	for _, token := range want {
		if !strings.Contains(body, token) {
			t.Fatalf("portal page missing %q", token)
		}
	}
}

func TestLoginPagesSmoke(t *testing.T) {
	tmpl := parseTemplates(t, "login.html")

	t.Run("install login", func(t *testing.T) {
		auth := NewAuthHandler(tmpl, "install-secret")
		mux := http.NewServeMux()
		auth.RegisterRoutes(mux)

		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		want := []string{"Install Portal Login", "page-hero-compact", "Continue"}
		for _, token := range want {
			if !strings.Contains(body, token) {
				t.Fatalf("install login page missing %q", token)
			}
		}
	})

	t.Run("admin login", func(t *testing.T) {
		admin := NewAdminLoginHandler(tmpl, "admin-secret")
		mux := http.NewServeMux()
		admin.RegisterRoutes(mux)

		req := httptest.NewRequest("GET", "/admin/login", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		want := []string{"Admin Login", "page-hero-compact", "Continue"}
		for _, token := range want {
			if !strings.Contains(body, token) {
				t.Fatalf("admin login page missing %q", token)
			}
		}
	})
}

func TestResultTemplateSmoke(t *testing.T) {
	resultTmpl := parseTemplates(t, "result.html")
	h := &PortalHandler{resultTmpl: resultTmpl}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.renderResult(w, true, "ok", "BotX", "GuildX", "123")

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		want := []string{"Install Result", "Success", "Back To Portal", "Guild ID"}
		for _, token := range want {
			if !strings.Contains(body, token) {
				t.Fatalf("success result page missing %q", token)
			}
		}
	})

	t.Run("failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		h.renderResult(w, false, "failed", "", "", "")

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		want := []string{"Install Result", "Failed", "Back To Portal"}
		for _, token := range want {
			if !strings.Contains(body, token) {
				t.Fatalf("failure result page missing %q", token)
			}
		}
	})
}

func TestStylesheetSmoke(t *testing.T) {
	root := repoRootFromThisFile(t)
	stylePath := filepath.Join(root, "web", "static", "style.css")
	content, err := os.ReadFile(stylePath)
	if err != nil {
		t.Fatalf("read stylesheet: %v", err)
	}
	css := string(content)
	want := []string{
		".page-hero",
		".status-chip",
		".copy-link-btn",
		"@media (max-width: 720px)",
	}
	for _, token := range want {
		if !strings.Contains(css, token) {
			t.Fatalf("stylesheet missing %q", token)
		}
	}
}

func TestLayoutTemplateNoRemoteFonts(t *testing.T) {
	root := repoRootFromThisFile(t)
	layoutPath := filepath.Join(root, "web", "templates", "layout.html")
	content, err := os.ReadFile(layoutPath)
	if err != nil {
		t.Fatalf("read layout template: %v", err)
	}
	layout := string(content)
	disallow := []string{
		"fonts.googleapis.com",
		"fonts.gstatic.com",
	}
	for _, token := range disallow {
		if strings.Contains(layout, token) {
			t.Fatalf("layout should not reference remote font host %q", token)
		}
	}
}
