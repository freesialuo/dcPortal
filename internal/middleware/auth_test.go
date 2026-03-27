package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAuthBearerToken(t *testing.T) {
	token := "secret-token"

	handler := AdminAuth(token)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	// Valid token
	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("valid token: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Invalid token
	req = httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	// No token
	req = httptest.NewRequest("GET", "/admin", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("no token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminAuthCookie(t *testing.T) {
	token := "secret-token"

	handler := AdminAuth(token)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "admin_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("cookie auth: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAdminAuthRejectsQueryParam(t *testing.T) {
	token := "secret-token"

	handler := AdminAuth(token)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin?token="+token, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("query param auth should be rejected: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminAuthWithRedirect(t *testing.T) {
	token := "secret-token"
	handler := AdminAuthWithRedirect(token, "/")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if got := w.Header().Get("Location"); got != "/?next=%2Fadmin" {
		t.Fatalf("Location = %q, want /?next=%%2Fadmin", got)
	}
}

func TestAdminAuthWithRedirectAllowsValidToken(t *testing.T) {
	token := "secret-token"
	handler := AdminAuthWithRedirect(token, "/")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "admin_token", Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
