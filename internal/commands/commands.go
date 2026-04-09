// Package commands implements all dojo slash commands.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DojoGenesis/dojo-cli/internal/client"
	"github.com/DojoGenesis/dojo-cli/internal/config"
	"github.com/DojoGenesis/dojo-cli/internal/hooks"
	"github.com/DojoGenesis/dojo-cli/internal/plugins"
	"github.com/fatih/color"
	gcolor "github.com/gookit/color"
)

// Registry maps slash command names to handler functions.
type Registry struct {
	cfg     *config.Config
	gw      *client.Client
	cmds    map[string]Command
	plgs    []plugins.Plugin
	runner  *hooks.Runner
	session *string // pointer to REPL's active session ID
}

// Command is a callable slash command.
type Command struct {
	Name    string
	Aliases []string
	Usage   string
	Short   string
	Run     func(ctx context.Context, args []string) error
}

// New builds the command registry. session is a pointer to the REPL's active session ID
// so that /session new and /session <id> can update it across turns.
func New(cfg *config.Config, gw *client.Client, plgs []plugins.Plugin, session *string) *Registry {
	r := &Registry{
		cfg:     cfg,
		gw:      gw,
		cmds:    make(map[string]Command),
		plgs:    plgs,
		runner:  hooks.New(plgs),
		session: session,
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
	r.add(r.sessionCmd())
	r.add(r.runCmd())
}

// ─── /help ────────────────────────────────────────────────────────────────────

func (r *Registry) helpCmd() Command {
	return Command{
		Name:  "help",
		Usage: "/help",
		Short: "List available slash commands",
		Run: func(ctx context.Context, args []string) error {
			fmt.Println()
			// Section header: warm-amber + bold
			color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("Dojo CLI — slash commands"))
			fmt.Println()
			fmt.Println()
			cmds := []struct{ cmd, desc string }{
				{"/help", "show this message"},
				{"/health", "gateway health + stats"},
				{"/home", "workspace state overview"},
				{"/model [ls]", "list available models and providers"},
				{"/tools [ls]", "list registered MCP tools"},
				{"/agent ls", "list agents registered in the gateway"},
				{"/agent dispatch <mode> <msg>", "create agent and stream response"},
				{"/agent chat <id> <msg>", "chat with existing agent by ID"},
				{"/skill ls [filter]", "list skills (optionally filter by name)"},
				{"/session", "show active session ID"},
				{"/session new", "start a fresh session"},
				{"/session <id>", "resume a prior session by ID"},
				{"/run <task>", "submit multi-step orchestration plan"},
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
				// Command names in golden-orange, descriptions in cloud-gray
				fmt.Printf("  %-32s", gcolor.HEX("#f4a261").Sprint(c.cmd))
				fmt.Println(gcolor.HEX("#94a3b8").Sprint(c.desc))
			}
			fmt.Println()
			fmt.Println(gcolor.HEX("#94a3b8").Sprint("  Type a message without / to chat with the gateway."))
			fmt.Println(gcolor.HEX("#94a3b8").Sprint("  Ctrl+C or type exit to quit."))
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

			fmt.Println()
			// Section header: warm-amber + bold
			color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("  Dojo Workspace"))
			fmt.Println()
			fmt.Println(gcolor.HEX("#94a3b8").Sprint("  " + r.cfg.Gateway.URL))
			fmt.Println()

			fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("gateway"), colorStatus(h.Status))
			fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("agents"), color.WhiteString("%d", len(agents)))
			fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("seeds"), color.WhiteString("%d", len(seeds)))
			fmt.Printf("  %-18s %s\n", gcolor.HEX("#f4a261").Sprint("plugins"), color.WhiteString("%d", len(r.plgs)))
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
				color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Models (%d)\n\n", len(models)))
				for _, m := range models {
					fmt.Printf("  %s  %s\n",
						gcolor.HEX("#f4a261").Sprintf("%-42s", m.ID),
						gcolor.HEX("#94a3b8").Sprint(m.Provider),
					)
				}
				fmt.Println()
				return nil
			}

			fmt.Println()
			color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Providers (%d)\n\n", len(providers)))
			for _, p := range providers {
				status := colorStatus(p.Status)
				caps := ""
				if p.Info != nil && len(p.Info.Capabilities) > 0 {
					caps = gcolor.HEX("#94a3b8").Sprintf(" [%s]", strings.Join(p.Info.Capabilities, ", "))
				}
				fmt.Printf("  %s  %s%s\n", gcolor.HEX("#f4a261").Sprintf("%-20s", p.Name), status, caps)
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
			color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Tools (%d)\n\n", len(tools)))

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
				// Glass-effect section divider
				fmt.Printf("  %s %s %s\n",
					gcolor.HEX("#1a3a4a").Sprint("────"),
					gcolor.HEX("#e8b04a").Sprint("["+n+"]"),
					gcolor.HEX("#1a3a4a").Sprint("────"),
				)
				for _, t := range ns[n] {
					fmt.Printf("    %s  %s\n",
						gcolor.HEX("#f4a261").Sprintf("%-34s", t.Name),
						gcolor.HEX("#94a3b8").Sprint(truncate(t.Description, 60)),
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
		Usage:   "/agent [ls|dispatch <mode> <msg>|chat <id> <msg>]",
		Short:   "List, create, or chat with agents",
		Run: func(ctx context.Context, args []string) error {
			sub := "ls"
			if len(args) > 0 {
				sub = strings.ToLower(args[0])
			}

			switch sub {
			case "dispatch":
				// /agent dispatch [mode] <msg...>
				// mode is optional — defaults to "balanced"
				validModes := map[string]bool{
					"focused": true, "balanced": true,
					"exploratory": true, "deliberate": true,
				}
				mode := "balanced"
				var msgArgs []string
				if len(args) >= 2 && validModes[args[1]] {
					mode = args[1]
					msgArgs = args[2:]
				} else {
					msgArgs = args[1:]
				}
				if len(msgArgs) == 0 {
					return fmt.Errorf("usage: /agent dispatch [focused|balanced|exploratory|deliberate] <message>")
				}
				message := strings.Join(msgArgs, " ")

				fmt.Println()
				fmt.Println(gcolor.HEX("#94a3b8").Sprintf("  Creating agent (mode: %s)...", mode))

				agentResp, err := r.gw.CreateAgent(ctx, client.CreateAgentRequest{
					WorkspaceRoot: ".",
					ActiveMode:    mode,
				})
				if err != nil {
					return fmt.Errorf("could not create agent: %w", err)
				}

				shortID := agentResp.AgentID
				if len(shortID) > 8 {
					shortID = shortID[:8]
				}
				fmt.Printf("  %s %s",
					gcolor.HEX("#f4a261").Sprint("Agent:"),
					gcolor.HEX("#e8b04a").Sprint(shortID),
				)
				if agentResp.Disposition != nil {
					fmt.Printf("  %s", gcolor.HEX("#94a3b8").Sprintf(
						"pacing=%s depth=%s",
						agentResp.Disposition.Pacing,
						agentResp.Disposition.Depth,
					))
				}
				fmt.Println()
				fmt.Println()

				color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("  dojo  "))
				return r.streamAgentChat(ctx, agentResp.AgentID, message)

			case "chat":
				// /agent chat <id> <msg...>
				if len(args) < 3 {
					return fmt.Errorf("usage: /agent chat <agent-id> <message>")
				}
				agentID := args[1]
				message := strings.Join(args[2:], " ")
				fmt.Println()
				color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("  dojo  "))
				return r.streamAgentChat(ctx, agentID, message)

			default: // ls
				agents, err := r.gw.Agents(ctx)
				if err != nil {
					return fmt.Errorf("could not fetch agents: %w", err)
				}
				fmt.Println()
				color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Agents (%d)\n\n", len(agents)))
				if len(agents) == 0 {
					fmt.Println(gcolor.HEX("#94a3b8").Sprint("  No agents registered. Start the gateway with agent configs."))
					fmt.Println()
					return nil
				}
				for _, a := range agents {
					status := colorStatus(a.Status)
					fmt.Printf("  %s  %s\n",
						gcolor.HEX("#f4a261").Sprintf("%-32s", a.AgentID),
						status,
					)
					if a.Disposition != nil {
						fmt.Println(gcolor.HEX("#94a3b8").Sprintf("    tone=%s pacing=%s", a.Disposition.Tone, a.Disposition.Pacing))
					}
				}
				fmt.Println()
				return nil
			}
		},
	}
}

