package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DojoGenesis/cli/internal/client"
	"github.com/DojoGenesis/cli/internal/config"
	"github.com/DojoGenesis/cli/internal/state"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails application struct.
// Its exported methods are automatically bound to the frontend via wailsjs.
type App struct {
	ctx         context.Context
	gw          *client.Client
	cfg         *config.Config
	st          *state.State
	pilotCancel context.CancelFunc
	sessionID   string
}

// AppConfig is returned by GetConfig and consumed directly by the frontend.
type AppConfig struct {
	GatewayURL string `json:"gateway_url"`
	Version    string `json:"version"`
	SessionID  string `json:"session_id"`
}

// NewApp creates a bare App. Wails calls OnStartup before any bound method is invoked.
func NewApp() *App {
	return &App{}
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────

// OnStartup is called by Wails after the application window is ready.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.sessionID = fmt.Sprintf("dojo-desktop-%d", time.Now().UnixMilli())

	cfg, err := config.Load()
	if err != nil {
		runtime.LogWarningf(ctx, "Config load error: %v", err)
		cfg = &config.Config{}
		cfg.Gateway.URL = config.DefaultGatewayURL
		cfg.Gateway.Timeout = "60s"
	}
	a.cfg = cfg

	a.gw = client.New(cfg.Gateway.URL, cfg.Gateway.Token, cfg.Gateway.Timeout)

	st, err := state.Load()
	if err != nil {
		runtime.LogWarningf(ctx, "State load error: %v", err)
		st = &state.State{}
	}
	a.st = st
}

// OnShutdown is called by Wails when the application is closing.
func (a *App) OnShutdown(ctx context.Context) {
	if a.pilotCancel != nil {
		a.pilotCancel()
	}
}

// ─── Configuration ────────────────────────────────────────────────────────────

// GetConfig returns the current application and gateway configuration.
func (a *App) GetConfig() AppConfig {
	url := config.DefaultGatewayURL
	if a.cfg != nil && a.cfg.Gateway.URL != "" {
		url = a.cfg.Gateway.URL
	}
	return AppConfig{
		GatewayURL: url,
		Version:    "1.0.0",
		SessionID:  a.sessionID,
	}
}

// ─── Health ───────────────────────────────────────────────────────────────────

// CheckHealth pings the gateway and returns true if reachable.
func (a *App) CheckHealth() bool {
	if a.gw == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := a.gw.Health(ctx)
	return err == nil
}

// ─── Providers / Models ───────────────────────────────────────────────────────

// GetProviders returns available AI providers from the gateway.
func (a *App) GetProviders() ([]client.Provider, error) {
	if a.gw == nil {
		return nil, fmt.Errorf("gateway client not initialised")
	}
	return a.gw.Providers(context.Background())
}

// GetModels returns available models from the gateway.
func (a *App) GetModels() ([]client.Model, error) {
	if a.gw == nil {
		return nil, fmt.Errorf("gateway client not initialised")
	}
	return a.gw.Models(context.Background())
}

// ─── Skills ───────────────────────────────────────────────────────────────────

// GetSkills searches skills by query. An empty query returns all skills.
func (a *App) GetSkills(query string) ([]client.Skill, error) {
	if a.gw == nil {
		return nil, fmt.Errorf("gateway client not initialised")
	}
	ctx := context.Background()
	if strings.TrimSpace(query) != "" {
		return a.gw.SearchSkills(ctx, query)
	}
	return a.gw.Skills(ctx)
}

// ─── Agents ───────────────────────────────────────────────────────────────────

// GetAgents returns all agents from the gateway.
func (a *App) GetAgents() ([]client.Agent, error) {
	if a.gw == nil {
		return nil, fmt.Errorf("gateway client not initialised")
	}
	return a.gw.Agents(context.Background())
}

// ─── Chat streaming ───────────────────────────────────────────────────────────

