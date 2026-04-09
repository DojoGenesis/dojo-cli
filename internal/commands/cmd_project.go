package commands

// cmd_project.go — /projects command (workspace view).
// NOTE: /project (singular) lives in project_cmds.go.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gcolor "github.com/gookit/color"
)

// ─── /projects ──────────────────────────────────────────────────────────────

func (r *Registry) projectsCmd() Command {
	return Command{
		Name:  "projects",
		Usage: "/projects ls",
		Short: "Local workspace view — cwd, plugins, session",
		Run: func(ctx context.Context, args []string) error {
			fmt.Println()
			gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprint("  Projects — local workspace"))
			fmt.Println()
			fmt.Println()

			// Current working directory name as the project
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "(unknown)"
			}
			project := filepath.Base(cwd)
			printKV("project", project)
			printKV("path", cwd)

			// Check for .ada/disposition.yaml
			adaPath := filepath.Join(cwd, ".ada", "disposition.yaml")
			if data, readErr := os.ReadFile(adaPath); readErr == nil {
				// Extract active_mode from the YAML with a simple scan (no yaml dep needed)
				activeMode := ""
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "active_mode:") {
						activeMode = strings.TrimSpace(strings.TrimPrefix(line, "active_mode:"))
						break
					}
				}
				if activeMode == "" {
					activeMode = "(set)"
				}
				printKV("disposition", activeMode)
			} else {
				printKV("disposition", gcolor.HEX("#94a3b8").Sprint("no .ada/disposition.yaml"))
			}

			printKV("plugins loaded", fmt.Sprintf("%d", len(r.plgs)))
			printKV("session", *r.session)
			fmt.Println()
			return nil
		},
	}
}
