// Package commands implements all dojo slash commands.
package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DojoGenesis/dojo-cli/internal/client"
	"github.com/DojoGenesis/dojo-cli/internal/config"
	"github.com/DojoGenesis/dojo-cli/internal/hooks"
	"github.com/DojoGenesis/dojo-cli/internal/plugins"
	"github.com/fatih/color"
)

// Registry maps slash command names to handler functions.
type Registry struct {
	cfg    *config.Config
	gw     *client.Client
	cmds   map[string]Command
	plgs   []plugins.Plugin
	runner *hooks.Runner
}

// Command is a callable slash command.
type Command struct {
	Name    string
	Aliases []string
	Usage   string
	Short   string
	Run     func(ctx context.Context, args []string) error
}

// New builds the command registry.
func New(cfg *config.Config, gw *client.Client, plgs []plugins.Plugin) *Registry {
	r := &Registry{
		cfg:    cfg,
		gw:     gw,
		cmds:   make(map[string]Command),
		plgs:   plgs,
		runner: hooks.New(plgs),
	}
	r.register()
	return r
}

// Runner returns the hook runner so the REPL can fire events.
func (r *Registry) Runner() *hooks.Runner {
	return r.runner
}

// Dispatch finds and executes a slash command. Input should be the full line
// after the leading "/", e.g. "skill ls" or "chat hello world".
func (r *Registry) Dispatch(ctx context.Context, input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}
	name := strings.ToLower(parts[0])
	args := parts[1:]

	// Exact match
	if cmd, ok := r.cmds[name]; ok {
		return cmd.Run(ctx, args)
	}
	// Alias scan
	for _, cmd := range r.cmds {
		for _, a := range cmd.Aliases {
			if a == name {
				return cmd.Run(ctx, args)
			}
		}
	}
	return fmt.Errorf("unknown command /%s — type /help for a list", name)
}

func (r *Registry) add(cmd Command) {
	r.cmds[cmd.Name] = cmd
}

// ─── Registration ─────────────────────────────────────────────────────────────

func (r *Registry) register() {
	r.add(r.helpCmd())
	r.add(r.healthCmd())
	r.add(r.homeCmd())
	r.add(r.modelCmd())
	r.add(r.toolsCmd())
	r.add(r.agentCmd())
	r.add(r.skillCmd())
	r.add(r.gardenCmd())
	r.add(r.trailCmd())
	r.add(r.traceCmd())
	r.add(r.pilotCmd())
	r.add(r.hooksCmd())
	r.add(r.settingsCmd())
}

// ─── /help ────────────────────────────────────────────────────────────────────

func (r *Registry) helpCmd() Command {
	return Command{
		Name:  "help",
		Usage: "/help",
		Short: "List available slash commands",
		Run: func(ctx context.Context, args []string) error {
			header := color.New(color.FgGreen, color.Bold)
			sub := color.New(color.FgHiBlack)
			fmt.Println()
			header.Println("Dojo CLI — slash commands")
			fmt.Println()
			cmds := []struct{ cmd, desc string }{
				{"/help", "show this message"},
				{"/health", "gateway health + stats"},
				{"/home", "workspace state overview"},
				{"/model [ls]", "list available models and providers"},
				{"/tools [ls]", "list registered MCP tools"},
				{"/agent ls", "list agents registered in the gateway"},
				{"/skill ls [filter]", "list skills (optionally filter by name)"},
				{"/garden ls", "list memory seeds"},
				{"/garden stats", "memory garden statistics"},
				{"/garden plant <text>", "plant a new seed into the garden"},
				{"/trail", "show memory timeline"},
				{"/trace", "show trace info and guidance"},
				{"/pilot", "live SSE event stream (Ctrl+C to stop)"},
				{"/hooks ls", "list loaded hook rules from plugins"},
				{"/hooks fire <event>", "manually fire a hook event (for testing)"},
				{"/settings", "show config file path and active settings"},
			}
			for _, c := range cmds {
				fmt.Printf("  %-32s", color.CyanString(c.cmd))
				sub.Println(c.desc)
			}
			fmt.Println()
			fmt.Println("  Type a message without / to chat with the gateway.")
			fmt.Println("  Ctrl+C or type exit to quit.")
			fmt.Println()
			return nil
		},
	}
}

