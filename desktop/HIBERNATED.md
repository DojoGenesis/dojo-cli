# cli/desktop/ — HIBERNATED

**Status:** Hibernated  
**Date:** 2026-04-11  
**Decision:** Strategic scout concluded feature development is premature.

## What Exists

- Wails v2 + Svelte 5 scaffold (commit `cc81104`)
- Go backend bridge: 10 exported methods wrapping `internal/client.Client`
- Frontend: single ChatView, StatusBar, rAF-batched chat store
- 5 of 10 Go methods have no frontend consumer (skills, agents, pilot, config, sessions)
- No tests. Not smoke-tested against Gateway.

## Why Hibernated

1. CLI is near-shippable (one TODO item remaining). Desktop splits focus.
2. Six web frontend specs are written and commission-ready — desktop duplicates their scope.
3. No user demand signal for a desktop app. CLI TUI covers the same surface.
4. The Go bridge is architecturally sound but the frontend is too thin to ship.

## Revival Conditions

All three must be true:

1. CLI has shipped (`v1.0.0` tagged, TODO.md fully resolved)
2. Gateway API surface is stable (no breaking `/api/chat/stream` changes)
3. A user demand signal emerges that the TUI cannot satisfy

## Optional: Chat Wedge Gate

After CLI ships, one bounded session (4-8h max) may smoke-test the existing chat view against a live Gateway. If it works cleanly, it becomes a demo artifact. If bitrot is found, archive the directory.

## Architecture Notes

- No `go.mod` — shares parent module (`github.com/DojoGenesis/cli`)
- Wails v2 dep (`v2.12.0`) remains in cli/go.mod
- Imports: `internal/client`, `internal/config`, `internal/state`
- HTMLCraftStudio is the reference Wails v2 app in this stack (140 tests, backend frozen)
