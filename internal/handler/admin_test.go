package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"dcportal/internal/discord"
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