// streamAgentChat sends a message to an agent and streams the SSE response.
// Thinking and tool-call events are rendered in dim colors; text is printed inline.
func (r *Registry) streamAgentChat(ctx context.Context, agentID, message string) error {
	req := client.AgentChatRequest{
		Message: message,
		Stream:  true,
	}

	err := r.gw.AgentChatStream(ctx, agentID, req, func(chunk client.SSEChunk) {
		switch chunk.Event {
		case "thinking":
			fmt.Print(gcolor.HEX("#94a3b8").Sprint("\n  [Thinking] " + truncate(chunk.Data, 80)))
		case "tool_call":
			fmt.Print(gcolor.HEX("#457b9d").Sprintf("\n  [Tool: %s]", truncate(chunk.Data, 60)))
		case "tool_result":
			// absorbed into the response
		default:
			if text := agentExtractText(chunk.Data); text != "" {
				fmt.Print(text)
			}
		}
	})

	fmt.Println()
	fmt.Println()
	return err
}

// agentExtractText pulls readable text from an agent SSE data field.
func agentExtractText(data string) string {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err == nil {
		for _, key := range []string{"text", "content", "message", "delta"} {
			if v, ok := m[key].(string); ok {
				return v
			}
		}
		return ""
	}
	return data
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
			if filter != "" {
				color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Skills matching %q (%d)\n\n", filter, len(skills)))
			} else {
				color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Skills (%d)\n\n", len(skills)))
			}

			if len(skills) == 0 {
				fmt.Println(gcolor.HEX("#94a3b8").Sprint("  No skills found."))
				fmt.Println()
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
				// Glass-effect section divider
				fmt.Printf("  %s %s %s\n",
					gcolor.HEX("#1a3a4a").Sprint("────"),
					gcolor.HEX("#e8b04a").Sprint("["+cat+"]"),
					gcolor.HEX("#1a3a4a").Sprint("────"),
				)
				for _, s := range cats[cat] {
					plugin := ""
					if s.Plugin != "" {
						plugin = gcolor.HEX("#94a3b8").Sprintf("(%s)", s.Plugin)
					}
					fmt.Printf("    %s %s\n",
						gcolor.HEX("#f4a261").Sprintf("%-40s", s.Name),
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
				color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("  Garden Stats"))
				fmt.Println()
				fmt.Println()
				for k, v := range stats {
					printKV(k, fmt.Sprintf("%v", v))
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
				color.New(color.Bold).Print(gcolor.HEX("#7fb88c").Sprint("  Seed planted"))
				fmt.Println()
				if seed != nil {
					printKV("id", seed.ID)
					printKV("name", seed.Name)
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
				color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Seeds (%d)\n\n", len(seeds)))
				if len(seeds) == 0 {
					fmt.Println(gcolor.HEX("#94a3b8").Sprint("  Garden is empty. Use /garden plant <text> to add a seed."))
					fmt.Println()
					return nil
				}
				for _, s := range seeds {
					fmt.Printf("  %s  %s\n",
						gcolor.HEX("#f4a261").Sprintf("%-44s", s.Name),
						gcolor.HEX("#94a3b8").Sprint(truncate(s.Content, 50)),
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
			color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Memory Trail (%d)\n\n", len(memories)))
			if len(memories) == 0 {
				fmt.Println(gcolor.HEX("#94a3b8").Sprint("  No memory entries yet."))
				fmt.Println()
				return nil
			}
			for _, m := range memories {
				fmt.Printf("  %s  %s\n",
					gcolor.HEX("#94a3b8").Sprintf("%-20s", m.CreatedAt),
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
			color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("  Dojo Trace"))
			fmt.Println()
			fmt.Println()
			// Trace context in info-steel
			fmt.Println(gcolor.HEX("#457b9d").Sprint("  Trace follows the active session's decision and tool-use history."))
			fmt.Println(gcolor.HEX("#457b9d").Sprint("  Connect the gateway with --trace to enable full trace output."))
			fmt.Println()
			printKV("gateway", r.cfg.Gateway.URL)
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
			color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("  Pilot — live event stream  (Ctrl+C to stop)"))
			fmt.Println()
			fmt.Println()

			clientID := fmt.Sprintf("dojo-cli-%d", time.Now().UnixMilli())
			fmt.Println(gcolor.HEX("#94a3b8").Sprintf("  client_id: %s", clientID))
			fmt.Println()

			return r.gw.PilotStream(ctx, clientID, func(chunk client.SSEChunk) {
				ev := chunk.Event
				if ev == "" {
					ev = "message"
				}
				fmt.Printf("  %s  %s\n",
					// Pilot events in info-steel
					gcolor.HEX("#457b9d").Sprintf("%-16s", ev),
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
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  done"))
				fmt.Println()

			default: // ls
				// Count total rules
				totalRules := 0
				for _, p := range r.plgs {
					totalRules += len(p.HookRules)
				}

				fmt.Println()
				color.New(color.Bold).Printf(gcolor.HEX("#e8b04a").Sprintf("  Hooks (%d rules across %d plugins)\n\n", totalRules, len(r.plgs)))

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
						gcolor.HEX("#1a3a4a").Sprint("────"),
						gcolor.HEX("#e8b04a").Sprint("["+p.Name+"]"),
						gcolor.HEX("#1a3a4a").Sprint("────"),
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
								color.WhiteString("%-10s", h.Type),
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

// ─── /settings ───────────────────────────────────────────────────────────────

func (r *Registry) settingsCmd() Command {
	return Command{
		Name:    "settings",
		Aliases: []string{"config", "cfg"},
		Usage:   "/settings",
		Short:   "Show active config and settings file path",
		Run: func(ctx context.Context, args []string) error {
			fmt.Println()
			color.New(color.Bold).Print(gcolor.HEX("#e8b04a").Sprint("  Active Settings"))
			fmt.Println()
			fmt.Println()
			printKV("config file", config.SettingsPath())
			printKV("gateway.url", r.cfg.Gateway.URL)
			printKV("gateway.timeout", r.cfg.Gateway.Timeout)
			printKV("plugins.path", r.cfg.Plugins.Path)
			printKV("plugins loaded", fmt.Sprintf("%d", len(r.plgs)))
			printKV("defaults.provider", orDefault(r.cfg.Defaults.Provider, "(auto)"))
			printKV("defaults.disposition", r.cfg.Defaults.Disposition)
			printKV("defaults.model", orDefault(r.cfg.Defaults.Model, "(auto)"))
			fmt.Println()
			return nil
		},
	}
}

// ─── /session ────────────────────────────────────────────────────────────────

func (r *Registry) sessionCmd() Command {
	return Command{
		Name:  "session",
		Usage: "/session [new|<id>]",
		Short: "Show or change the active session ID",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				fmt.Println()
				printKV("session", *r.session)
				fmt.Println()
				return nil
			}
			switch args[0] {
			case "new":
				*r.session = fmt.Sprintf("dojo-cli-%s", time.Now().Format("20060102-150405"))
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  New session started"))
				printKV("session", *r.session)
				fmt.Println()
			default:
				*r.session = args[0]
				fmt.Println()
				fmt.Println(gcolor.HEX("#7fb88c").Sprint("  Session resumed"))
				printKV("session", *r.session)
				fmt.Println()
			}
			return nil
		},
	}
}

// ─── /run ─────────────────────────────────────────────────────────────────────

func (r *Registry) runCmd() Command {
	return Command{
		Name:  "run",
		Usage: "/run <task description>",
		Short: "Submit a multi-step orchestration plan and watch it execute",
		Run: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: /run <task description>")
			}
			task := strings.Join(args, " ")
			planID := fmt.Sprintf("plan-%d", time.Now().UnixMilli())

			req := client.OrchestrateRequest{
				Plan: client.ExecutionPlan{
					ID:   planID,
					Name: task,
					DAG: []client.ToolInvocation{
						{
							ID:        "step1",
							ToolName:  "chat",
							Input:     map[string]any{"message": task, "session_id": *r.session},
							DependsOn: []string{},
						},
					},
				},
			}

			fmt.Println()
			fmt.Println(gcolor.HEX("#94a3b8").Sprintf("  Submitting plan: %s", truncate(task, 60)))

			status, err := r.gw.Orchestrate(ctx, req)
			if err != nil {
				return fmt.Errorf("orchestration failed: %w", err)
			}

			printKV("execution_id", status.ExecutionID)
			printKV("status", colorStatus(status.Status))
			fmt.Println()

			// Poll every second until completed or failed.
			seen := map[string]string{} // node id → last printed status
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					dag, err := r.gw.OrchestrationDAG(ctx, status.ExecutionID)
					if err != nil {
						fmt.Println(gcolor.HEX("#e63946").Sprint("  poll error: " + err.Error()))
						continue
					}

					for _, node := range dag.Nodes {
						id, _ := node["id"].(string)
						st, _ := node["status"].(string)
						if id == "" || seen[id] == st {
							continue
						}
						seen[id] = st

						icon := "○"
						switch st {
						case "completed":
							icon = gcolor.HEX("#7fb88c").Sprint("✓")
						case "running":
							icon = gcolor.HEX("#e8b04a").Sprint("⟳")
						case "failed":
							icon = gcolor.HEX("#e63946").Sprint("✗")
						}

						line := fmt.Sprintf("  [%s %s] %s", icon, gcolor.HEX("#f4a261").Sprint(id), colorStatus(st))
						if out, _ := node["output"].(string); out != "" {
							line += "  " + gcolor.HEX("#94a3b8").Sprint(truncate(out, 60))
						}
						fmt.Println(line)
					}

					if dag.Status == "completed" || dag.Status == "failed" {
						fmt.Printf("\n  %s\n\n", colorStatus(dag.Status))
						return nil
					}
				}
			}
		},
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// printKV prints a key-value pair: key in cloud-gray, value in white.
func printKV(key, value string) {
	fmt.Printf("%s%s\n",
		gcolor.HEX("#94a3b8").Sprintf("  %-24s", key),
		color.WhiteString(value),
	)
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
		return gcolor.HEX("#7fb88c").Sprint(s) // soft-sage
	case "loading", "starting":
		return gcolor.HEX("#e8b04a").Sprint(s) // warm-amber
	case "", "unknown":
		return gcolor.HEX("#94a3b8").Sprint("unknown") // cloud-gray
	default:
		return gcolor.HEX("#e63946").Sprint(s) // danger-red
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
