// Package plugins scans a plugins directory for CoworkPlugins-format plugin directories.
package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Plugin holds the metadata and hook rules discovered for a single plugin.
type Plugin struct {
	Name        string
	Description string
	Version     string
	Path        string // absolute path to plugin directory
	HookRules   []HookRule
	AgentCount  int
	SkillCount  int
}

// HookRule is a single entry in hooks.json — one event + one list of hook definitions.
type HookRule struct {
	Event   string
	Matcher string
	If      string
	Hooks   []HookDef
}

// HookDef is an individual hook action within a rule.
type HookDef struct {
	Type    string `json:"type"`    // "command", "prompt", "agent", "http"
	Command string `json:"command"` // shell command string (type=command)
	Prompt  string `json:"prompt"`  // prompt text (type=prompt)
	Model   string `json:"model"`   // model override (type=prompt)
	URL     string `json:"url"`     // target URL (type=http)
	Async   bool   `json:"async"`
}

// pluginMeta is the JSON shape of plugin.json (or .claude-plugin/plugin.json).
type pluginMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// hookEntry is one element inside an event's array in hooks.json.
type hookEntry struct {
	Matcher string    `json:"matcher"`
	If      string    `json:"if"`
	Hooks   []HookDef `json:"hooks"`
}

// Scan reads a plugins root directory and returns all discovered plugins.
// Each subdirectory is checked for a plugin.json (or .claude-plugin/plugin.json).
// Missing or unreadable files are skipped with a best-effort approach.
func Scan(pluginsRoot string) ([]Plugin, error) {
	entries, err := os.ReadDir(pluginsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var plugins []Plugin
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pluginDir := filepath.Join(pluginsRoot, e.Name())
		p, ok := scanOne(pluginDir)
		if ok {
			plugins = append(plugins, p)
		}
	}
	return plugins, nil
}

// scanOne attempts to parse one plugin directory. Returns (plugin, true) on success.
func scanOne(dir string) (Plugin, bool) {
	// Locate plugin.json — check .claude-plugin/ first, then root.
	meta, ok := loadPluginMeta(dir)
	if !ok {
		return Plugin{}, false
	}

	p := Plugin{
		Name:        meta.Name,
		Description: meta.Description,
		Version:     meta.Version,
		Path:        dir,
	}

	// Fall back to directory name if plugin.json has no name.
	if p.Name == "" {
		p.Name = filepath.Base(dir)
	}

	// Load hooks/hooks.json
	p.HookRules = loadHooks(dir, p.Name)

	// Count agents (agents/*.md)
	p.AgentCount = countFiles(filepath.Join(dir, "agents"), "*.md")

	// Count skills (skills/*/SKILL.md)
	p.SkillCount = countSkills(filepath.Join(dir, "skills"))

	return p, true
}

// loadPluginMeta looks for plugin.json in .claude-plugin/ then root.
func loadPluginMeta(dir string) (pluginMeta, bool) {
	candidates := []string{
		filepath.Join(dir, ".claude-plugin", "plugin.json"),
		filepath.Join(dir, "plugin.json"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m pluginMeta
		if err := json.Unmarshal(data, &m); err == nil {
			return m, true
		}
	}
	return pluginMeta{}, false
}

// loadHooks reads hooks/hooks.json and converts the map-of-event format into []HookRule.
// Format: { "EventName": [ { "matcher": "...", "if": "...", "hooks": [...] } ] }
func loadHooks(pluginDir, pluginName string) []HookRule {
	hooksPath := filepath.Join(pluginDir, "hooks", "hooks.json")
	data, err := os.ReadFile(hooksPath)
	if err != nil {
		return nil
	}

	// hooks.json is an object keyed by event name.
	var raw map[string][]hookEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	var rules []HookRule
	for event, entries := range raw {
		for _, entry := range entries {
			rules = append(rules, HookRule{
				Event:   event,
				Matcher: entry.Matcher,
				If:      entry.If,
				Hooks:   entry.Hooks,
			})
		}
	}
	return rules
}

// countFiles counts files matching a glob pattern inside dir.
func countFiles(dir, pattern string) int {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return 0
	}
	return len(matches)
}

// countSkills counts subdirectories of skillsDir that contain a SKILL.md file.
func countSkills(skillsDir string) int {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillMD := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		if _, err := os.Stat(skillMD); err == nil {
			count++
		}
	}
	return count
}
