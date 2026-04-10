package commands

// cmd_system.go — /health, /settings, /hooks, /trace, /init, /home commands.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DojoGenesis/dojo-cli/internal/bootstrap"
	"github.com/DojoGenesis/dojo-cli/internal/config"
	"github.com/DojoGenesis/dojo-cli/internal/tui"
	gcolor "github.com/gookit/color"
)

// ─── /health ────────────────────────────────────────────────────────────────

func (r *Registry) healthCmd() Command {
	return Command{
		Name:    "health",
		Aliases: []string{"ping", "status"},
		Usage:   "/health",
		Short:   "Gateway health + stats",
		Run: func(ctx context.Context, args []string) error {
			h, err := r.gw.Health(ctx)
			if err != nil {
				return fmt.Errorf("gateway unreachable: %w", err)
			}
			fmt.Println()
			printKV("status", colorStatus(h.Status))
			printKV("version", h.Version)
			printKV("uptime", fmt.Sprintf("%ds", h.UptimeSeconds))
			printKV("requests", fmt.Sprintf("%d", h.RequestsProcessed))
			for name, st := range h.Providers {
				printKV("provider/"+name, colorStatus(st))
			}
			printKV("gateway", r.cfg.Gateway.URL)
			fmt.Println()
			return nil
		},
	}
}

// ─── /home ──────────────────────────────────────────────────────────────────

func (r *Registry) homeCmd() Command {
	return Command{
		Name:    "home",
		Aliases: []string{"ws", "workspace"},
		Usage:   "/home [plain]",
		Short:   "Workspace state overview",
		Run: func(ctx context.Context, args []string) error {
			// /home plain — text-only fallback
			if len(args) > 0 && args[0] == "plain" {
				return r.homePlain(ctx)
			}

			// Default: Bubbletea TUI panel
			model := tui.NewHomeModel(r.cfg, r.gw, *r.session, len(r.plgs))
			p := tea.NewProgram(model, tea.WithAltScreen())
			_, err := p.Run()
			return err
		},
	}
}

func (r *Registry) homePlain(ctx context.Context) error {
	h, err := r.gw.Health(ctx)
	if err != nil {
		return fmt.Errorf("gateway unreachable: %w", err)
	}
	agents, agentErr := r.gw.Agents(ctx)
	seeds, seedErr := r.gw.Seeds(ctx)

	fmt.Println()
	gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprint("  Dojo Workspace"))
	fmt.Println()
	fmt.Println(gcolor.HEX("#94a3b8").Sprint("  " + r.cfg.Gateway.URL))
	fmt.Println()

	fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("gateway"), colorStatus(h.Status))
	if agentErr != nil {
		fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("agents"), gcolor.HEX("#e63946").Sprint("unavailable"))
	} else {
		fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("agents"), gcolor.White.Sprintf("%d", len(agents)))
	}
	if seedErr != nil {
		fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("seeds"), gcolor.HEX("#e63946").Sprint("unavailable"))
	} else {
		fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("seeds"), gcolor.White.Sprintf("%d", len(seeds)))
	}
	fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("plugins"), gcolor.White.Sprintf("%d", len(r.plgs)))
	fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("session"), gcolor.HEX("#e8b04a").Sprint(*r.session))
	fmt.Println()
	return nil
}

// ─── /settings ──────────────────────────────────────────────────────────────

func (r *Registry) settingsCmd() Command {
	return Command{
		Name:    "settings",
		Aliases: []string{"config", "cfg"},
		Usage:   "/settings [providers|set <provider> <key>]",
		Short:   "Show active config and settings, or manage provider keys",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprint("  Active Settings"))
				fmt.Println()
				fmt.Println()
				printKV("config file", config.SettingsPath())
				printKV("plugins loaded", fmt.Sprintf("%d", len(r.plgs)))
				fmt.Print(r.cfg.EffectiveString())
				fmt.Println()
				return nil
			}

			sub := strings.ToLower(args[0])
			switch sub {
			case "effective":
				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprint("  Effective Configuration"))
				fmt.Println()
				fmt.Println(gcolor.HEX("#94a3b8").Sprint("  (file + env + flags, in priority order)"))
				fmt.Println()
				fmt.Print(r.cfg.EffectiveString())
				fmt.Println()

			case "providers":
				providerSettings, err := r.gw.GetProviderSettings(ctx)
				if err != nil {
					return fmt.Errorf("could not fetch provider settings: %w", err)
				}
				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprint("  Provider Configuration"))
				fmt.Println()
				fmt.Println()
				for k, v := range providerSettings {
					printKV(k, colorStatus(fmt.Sprintf("%v", v)))
				}
				fmt.Println()

			case "set":
				// /settings set <provider> <key>
				if len(args) < 3 {
					return fmt.Errorf("usage: /settings set <provider> <api-key>")
				}
				provider := args[1]
				apiKey := args[2]
				if err := r.gw.SetProviderKey(ctx, provider, apiKey); err != nil {
					return fmt.Errorf("could not set provider key: %w", err)
				}
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Provider key updated"))
				printKV("provider", provider)
				fmt.Println()

			default:
				return fmt.Errorf("unknown settings subcommand %q — use: effective, providers, set", sub)
			}
			return nil
		},
	}
}

// ─── /hooks ─────────────────────────────────────────────────────────────────

