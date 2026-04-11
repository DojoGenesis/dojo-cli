package commands

// cmd_bloom.go — /bloom command: fullscreen animated bonsai garden TUI.

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DojoGenesis/cli/internal/spirit"
	"github.com/DojoGenesis/cli/internal/state"
	"github.com/DojoGenesis/cli/internal/tui"
)

// bloomCmd returns the /bloom command.
func (r *Registry) bloomCmd() Command {
	return Command{
		Name:    "bloom",
		Aliases: []string{"tree", "garden", "zen"},
		Usage:   "/bloom",
		Short:   "Watch your bonsai grow — animated zen garden",
		Run: func(ctx context.Context, args []string) error {
			st, err := state.Load()
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			belt := spirit.CurrentBelt(st.Spirit.XP)
			model := tui.NewBloomModel(st.Spirit, belt)

			p := tea.NewProgram(model, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("bloom: %w", err)
			}
			return nil
		},
	}
}
