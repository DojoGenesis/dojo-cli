# Changelog

All notable changes to the Dojo CLI are documented here.

This project adheres to [Keep a Changelog v1.1.0](https://keepachangelog.com/en/1.1.0/) and [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- `/craft` command group — DojoCraft practitioner workbench with 15 offline + online test cases (`1036875`, `ad4ece2`)
- Desktop scaffold (`cli/desktop/`) hibernated pending CLI v1.0.0 ship; Wails v2 + Svelte 5 build is clean (`8f456a2`, `4b5eb7c`)

### Changed

- goreleaser brew token: corrected cross-repo homebrew-tap push (`6995170`)
- Embed anchor committed for `desktop/frontend/dist` to satisfy CI (`af08841`)

### Fixed

- Atomic writes for all config/state JSON files — eliminates partial-write corruption (`e58945f`)
- Redact secret positional args in activity log (`a965ac4`)
- Enforce project-root boundary in `/code` read — prevents path traversal (`abeb071`)
- smoke-craft.sh: use REPL stdin pipe instead of `--one-shot` flag (`956b970`)

### Docs

- README: fill gaps in coverage (`190e24e`)

---

## [1.0.0] — 2026-04-12

### Added

- NL-to-DAG construction: `/run <task> --dag` command (TODO 3.4) (`10a1648`)
- goreleaser release workflow: auto-build on tag push (`723e958`)

### Fixed

- Resolve `WorkspaceRoot` to absolute path instead of hardcoded `"."` (`5c491e0`)
- Shell completions: add 13 missing commands to zsh/bash/fish + REPL (`723e958`)

### Docs

- Mark `TODO.md` fully resolved — all phases complete, CLI shippable (`6114dd5`)

---

## [0.2.0] — 2026-04-11

### Added

- Skills semantic clustering package + telemetry sink/providers tests (`27082ae`)
- Persist disposition profiles to `settings.json`; add `/settings profile` subcommand (`7e97309`)
- `/bloom`, `/code`, `/guide` commands; art animations; War Room updates (`f37f8ef`)
- Specialist awareness + DAG panel + garden context (pilot) (`f37f8ef`)
- Multi-panel layout + telemetry sink (pilot) (`14ec816`)
- Typed event parser + real-time cost tracking (pilot) (`5081c97`)
- War Room TUI mode — split-panel Scout vs Challenger debate (`1a27480`)
- Dojo Spirit — belt ranks, achievements, streaks, koans, profile card (`216ab92`)
- `/telemetry` CLI commands: sessions, costs, tools, summary (`adfd19c`)
- Comprehensive CLI improvements Phases 1–3 (`3270d3d`)
- `/init` command — populate empty workspace with ecosystem assets (`ba393c0`)
- Desktop scaffold and CI workflow (`8f456a2`)

### Changed

- Desktop: hibernate `cli/desktop/` — feature dev gated behind CLI v1.0.0 ship (`4b5eb7c`)
- Stop tracking binaries; stage providers + test runner updates (`2a473d3`)

### Fixed

- Agent dispatch guard, `/run` command routing, persistent SSE connection (`d9718d6`)
- Send `workspace_root` with every chat request for file tool resolution (`009208a`)
- Nightly CI: post-sweep cleanup (`f7f521c`)
- Correct telemetry API default URL (`89dd205`)

### Docs

- Add `llms.txt` for LLM/Context7 discoverability (`d9718d6`)

### Chore

- Add Apache 2.0 LICENSE file (`e3b7044`)
- Commit hygiene: stop tracking binaries (`2a473d3`)
- CI workflow, desktop scaffold, build scripts, gitignore (`8f456a2`)

---

## [0.1.0] — 2026-04-09

### Added

- Phase 7: MCP apps, agent channels, workflows, CAS, documents — initial CLI feature set (`feat: Phase 7`)