// ─── /health ─────────────────────────────────────────────────────────────────

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
			label := color.New(color.FgHiBlack)
			val := color.New(color.FgGreen, color.Bold)
			fmt.Println()
			printKV(label, val, "status", h.Status)
			printKV(label, val, "version", h.Version)
			printKV(label, val, "uptime", fmt.Sprintf("%ds", h.UptimeSeconds))
			printKV(label, val, "requests", fmt.Sprintf("%d", h.RequestsProcessed))
			for name, st := range h.Providers {
				printKV(label, val, "provider/"+name, st)
			}
			printKV(label, val, "gateway", r.cfg.Gateway.URL)
			fmt.Println()
			return nil
		},
	}
}

// ─── /home ───────────────────────────────────────────────────────────────────

func (r *Registry) homeCmd() Command {
	return Command{
		Name:    "home",
		Aliases: []string{"ws", "workspace"},
		Usage:   "/home",
		Short:   "Workspace state overview",
		Run: func(ctx context.Context, args []string) error {
			h, err := r.gw.Health(ctx)
			if err != nil {
				return fmt.Errorf("gateway unreachable: %w", err)
			}
			agents, _ := r.gw.Agents(ctx)
			seeds, _ := r.gw.Seeds(ctx)

			header := color.New(color.FgGreen, color.Bold)
			dim := color.New(color.FgHiBlack)

			fmt.Println()
			header.Println("  Dojo Workspace")
			dim.Printf("  %s\n\n", r.cfg.Gateway.URL)

			fmt.Printf("  %-18s %s\n", color.CyanString("gateway"), colorStatus(h.Status))
			fmt.Printf("  %-18s %s\n", color.CyanString("agents"), color.WhiteString("%d", len(agents)))
			fmt.Printf("  %-18s %s\n", color.CyanString("seeds"), color.WhiteString("%d", len(seeds)))
			fmt.Printf("  %-18s %s\n", color.CyanString("plugins"), color.WhiteString("%d", len(r.plgs)))
			fmt.Println()
			return nil
		},
	}
}

// ─── /model ──────────────────────────────────────────────────────────────────

