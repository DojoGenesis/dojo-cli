# Dojo CLI — Consolidated Improvement Plan

**Date:** 2026-04-09
**Baseline commit:** ba393c04fb18 (main)
**Status:** All tests passing, codebase stable

---

## Diagnosis

The CLI is functional and well-tested (118 tests, 2,478 test LOC), but is accumulating three kinds of drift:

1. **Structural drift** — `commands.go` is 2,026 lines carrying 22 command handlers, help rendering, state mutation, streaming presentation, and gateway bridging in one file
2. **UX drift** — 47+ commands in a flat `/help` list; session resume is cosmetic (shows last session ID but always creates new); no plain-output mode for piped/CI usage
3. **Configuration drift** — no validation at load time; dual color libraries (`fatih/color` + `gookit/color`); TODO.md is stale (6+ items already implemented)

## What's already solid (do not touch)

- Core architecture: clean package split (config, client, commands, repl, plugins, hooks, state, tui)
- Client library: 40+ typed methods covering all gateway endpoints
- Plugin/hook system: full lifecycle with async/sync firing
- Test coverage: 8 test files, all passing with `-race`
- Visual identity: sunset gradient, vitality prompt, bonsai sigil

## Stale items in TODO.md (already implemented)

These should be removed from TODO.md as they're already done:
- `var version` (main.go:20 — was `const`, now `var`)
- `SessionID` in `ChatRequest` (client.go:303) + passed in `repl.chat()` (repl.go:253)
- Session ID generated at startup (repl.go:57)
- Session ID printed in welcome banner (repl.go:466-469)
- `/session` command with `new` and resume-by-ID (commands.go:122)
- `CreateAgent()`, `AgentChatStream()` (client.go:466, 490)
- `Orchestrate()`, `OrchestrationDAG()` (client.go:554, 573)
- Shell completions wired (main.go:117-176)

---

## Phase 1: Structural Consolidation (parallel tracks)

### Track A — Command File Split
**Priority:** P1
**Files:** `internal/commands/` (modify commands.go, create new files)
**Goal:** Break 2,026-line commands.go into feature-slice files

Split into:
| File | Commands | Approx lines |
|------|----------|------|
| `commands.go` | Registry, Dispatch, add() only | ~100 |
| `cmd_help.go` | helpCmd() + printKV/colorStatus helpers | ~100 |
| `cmd_system.go` | healthCmd, settingsCmd, hooksCmd, traceCmd, initCmd, activityCmd | ~300 |
| `cmd_model.go` | modelCmd, modelList, modelSet, modelDirect | ~200 |
| `cmd_agent.go` | agentCmd + all agent subcommands | ~250 |
| `cmd_session.go` | sessionCmd | ~80 |
| `cmd_garden.go` | gardenCmd + all garden subcommands | ~200 |
| `cmd_memory.go` | trailCmd, snapshotCmd | ~200 |
| `cmd_project.go` | projectCmd, projectsCmd | ~300 |
| `cmd_apps.go` | appsCmd + all app subcommands | ~200 |
| `cmd_workflow.go` | workflowCmd, runCmd, docCmd, pilotCmd, practiceCmd | ~200 |

Rules:
- Package stays `commands` — no new packages
- All files share the `*Registry` receiver
- No behavioral changes — pure file split
- All existing tests must pass unchanged

### Track B — Typed Streaming Event Renderer
**Priority:** P2
**Files:** `internal/repl/renderer.go` (new), modify `internal/repl/repl.go`
**Goal:** Replace thin `extractText()` with a typed renderer layer

Create `renderer.go` with:
```go
type EventType int
const (
    EventText EventType = iota
    EventThinking
    EventToolCall
    EventToolResult
    EventArtifact
    EventWarning
    EventDone
)

type RenderEvent struct {
    Type    EventType
    Content string
    Meta    map[string]string // tool name, artifact ID, etc.
}

func ClassifyChunk(chunk client.SSEChunk) RenderEvent
func (re RenderEvent) Render(plain bool) string
```

- `ClassifyChunk` replaces `extractText`, returning typed events
- `Render(plain)` handles both styled and plain-text output
- `repl.chat()` calls `ClassifyChunk` → `Render` instead of `extractText` directly
- Agent streaming (`/agent chat`, `/agent dispatch`) also uses the same renderer

### Track C — Config Validation
**Priority:** P2
**Files:** `internal/config/config.go`, `internal/config/config_test.go`
**Goal:** Validate config at load time, add effective-config inspection

Add:
- `Validate()` method on `Config` — checks URL format, timeout parseable, disposition in allowed set
- Env var coverage for `DOJO_DISPOSITION` and `DOJO_MODEL` (currently missing)
- `/settings effective` subcommand showing merged config (file + env + flags)

### Track D — TODO.md Cleanup + One-Shot Exit Codes
**Priority:** P3
**Files:** `TODO.md`, `cmd/dojo/main.go`
**Goal:** Remove stale items, fix exit codes

- Remove all implemented items from TODO.md (listed above)
- Add proper exit codes to one-shot mode: 0=success, 1=gateway error, 2=config error
- Add `--plain` flag for non-interactive output formatting

---

## Phase 2: UX Refinement (sequential, after Phase 1)

### Track E — Help Grouping
**Depends on:** Track A (cmd_help.go exists)
**Goal:** Group 47+ commands into categories in `/help` output

Categories:
- **Chat** — (default), /model, /session
- **Agents** — /agent dispatch/chat/ls/info/channels/bind/unbind
- **Memory** — /garden, /trail, /snapshot
- **Workspace** — /home, /projects, /project, /apps
- **Orchestration** — /run, /workflow, /pilot
- **System** — /health, /settings, /hooks, /trace, /init, /activity
- **Practice** — /practice, /doc

Add maturity labels: commands marked `[beta]` or `[experimental]` in help output.

### Track F — Session Resume
**Depends on:** Track A (cmd_session.go exists)
**Goal:** Make session resume real, not cosmetic

Current behavior: shows last session but always generates a fresh one.
Target:
- `dojo` (no flags) → new session (current behavior, keep it)
- `dojo --resume` → actually resume last session ID
- `/session resume` → switch to last session mid-conversation
- Clear indicator in prompt when resumed vs fresh

### Track G — Color Library Unification
**Depends on:** All Phase 1 tracks
**Goal:** Drop `fatih/color`, use `gookit/color` everywhere

- Replace all `color.New(...)`, `color.Red(...)`, `color.NoColor` with gookit equivalents
- Remove `fatih/color` from go.mod
- Add `--no-color` support via `gcolor.Disable()`

---

## Sequencing

```
Phase 1 (parallel):
  Track A ──────────────┐
  Track B ──────────────┤──→ go test ./... gate
  Track C ──────────────┤
  Track D ──────────────┘

Phase 2 (sequential, after gate):
  Track E (help grouping)
  Track F (session resume)
  Track G (color unification)
```

## Dispatch Model

- **Track A:** Sonnet agent (structural, high file count, well-defined rules)
- **Track B:** Sonnet agent (new file + targeted repl.go edits)
- **Track C:** Sonnet agent (config module, self-contained)
- **Track D:** Main thread (small, fast, touches entry point)
- **Tracks E-G:** Main thread or Sonnet (after Phase 1 integration)

## Acceptance Gate

After all Phase 1 tracks complete:
```bash
cd /Users/alfonsomorales/ZenflowProjects/dojo-cli
go build ./...   # must compile
go test ./... -count=1 -race  # all 118 tests pass
go vet ./...     # clean
```
