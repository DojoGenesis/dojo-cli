package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ─── BuiltinPresets ──────────────────────────────────────────────────────────

func TestBuiltinPresets_ReturnsFour(t *testing.T) {
	presets := BuiltinPresets()
	if len(presets) != 4 {
		t.Fatalf("BuiltinPresets() returned %d presets, want 4", len(presets))
	}
	names := map[string]bool{}
	for _, p := range presets {
		names[p.Name] = true
	}
	for _, want := range []string{"focused", "balanced", "exploratory", "deliberate"} {
		if !names[want] {
			t.Errorf("BuiltinPresets() missing preset %q", want)
		}
	}
}

// ─── LoadDispositionPresets — missing dir ────────────────────────────────────

func TestLoadPresets_MissingDir(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	// No ~/.dojo/dispositions/ directory exists — should return builtins.
	presets, err := LoadDispositionPresets()
	if err != nil {
		t.Fatalf("LoadDispositionPresets() returned error: %v", err)
	}
	if len(presets) != 4 {
		t.Errorf("LoadDispositionPresets() returned %d presets, want 4 builtins", len(presets))
	}
}

// ─── SaveDispositionPreset + LoadDispositionPresets roundtrip ────────────────

func TestSaveAndLoadPreset(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmp)
	defer func() { os.Setenv("HOME", origHome) }()

	custom := DispositionPreset{
		Name:       "custom",
		Pacing:     "swift",
		Depth:      "thorough",
		Tone:       "warm",
		Initiative: "autonomous",
	}
	if err := SaveDispositionPreset(custom); err != nil {
		t.Fatalf("SaveDispositionPreset() error: %v", err)
	}

	// Verify the file was written.
	path := filepath.Join(DispositionsDir(), "custom.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read saved preset file: %v", err)
	}
	var loaded DispositionPreset
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("saved preset is not valid JSON: %v", err)
	}
	if loaded.Name != "custom" || loaded.Tone != "warm" {
		t.Errorf("saved preset mismatch: got %+v", loaded)
	}

	// Load all — should include custom + 4 builtins = 5.
	presets, err := LoadDispositionPresets()
	if err != nil {
		t.Fatalf("LoadDispositionPresets() error: %v", err)
	}
	if len(presets) != 5 {
		t.Errorf("LoadDispositionPresets() returned %d presets, want 5 (1 custom + 4 builtins)", len(presets))
	}

	found := false
	for _, p := range presets {
		if p.Name == "custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("custom preset not found in loaded presets")
	}
}

// ─── mergeBuiltins ──────────────────────────────────────────────────────────

func TestMergeBuiltins(t *testing.T) {
	// User override of "focused" — should not duplicate it.
	userPresets := []DispositionPreset{
		{Name: "focused", Pacing: "deliberate", Depth: "exhaustive", Tone: "warm", Initiative: "autonomous"},
	}
	merged := mergeBuiltins(userPresets)

	// 1 user + 3 remaining builtins (balanced, exploratory, deliberate) = 4
	if len(merged) != 4 {
		t.Fatalf("mergeBuiltins() returned %d presets, want 4", len(merged))
	}

	// The user's "focused" should keep user values, not be overwritten.
	for _, p := range merged {
		if p.Name == "focused" {
			if p.Pacing != "deliberate" {
				t.Errorf("user focused.Pacing was overwritten: got %q, want %q", p.Pacing, "deliberate")
			}
			return
		}
	}
	t.Error("focused preset missing from merged results")
}
