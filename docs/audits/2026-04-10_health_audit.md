# Dojo CLI — Health Audit

**Date:** 2026-04-10
**Auditor:** Claude Code (automated)
**Baseline commit:** d9718d6 (main)
**Scope:** Full repository — all Go packages, configuration, documentation, test coverage

---

## Executive Summary

The codebase is **buildable, vet-clean, and fully passing** — all 12 tested packages pass with `-race` on. The biggest risk is critically low test coverage in the two most-used packages (`client` at 10%, `commands` at 6.6%): a gateway protocol change or command regression would be invisible to CI. The highest-priority action is adding a GitHub Actions CI workflow (no CI exists) and raising client/commands coverage to a safe floor before the v1 release.

---

## Health Dashboard

| Dimension | Status | Summary |
|---|---|---|
| Critical Issues | **GREEN** | `go build`, `go test`, `go vet` all clean. No broken imports. |
| Security | **YELLOW** | Hook runner executes arbitrary shell via `sh -c`; no plugin URL allowlist or signature check. |
| Testing | **YELLOW** | `client` 10%, `commands` 6.6%, 9 packages at 0%. No CI pipeline. |
| Technical Debt | **YELLOW** | Stale Makefile comment; improvement-plan.md not updated after Phase 1 completion; Phase 3 backlog untracked. |
| Documentation | **YELLOW** | README excellent. No CI badge. Makefile carries a stale `const`/`var` note. improvement-plan.md predates completed work. |

---

## Findings

### Dimension 1: Critical Issues — GREEN

**F1.1 — Build, test, and vet are all clean**
- `go build ./...`: exits 0, no output.
- `go test ./... -count=1 -race`: all 12 tested packages pass.
- `go vet ./...`: exits 0, no output.
- No missing dependencies, broken imports, or mainline failures detected.

No RED or YELLOW findings in this dimension.

---

### Dimension 2: Security — YELLOW

**F2.1 — Hook runner executes arbitrary shell commands from plugin manifests** (YELLOW)
- **File:** `internal/hooks/runner.go:186`
- **Code:** `exec.CommandContext(ctx, "sh", "-c", command)`
- **Impact:** Any `plugin.json` that declares a `type: command` hook can run arbitrary shell code on the user's machine. The `command` string is taken verbatim from the plugin manifest without sanitization or validation.
- **Context:** This is a design-level trade-off: hook plugins are intended to run commands. The risk is social engineering — a user tricked into installing a malicious plugin URL.
- **Missing control:** There is no allowlist of trusted plugin sources, no signature verification, and no sandboxing of hook execution.

**F2.2 — API keys handled correctly** (GREEN)
- API keys are read exclusively from environment variables (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `KIMI_API_KEY`) via `internal/providers/providers.go:85-90`.
- No hardcoded credentials found anywhere in the codebase.
- Bearer token passed via `Authorization` header over HTTP to the gateway — safe as long as gateway is localhost or HTTPS.

**F2.3 — No TLS bypass found** (GREEN)
- No `InsecureSkipVerify` or custom `TLSClientConfig` in any HTTP client construction.

---

### Dimension 3: Sustainability — Testing — YELLOW

**F3.1 — `client` package critically undertested at 10.4%** (YELLOW)
- **File:** `internal/client/client.go` (the gateway API layer)
- **Impact:** The client package contains 40+ typed gateway methods. At 10% coverage, regressions in request serialization, SSE parsing, or error handling would be undetected.
- **Root cause:** Tests use mock HTTP servers for happy paths but miss most error branches and streaming behavior.

**F3.2 — `commands` package critically undertested at 6.6%** (YELLOW)
- **File:** `internal/commands/` (5,181 lines across 21 files)
- **Impact:** All user-facing command behavior — `/agent`, `/project`, `/skill`, `/telemetry`, `/pilot`, etc. — has near-zero test coverage. Command regressions are silent.
- **Note:** TUI-heavy commands are difficult to test; the gap is worst in `cmd_workflow.go` (896 lines, 0% coverage) and `cmd_telemetry.go` (458 lines).

**F3.3 — Nine packages have zero test files** (YELLOW)
- Packages: `cmd/dojo`, `activity`, `art`, `artifacts`, `guide`, `project`, `providers`, `telemetry`, `trace`
- The highest-risk zero-coverage packages are `providers` (direct API key relay to Anthropic/OpenAI) and `telemetry` (data sink).

**F3.4 — No CI pipeline** (YELLOW)
- **Missing:** No `.github/workflows/` directory.
- **Impact:** Test gate is manual-only (`make test`). A PR or push can break tests without any automated signal.

**Well-covered packages (reference):**
- `skills`: 96.4%, `config`: 84.5%, `hooks`: 83.1%, `orchestration`: 83.1%, `spirit`: 77.4%, `state`: 74.4%

---

### Dimension 4: Sustainability — Technical Debt — YELLOW

**F4.1 — Stale Makefile comment about `const` vs `var` version** (YELLOW)
- **File:** `Makefile:7-9`
- **Comment says:** "cmd/dojo/main.go declares `const version = "0.1.0"`. Go's -X ldflags only patches `var` symbols."
- **Reality:** `main.go:20` already declares `var version = "0.1.0"`. The comment is stale — this work was done but the note was not removed.
- **Impact:** Misleads future contributors into thinking ldflags won't work.

