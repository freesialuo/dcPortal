package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchangeCode(t *testing.T) {
	// Mock Discord API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", 404)
			return
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %q, want application/x-www-form-urlencoded", ct)
		}

		// Verify Basic Auth
		user, pass, ok := r.BasicAuth()
		if !ok || user != "test-client-id" || pass != "test-client-secret" {
			t.Errorf("basic auth: user=%q pass=%q ok=%v", user, pass, ok)
		}

		// Verify form data
		r.ParseForm()
		if r.FormValue("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.FormValue("grant_type"))
		}
		if r.FormValue("code") != "test-code" {
			t.Errorf("code = %q", r.FormValue("code"))
		}
		if r.FormValue("redirect_uri") != "http://localhost:8080/callback" {
			t.Errorf("redirect_uri = %q", r.FormValue("redirect_uri"))
		}

		// Return mock response
		resp := TokenResponse{
			AccessToken:  "mock-access-token",
			TokenType:    "Bearer",
			ExpiresIn:    604800,
			RefreshToken: "mock-refresh-token",
			Scope:        "bot",
			Guild: &GuildResponse{
				ID:   "123456789",
				Name: "Test Guild",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClientWithBase(server.URL, server.Client())
	resp, err := client.ExchangeCode("test-client-id", "test-client-secret", "test-code", "http://localhost:8080/callback")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}

	if resp.AccessToken != "mock-access-token" {
		t.Errorf("AccessToken = %q", resp.AccessToken)
	}
	if resp.Guild == nil {
		t.Fatal("expected guild info")
	}
	if resp.Guild.ID != "123456789" {
		t.Errorf("Guild.ID = %q", resp.Guild.ID)
	}
	if resp.Guild.Name != "Test Guild" {
		t.Errorf("Guild.Name = %q", resp.Guild.Name)
	}
}

func TestExchangeCodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()

	client := NewClientWithBase(server.URL, server.Client())
	_, err := client.ExchangeCode("id", "secret", "bad-code", "http://localhost/callback")
	if err == nil {
		t.Fatal("expected error for bad code")
	}
}

func TestRevokeToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2/token/revoke" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		r.ParseForm()
		if r.FormValue("token") != "some-token" {
			t.Errorf("token = %q", r.FormValue("token"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithBase(server.URL, server.Client())
	err := client.RevokeToken("id", "secret", "some-token")
	if err != nil {
		t.Fatalf("RevokeToken: %v", err)
	}
}
