# Dojo CLI — Integration Todo

## Status: Phase 1 complete, Phase 2 in progress (agents running)

---

## CRITICAL BUG — Fix before next test

### Session ID required by /v1/chat

`POST /v1/chat` returns **400** when `session_id` is missing — it is not optional.

**File:** `internal/repl/repl.go` → `chat()` function
**File:** `internal/client/client.go` → `ChatRequest` struct

**Fix:**
1. Add `SessionID string` to `client.ChatRequest` (already field exists in server's `ChatRequest`)
2. In `repl.New()`, generate a session ID once: `r.session = fmt.Sprintf("dojo-%d", time.Now().UnixNano())`
3. Pass `SessionID: r.session` in every `client.ChatRequest` inside `repl.chat()`

The session ID is what ties successive chat turns together in the gateway's memory system. Same session → accumulated context. New REPL invocation → new session (correct behavior).

---

## Integration Spec 1 — Session Continuity

**Goal:** Make the REPL a stateful participant in the gateway's memory system, not a sequence of disconnected single-turn calls.

### What the gateway provides

- `POST /v1/chat` requires `session_id string` — the same value across turns gives the memory manager a single conversation to compress and recall
- The gateway stores turn history in SQLite under that session ID
- The session is never explicitly "closed" — it stays open until TTL or explicit deletion

### What the REPL needs to do

1. **Generate session ID at startup** in `repl.New()`:
   ```go
   r.session = fmt.Sprintf("dojo-cli-%s", time.Now().Format("20060102-150405"))
   ```
   Print it in the welcome banner so the user can reference it: `session: dojo-cli-20260409-142301`

2. **Persist session across turns** — pass `session_id: r.session` in every `ChatStream` call. The gateway uses this to look up prior turns in memory.

3. **Add `/session` command** to `commands.go`:
   ```
   /session           show current session ID
   /session new       start a new session (generates a fresh ID, breaks continuity)
   /session <id>      resume a named session (connects to prior gateway memory)
   ```
   Requires the registry to hold a reference to the REPL's session — pass a `*string` pointer or add a setter.

4. **Add `UserID` support** — `POST /v1/chat` also accepts `user_id`. Read from `cfg.Auth.UserID` (add to config). Defaults to empty string (guest user, which gateway allows).

### Files to change
- `internal/repl/repl.go` — session field, new() init, chat() call, welcome banner
- `internal/client/client.go` — `ChatRequest.SessionID string` field
- `internal/commands/commands.go` — add `/session` command, pass session pointer to registry
- `internal/config/config.go` — add `Auth.UserID` field

### Acceptance criteria
- `dojo chat` does not return a 400 error
- Asking a follow-up question ("what did I just say?") returns a coherent response that references the prior turn
- `/session new` resets continuity; gateway treats next turn as fresh context
- Session ID printed at startup

---

## Integration Spec 2 — Agent Dispatch

**Goal:** Let the user dispatch a named agent from the REPL, pass it a task, and stream the response — going through the gateway's full ADA disposition + tool loop.

### What the gateway provides

**Create an agent:**
```
POST /v1/gateway/agents
Body: { "workspace_root": "/path", "active_mode": "balanced" }
Returns: { "agent_id": "uuid", "status": "created", "disposition": {...} }
```
The agent is initialized with the ADA disposition from the workspace root's `.ada/disposition.yaml` (or default if none). `active_mode` matches the disposition preset names: `"focused"`, `"balanced"`, `"exploratory"`, `"deliberate"`.

**Chat with an agent:**
```
POST /v1/gateway/agents/:id/chat
Body: { "message": "...", "stream": true, "user_id": "" }
Returns: SSE stream — same event format as /v1/chat
```
The agent has tool access (full tool registry), DAG planning capability, and the disposition shaping its pacing/depth/tone.

**List existing agents:**
```
GET /v1/gateway/agents
Returns: { "agents": [{agent_id, status, disposition, channels}], "total": N }
```
Agents persist in-memory for the gateway's lifetime. They are re-used if their ID is known.

### New client methods needed

```go
// CreateAgent creates a new agent and returns its ID.
func (c *Client) CreateAgent(ctx context.Context, workspaceRoot, activeMode string) (string, error)

// AgentChatStream streams a conversation with a specific agent.
// Same SSEChunk callback as ChatStream.
func (c *Client) AgentChatStream(ctx context.Context, agentID, message string, onChunk func(SSEChunk)) error
```

### New command: `/agent dispatch`

Extend `agentCmd()` in `commands.go`:

```
/agent ls                     list agents (existing)
/agent dispatch <mode> <msg>  create agent with mode, send message, stream response
/agent chat <id> <msg>        chat with existing agent by ID
```

`mode` values: `focused`, `balanced`, `exploratory`, `deliberate` (maps to `active_mode`).

Workspace root defaults to `os.Getwd()` — the agent initializes from whatever `.ada/disposition.yaml` is in scope.

**UX flow:**
```
dojo › /agent dispatch exploratory what tensions exist in this codebase?

  Creating agent (mode: exploratory)...
  Agent: a3f9bc2e  disposition: pacing=deliberate depth=exhaustive

  dojo  [Thinking... analyzing codebase structure]
  [Tool: file_ops → listing root]
  [Tool: web_search → recent Go patterns]

  Here are three tensions I found...
```

Display thinking/tool events differently from text chunks (dim color, prefix with `[Tool: ...]`).

### Files to change
- `internal/client/client.go` — `CreateAgent()`, `AgentChatStream()`, `AgentChatRequest` type
- `internal/commands/commands.go` — extend `agentCmd()` with `dispatch` and `chat` subcommands
- `internal/repl/repl.go` — `extractText()` needs to handle agent SSE event format (thinking, tool_call, text_chunk events — different from plain chat delta)

### Acceptance criteria
- `/agent dispatch balanced hello` creates an agent and streams a response
- Tool calls are visible in the stream as `[Tool: tool_name]` lines
- `/agent ls` shows the newly created agent with its ID
- `/agent chat <id> follow-up question` continues with the same agent

---

## Integration Spec 3 — Orchestration Dispatch

**Goal:** Let the user submit a multi-step task as a DAG execution plan and watch it execute node by node.

### What the gateway provides

**Submit a plan:**
```
POST /v1/gateway/orchestrate
Body: {
  "plan": {
    "id": "uuid",
    "name": "Research and summarize X",
    "dag": [
      { "id": "step1", "tool_name": "web_search", "input": {"query": "X"}, "depends_on": [] },
      { "id": "step2", "tool_name": "summarize", "input": {"text": "{{step1.output}}"}, "depends_on": ["step1"] }
    ]
  },
  "user_id": ""
}
Returns: { "execution_id": "uuid", "plan_id": "uuid", "status": "submitted" }
```
Execution is async — runs in a goroutine server-side.

**Poll DAG status:**
```
GET /v1/gateway/orchestrate/:id/dag
Returns: { "execution_id": "...", "status": "running|completed|failed", "plan": {...}, "nodes": [...] }
```

### New command: `/run`

```
/run <natural language task>
```

Flow:
1. The CLI sends the task description to `/v1/chat` with `session_id` and gets back a structured DAG plan (the gateway's intent classifier already routes complex multi-step tasks to its orchestration path)
2. OR: the CLI constructs a simple DAG directly for common patterns (web_search + summarize is the MVP)
3. Polls `/v1/gateway/orchestrate/:id/dag` every 1s and prints node status changes in real time

**MVP approach** (avoid complexity of LLM-generated DAGs): just ask the gateway's chat endpoint to orchestrate and let it decide. The `/run` command is a wrapper around chat that sets the expectation for a multi-step result.

**Full approach** (Phase 3): parse natural language into a DAG client-side using a simple template library, then submit via `/v1/gateway/orchestrate`.

### New client methods needed

```go
type OrchestrateRequest struct {
    Plan   ExecutionPlan `json:"plan"`
    UserID string        `json:"user_id,omitempty"`
}

type ExecutionPlan struct {
    ID   string            `json:"id"`
    Name string            `json:"name"`
    DAG  []ToolInvocation  `json:"dag"`
}

type ToolInvocation struct {
    ID        string         `json:"id"`
    ToolName  string         `json:"tool_name"`
    Input     map[string]any `json:"input"`
    DependsOn []string       `json:"depends_on,omitempty"`
}

type OrchestrationStatus struct {
    ExecutionID string `json:"execution_id"`
    Status      string `json:"status"` // submitted, running, completed, failed
    PlanID      string `json:"plan_id"`
}

func (c *Client) Orchestrate(ctx context.Context, req OrchestrateRequest) (*OrchestrationStatus, error)
func (c *Client) OrchestrationDAG(ctx context.Context, executionID string) (map[string]any, error)
```

### Files to change
- `internal/client/client.go` — `Orchestrate()`, `OrchestrationDAG()`, plan types
- `internal/commands/commands.go` — add `runCmd()`
- `internal/repl/repl.go` — register `/run` in readline autocomplete

### Acceptance criteria
- `/run research the MCP protocol and summarize` submits a plan and prints live node status
- Status updates print as nodes complete: `[step1 ✓] [step2 running...]`
- Final output (if available in DAG response) is printed after completion
- Error nodes show the error message

---

## Remaining Phase 2 work (agents running)

These are being built by background agents right now:

- `internal/plugins/scanner.go` — CoworkPlugins format scanner
- `internal/hooks/runner.go` — PreCommand/PostCommand hook runner
- `/hooks ls` command
- `/garden plant <text>` (POST /v1/seeds)
- `/trail` (GET /v1/memory timeline)
- `/pilot` (GET /events live SSE tail)
- `/trace` (trace guidance)

---

## Phase 3 (future)

- `/run` with full DAG construction from natural language
- `dojo --one-shot "task"` flag for non-interactive use
- Shell completions: `dojo completion zsh`
- `~/.dojo/plugins/` auto-install from CoworkPlugins git URL
- Agent persistence across REPL sessions (store agent ID in `~/.dojo/state.json`)
- Disposition presets as named profiles in `settings.json`
