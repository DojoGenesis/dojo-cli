// Package guide defines interactive step-by-step guides for learning dojo-cli features.
// Guides track progress in state.GuideProgress and award XP on step completion.
package guide

import (
	"fmt"
	"strings"

	"github.com/DojoGenesis/cli/internal/state"
)

// Step is one interactive step in a guide.
type Step struct {
	Title   string // short label, e.g. "Check the gateway"
	Action  string // instruction shown to the user
	Command string // command prefix to match (with leading slash), e.g. "/health"
	XP      int    // XP awarded on completion
	Hint    string // extra context shown in /guide status
}

// Guide is a named sequence of steps that walks a user through a feature.
type Guide struct {
	ID    string
	Title string
	Short string
	Steps []Step
}

// All is the full catalogue of built-in guides.
var All = []Guide{
	{
		ID:    "welcome",
		Title: "Your First 5 Minutes",
		Short: "Tour the essential commands",
		Steps: []Step{
			{
				Title:   "Check the gateway",
				Action:  "Run /health to verify your gateway connection.",
				Command: "/health",
				XP:      15,
				Hint:    "/health tells you if the Dojo gateway is reachable and its current version.",
			},
			{
				Title:   "Inspect your workspace",
				Action:  "Run /home to see your current workspace state.",
				Command: "/home",
				XP:      15,
				Hint:    "/home shows session info, loaded plugins, active agents, and key settings.",
			},
			{
				Title:   "List available models",
				Action:  "Run /model ls to see AI models available in Dojo.",
				Command: "/model",
				XP:      15,
				Hint:    "You can also run /model set <name> to switch the active model.",
			},
			{
				Title:   "Daily reflection",
				Action:  "Run /practice to complete a daily reflection prompt.",
				Command: "/practice",
				XP:      15,
				Hint:    "/practice rotates prompts by day of week. Return daily to build your streak.",
			},
			{
				Title:   "View your profile card",
				Action:  "Run /card to see your XP, belt rank, and achievements.",
				Command: "/card",
				XP:      15,
				Hint:    "Aliases: /profile, /rank, /belt",
			},
		},
	},
	{
		ID:    "spirit",
		Title: "XP & Belts",
		Short: "Master the spirit system",
		Steps: []Step{
			{
				Title:   "Meet the sensei",
				Action:  "Run /sensei to receive a koan.",
				Command: "/sensei",
				XP:      15,
				Hint:    "The sensei unlocks deeper wisdom as your belt rank increases.",
			},
			{
				Title:   "View your activity trail",
				Action:  "Run /trail to see your command history.",
				Command: "/trail",
				XP:      15,
				Hint:    "The trail shows every command you've run, with timestamps — good for reflection.",
			},
			{
				Title:   "Dispatch your first agent",
				Action:  "Run /agent dispatch balanced Hello! What can you do?",
				Command: "/agent dispatch",
				XP:      20,
				Hint:    "Agents earn bonus XP. Modes: focused, balanced, exploratory, deliberate.",
			},
			{
				Title:   "Check your belt progress",
				Action:  "Run /card to see how much XP you've earned.",
				Command: "/card",
				XP:      15,
				Hint:    "Belt promotions are announced automatically — keep running commands to level up.",
			},
		},
	},
	{
		ID:    "agents",
		Title: "Working with Agents",
		Short: "Create and chat with AI agents",
		Steps: []Step{
			{
				Title:   "List existing agents",
				Action:  "Run /agent ls to see your agent roster.",
				Command: "/agent ls",
				XP:      15,
				Hint:    "Agents persist between sessions — you can have many running at once.",
			},
			{
				Title:   "Dispatch a new agent",
				Action:  "Run /agent dispatch balanced Hello! What can you help me with?",
				Command: "/agent dispatch",
				XP:      20,
				Hint:    "Copy the agent ID from the output — you'll use it to chat with /agent chat.",
			},
			{
				Title:   "Browse available skills",
				Action:  "Run /skill ls to see a category summary of all loaded skills.",
				Command: "/skill",
				XP:      15,
				Hint:    "Pick a category from the summary and run /skill ls <category> to drill in. Use /skill ls all for the full list, or /skill search <query> to search.",
			},
		},
	},
	{
		ID:    "memory",
		Title: "The Memory Garden",
		Short: "Plant and harvest memory seeds",
		Steps: []Step{
			{
				Title:   "Open the garden",
				Action:  "Run /garden ls to list your memory seeds.",
				Command: "/garden",
				XP:      15,
				Hint:    "The garden is a persistent knowledge store. Seeds survive across sessions.",
			},
			{
				Title:   "Plant a seed",
				Action:  `Run /garden plant "My first Dojo memory" to add a seed.`,
				Command: "/garden plant",
				XP:      20,
				Hint:    "Seeds can be harvested with /garden harvest or searched with /garden search.",
			},
			{
				Title:   "Check garden statistics",
				Action:  "Run /garden stats to see your garden at a glance.",
				Command: "/garden stats",
				XP:      15,
				Hint:    "Stats show total seeds, recent activity, and storage usage.",
			},
		},
	},

	// ─── Extended guides ───────────────────────────────────────────────────────

	{
		ID:    "models",
		Title: "Switching AI Models",
		Short: "Choose and query different models",
		Steps: []Step{
			{
				Title:   "List available models",
				Action:  "Run /model ls to see every model and provider registered with Dojo.",
				Command: "/model ls",
				XP:      15,
				Hint:    "Models are grouped by provider. Only models whose API keys are configured appear.",
			},
			{
				Title:   "Switch the active model",
				Action:  "Run /model set <provider> <model> to change the default model.",
				Command: "/model set",
				XP:      20,
				Hint:    "Example: /model set anthropic claude-sonnet-4-6. The new model persists for the session.",
			},
			{
				Title:   "Send a direct API call",
				Action:  "Run /model direct <provider> <model> <message> to query a model directly.",
				Command: "/model direct",
				XP:      20,
				Hint:    "/model direct bypasses sessions and routing — useful for quick one-shot tests.",
			},
		},
	},

	{
		ID:    "sessions",
		Title: "Session Management",
		Short: "Create, switch, and resume sessions",
		Steps: []Step{
			{
				Title:   "View your current session",
				Action:  "Run /session to see the active session ID and metadata.",
				Command: "/session",
				XP:      15,
				Hint:    "Session IDs follow the pattern dojo-cli-YYYYMMDD-HHMMSS.",
			},
			{
				Title:   "Start a fresh session",
				Action:  "Run /session new to open a clean conversation context.",
				Command: "/session new",
				XP:      15,
				Hint:    "The old session stays in history — you can resume it later with /session resume.",
			},
			{
				Title:   "Resume the last session",
				Action:  "Run /session resume to restore the most recent session context.",
				Command: "/session resume",
				XP:      20,
				Hint:    "Resuming preserves conversation history and tool context from where you left off.",
			},
		},
	},

	{
		ID:    "plugins",
		Title: "Plugin Ecosystem",
		Short: "Install and manage CoworkPlugins",
		Steps: []Step{
			{
				Title:   "List installed plugins",
				Action:  "Run /plugin ls to see all plugins currently loaded.",
				Command: "/plugin ls",
				XP:      15,
				Hint:    "Each plugin entry shows name, version, skill count, agent count, and hook count.",
			},
			{
				Title:   "Install a plugin",
				Action:  "Run /plugin install <git-url> to clone and register a plugin.",
				Command: "/plugin install",
				XP:      25,
				Hint:    "Plugins are cloned to your plugins path (see /settings). Restart to activate hooks.",
			},
			{
				Title:   "Verify the install",
				Action:  "Run /home to confirm the new plugin appears in your workspace overview.",
				Command: "/home",
				XP:      15,
				Hint:    "/home lists loaded plugins, agents, and skills — a quick sanity check after install.",
			},
			{
				Title:   "Browse new skills",
				Action:  "Run /skill ls to see the updated category summary including the plugin's new skills.",
				Command: "/skill ls",
				XP:      15,
				Hint:    "The summary shows total counts by category. Drill into a category with /skill ls <category>, or page through all with /skill ls all.",
			},
		},
	},

	{
		ID:    "skills-deep",
		Title: "Skills Deep Dive",
		Short: "Navigate, search, fetch, and inspect skills",
		Steps: []Step{
			{
				Title:   "Open the skill navigator",
				Action:  "Run /skill ls to see a summary of all skills grouped by category.",
				Command: "/skill ls",
				XP:      15,
				Hint:    "/skill ls is the entry point: it shows every category, its skill count, and a mini bar chart. No matter how many plugins you've installed, the summary stays scannable.",
			},
			{
				Title:   "Drill into a category",
				Action:  "Run /skill ls <category> to list skills in that category (e.g. /skill ls engineering).",
				Command: "/skill ls",
				XP:      15,
				Hint:    "Category names come from the summary. Large categories paginate automatically — add p2, p3, … to navigate (e.g. /skill ls engineering p2). /skill ls all shows every skill across all categories.",
			},
			{
				Title:   "Search across all skills",
				Action:  "Run /skill search <query> to find skills by name, description, or trigger.",
				Command: "/skill search",
				XP:      15,
				Hint:    "Search now scans the full skill catalogue — not just the first page. Try /skill search deploy or /skill search memory.",
			},
			{
				Title:   "Browse CAS tags",
				Action:  "Run /skill tags to see all skills registered in the content-addressable store.",
				Command: "/skill tags",
				XP:      15,
				Hint:    "Tags are the CAS layer beneath /skill ls. Each tag maps a skill name to a versioned hash. Useful for auditing what the gateway actually has vs. what your plugins provide.",
			},
			{
				Title:   "Fetch a skill by name",
				Action:  "Run /skill get <name> to retrieve a skill's full SKILL.md content.",
				Command: "/skill get",
				XP:      20,
				Hint:    "Content is pulled from the CAS on the gateway, not your local plugin files. Use this to verify what version of a skill agents are actually running.",
			},
			{
				Title:   "Inspect a skill by hash",
				Action:  "Run /skill inspect <hash> to view raw CAS content by reference.",
				Command: "/skill inspect",
				XP:      20,
				Hint:    "CAS hashes are content-addressed and stable — the same skill content always produces the same hash. Useful for pinning a specific skill version or debugging CAS drift.",
			},
		},
	},

	{
		ID:    "dispositions",
		Title: "AI Dispositions",
		Short: "Shape how Dojo thinks and responds",
		Steps: []Step{
			{
				Title:   "List all dispositions",
				Action:  "Run /disposition ls to see every preset available.",
				Command: "/disposition ls",
				XP:      15,
				Hint:    "Dispositions set pacing, depth, tone, and initiative. Think of them as personas.",
			},
			{
				Title:   "Inspect a preset",
				Action:  "Run /disposition show <name> to see a disposition's full configuration.",
				Command: "/disposition show",
				XP:      15,
				Hint:    "Each preset has 4 dials: pacing (fast/measured), depth, tone, initiative.",
			},
			{
				Title:   "Switch your active disposition",
				Action:  "Run /disposition set <name> to activate a preset.",
				Command: "/disposition set",
				XP:      20,
				Hint:    "The new disposition applies to all subsequent chat turns until you change it again.",
			},
			{
				Title:   "Feel the difference",
				Action:  "Run /sensei — notice how the tone changes with your new disposition.",
				Command: "/sensei",
				XP:      15,
				Hint:    "Dispositions influence every response. Try switching between extremes to feel the contrast.",
			},
		},
	},

	{
		ID:    "hooks",
		Title: "Automation with Hooks",
		Short: "Trigger actions on CLI events",
		Steps: []Step{
			{
				Title:   "List registered hooks",
				Action:  "Run /hooks ls to see all hook rules loaded from your plugins.",
				Command: "/hooks ls",
				XP:      15,
				Hint:    "Hooks fire on events like PreCommand, PostCommand, AgentDone, and more.",
			},
			{
				Title:   "Fire a hook manually",
				Action:  "Run /hooks fire <event> to trigger a hook event by name.",
				Command: "/hooks fire",
				XP:      20,
				Hint:    "Useful for testing hooks without waiting for a real event. Event names are case-sensitive.",
			},
			{
				Title:   "Verify hook execution",
				Action:  "Run /trail to see hook activity logged in your activity trail.",
				Command: "/trail",
				XP:      15,
				Hint:    "Hook executions appear as entries in the trail. Check timestamps to confirm firing order.",
			},
		},
	},

	{
		ID:    "snapshots",
		Title: "Snapshots & Restore",
		Short: "Save and restore session state",
		Steps: []Step{
			{
				Title:   "Save a snapshot",
				Action:  "Run /snapshot save to capture the current session state.",
				Command: "/snapshot save",
				XP:      20,
				Hint:    "Snapshots capture conversation context, active agents, and settings at a point in time.",
			},
			{
				Title:   "Restore from a snapshot",
				Action:  "Run /snapshot restore to reload a previously saved state.",
				Command: "/snapshot restore",
				XP:      20,
				Hint:    "Restoring replaces the current session state — useful after a crash or context loss.",
			},
			{
				Title:   "Export a snapshot",
				Action:  "Run /snapshot export to write the snapshot to a portable file.",
				Command: "/snapshot export",
				XP:      15,
				Hint:    "Exported snapshots can be shared or checked into version control for reproducibility.",
			},
		},
	},

	{
		ID:    "pilot",
		Title: "Live Event Streaming",
		Short: "Watch the gateway in real time",
		Steps: []Step{
			{
				Title:   "Start the plain event stream",
				Action:  "Run /pilot plain to watch raw gateway events in text mode.",
				Command: "/pilot plain",
				XP:      15,
				Hint:    "Press Ctrl+C to exit. Each line is one SSE event from the gateway.",
			},
			{
				Title:   "Generate some events",
				Action:  "Run /agent dispatch balanced Generate some activity for the stream.",
				Command: "/agent dispatch",
				XP:      20,
				Hint:    "After dispatching, re-open /pilot plain to see the agent events flow through.",
			},
			{
				Title:   "Try the TUI dashboard",
				Action:  "Run /pilot to open the full live event dashboard.",
				Command: "/pilot",
				XP:      15,
				Hint:    "The TUI shows events in a live-updating panel. Much richer than plain mode.",
			},
		},
	},

	{
		ID:    "trail-deep",
		Title: "Activity Trail",
		Short: "Log, search, and annotate your history",
		Steps: []Step{
			{
				Title:   "Review your trail",
				Action:  "Run /trail to view recent activity across all sessions.",
				Command: "/trail",
				XP:      15,
				Hint:    "The trail is an append-only log. Every command, chat, and hook shows up here.",
			},
			{
				Title:   "Add a manual note",
				Action:  `Run /trail add "First guided session complete" to log a note.`,
				Command: "/trail add",
				XP:      20,
				Hint:    "Notes appear in the trail with a timestamp — useful for marking milestones.",
			},
			{
				Title:   "Search the trail",
				Action:  "Run /trail search <keyword> to filter entries by content.",
				Command: "/trail search",
				XP:      15,
				Hint:    "Search is case-insensitive. Great for finding when you last ran a specific command.",
			},
		},
	},

	{
		ID:    "settings",
		Title: "Configuring Dojo",
		Short: "View and update your CLI settings",
		Steps: []Step{
			{
				Title:   "View current config",
				Action:  "Run /settings to print all active configuration values.",
				Command: "/settings",
				XP:      15,
				Hint:    "Config is loaded from ~/.dojo/settings.json and can be overridden by env vars.",
			},
			{
				Title:   "Check provider config",
				Action:  "Run /settings providers to see which AI providers are configured.",
				Command: "/settings providers",
				XP:      15,
				Hint:    "A provider needs an API key to be active. Missing keys show as unconfigured.",
			},
			{
				Title:   "Set a provider API key",
				Action:  "Run /settings set <provider> <api-key> to register a provider key.",
				Command: "/settings set",
				XP:      20,
				Hint:    "Keys are pushed to the gateway at session start. Restart is not required.",
			},
		},
	},

	{
		ID:    "tools",
		Title: "MCP Tools",
		Short: "Discover and exercise your tool surface",
		Steps: []Step{
			{
				Title:   "List all tools",
				Action:  "Run /tools to see every MCP tool registered with the gateway.",
				Command: "/tools",
				XP:      15,
				Hint:    "Tools come from MCP servers wired to the gateway. Each has a name, description, and schema.",
			},
			{
				Title:   "Ask an agent about tools",
				Action:  "Run /agent dispatch balanced What tools do you have access to?",
				Command: "/agent dispatch",
				XP:      20,
				Hint:    "Agents can describe their tool surface in natural language — useful for discovery.",
			},
			{
				Title:   "Inspect execution traces",
				Action:  "Run /trace to review recent tool call traces.",
				Command: "/trace",
				XP:      15,
				Hint:    "Traces show which tools were called, with what args, and what they returned.",
			},
		},
	},

	{
		ID:    "warroom",
		Title: "The War Room",
		Short: "Challenge your thinking with Scout vs Challenger",
		Steps: []Step{
			{
				Title:   "Open a war room debate",
				Action:  "Run /warroom <topic> to start a Scout vs Challenger analysis.",
				Command: "/warroom",
				XP:      20,
				Hint:    "Example: /warroom Should we use microservices or a monolith? Two agents debate both sides.",
			},
			{
				Title:   "Capture the insight",
				Action:  `Run /trail add "War room insight: <your takeaway>" to log what you learned.`,
				Command: "/trail add",
				XP:      15,
				Hint:    "Good debates produce decisions. Trail notes make them findable later.",
			},
			{
				Title:   "Plant a seed from the debate",
				Action:  `Run /garden plant "War room conclusion: <key decision>" to persist the outcome.`,
				Command: "/garden plant",
				XP:      20,
				Hint:    "Seeds are the long-term memory of your dojo. War room conclusions belong here.",
			},
		},
	},

	{
		ID:    "apps",
		Title: "External Apps",
		Short: "Launch and call MCP app servers",
		Steps: []Step{
			{
				Title:   "Check app status",
				Action:  "Run /apps status to see which MCP app servers are running.",
				Command: "/apps status",
				XP:      15,
				Hint:    "Apps are MCP servers managed by Dojo. They expose tools callable from any session.",
			},
			{
				Title:   "List available apps",
				Action:  "Run /apps ls to see all configured apps whether or not they're running.",
				Command: "/apps ls",
				XP:      15,
				Hint:    "Configured apps come from your plugin hooks and settings.",
			},
			{
				Title:   "Launch an app",
				Action:  "Run /apps launch <name> to start an MCP app server.",
				Command: "/apps launch",
				XP:      20,
				Hint:    "Once launched, an app's tools appear in /tools and are available to agents.",
			},
		},
	},
}

