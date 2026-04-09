package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── Load defaults ────────────────────────────────────────────────────────────

func TestLoad_NoSettingsFile_ReturnsDefaults(t *testing.T) {
	// Point the dojo dir at a temp directory that has no settings.json.
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	// Clear env overrides that might bleed in from the test environment.
	t.Setenv("DOJO_GATEWAY_URL", "")
	t.Setenv("DOJO_GATEWAY_TOKEN", "")
	t.Setenv("DOJO_PLUGINS_PATH", "")
	t.Setenv("DOJO_PROVIDER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if cfg.Gateway.URL != "http://localhost:7340" {
		t.Errorf("default Gateway.URL: got %q, want %q", cfg.Gateway.URL, "http://localhost:7340")
	}
	if cfg.Gateway.Timeout != "60s" {
		t.Errorf("default Gateway.Timeout: got %q, want %q", cfg.Gateway.Timeout, "60s")
	}
	if cfg.Defaults.Disposition != "balanced" {
		t.Errorf("default Disposition: got %q, want %q", cfg.Defaults.Disposition, "balanced")
	}
}

// ─── Load with file overrides ─────────────────────────────────────────────────

func TestLoad_WithSettingsFile_AppliesOverrides(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	// Ensure no env overrides interfere.
	t.Setenv("DOJO_GATEWAY_URL", "")
	t.Setenv("DOJO_GATEWAY_TOKEN", "")
	t.Setenv("DOJO_PLUGINS_PATH", "")
	t.Setenv("DOJO_PROVIDER", "")

	// Create ~/.dojo/settings.json with custom values.
	dojoDir := filepath.Join(tmp, ".dojo")
	if err := os.MkdirAll(dojoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	overrides := map[string]any{
		"gateway": map[string]any{
			"url":     "http://custom-gateway:8080",
			"timeout": "30s",
			"token":   "mytoken",
		},
		"defaults": map[string]any{
			"provider":    "openai",
			"disposition": "focused",
			"model":       "gpt-4",
		},
	}
	data, _ := json.Marshal(overrides)
	if err := os.WriteFile(filepath.Join(dojoDir, "settings.json"), data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Gateway.URL != "http://custom-gateway:8080" {
		t.Errorf("Gateway.URL: got %q, want %q", cfg.Gateway.URL, "http://custom-gateway:8080")
	}
	if cfg.Gateway.Timeout != "30s" {
		t.Errorf("Gateway.Timeout: got %q, want %q", cfg.Gateway.Timeout, "30s")
	}
	if cfg.Gateway.Token != "mytoken" {
		t.Errorf("Gateway.Token: got %q, want %q", cfg.Gateway.Token, "mytoken")
	}
	if cfg.Defaults.Provider != "openai" {
		t.Errorf("Defaults.Provider: got %q, want %q", cfg.Defaults.Provider, "openai")
	}
	if cfg.Defaults.Model != "gpt-4" {
		t.Errorf("Defaults.Model: got %q, want %q", cfg.Defaults.Model, "gpt-4")
	}
}

// ─── DojoDir ─────────────────────────────────────────────────────────────────

func TestDojoDir_ContainsDotDojo(t *testing.T) {
	d := DojoDir()
	if !strings.HasSuffix(d, ".dojo") {
		t.Errorf("DojoDir() = %q; expected it to end with '.dojo'", d)
	}
	// Must be an absolute path rooted at the real home dir.
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(d, home) {
		t.Errorf("DojoDir() = %q; expected prefix %q", d, home)
	}
}

// ─── Env var overrides ────────────────────────────────────────────────────────

func TestLoad_EnvVar_GatewayURL(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	t.Setenv("DOJO_GATEWAY_URL", "http://env-override:9999")
	t.Setenv("DOJO_GATEWAY_TOKEN", "")
	t.Setenv("DOJO_PLUGINS_PATH", "")
	t.Setenv("DOJO_PROVIDER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Gateway.URL != "http://env-override:9999" {
		t.Errorf("Gateway.URL after env override: got %q, want %q", cfg.Gateway.URL, "http://env-override:9999")
	}
}

func TestLoad_EnvVar_Token(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	t.Setenv("DOJO_GATEWAY_URL", "")
	t.Setenv("DOJO_GATEWAY_TOKEN", "env-token-xyz")
	t.Setenv("DOJO_PLUGINS_PATH", "")
	t.Setenv("DOJO_PROVIDER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Gateway.Token != "env-token-xyz" {
		t.Errorf("Gateway.Token: got %q, want %q", cfg.Gateway.Token, "env-token-xyz")
	}
}

func TestLoad_EnvVar_Provider(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	t.Setenv("DOJO_GATEWAY_URL", "")
	t.Setenv("DOJO_GATEWAY_TOKEN", "")
	t.Setenv("DOJO_PLUGINS_PATH", "")
	t.Setenv("DOJO_PROVIDER", "anthropic")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Defaults.Provider != "anthropic" {
		t.Errorf("Defaults.Provider: got %q, want %q", cfg.Defaults.Provider, "anthropic")
	}
}

func TestLoad_EnvVar_PluginsPath(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	customPluginsPath := filepath.Join(tmp, "myplugins")
	t.Setenv("DOJO_GATEWAY_URL", "")
	t.Setenv("DOJO_GATEWAY_TOKEN", "")
	t.Setenv("DOJO_PLUGINS_PATH", customPluginsPath)
	t.Setenv("DOJO_PROVIDER", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Plugins.Path != customPluginsPath {
		t.Errorf("Plugins.Path: got %q, want %q", cfg.Plugins.Path, customPluginsPath)
	}
}

// ─── SettingsPath ─────────────────────────────────────────────────────────────

func TestSettingsPath_EndsWithSettingsJSON(t *testing.T) {
	p := SettingsPath()
	if !strings.HasSuffix(p, "settings.json") {
		t.Errorf("SettingsPath() = %q; expected suffix 'settings.json'", p)
	}
}

// ─── Invalid JSON in settings file ───────────────────────────────────────────

func TestLoad_InvalidJSON_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	t.Setenv("DOJO_GATEWAY_URL", "")
	t.Setenv("DOJO_GATEWAY_TOKEN", "")
	t.Setenv("DOJO_PLUGINS_PATH", "")
	t.Setenv("DOJO_PROVIDER", "")

	dojoDir := filepath.Join(tmp, ".dojo")
	os.MkdirAll(dojoDir, 0o755)
	os.WriteFile(filepath.Join(dojoDir, "settings.json"), []byte("{invalid json"), 0o644)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid JSON settings file, got nil")
	}
}
