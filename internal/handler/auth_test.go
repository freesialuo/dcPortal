package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func setupAuthHandler(t *testing.T) *AuthHandler {
	t.Helper()
	tmpl := template.Must(template.New("layout.html").Parse(`{{block "content" .}}{{.Error}}|{{.Next}}{{end}}`))
	tmpl = template.Must(tmpl.New("login.html").Parse(`{{define "content"}}{{.Error}}|{{.Next}}{{end}}`))
	return NewAuthHandler(tmpl, "secret-token")
}

func TestAuthIndexRequiresLogin(t *testing.T) {
	h := setupAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthIndexRedirectWhenTokenPresent(t *testing.T) {
	h := setupAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "admin_token", Value: "secret-token"})
	w := httptest.NewRecorder()
	h.index(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if got := w.Header().Get("Location"); got != "/admin" {
		t.Fatalf("Location = %q, want /admin", got)
	}
}

func TestAuthLoginSuccessSetsCookieAndRedirects(t *testing.T) {
	h := setupAuthHandler(t)

	form := url.Values{
		"token": {"secret-token"},
		"next":  {"/portal"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.login(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if got := w.Header().Get("Location"); got != "/portal" {
		t.Fatalf("Location = %q, want /portal", got)
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "admin_token=") {
		t.Fatalf("Set-Cookie should contain admin_token, got %q", w.Header().Get("Set-Cookie"))
	}
}

func TestAuthLoginRejectsInvalidToken(t *testing.T) {
	h := setupAuthHandler(t)

	form := url.Values{
		"token": {"wrong-token"},
		"next":  {"/portal"},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(w.Body.String(), "Invalid ADMIN token") {
		t.Fatalf("body should include invalid token message, got %q", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "/portal") {
		t.Fatalf("body should keep next path on invalid token, got %q", w.Body.String())
	}
}

func TestAuthLogoutClearsCookie(t *testing.T) {
	h := setupAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	w := httptest.NewRecorder()
	h.logout(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if got := w.Header().Get("Location"); got != "/" {
		t.Fatalf("Location = %q, want /", got)
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age=0") &&
		!strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age=-1") {
		t.Fatalf("Set-Cookie should clear admin_token, got %q", w.Header().Get("Set-Cookie"))
	}
}

func TestSanitizeNextPath(t *testing.T) {
	if got := sanitizeNextPath("https://evil.com"); got != "/admin" {
		t.Fatalf("sanitize external = %q, want /admin", got)
	}
	if got := sanitizeNextPath("//evil"); got != "/admin" {
		t.Fatalf("sanitize protocol-relative = %q, want /admin", got)
	}
	if got := sanitizeNextPath("/portal"); got != "/portal" {
		t.Fatalf("sanitize local path = %q, want /portal", got)
	}
}
