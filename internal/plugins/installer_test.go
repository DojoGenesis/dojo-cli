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

// --- New tests for multi-plugin / monorepo / subdirectory URL cases ---

func TestParseGitHubSubdirURL(t *testing.T) {
	tests := []struct {
		input      string
		wantClone  string
		wantPath   string
		wantBranch string
		wantOK     bool
	}{
		{
			"https://github.com/anthropics/claude-plugins-official/tree/main/plugins/agent-sdk-dev",
			"https://github.com/anthropics/claude-plugins-official",
			"plugins/agent-sdk-dev",
			"main",
			true,
		},
		{
			"https://github.com/foo/bar/tree/feature/x/deep/subdir",
			"https://github.com/foo/bar",
			"x/deep/subdir",
			"feature",
			true,
		},
		// Not a subdirectory URL — root repo URL.
		{
			"https://github.com/foo/bar",
			"", "", "", false,
		},
		// Non-GitHub URL — should not match.
		{
			"https://gitlab.com/foo/bar/tree/main/plugin",
			"", "", "", false,
		},
	}

	for _, tc := range tests {
		gotClone, gotPath, gotBranch, gotOK := parseGitHubSubdirURL(tc.input)
		if gotOK != tc.wantOK {
			t.Errorf("parseGitHubSubdirURL(%q): ok=%v, want %v", tc.input, gotOK, tc.wantOK)
			continue
		}
		if !tc.wantOK {
			continue
		}
		if gotClone != tc.wantClone {
			t.Errorf("parseGitHubSubdirURL(%q): cloneURL=%q, want %q", tc.input, gotClone, tc.wantClone)
		}
		if gotPath != tc.wantPath {
			t.Errorf("parseGitHubSubdirURL(%q): subpath=%q, want %q", tc.input, gotPath, tc.wantPath)
		}
		if gotBranch != tc.wantBranch {
			t.Errorf("parseGitHubSubdirURL(%q): branch=%q, want %q", tc.input, gotBranch, tc.wantBranch)
		}
	}
}

// TestExtractMonorepoPlugins_Depth1 verifies that plugins at depth 1 are discovered.
func TestExtractMonorepoPlugins_Depth1(t *testing.T) {
	tmp := t.TempDir()
	destDir := t.TempDir()

	// Build a fake monorepo:
	//   tmp/
	//     plugin-alpha/plugin.json
	//     plugin-beta/plugin.json
	//     .hidden/plugin.json  (should be skipped)
	//     not-a-plugin/README.md
	for _, name := range []string{"plugin-alpha", "plugin-beta"} {
		dir := filepath.Join(tmp, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{"name":"`+name+`"}`), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Hidden dir — should be skipped.
	hiddenDir := filepath.Join(tmp, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "plugin.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Non-plugin dir.
	if err := os.MkdirAll(filepath.Join(tmp, "not-a-plugin"), 0755); err != nil {
		t.Fatal(err)
	}

	results, err := extractMonorepoPlugins(tmp, destDir)
	if err != nil {
		t.Fatalf("extractMonorepoPlugins: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 plugins, got %d: %+v", len(results), results)
	}

	names := map[string]bool{}
	for _, r := range results {
		names[r.Name] = true
		if _, err := os.Stat(r.Path); err != nil {
			t.Errorf("plugin path %s does not exist", r.Path)
		}
		if !hasPluginJSON(r.Path) {
			t.Errorf("plugin at %s has no plugin.json", r.Path)
		}
	}
	for _, want := range []string{"plugin-alpha", "plugin-beta"} {
		if !names[want] {
			t.Errorf("expected plugin %q in results", want)
		}
	}
}

// TestExtractMonorepoPlugins_Depth2 verifies nested plugins (e.g. plugins/agent-sdk-dev/).
func TestExtractMonorepoPlugins_Depth2(t *testing.T) {
	tmp := t.TempDir()
	destDir := t.TempDir()

	// Build:
	//   tmp/
	//     plugins/
	//       agent-sdk-dev/plugin.json
	//       another-plugin/plugin.json
	pluginsDir := filepath.Join(tmp, "plugins")
	for _, name := range []string{"agent-sdk-dev", "another-plugin"} {
		dir := filepath.Join(pluginsDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{"name":"`+name+`"}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	results, err := extractMonorepoPlugins(tmp, destDir)
	if err != nil {
		t.Fatalf("extractMonorepoPlugins: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 plugins, got %d: %+v", len(results), results)
	}
	for _, r := range results {
		if !hasPluginJSON(r.Path) {
			t.Errorf("plugin at %s has no plugin.json after extraction", r.Path)
		}
	}
}

// TestExtractMonorepoPlugins_Empty returns an error-free empty slice when no plugins exist.
func TestExtractMonorepoPlugins_Empty(t *testing.T) {
	tmp := t.TempDir()
	destDir := t.TempDir()

	// Repo with no plugin.json anywhere.
	if err := os.MkdirAll(filepath.Join(tmp, "src"), 0755); err != nil {
		t.Fatal(err)
	}

	results, err := extractMonorepoPlugins(tmp, destDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 plugins, got %d", len(results))
	}
}

// TestCopyDir verifies that copyDir reproduces directory trees faithfully.
func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	dst = filepath.Join(dst, "copy") // copyDir must create the destination.

	// src/a.txt, src/sub/b.txt
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	for _, rel := range []string{"a.txt", filepath.Join("sub", "b.txt")} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Errorf("expected %s to exist in dst", rel)
		}
	}
}
