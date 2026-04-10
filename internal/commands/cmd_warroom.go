package commands

// cmd_warroom.go — /warroom command: split-panel Scout vs Challenger debate TUI.

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DojoGenesis/dojo-cli/internal/tui"
)

// warroomCmd returns the /warroom command.
func (r *Registry) warroomCmd() Command {
	return Command{
		Name:    "warroom",
		Aliases: []string{"war", "debate"},
		Usage:   "/warroom [topic]",
		Short:   "Split-panel debate: Scout vs Challenger",
		Run: func(ctx context.Context, args []string) error {
			sessionID := fmt.Sprintf("warroom-%d", time.Now().UnixMilli())
			if r.session != nil && *r.session != "" {
				sessionID = *r.session + "-warroom"
			}

			topic := strings.Join(args, " ")

			model := tui.NewWarRoomModel(
				r.cfg.Gateway.URL,
				r.cfg.Gateway.Token,
				r.cfg.Defaults.Model,
				r.cfg.Defaults.Provider,
				sessionID,
				topic,
			)

			p := tea.NewProgram(model, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
}
