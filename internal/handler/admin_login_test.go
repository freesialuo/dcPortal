package handler

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func setupAdminLoginHandler(t *testing.T) *AdminLoginHandler {
	t.Helper()
	tmpl := template.Must(template.New("layout.html").Parse(`{{block "content" .}}{{.Error}}|{{.Next}}{{end}}`))
	tmpl = template.Must(tmpl.New("login.html").Parse(`{{define "content"}}{{.Error}}|{{.Next}}{{end}}`))
	return NewAdminLoginHandler(tmpl, "admin-secret")
}

func TestAdminLoginIndexRequiresLogin(t *testing.T) {
	h := setupAdminLoginHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	w := httptest.NewRecorder()
	h.index(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminLoginIndexRedirectWhenAdminTokenPresent(t *testing.T) {
	h := setupAdminLoginHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	req.AddCookie(&http.Cookie{Name: "admin_token", Value: "admin-secret"})
	w := httptest.NewRecorder()
	h.index(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if got := w.Header().Get("Location"); got != "/admin" {
		t.Fatalf("Location = %q, want /admin", got)
	}
}

func TestAdminLoginSuccessSetsCookieAndRedirects(t *testing.T) {
	h := setupAdminLoginHandler(t)

	form := url.Values{
		"token": {"admin-secret"},
		"next":  {"/admin"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.login(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if got := w.Header().Get("Location"); got != "/admin" {
		t.Fatalf("Location = %q, want /admin", got)
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "admin_token=") {
		t.Fatalf("Set-Cookie should contain admin_token, got %q", w.Header().Get("Set-Cookie"))
	}
}

func TestAdminLoginRejectsInvalidToken(t *testing.T) {
	h := setupAdminLoginHandler(t)

	form := url.Values{
		"token": {"wrong"},
		"next":  {"/admin"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(w.Body.String(), "Invalid ADMIN token") {
		t.Fatalf("body = %q", w.Body.String())
	}
}

func TestAdminLogoutClearsCookie(t *testing.T) {
	h := setupAdminLoginHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	w := httptest.NewRecorder()
	h.logout(w, req)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if got := w.Header().Get("Location"); got != "/admin/login" {
		t.Fatalf("Location = %q, want /admin/login", got)
	}
}
