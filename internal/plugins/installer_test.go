package plugins

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/DojoGenesis/plugins.git", "plugins"},
		{"https://github.com/DojoGenesis/plugins", "plugins"},
		{"github.com/foo/bar", "bar"},
		{"github.com/foo/bar.git", "bar"},
		{"https://github.com/user/my-plugin.git/", "my-plugin"},
		{"git@github.com:user/repo.git", "repo"},
		{"https://gitlab.com/group/subgroup/project.git", "project"},
		{"simple-name", "simple-name"},
	}

	for _, tc := range tests {
		got := repoName(tc.input)
		if got != tc.want {
			t.Errorf("repoName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/foo/bar", "https://github.com/foo/bar"},
		{"http://github.com/foo/bar", "http://github.com/foo/bar"},
		{"github.com/foo/bar", "https://github.com/foo/bar"},
		{"  github.com/foo/bar  ", "https://github.com/foo/bar"},
	}

	for _, tc := range tests {
		got := normalizeURL(tc.input)
		if got != tc.want {
			t.Errorf("normalizeURL(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestInstall_AlreadyExists(t *testing.T) {
	tmp := t.TempDir()
	existing := filepath.Join(tmp, "plugins")
	if err := os.MkdirAll(existing, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := Install("https://github.com/DojoGenesis/plugins.git", tmp)
	if err == nil {
		t.Fatal("expected error for already-existing directory, got nil")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestInstall_InvalidURL(t *testing.T) {
	// Skip if git is not available.
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping")
	}

	tmp := t.TempDir()
	_, err := Install("https://invalid.example.com/no-such-repo/does-not-exist-ever.git", tmp)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestInstall_MissingPluginJSON(t *testing.T) {
	// Simulate a successful clone that lacks plugin.json.
	// We create a temp dir manually and call hasPluginJSON to verify behavior.
	tmp := t.TempDir()
	fakeClone := filepath.Join(tmp, "fake-plugin")
	if err := os.MkdirAll(fakeClone, 0755); err != nil {
		t.Fatal(err)
	}

	if hasPluginJSON(fakeClone) {
		t.Error("hasPluginJSON should return false for directory without plugin.json")
	}

	// Now add plugin.json and check again.
	if err := os.WriteFile(filepath.Join(fakeClone, "plugin.json"), []byte(`{"name":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if !hasPluginJSON(fakeClone) {
		t.Error("hasPluginJSON should return true after creating plugin.json")
	}
}

func TestInstall_MissingPluginJSON_ClaudePluginDir(t *testing.T) {
	tmp := t.TempDir()
	fakeClone := filepath.Join(tmp, "nested-plugin")
	cpDir := filepath.Join(fakeClone, ".claude-plugin")
	if err := os.MkdirAll(cpDir, 0755); err != nil {
		t.Fatal(err)
	}

	if hasPluginJSON(fakeClone) {
		t.Error("hasPluginJSON should return false when .claude-plugin/ exists but has no plugin.json")
	}

	if err := os.WriteFile(filepath.Join(cpDir, "plugin.json"), []byte(`{"name":"nested"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if !hasPluginJSON(fakeClone) {
		t.Error("hasPluginJSON should return true for .claude-plugin/plugin.json")
	}
}

func TestUninstall(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "test-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Add plugin.json so safety check passes.
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{"name":"test-plugin"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Uninstall should succeed.
	if err := Uninstall("test-plugin", tmp); err != nil {
		t.Fatalf("Uninstall returned error: %v", err)
	}

	// Directory should be gone.
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		t.Error("expected plugin directory to be removed after Uninstall")
	}
}

func TestUninstall_NotFound(t *testing.T) {
	tmp := t.TempDir()
	err := Uninstall("nonexistent", tmp)
	if err == nil {
		t.Fatal("expected error for nonexistent plugin, got nil")
	}
}

func TestUninstall_NoPluginJSON(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "bad-dir")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Directory exists but has no plugin.json — safety check should reject.
	err := Uninstall("bad-dir", tmp)
	if err == nil {
		t.Fatal("expected error for directory without plugin.json, got nil")
	}

	// Directory should still exist (not removed).
	if _, err := os.Stat(pluginDir); err != nil {
		t.Error("directory should NOT have been removed when plugin.json is missing")
	}
}