func (r *Registry) hooksCmd() Command {
	return Command{
		Name:  "hooks",
		Usage: "/hooks [ls|fire <event>]",
		Short: "List loaded hook rules or fire an event manually",
		Run: func(ctx context.Context, args []string) error {
			sub := "ls"
			if len(args) > 0 {
				sub = strings.ToLower(args[0])
			}

			switch sub {
			case "fire":
				if len(args) < 2 {
					return fmt.Errorf("usage: /hooks fire <event>")
				}
				event := args[1]
				fmt.Printf("\n  Firing event %q ...\n\n", event)
				if err := r.runner.Fire(ctx, event, nil); err != nil {
					return fmt.Errorf("hook fire error: %w", err)
				}
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  done"))
				fmt.Println()

			default: // ls
				// Count total rules
				totalRules := 0
				for _, p := range r.plgs {
					totalRules += len(p.HookRules)
				}

				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprintf("  Hooks (%d rules across %d plugins)\n\n", totalRules, len(r.plgs)))

				if totalRules == 0 {
					fmt.Println(gcolor.HEX("#94a3b8").Sprintf("  No hook rules loaded. Place plugins in %s", r.cfg.Plugins.Path))
					fmt.Println()
					return nil
				}

				for _, p := range r.plgs {
					if len(p.HookRules) == 0 {
						continue
					}
					// Glass-effect section divider
					fmt.Printf("  %s %s %s\n",
						gcolor.HEX("#64748b").Sprint("────"),
						gcolor.HEX("#e8b04a").Sprint("["+p.Name+"]"),
						gcolor.HEX("#64748b").Sprint("────"),
					)
					for _, rule := range p.HookRules {
						for _, h := range rule.Hooks {
							asyncLabel := ""
							if h.Async {
								asyncLabel = gcolor.HEX("#94a3b8").Sprint("  (async)")
							}
							cmd := h.Command
							if cmd == "" {
								cmd = h.Prompt
							}
							if cmd == "" {
								cmd = h.URL
							}
							fmt.Printf("    %s  %s  %s%s\n",
								gcolor.HEX("#f4a261").Sprintf("%-20s", rule.Event),
								gcolor.White.Sprintf("%-10s", h.Type),
								gcolor.HEX("#94a3b8").Sprint(truncate(cmd, 50)),
								asyncLabel,
							)
						}
					}
				}
				fmt.Println()
			}
			return nil
		},
	}
}

// ─── /trace ─────────────────────────────────────────────────────────────────

func (r *Registry) traceCmd() Command {
	return Command{
		Name:  "trace",
		Usage: "/trace [<id>]",
		Short: "Inspect an execution trace by ID, or show guidance",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				// Guidance mode
				fmt.Println()
				gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprint("  Dojo Trace"))
				fmt.Println()
				fmt.Println()
				fmt.Println(gcolor.HEX("#457b9d").Sprint("  Trace follows the active session's decision and tool-use history."))
				fmt.Println(gcolor.HEX("#457b9d").Sprint("  Connect the gateway with --trace to enable full trace output."))
				fmt.Println()
				printKV("gateway", r.cfg.Gateway.URL)
				fmt.Println(gcolor.HEX("#94a3b8").Sprint("  hint: /trace <id>  — provide a trace ID to inspect"))
				fmt.Println()
				return nil
			}

			traceID := args[0]
			data, err := r.gw.GetTrace(ctx, traceID)
			if err != nil {
				return fmt.Errorf("could not fetch trace: %w", err)
			}
			fmt.Println()
			gcolor.Bold.Print(gcolor.HEX("#e8b04a").Sprintf("  Trace: %s\n\n", traceID))
			for k, v := range data {
				switch val := v.(type) {
				case map[string]any, []any:
					// Format nested structures as indented JSON
					b, jsonErr := json.MarshalIndent(val, "    ", "  ")
					if jsonErr != nil {
						printKV(k, fmt.Sprintf("%v", val))
					} else {
						fmt.Printf("%s\n    %s\n",
							gcolor.HEX("#94a3b8").Sprintf("  %-24s", k),
							gcolor.White.Sprint(string(b)),
						)
					}
				default:
					printKV(k, fmt.Sprintf("%v", val))
				}
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── /init ──────────────────────────────────────────────────────────────────

func (r *Registry) initCmd() Command {
	return Command{
		Name:    "init",
		Aliases: []string{"setup", "bootstrap"},
		Usage:   "/init [--force] [--gateway <url>] [--plugins-source <path>]",
		Short:   "Initialize Dojo workspace with plugins, dispositions, and seeds",
		Run: func(ctx context.Context, args []string) error {
			opts := bootstrap.Options{
				GatewayURL: r.cfg.Gateway.URL,
			}
			for i := 0; i < len(args); i++ {
				switch args[i] {
				case "--force":
					opts.Force = true
				case "--gateway":
					if i+1 < len(args) {
						i++
						opts.GatewayURL = args[i]
					}
				case "--plugins-source":
					if i+1 < len(args) {
						i++
						opts.PluginsSource = args[i]
					}
				case "--skip-seeds":
					opts.SkipSeeds = true
				}
			}

			_, err := bootstrap.Run(ctx, opts, r.gw, os.Stdout)
			return err
		},
	}
}
