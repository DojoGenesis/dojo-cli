package hooks

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DojoGenesis/dojo-cli/internal/plugins"
)

// ─── New() safety ─────────────────────────────────────────────────────────────

func TestNew_NilPlugins_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New(nil) panicked: %v", r)
		}
	}()
	r := New(nil)
	if r == nil {
		t.Fatal("New(nil) returned nil runner")
	}
}

func TestNew_PluginsWithNoHooks_Works(t *testing.T) {
	ps := []plugins.Plugin{
		{Name: "empty-plugin", Version: "1.0", HookRules: nil},
	}
	r := New(ps)
	if r == nil {
		t.Fatal("New() returned nil")
	}
	// Fire should return nil with no matching hooks.
	err := r.Fire(context.Background(), EventPreCommand, nil)
	if err != nil {
		t.Errorf("Fire() with no hooks returned error: %v", err)
	}
}

// ─── Fire() with unknown event ────────────────────────────────────────────────

func TestFire_UnknownEvent_NoError(t *testing.T) {
	ps := []plugins.Plugin{
		{
			Name:    "test-plugin",
			Version: "1.0",
			HookRules: []plugins.HookRule{
				{
					Event: EventPostCommand,
					Hooks: []plugins.HookDef{
						{Type: "command", Command: "echo test"},
					},
				},
			},
		},
	}
	r := New(ps)
	err := r.Fire(context.Background(), "NonExistentEvent", nil)
	if err != nil {
		t.Errorf("Fire() with unknown event returned unexpected error: %v", err)
	}
}

// ─── Fire() with "command" type hook ─────────────────────────────────────────

func TestFire_CommandHook_ExecutesScript(t *testing.T) {
	// Create a temp file that the hook will touch — proves sh -c execution works.
	tmp := t.TempDir()
	markerFile := filepath.Join(tmp, "hook-ran.txt")

	ps := []plugins.Plugin{
		{
			Name:    "cmd-plugin",
			Version: "1.0",
			Path:    tmp,
			HookRules: []plugins.HookRule{
				{
					Event: EventPreCommand,
					Hooks: []plugins.HookDef{
						{
							Type:    "command",
							Command: "touch " + markerFile,
							Async:   false,
						},
					},
				},
			},
		},
	}

	r := New(ps)
	err := r.Fire(context.Background(), EventPreCommand, map[string]any{"command": "/help"})
	if err != nil {
		t.Fatalf("Fire() returned error: %v", err)
	}

	if _, statErr := os.Stat(markerFile); os.IsNotExist(statErr) {
		t.Errorf("hook command did not run: marker file %q was not created", markerFile)
	}
}

// ─── Fire() with async hook ───────────────────────────────────────────────────

func TestFire_AsyncHook_ReturnsBeforeCompletion(t *testing.T) {
	// Use a sleep command as the hook body; Fire() should return before it finishes.
	tmp := t.TempDir()
	markerFile := filepath.Join(tmp, "async-done.txt")

	// The hook sleeps briefly then touches the marker.
	// Fire() must return before the marker appears.
	ps := []plugins.Plugin{
		{
			Name:    "async-plugin",
			Version: "1.0",
			Path:    tmp,
			HookRules: []plugins.HookRule{
				{
					Event: EventPostCommand,
					Hooks: []plugins.HookDef{
						{
							Type:    "command",
							Command: "sleep 0.3 && touch " + markerFile,
							Async:   true,
						},
					},
				},
			},
		},
	}

	r := New(ps)

	start := time.Now()
	err := r.Fire(context.Background(), EventPostCommand, nil)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Fire() returned error: %v", err)
	}

	// Fire() should return well before the 300 ms sleep completes.
	if elapsed >= 200*time.Millisecond {
		t.Errorf("Fire() took %v — async hook should have returned immediately", elapsed)
	}

	// Marker should NOT exist yet right after Fire() returns.
	if _, statErr := os.Stat(markerFile); !os.IsNotExist(statErr) {
		t.Logf("marker appeared faster than expected (flaky if machine is very fast)")
	}

	// Give the goroutine time to finish so we don't leave zombie processes.
	time.Sleep(500 * time.Millisecond)
}

// ─── Fire() cancelled context prevents new async hooks ───────────────────────

func TestFire_CancelledContext_AsyncHookNotStarted(t *testing.T) {
	tmp := t.TempDir()
	markerFile := filepath.Join(tmp, "cancelled.txt")

	ps := []plugins.Plugin{
		{
			Name:    "cancel-plugin",
			Version: "1.0",
			Path:    tmp,
			HookRules: []plugins.HookRule{
				{
					Event: EventPreCommand,
					Hooks: []plugins.HookDef{
						{
							Type:    "command",
							Command: "touch " + markerFile,
							Async:   true,
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before firing

	r := New(ps)
	err := r.Fire(ctx, EventPreCommand, nil)
	if err != nil {
		t.Fatalf("Fire() with cancelled context returned error: %v", err)
	}

	// Allow a brief window; the async hook should NOT have run.
	time.Sleep(50 * time.Millisecond)
	if _, statErr := os.Stat(markerFile); !os.IsNotExist(statErr) {
		t.Errorf("async hook ran despite cancelled context; marker file was created")
	}
}

// ─── Non-command hooks are skipped ───────────────────────────────────────────

func TestFire_NonCommandHooks_Skipped(t *testing.T) {
	// Phase 1: only "command" type hooks execute. prompt/agent/http are silently skipped.
	ps := []plugins.Plugin{
		{
			Name:    "skip-plugin",
			Version: "1.0",
			HookRules: []plugins.HookRule{
				{
					Event: EventPreCommand,
					Hooks: []plugins.HookDef{
						{Type: "prompt", Prompt: "do something"},
						{Type: "agent", Command: "some-agent"},
						{Type: "http", URL: "http://example.com"},
					},
				},
			},
		},
	}
	r := New(ps)
	err := r.Fire(context.Background(), EventPreCommand, nil)
	if err != nil {
		t.Errorf("Fire() with non-command hooks returned error: %v", err)
	}
}

// ─── Event name matching is case-insensitive ──────────────────────────────────

func TestFire_CaseInsensitiveEventMatch(t *testing.T) {
	tmp := t.TempDir()
	markerFile := filepath.Join(tmp, "case-match.txt")

	ps := []plugins.Plugin{
		{
			Name:    "case-plugin",
			Version: "1.0",
			Path:    tmp,
			HookRules: []plugins.HookRule{
				{
					// Rule uses mixed case
					Event: "precommand",
					Hooks: []plugins.HookDef{
						{Type: "command", Command: "touch " + markerFile, Async: false},
					},
				},
			},
		},
	}

	r := New(ps)
	// Fire with the canonical constant (different case)
	err := r.Fire(context.Background(), EventPreCommand, nil) // "PreCommand"
	if err != nil {
		t.Fatalf("Fire() returned error: %v", err)
	}

	if _, statErr := os.Stat(markerFile); os.IsNotExist(statErr) {
		t.Errorf("case-insensitive event match failed: marker file not created")
	}
}