**F4.2 — `improvement-plan.md` not updated after Phase 1 completion** (YELLOW)
- **File:** `docs/improvement-plan.md`
- **Dated:** 2026-04-09. Lists Track A (command file split), Track B (typed renderer), Track C (config validation), Track G (color unification) as pending Phase 1 work.
- **Reality:** All four tracks are complete: `commands.go` is 167 lines with 20+ feature-slice files; `repl/renderer.go` exists; `config.Validate()` is implemented and tested; `fatih/color` is absent from `go.mod`.
- **Impact:** The plan looks like open work when it is done. Phase 3 backlog items are buried without clear priority or ownership.

**F4.3 — Phase 3 backlog items are undocumented engineering debt** (YELLOW)
- **File:** `TODO.md:89-95`
- **Items:** plugin auto-install from CoworkPlugins git URL; disposition presets as named `settings.json` profiles; `Auth.UserID` non-guest identity; DAG natural-language construction.
- **Impact:** These are tracked in a flat TODO list with no priority or effort estimate.

**F4.4 — `cmd_workflow.go` is the largest file at 896 lines** (YELLOW, mild)
- **File:** `internal/commands/cmd_workflow.go`
- **Impact:** Houses `/run`, `/workflow`, `/pilot`, `/practice`, `/doc` — distinct concerns. Not blocking, but a future split would help maintainability.

---

### Dimension 5: Sustainability — Documentation — YELLOW

**F5.1 — No CI configuration means no automated badge or health signal** (YELLOW)
- README has no CI status badge. Contributors cannot see the build/test status without cloning locally.

**F5.2 — Stale Makefile comment misleads contributors** (YELLOW)
- See F4.1. The comment creates a false impression of an unresolved issue.

**F5.3 — improvement-plan.md mismatch with current code** (YELLOW)
- See F4.2. A reader of `docs/improvement-plan.md` would think Tracks A-G are unstarted. The doc should either be marked complete or replaced with the current Phase 3 plan.

**F5.4 — No architecture diagram** (YELLOW, low priority)
- The README explains all commands and surfaces well, but has no diagram showing the relationship between the CLI, gateway, CAS, plugin system, and hook runner.

**Well-documented:**
- README: complete, accurate, covers all 50+ commands, ADA system, Spirit, plugin system, quick start, all env vars.
- Package-level documentation is minimal but sufficient given the README depth.

---

## Action Items

| # | Task | Priority | Files | Effort | Acceptance Criteria |
|---|---|---|---|---|---|
| 1 | Add GitHub Actions CI workflow | P0 | `.github/workflows/ci.yml` (new) | 1-2h | Push to any branch triggers `go build ./...` + `go test ./... -race`; PR cannot merge with red CI |
| 2 | Add plugin source warning before `git clone` in installer | P1 | `internal/plugins/installer.go:76` | 1h | `git clone` is preceded by a user-visible warning listing the URL and requiring confirmation (or `--yes` flag) before executing |
| 3 | Raise `client` package coverage to ≥ 40% | P1 | `internal/client/client_test.go` | 4-6h | `go test ./internal/client/... -cover` reports ≥ 40%; at minimum: SSE stream parsing, error response handling, and auth header injection are tested |
| 4 | Raise `commands` package coverage to ≥ 25% | P1 | `internal/commands/commands_test.go` | 4-8h | `go test ./internal/commands/... -cover` reports ≥ 25%; at minimum `/session`, `/model`, `/agent ls`, and `/skill ls` dispatch paths are tested |
| 5 | Remove stale `const`→`var` comment from Makefile | P1 | `Makefile:7-10` | 15min | The four-line comment block is deleted; `VERSION` override via ldflags is verified by running `make build && ./dojo --version` with a tag set |
| 6 | Mark improvement-plan.md Phase 1 tracks as COMPLETE | P1 | `docs/improvement-plan.md` | 30min | Each completed track (A, B, C, G) is marked `[DONE]`; Phase 3 backlog is the only open section |
| 7 | Add tests for `providers` package | P2 | `internal/providers/providers_test.go` (new) | 2-3h | `LoadAPIKeys()`, `HasDirectAccess()`, and `KeyForProvider()` are tested; `go test ./internal/providers/... -cover` ≥ 60% |
| 8 | Add tests for `telemetry` sink | P2 | `internal/telemetry/sink_test.go` (new) | 2h | Sink write, batch flush, and error handling are tested; coverage ≥ 50% |
| 9 | Split `cmd_workflow.go` into feature slices | P2 | `internal/commands/cmd_workflow.go` | 2-3h | `/run`, `/pilot`, `/workflow`, `/practice`, `/doc` each in their own file; all existing tests pass |
| 10 | Convert Phase 3 TODO items to structured action items | P2 | `TODO.md` | 30min | Each Phase 3 item has a priority, effort estimate, and acceptance criterion; resolved items removed |
| 11 | Add architecture diagram to docs | P3 | `docs/architecture.md` (new) | 2-3h | Diagram shows CLI → Gateway → CAS / Plugin → Hook Runner data flows; linked from README |

---

## Trend

No prior audits found in `docs/audits/`. This is the baseline audit.

---

*Generated by `/system-health:health-audit` on 2026-04-10.*
