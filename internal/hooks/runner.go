// Package hooks runs hook rules from loaded plugins.
package hooks

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/DojoGenesis/dojo-cli/internal/plugins"
)

// Event names that dojo-cli fires.
const (
	EventPreCommand  = "PreCommand"
	EventPostCommand = "PostCommand"
	EventPostSkill   = "PostSkill"
	EventPostAgent   = "PostAgent"
	EventSessionEnd  = "SessionEnd"
)

// Runner executes hook rules for a given event.
type Runner struct {
	plugins []plugins.Plugin
}

// New creates a Runner from a list of loaded plugins.
func New(ps []plugins.Plugin) *Runner {
	return &Runner{plugins: ps}
}

// Fire runs all hook rules matching the given event name across all plugins.
// Only "command" type hooks are executed in Phase 1; prompt/agent/http hooks
// are recognised but skipped.
// Async hooks are run in a goroutine. Sync hooks block until completion.
// ctx cancellation prevents async hooks from starting but does not kill
// already-running processes.
func (r *Runner) Fire(ctx context.Context, event string, payload map[string]any) error {
	for _, p := range r.plugins {
		for _, rule := range p.HookRules {
			if !strings.EqualFold(rule.Event, event) {
				continue
			}
			for _, h := range rule.Hooks {
				if h.Type != "command" {
					// Phase 1: skip prompt / agent / http hooks.
					continue
				}
				cmd := h.Command
				pluginName := p.Name
				pluginPath := p.Path
				isAsync := h.Async

				if isAsync {
					go func() {
						select {
						case <-ctx.Done():
							return
						default:
						}
						if err := runCommand(ctx, cmd, pluginPath); err != nil {
							log.Printf("[hooks] async hook error (%s/%s): %v", pluginName, event, err)
						}
					}()
				} else {
					if err := runCommand(ctx, cmd, pluginPath); err != nil {
						return fmt.Errorf("hook error (%s/%s): %w", pluginName, event, err)
					}
				}
			}
		}
	}
	return nil
}

// runCommand executes a shell command string with CLAUDE_PLUGIN_ROOT set to
// the plugin's directory. The command is passed to sh -c so it can use
// shell expansion (e.g. variable substitution, quoting).
func runCommand(ctx context.Context, command, pluginRoot string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Env = append(cmd.Environ(),
		"CLAUDE_PLUGIN_ROOT="+pluginRoot,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return err
	}
	return nil
}