func (r *Registry) modelCmd() Command {
	return Command{
		Name:    "model",
		Aliases: []string{"models", "providers"},
		Usage:   "/model [ls]",
		Short:   "List models and providers",
		Run: func(ctx context.Context, args []string) error {
			providers, err := r.gw.Providers(ctx)
			if err != nil {
				// fallback to /v1/models
				models, err2 := r.gw.Models(ctx)
				if err2 != nil {
					return fmt.Errorf("could not fetch models: %w", err)
				}
				fmt.Println()
				header := color.New(color.FgGreen, color.Bold)
				header.Printf("  Models (%d)\n\n", len(models))
				for _, m := range models {
					fmt.Printf("  %s  %s\n",
						color.CyanString("%-42s", m.ID),
						color.HiBlackString(m.Provider),
					)
				}
				fmt.Println()
				return nil
			}

			fmt.Println()
			header := color.New(color.FgGreen, color.Bold)
			header.Printf("  Providers (%d)\n\n", len(providers))
			for _, p := range providers {
				status := colorStatus(p.Status)
				caps := ""
				if p.Info != nil && len(p.Info.Capabilities) > 0 {
					caps = color.HiBlackString(" [%s]", strings.Join(p.Info.Capabilities, ", "))
				}
				fmt.Printf("  %s  %s%s\n", color.CyanString("%-20s", p.Name), status, caps)
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── /tools ──────────────────────────────────────────────────────────────────

func (r *Registry) toolsCmd() Command {
	return Command{
		Name:    "tools",
		Aliases: []string{"tool"},
		Usage:   "/tools [ls]",
		Short:   "List registered MCP tools",
		Run: func(ctx context.Context, args []string) error {
			tools, err := r.gw.Tools(ctx)
			if err != nil {
				return fmt.Errorf("could not fetch tools: %w", err)
			}
			fmt.Println()
			header := color.New(color.FgGreen, color.Bold)
			header.Printf("  Tools (%d)\n\n", len(tools))

			// Group by namespace
			ns := map[string][]client.Tool{}
			order := []string{}
			for _, t := range tools {
				n := t.Namespace
				if n == "" {
					n = "builtin"
				}
				if _, seen := ns[n]; !seen {
					order = append(order, n)
				}
				ns[n] = append(ns[n], t)
			}
			for _, n := range order {
				color.New(color.FgYellow).Printf("  [%s]\n", n)
				for _, t := range ns[n] {
					fmt.Printf("    %s  %s\n",
						color.CyanString("%-34s", t.Name),
						color.HiBlackString(truncate(t.Description, 60)),
					)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── /agent ──────────────────────────────────────────────────────────────────

func (r *Registry) agentCmd() Command {
	return Command{
		Name:    "agent",
		Aliases: []string{"agents"},
		Usage:   "/agent ls",
		Short:   "List agents registered in the gateway",
		Run: func(ctx context.Context, args []string) error {
			agents, err := r.gw.Agents(ctx)
			if err != nil {
				return fmt.Errorf("could not fetch agents: %w", err)
			}
			fmt.Println()
			header := color.New(color.FgGreen, color.Bold)
			header.Printf("  Agents (%d)\n\n", len(agents))
			if len(agents) == 0 {
				color.HiBlack("  No agents registered. Start the gateway with agent configs.\n")
				fmt.Println()
				return nil
			}
			for _, a := range agents {
				status := colorStatus(a.Status)
				fmt.Printf("  %s  %s\n",
					color.CyanString("%-32s", a.AgentID),
					status,
				)
				if a.Disposition != nil {
					color.HiBlack("    tone=%s pacing=%s\n", a.Disposition.Tone, a.Disposition.Pacing)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── /skill ──────────────────────────────────────────────────────────────────

func (r *Registry) skillCmd() Command {
	return Command{
		Name:    "skill",
		Aliases: []string{"skills"},
		Usage:   "/skill ls [filter]",
		Short:   "List skills (optionally filter by name)",
		Run: func(ctx context.Context, args []string) error {
			// args[0] may be "ls" or a filter term
			filter := ""
			for _, a := range args {
				if a != "ls" {
					filter = strings.ToLower(a)
				}
			}

			skills, err := r.gw.Skills(ctx)
			if err != nil {
				return fmt.Errorf("could not fetch skills: %w", err)
			}

			// Filter
			if filter != "" {
				var filtered []client.Skill
				for _, s := range skills {
					if strings.Contains(strings.ToLower(s.Name), filter) ||
						strings.Contains(strings.ToLower(s.Plugin), filter) {
						filtered = append(filtered, s)
					}
				}
				skills = filtered
			}

			fmt.Println()
			header := color.New(color.FgGreen, color.Bold)
			if filter != "" {
				header.Printf("  Skills matching %q (%d)\n\n", filter, len(skills))
			} else {
				header.Printf("  Skills (%d)\n\n", len(skills))
			}

			if len(skills) == 0 {
				color.HiBlack("  No skills found.\n\n")
				return nil
			}

			// Group by category
			cats := map[string][]client.Skill{}
			order := []string{}
			for _, s := range skills {
				cat := s.Category
				if cat == "" {
					cat = "general"
				}
				if _, seen := cats[cat]; !seen {
					order = append(order, cat)
				}
				cats[cat] = append(cats[cat], s)
			}
			for _, cat := range order {
				color.New(color.FgYellow).Printf("  [%s]\n", cat)
				for _, s := range cats[cat] {
					plugin := ""
					if s.Plugin != "" {
						plugin = color.HiBlackString("(%s)", s.Plugin)
					}
					fmt.Printf("    %s %s\n",
						color.CyanString("%-40s", s.Name),
						plugin,
					)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── /garden ─────────────────────────────────────────────────────────────────

func (r *Registry) gardenCmd() Command {
	return Command{
		Name:    "garden",
		Aliases: []string{"seeds", "memory"},
		Usage:   "/garden [ls|stats|plant <text>|harvest]",
		Short:   "Memory garden — list seeds, show stats, or plant new seeds",
		Run: func(ctx context.Context, args []string) error {
			sub := "ls"
			if len(args) > 0 {
				sub = strings.ToLower(args[0])
			}

			switch sub {
			case "stats":
				stats, err := r.gw.GardenStats(ctx)
				if err != nil {
					return fmt.Errorf("could not fetch garden stats: %w", err)
				}
				fmt.Println()
				header := color.New(color.FgGreen, color.Bold)
				header.Println("  Garden Stats")
				fmt.Println()
				label := color.New(color.FgHiBlack)
				val := color.New(color.FgWhite)
				for k, v := range stats {
					printKV(label, val, k, fmt.Sprintf("%v", v))
				}
				fmt.Println()

			case "plant":
				// /garden plant <text...>
				if len(args) < 2 {
					return fmt.Errorf("usage: /garden plant <text>")
				}
				text := strings.Join(args[1:], " ")
				title := "Planted " + time.Now().Format("2006-01-02")
				req := client.CreateSeedRequest{
					Name:    title,
					Content: text,
				}
				seed, err := r.gw.CreateSeed(ctx, req)
				if err != nil {
					return fmt.Errorf("could not plant seed: %w", err)
				}
				fmt.Println()
				color.New(color.FgGreen, color.Bold).Println("  Seed planted")
				label := color.New(color.FgHiBlack)
				val := color.New(color.FgWhite)
				if seed != nil {
					printKV(label, val, "id", seed.ID)
					printKV(label, val, "name", seed.Name)
				}
				fmt.Println()

			case "harvest": // alias for ls
				fallthrough
			default: // ls
				seeds, err := r.gw.Seeds(ctx)
				if err != nil {
					return fmt.Errorf("could not fetch seeds: %w", err)
				}
				fmt.Println()
				header := color.New(color.FgGreen, color.Bold)
				header.Printf("  Seeds (%d)\n\n", len(seeds))
				if len(seeds) == 0 {
					color.HiBlack("  Garden is empty. Use /garden plant <text> to add a seed.\n\n")
					return nil
				}
				for _, s := range seeds {
					fmt.Printf("  %s  %s\n",
						color.CyanString("%-44s", s.Name),
						color.HiBlackString(truncate(s.Content, 50)),
					)
				}
				fmt.Println()
			}
			return nil
		},
	}
}

// ─── /trail ──────────────────────────────────────────────────────────────────

func (r *Registry) trailCmd() Command {
	return Command{
		Name:  "trail",
		Usage: "/trail",
		Short: "Show memory timeline",
		Run: func(ctx context.Context, args []string) error {
			memories, err := r.gw.Memories(ctx)
			if err != nil {
				return fmt.Errorf("could not fetch memory trail: %w", err)
			}
			fmt.Println()
			header := color.New(color.FgGreen, color.Bold)
			header.Printf("  Memory Trail (%d)\n\n", len(memories))
			if len(memories) == 0 {
				color.HiBlack("  No memory entries yet.\n\n")
				return nil
			}
			for _, m := range memories {
				fmt.Printf("  %s  %s\n",
					color.HiBlackString("%-20s", m.CreatedAt),
					color.WhiteString(truncate(m.Content, 80)),
				)
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── /trace ──────────────────────────────────────────────────────────────────

func (r *Registry) traceCmd() Command {
	return Command{
		Name:  "trace",
		Usage: "/trace",
		Short: "Show trace info and guidance",
		Run: func(ctx context.Context, args []string) error {
			fmt.Println()
			header := color.New(color.FgGreen, color.Bold)
			dim := color.New(color.FgHiBlack)
			header.Println("  Dojo Trace")
			fmt.Println()
			dim.Println("  Trace follows the active session's decision and tool-use history.")
			dim.Println("  Connect the gateway with --trace to enable full trace output.")
			fmt.Println()
			printKV(color.New(color.FgHiBlack), color.New(color.FgWhite), "gateway", r.cfg.Gateway.URL)
			fmt.Println()
			return nil
		},
	}
}

// ─── /pilot ──────────────────────────────────────────────────────────────────

func (r *Registry) pilotCmd() Command {
	return Command{
		Name:  "pilot",
		Usage: "/pilot",
		Short: "Live SSE event stream (Ctrl+C to stop)",
		Run: func(ctx context.Context, args []string) error {
			fmt.Println()
			color.New(color.FgGreen, color.Bold).Println("  Pilot — live event stream  (Ctrl+C to stop)")
			fmt.Println()

			clientID := fmt.Sprintf("dojo-cli-%d", time.Now().UnixMilli())
			color.HiBlack("  client_id: %s\n\n", clientID)

			return r.gw.PilotStream(ctx, clientID, func(chunk client.SSEChunk) {
				ev := chunk.Event
				if ev == "" {
					ev = "message"
				}
				fmt.Printf("  %s  %s\n",
					color.YellowString("%-16s", ev),
					color.WhiteString(truncate(chunk.Data, 100)),
				)
			})
		},
	}
}

// ─── /hooks ──────────────────────────────────────────────────────────────────

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
				color.New(color.FgGreen).Println("  done")
				fmt.Println()

			default: // ls
				// Count total rules
				totalRules := 0
				for _, p := range r.plgs {
					totalRules += len(p.HookRules)
				}

				fmt.Println()
				header := color.New(color.FgGreen, color.Bold)
				header.Printf("  Hooks (%d rules across %d plugins)\n\n", totalRules, len(r.plgs))

				if totalRules == 0 {
					color.HiBlack("  No hook rules loaded. Place plugins in %s\n\n", r.cfg.Plugins.Path)
					return nil
				}

				for _, p := range r.plgs {
					if len(p.HookRules) == 0 {
						continue
					}
					color.New(color.FgYellow).Printf("  [%s]\n", p.Name)
					for _, rule := range p.HookRules {
						for _, h := range rule.Hooks {
							asyncLabel := ""
							if h.Async {
								asyncLabel = color.HiBlackString("  (async)")
							}
							cmd := h.Command
							if cmd == "" {
								cmd = h.Prompt
							}
							if cmd == "" {
								cmd = h.URL
							}
							fmt.Printf("    %s  %s  %s%s\n",
								color.CyanString("%-20s", rule.Event),
								color.WhiteString("%-10s", h.Type),
								color.HiBlackString(truncate(cmd, 50)),
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

// ─── /settings ───────────────────────────────────────────────────────────────

func (r *Registry) settingsCmd() Command {
	return Command{
		Name:    "settings",
		Aliases: []string{"config", "cfg"},
		Usage:   "/settings",
		Short:   "Show active config and settings file path",
		Run: func(ctx context.Context, args []string) error {
			label := color.New(color.FgHiBlack)
			val := color.New(color.FgWhite)
			fmt.Println()
			color.New(color.FgGreen, color.Bold).Println("  Active Settings")
			fmt.Println()
			printKV(label, val, "config file", config.SettingsPath())
			printKV(label, val, "gateway.url", r.cfg.Gateway.URL)
			printKV(label, val, "gateway.timeout", r.cfg.Gateway.Timeout)
			printKV(label, val, "plugins.path", r.cfg.Plugins.Path)
			printKV(label, val, "plugins loaded", fmt.Sprintf("%d", len(r.plgs)))
			printKV(label, val, "defaults.provider", orDefault(r.cfg.Defaults.Provider, "(auto)"))
			printKV(label, val, "defaults.disposition", r.cfg.Defaults.Disposition)
			printKV(label, val, "defaults.model", orDefault(r.cfg.Defaults.Model, "(auto)"))
			fmt.Println()
			return nil
		},
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func printKV(label, val *color.Color, key, value string) {
	label.Printf("  %-24s", key)
	val.Println(value)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func colorStatus(s string) string {
	switch strings.ToLower(s) {
	case "ok", "healthy", "active", "running", "ready":
		return color.GreenString(s)
	case "loading", "starting":
		return color.YellowString(s)
	case "", "unknown":
		return color.HiBlackString("unknown")
	default:
		return color.RedString(s)
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
