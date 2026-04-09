package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ─── Scan on non-existent directory ──────────────────────────────────────────

func TestScan_NonExistentDirectory_ReturnsNilNil(t *testing.T) {
	result, err := Scan("/tmp/definitely-does-not-exist-dojo-test-12345")
	if err != nil {
		t.Errorf("expected nil error for non-existent dir, got: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil slice for non-existent dir, got: %v", result)
	}
}

// ─── Scan on empty directory ──────────────────────────────────────────────────

func TestScan_EmptyDirectory_ReturnsEmptySlice(t *testing.T) {
	tmp := t.TempDir()
	result, err := Scan(tmp)
	if err != nil {
		t.Fatalf("Scan on empty dir returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(result))
	}
}

// ─── Scan with valid plugin ───────────────────────────────────────────────────

func TestScan_ValidPlugin_PopulatesFields(t *testing.T) {
	// Create:
	//   <root>/myplugin/plugin.json          { "name": "test", "version": "0.1" }
	//   <root>/myplugin/agents/agent1.md
	//   <root>/myplugin/skills/skill1/SKILL.md
	//   <root>/myplugin/hooks/hooks.json     { "PostCommand": [...] }
	root := t.TempDir()
	pluginDir := filepath.Join(root, "myplugin")

	// plugin.json
	mustMkdir(t, pluginDir)
	writeJSON(t, filepath.Join(pluginDir, "plugin.json"), map[string]any{
		"name":        "test",
		"description": "A test plugin",
		"version":     "0.1",
	})

	// agents/agent1.md
	mustMkdir(t, filepath.Join(pluginDir, "agents"))
	writeFile(t, filepath.Join(pluginDir, "agents", "agent1.md"), "# Agent 1")

	// skills/skill1/SKILL.md
	mustMkdir(t, filepath.Join(pluginDir, "skills", "skill1"))
	writeFile(t, filepath.Join(pluginDir, "skills", "skill1", "SKILL.md"), "# Skill 1")

	// hooks/hooks.json
	mustMkdir(t, filepath.Join(pluginDir, "hooks"))
	hooksData := map[string]any{
		"PostCommand": []map[string]any{
			{
				"matcher": "*",
				"hooks": []map[string]any{
					{"type": "command", "command": "echo hook"},
				},
			},
		},
	}
	writeJSON(t, filepath.Join(pluginDir, "hooks", "hooks.json"), hooksData)

	plugins, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	p := plugins[0]

	if p.Name != "test" {
		t.Errorf("Name: got %q, want %q", p.Name, "test")
	}
	if p.Version != "0.1" {
		t.Errorf("Version: got %q, want %q", p.Version, "0.1")
	}
	if p.Description != "A test plugin" {
		t.Errorf("Description: got %q, want %q", p.Description, "A test plugin")
	}
	if p.Path != pluginDir {
		t.Errorf("Path: got %q, want %q", p.Path, pluginDir)
	}
	if p.AgentCount != 1 {
		t.Errorf("AgentCount: got %d, want 1", p.AgentCount)
	}
	if p.SkillCount != 1 {
		t.Errorf("SkillCount: got %d, want 1", p.SkillCount)
	}
	if len(p.HookRules) != 1 {
		t.Errorf("HookRules: got %d rules, want 1", len(p.HookRules))
	} else {
		if p.HookRules[0].Event != "PostCommand" {
			t.Errorf("HookRules[0].Event: got %q, want %q", p.HookRules[0].Event, "PostCommand")
		}
		if len(p.HookRules[0].Hooks) != 1 {
			t.Errorf("HookRules[0].Hooks: got %d, want 1", len(p.HookRules[0].Hooks))
		} else if p.HookRules[0].Hooks[0].Type != "command" {
			t.Errorf("HookRules[0].Hooks[0].Type: got %q, want %q", p.HookRules[0].Hooks[0].Type, "command")
		}
	}
}

// ─── Scan with plugin.json in .claude-plugin subdir ──────────────────────────

func TestScan_ClaudePluginSubdir_Discovered(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "myclaudeplugin")
	mustMkdir(t, filepath.Join(pluginDir, ".claude-plugin"))
	writeJSON(t, filepath.Join(pluginDir, ".claude-plugin", "plugin.json"), map[string]any{
		"name":    "hidden-test",
		"version": "2.0",
	})

	plugins, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name != "hidden-test" {
		t.Errorf("Name: got %q, want %q", plugins[0].Name, "hidden-test")
	}
}

// ─── Scan skips non-directory entries ────────────────────────────────────────

func TestScan_SkipsFiles(t *testing.T) {
	root := t.TempDir()
	// Place a plain file (not a dir) at the root level.
	writeFile(t, filepath.Join(root, "notaplugin.txt"), "just a file")

	plugins, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins (file should be skipped), got %d", len(plugins))
	}
}

// ─── Scan directory with no plugin.json is skipped ───────────────────────────

func TestScan_DirectoryWithoutPluginJSON_IsSkipped(t *testing.T) {
	root := t.TempDir()
	// A directory with no plugin.json should not appear in results.
	mustMkdir(t, filepath.Join(root, "nodesc"))

	plugins, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d: directory without plugin.json should be skipped", len(plugins))
	}
}

// ─── Multiple skills counted correctly ───────────────────────────────────────

func TestScan_MultipleSkills_Counted(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "multi")
	mustMkdir(t, pluginDir)
	writeJSON(t, filepath.Join(pluginDir, "plugin.json"), map[string]any{"name": "multi", "version": "1.0"})

	// 3 skill subdirs, only 2 have SKILL.md.
	for _, name := range []string{"skill-a", "skill-b", "skill-c"} {
		mustMkdir(t, filepath.Join(pluginDir, "skills", name))
	}
	writeFile(t, filepath.Join(pluginDir, "skills", "skill-a", "SKILL.md"), "# A")
	writeFile(t, filepath.Join(pluginDir, "skills", "skill-b", "SKILL.md"), "# B")
	// skill-c has no SKILL.md — should not be counted.

	plugins, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].SkillCount != 2 {
		t.Errorf("SkillCount: got %d, want 2", plugins[0].SkillCount)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	writeFile(t, path, string(data))
}
