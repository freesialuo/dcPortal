package config

import (
	"os"
	"path/filepath"
	"testing"
)

func testTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "dcportal-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestLoad(t *testing.T) {
	dir := testTempDir(t)
	cfgFile := filepath.Join(dir, "config.yaml")

	content := `
server:
  port: 9090
admin:
  token: "test-secret-token"
install:
  token: "install-secret-token"
database:
  path: "./test.db"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Admin.Token != "test-secret-token" {
		t.Errorf("Token = %q, want %q", cfg.Admin.Token, "test-secret-token")
	}
	if cfg.Install.Token != "install-secret-token" {
		t.Errorf("Install.Token = %q, want %q", cfg.Install.Token, "install-secret-token")
	}
	if cfg.Database.Path != "./test.db" {
		t.Errorf("Path = %q, want %q", cfg.Database.Path, "./test.db")
	}
}

func TestLoadEnvOverride(t *testing.T) {
	dir := testTempDir(t)
	cfgFile := filepath.Join(dir, "config.yaml")

	content := `
server:
  port: 8080
admin:
  token: "original-token"
install:
  token: "original-install-token"
database:
  path: "./data.db"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DCPORTAL_PORT", "3000")
	t.Setenv("DCPORTAL_ADMIN_TOKEN", "env-token")
	t.Setenv("DCPORTAL_INSTALL_TOKEN", "env-install-token")
	t.Setenv("DCPORTAL_DB_PATH", "/tmp/override.db")

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.Server.Port)
	}
	if cfg.Admin.Token != "env-token" {
		t.Errorf("Token = %q, want %q", cfg.Admin.Token, "env-token")
	}
	if cfg.Install.Token != "env-install-token" {
		t.Errorf("Install.Token = %q, want %q", cfg.Install.Token, "env-install-token")
	}
	if cfg.Database.Path != "/tmp/override.db" {
		t.Errorf("Path = %q, want %q", cfg.Database.Path, "/tmp/override.db")
	}
}

func TestLoadRejectsDefaultToken(t *testing.T) {
	dir := testTempDir(t)
	cfgFile := filepath.Join(dir, "config.yaml")

	content := `
admin:
  token: "change-me-to-a-secure-token"
install:
  token: "install-secret-token"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("Load() should reject default token")
	}
}

func TestLoadRejectsDefaultInstallToken(t *testing.T) {
	dir := testTempDir(t)
	cfgFile := filepath.Join(dir, "config.yaml")

	content := `
admin:
  token: "admin-secret-token"
install:
  token: "change-me-to-a-secure-token"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("Load() should reject default install token")
	}
}

func TestLoadRejectsInvalidPortOverride(t *testing.T) {
	dir := testTempDir(t)
	cfgFile := filepath.Join(dir, "config.yaml")

	content := `
admin:
  token: "test-secret-token"
install:
  token: "install-secret-token"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DCPORTAL_PORT", "not-a-number")
	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("Load() should reject invalid DCPORTAL_PORT")
	}
}