// SendMessage sends a chat message and streams the response via Wails events.
// Emits "chat:chunk" events: {content: string, done: bool, error: string}
// Returns immediately; streaming happens in a background goroutine.
func (a *App) SendMessage(sessionID, message, provider, model string) error {
	if a.gw == nil {
		return fmt.Errorf("gateway client not initialised")
	}
	if sessionID == "" {
		sessionID = a.sessionID
	}

	req := client.ChatRequest{
		Message:   message,
		SessionID: sessionID,
		Provider:  provider,
		Model:     model,
		Stream:    true,
	}

	go func() {
		err := a.gw.ChatStream(context.Background(), req, func(chunk client.SSEChunk) {
			event := strings.TrimSpace(chunk.Event)

			// "complete" signals end of stream — emit done and stop.
			if event == "complete" {
				runtime.EventsEmit(a.ctx, "chat:chunk", map[string]interface{}{
					"content": "",
					"done":    true,
					"error":   "",
				})
				return
			}

			// Only forward actual text chunks; skip metadata events
			// (intent_classified, provider_selected, tool_invoked, etc.)
			if event != "chunk" && event != "response_chunk" {
				return
			}

			// Parse the JSON envelope to extract the text content field.
			var envelope struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(chunk.Data), &envelope); err != nil || envelope.Content == "" {
				return
			}

			runtime.EventsEmit(a.ctx, "chat:chunk", map[string]interface{}{
				"content": envelope.Content,
				"done":    false,
				"error":   "",
			})
		})

		if err != nil {
			// Skip context-cancelled errors (user navigated away, etc.)
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return
			}
			runtime.EventsEmit(a.ctx, "chat:chunk", map[string]interface{}{
				"content": "",
				"done":    true,
				"error":   err.Error(),
			})
			return
		}

		// Emit terminal done event.
		runtime.EventsEmit(a.ctx, "chat:chunk", map[string]interface{}{
			"content": "",
			"done":    true,
			"error":   "",
		})
	}()

	return nil
}

// ─── Agent message streaming ──────────────────────────────────────────────────

// SendAgentMessage sends a message to an existing agent and streams via Wails events.
// Emits "agent:chunk" events: {agent_id: string, content: string, done: bool, error: string}
// Returns immediately; streaming happens in a background goroutine.
func (a *App) SendAgentMessage(agentID, message string) error {
	if a.gw == nil {
		return fmt.Errorf("gateway client not initialised")
	}
	if agentID == "" {
		return fmt.Errorf("agentID must not be empty")
	}

	req := client.AgentChatRequest{
		Message: message,
		Stream:  true,
	}

	go func() {
		err := a.gw.AgentChatStream(context.Background(), agentID, req, func(chunk client.SSEChunk) {
			isDone := strings.TrimSpace(chunk.Event) == "done" ||
				strings.TrimSpace(chunk.Data) == "[DONE]"

			runtime.EventsEmit(a.ctx, "agent:chunk", map[string]interface{}{
				"agent_id": agentID,
				"content":  chunk.Data,
				"done":     isDone,
				"error":    "",
			})
		})

		if err != nil {
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				return
			}
			runtime.EventsEmit(a.ctx, "agent:chunk", map[string]interface{}{
				"agent_id": agentID,
				"content":  "",
				"done":     true,
				"error":    err.Error(),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "agent:chunk", map[string]interface{}{
			"agent_id": agentID,
			"content":  "",
			"done":     true,
			"error":    "",
		})
	}()

	return nil
}

// ─── Pilot stream ─────────────────────────────────────────────────────────────

// StartPilot begins streaming SSE events from the gateway /events endpoint.
// Emits "pilot:event" events: {event_type: string, data: string}
// Cancels any existing pilot stream before starting a new one.
func (a *App) StartPilot() error {
	if a.gw == nil {
		return fmt.Errorf("gateway client not initialised")
	}

	// Cancel any in-flight pilot stream.
	if a.pilotCancel != nil {
		a.pilotCancel()
	}

	pilotCtx, cancel := context.WithCancel(context.Background())
	a.pilotCancel = cancel

	go func() {
		err := a.gw.PilotStream(pilotCtx, a.sessionID, func(chunk client.SSEChunk) {
			runtime.EventsEmit(a.ctx, "pilot:event", map[string]interface{}{
				"event_type": chunk.Event,
				"data":       chunk.Data,
			})
		})

		if err != nil && err != context.Canceled &&
			!strings.Contains(err.Error(), "context canceled") {
			runtime.LogErrorf(a.ctx, "PilotStream error: %v", err)
		}
	}()

	return nil
}

// StopPilot cancels the active pilot stream (no-op if none running).
func (a *App) StopPilot() {
	if a.pilotCancel != nil {
		a.pilotCancel()
		a.pilotCancel = nil
	}
}

// ─── Session history ──────────────────────────────────────────────────────────

// GetRecentSessions returns recent session IDs from local state.
// Currently returns the last known session ID; extend as state tracking grows.
func (a *App) GetRecentSessions() []string {
	if a.st == nil {
		return []string{}
	}
	if a.st.LastSessionID != "" {
		return []string{a.st.LastSessionID}
	}
	return []string{}
}
