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

func TestAdminAuthQueryParam(t *testing.T) {
	token := "secret-token"

	handler := AdminAuth(token)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/admin?token="+token, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("query param auth: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Should set cookie
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "admin_token" && c.Value == token {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected admin_token cookie to be set")
	}
}