// Find returns the guide with the given ID, or nil.
func Find(id string) *Guide {
	for i := range All {
		if All[i].ID == id {
			return &All[i]
		}
	}
	return nil
}

// Active returns the guide currently in progress and its step index.
// Returns nil, 0 if no guide is active.
func Active(st *state.State) (*Guide, int) {
	if st.Guide.Active == "" {
		return nil, 0
	}
	g := Find(st.Guide.Active)
	if g == nil {
		return nil, 0
	}
	return g, st.Guide.Step
}

// IsCompleted reports whether the given guide ID has been finished.
func IsCompleted(st *state.State, id string) bool {
	for _, c := range st.Guide.Completed {
		if c == id {
			return true
		}
	}
	return false
}

// AdvanceResult holds the outcome of a successful step match.
type AdvanceResult struct {
	XP            int    // XP to award for this step
	GuideName     string // guide title
	GuideComplete bool   // true if this was the final step
	NextTitle     string // next step title (empty if guide complete)
	NextAction    string // next step action text (empty if guide complete)
	NextStep      int    // next step index, 1-based display
	TotalSteps    int    // total steps in the guide
}

// AdvanceStep checks whether line satisfies the current active guide step.
// If it does, st.Guide is updated in place and an AdvanceResult is returned.
// The caller is responsible for saving state and printing the result.
func AdvanceStep(st *state.State, line string) (*AdvanceResult, bool) {
	g, stepIdx := Active(st)
	if g == nil || stepIdx >= len(g.Steps) {
		return nil, false
	}

	step := g.Steps[stepIdx]
	lower := strings.ToLower(strings.TrimSpace(line))
	cmdLower := strings.ToLower(step.Command)

	if !strings.HasPrefix(lower, cmdLower) {
		return nil, false
	}

	xp := step.XP
	st.Guide.Step++
	nextIdx := st.Guide.Step

	res := &AdvanceResult{
		XP:         xp,
		GuideName:  g.Title,
		TotalSteps: len(g.Steps),
	}

	if nextIdx >= len(g.Steps) {
		// Guide complete
		st.Guide.Active = ""
		st.Guide.Step = 0
		st.Guide.Completed = append(st.Guide.Completed, g.ID)
		res.GuideComplete = true
	} else {
		next := g.Steps[nextIdx]
		res.NextTitle = next.Title
		res.NextAction = next.Action
		res.NextStep = nextIdx + 1
	}

	return res, true
}

// StepHint returns the hint for the current active step, or empty string.
func StepHint(st *state.State) string {
	g, stepIdx := Active(st)
	if g == nil || stepIdx >= len(g.Steps) {
		return ""
	}
	return g.Steps[stepIdx].Hint
}

// FormatStepBlock formats a step for display.
func FormatStepBlock(g *Guide, stepIdx int) string {
	if stepIdx >= len(g.Steps) {
		return ""
	}
	step := g.Steps[stepIdx]
	return fmt.Sprintf("  Step %d/%d — %s\n\n  %s\n",
		stepIdx+1, len(g.Steps), step.Title, step.Action)
}
