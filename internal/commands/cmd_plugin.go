package commands

// cmd_plugin.go — /plugin command for managing installed plugins.

import (
	"context"
	"fmt"

	"github.com/DojoGenesis/dojo-cli/internal/activity"
	"github.com/DojoGenesis/dojo-cli/internal/plugins"
	gcolor "github.com/gookit/color"
)

// pluginCmd returns the /plugin command with subcommands:
//
//	/plugin ls            — list installed plugins
//	/plugin install <url> — clone a plugin from a git URL
//	/plugin rm <name>     — remove an installed plugin
func (r *Registry) pluginCmd() Command {
	return Command{
		Name:    "plugin",
		Aliases: []string{"plugins"},
		Usage:   "/plugin [ls|install <url>|rm <name>]",
		Short:   "Manage installed plugins",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 || args[0] == "ls" {
				return r.pluginList(ctx)
			}
			switch args[0] {
			case "install":
				if len(args) < 2 {
					return fmt.Errorf("usage: /plugin install <git-url>")
				}
				return r.pluginInstall(ctx, args[1])
			case "rm", "remove", "uninstall":
				if len(args) < 2 {
					return fmt.Errorf("usage: /plugin rm <name>")
				}
				return r.pluginRemove(ctx, args[1])
			default:
				return fmt.Errorf("unknown subcommand %q — use ls, install, or rm", args[0])
			}
		},
	}
}

// pluginList prints all currently loaded plugins.
func (r *Registry) pluginList(ctx context.Context) error {
	fmt.Println()
	gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprint("  Installed plugins"))
	fmt.Println()
	fmt.Println()

	if len(r.plgs) == 0 {
		fmt.Println(gcolor.HEX("#94a3b8").Sprint("  No plugins installed."))
		fmt.Println()
		return nil
	}

	for _, p := range r.plgs {
		name := gcolor.HEX("#f4a261").Sprint(p.Name)
		ver := gcolor.HEX("#94a3b8").Sprint(orDefault(p.Version, "?"))
		fmt.Printf("  %s %s\n", name, ver)

		printKV("    path", p.Path)
		printKV("    hooks", fmt.Sprintf("%d rules", len(p.HookRules)))
		printKV("    skills", fmt.Sprintf("%d", p.SkillCount))
		fmt.Println()
	}

	return nil
}

// pluginInstall clones a plugin from a git URL and rescans.
func (r *Registry) pluginInstall(ctx context.Context, gitURL string) error {
	fmt.Println()
	fmt.Println(gcolor.HEX("#94a3b8").Sprintf("  Cloning %s ...", gitURL))

	dest, err := plugins.Install(gitURL, r.cfg.Plugins.Path)
	if err != nil {
		return fmt.Errorf("plugin install: %w", err)
	}

	activity.Log(activity.CommandRun, fmt.Sprintf("plugin installed from %s → %s", gitURL, dest))

	// Rescan plugins to pick up the new one.
	plgs, scanErr := plugins.Scan(r.cfg.Plugins.Path)
	if scanErr == nil {
		r.plgs = plgs
	}

	fmt.Println()
	gcolor.Bold.Print(gcolor.HEX("#7fb88c").Sprintf("  Plugin installed at %s", dest))
	fmt.Println()
	fmt.Println()
	return nil
}

// pluginRemove removes an installed plugin by name and rescans.
func (r *Registry) pluginRemove(ctx context.Context, name string) error {
	if err := plugins.Uninstall(name, r.cfg.Plugins.Path); err != nil {
		return fmt.Errorf("plugin remove: %w", err)
	}

	activity.Log(activity.CommandRun, fmt.Sprintf("plugin removed: %s", name))

	// Rescan plugins.
	plgs, scanErr := plugins.Scan(r.cfg.Plugins.Path)
	if scanErr == nil {
		r.plgs = plgs
	}

	fmt.Println()
	gcolor.Bold.Print(gcolor.HEX("#7fb88c").Sprintf("  Plugin %q removed", name))
	fmt.Println()
	fmt.Println()
	return nil
}
