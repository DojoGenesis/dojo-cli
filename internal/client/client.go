// Package client provides a typed HTTP+SSE client for the Dojo Genesis AgenticGateway.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to an AgenticGateway instance.
type Client struct {
	base   string
	token  string
	http   *http.Client
}

// New creates a Client. timeout is a Go duration string (e.g. "60s").
func New(baseURL, token, timeout string) *Client {
	d, err := time.ParseDuration(timeout)
	if err != nil {
		d = 60 * time.Second
	}
	return &Client{
		base:  strings.TrimRight(baseURL, "/"),
		token: token,
		http:  &http.Client{Timeout: d},
	}
}

// ─── Health ──────────────────────────────────────────────────────────────────

// HealthResponse is the /health response.
// Server: server/handle_health.go → HealthResponse
type HealthResponse struct {
	Status            string            `json:"status"`
	Version           string            `json:"version"`
	Timestamp         string            `json:"timestamp"`
	Providers         map[string]string `json:"providers"`
	Dependencies      map[string]string `json:"dependencies"`
	UptimeSeconds     int64             `json:"uptime_seconds"`
	RequestsProcessed int64             `json:"requests_processed"`
}

// Health fetches GET /health.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var h HealthResponse
	if err := c.get(ctx, "/health", &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// ─── Models / Providers ──────────────────────────────────────────────────────

// ProviderInfo carries optional metadata about a provider's capabilities.
// Mirrors provider.ProviderInfo fields that are relevant to the client.
type ProviderInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Provider is a single entry in the /v1/providers response.
// Server: server/handlers/models.go → ProviderStatus
type Provider struct {
	Name   string        `json:"name"`
	Status string        `json:"status"`
	Info   *ProviderInfo `json:"info,omitempty"`
	Error  string        `json:"error,omitempty"`
}

// providersEnvelope is the top-level wrapper for GET /v1/providers.
// Server returns: {"providers": [...], "count": N}
type providersEnvelope struct {
	Providers []Provider `json:"providers"`
	Count     int        `json:"count"`
}

// Providers fetches GET /v1/providers.
func (c *Client) Providers(ctx context.Context) ([]Provider, error) {
	var r providersEnvelope
	if err := c.get(ctx, "/v1/providers", &r); err != nil {
		return nil, err
	}
	return r.Providers, nil
}

// Model is a single entry in the /v1/models response.
// Server: server/handlers/models.go → provider.ModelInfo
// Server returns: {"models": [...], "count": N}
type Model struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
}

// modelsEnvelope is the top-level wrapper for GET /v1/models.
type modelsEnvelope struct {
	Models []Model `json:"models"`
	Count  int     `json:"count"`
}

// Models fetches GET /v1/models.
func (c *Client) Models(ctx context.Context) ([]Model, error) {
	var r modelsEnvelope
	if err := c.get(ctx, "/v1/models", &r); err != nil {
		return nil, err
	}
	return r.Models, nil
}

// ─── Tools ───────────────────────────────────────────────────────────────────

// Tool is a single entry in the tools list.
// Server: server/handle_gateway.go → handleGatewayListTools
// Returns: {"tools": [{name, description, parameters, namespace}], "count": N}
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Namespace   string         `json:"namespace"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// toolsEnvelope is the top-level wrapper for GET /v1/gateway/tools.
// Note: gateway uses "count", not "total".
type toolsEnvelope struct {
	Tools []Tool `json:"tools"`
	Count int    `json:"count"`
}

// ToolsResponse exposes the parsed tools list to callers.
type ToolsResponse struct {
	Tools []Tool
	Count int
}

// Tools fetches GET /v1/gateway/tools.
func (c *Client) Tools(ctx context.Context) ([]Tool, error) {
	var r toolsEnvelope
	if err := c.get(ctx, "/v1/gateway/tools", &r); err != nil {
		// fallback to /v1/tools (handlers/tools.go HandleListTools)
		// That endpoint returns: {"success":true,"count":N,"tools":[{name,description,parameters}]}
		var fallback struct {
			Tools []Tool `json:"tools"`
			Count int    `json:"count"`
		}
		if err2 := c.get(ctx, "/v1/tools", &fallback); err2 != nil {
			return nil, err
		}
		return fallback.Tools, nil
	}
	return r.Tools, nil
}

// ─── Agents ──────────────────────────────────────────────────────────────────

// AgentDisposition carries the disposition fields embedded in agent list entries.
type AgentDisposition struct {
	Pacing     string `json:"pacing"`
	Depth      string `json:"depth"`
	Tone       string `json:"tone"`
	Initiative string `json:"initiative"`
}

// Agent is a single entry in the /v1/gateway/agents response.
// Server: server/handle_gateway.go → handleGatewayListAgents
// Returns entries with: {agent_id, status, config?, disposition?, channels?}
type Agent struct {
	AgentID     string            `json:"agent_id"`
	Status      string            `json:"status"`
	Disposition *AgentDisposition `json:"disposition,omitempty"`
	Channels    []string          `json:"channels,omitempty"`
}

// agentsEnvelope is the top-level wrapper for GET /v1/gateway/agents.
// Server returns: {"agents": [...], "total": N, "limit": N, "offset": N}
type agentsEnvelope struct {
	Agents []Agent `json:"agents"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// Agents fetches GET /v1/gateway/agents.
func (c *Client) Agents(ctx context.Context) ([]Agent, error) {
	var r agentsEnvelope
	if err := c.get(ctx, "/v1/gateway/agents", &r); err != nil {
		return nil, err
	}
	return r.Agents, nil
}

// ─── Skills ──────────────────────────────────────────────────────────────────

// PortDefinition mirrors skill.PortDefinition from the server.
type PortDefinition struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

// Skill is a single entry in the /api/skills response.
// Server: workflow/api/handler.go → skillEntry
// Returns: {id, name, description, version, category, plugin, inputs, outputs, cas_hash}
type Skill struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Version     string           `json:"version,omitempty"`
	Category    string           `json:"category,omitempty"`
	Plugin      string           `json:"plugin,omitempty"`
	Inputs      []PortDefinition `json:"inputs"`
	Outputs     []PortDefinition `json:"outputs"`
	CASHash     string           `json:"cas_hash,omitempty"`
}

// skillsEnvelope is the top-level wrapper for GET /api/skills.
// Server returns: {"skills": [...], "total": N, "limit": N, "offset": N}
type skillsEnvelope struct {
	Skills []Skill `json:"skills"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// Skills fetches GET /api/skills.
func (c *Client) Skills(ctx context.Context) ([]Skill, error) {
	var r skillsEnvelope
	if err := c.get(ctx, "/api/skills", &r); err != nil {
		return nil, err
	}
	return r.Skills, nil
}

// ─── Seeds / Garden ──────────────────────────────────────────────────────────

// Seed is a single entry in the /v1/seeds response.
// Server: server/handlers/memory.go → memory.Seed
// Returns: {id, name, description, trigger, content, usage_count, last_used?, created_at, updated_at}
type Seed struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Trigger     string     `json:"trigger"`
	Content     string     `json:"content"`
	UsageCount  int        `json:"usage_count"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// seedsEnvelope is the top-level wrapper for GET /v1/seeds.
// Server returns: {"success": true, "count": N, "seeds": [...]}
type seedsEnvelope struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Seeds   []Seed `json:"seeds"`
}

// Seeds fetches GET /v1/seeds.
func (c *Client) Seeds(ctx context.Context) ([]Seed, error) {
	var r seedsEnvelope
	if err := c.get(ctx, "/v1/seeds", &r); err != nil {
		return nil, err
	}
	return r.Seeds, nil
}

// GardenStats fetches GET /v1/garden/stats.
func (c *Client) GardenStats(ctx context.Context) (map[string]any, error) {
	var r map[string]any
	if err := c.get(ctx, "/v1/garden/stats", &r); err != nil {
		return nil, err
	}
	return r, nil
}

// ─── Chat (streaming SSE) ────────────────────────────────────────────────────

// ChatRequest mirrors the /v1/chat request body.
// Server: server/handlers/chat.go → ChatRequest
type ChatRequest struct {
	Message   string `json:"message"`
	Model     string `json:"model,omitempty"`
	Stream    bool   `json:"stream"`
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

// SSEChunk is a parsed line from the SSE stream.
type SSEChunk struct {
	Event string
	Data  string
}

// ChatStream opens a streaming POST /v1/chat and calls onChunk for each SSE data line.
// Returns when the stream ends or ctx is cancelled.
func (c *Client) ChatStream(ctx context.Context, req ChatRequest, onChunk func(SSEChunk)) error {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/v1/chat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gateway returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return parseSSE(resp.Body, onChunk)
}

// parseSSE reads an SSE stream and calls onChunk for each data line.
func parseSSE(r io.Reader, onChunk func(SSEChunk)) error {
	scanner := bufio.NewScanner(r)
	var event string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				return nil
			}
			onChunk(SSEChunk{Event: event, Data: data})
			event = ""
		case line == "":
			event = ""
		}
	}
	return scanner.Err()
}

// ─── Seeds — create ──────────────────────────────────────────────────────────

// CreateSeedRequest is the body for POST /v1/seeds.
// Server: server/handlers/memory.go → CreateSeedRequest
type CreateSeedRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Trigger     string `json:"trigger,omitempty"`
	Content     string `json:"content"`
}

// CreateSeed posts a new seed to /v1/seeds.
func (c *Client) CreateSeed(ctx context.Context, req CreateSeedRequest) (*Seed, error) {
	var wrapper struct {
		Seed Seed `json:"seed"`
	}
	if err := c.post(ctx, "/v1/seeds", req, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Seed, nil
}

// ─── Memory ──────────────────────────────────────────────────────────────────

// Memory represents an entry from GET /v1/memory.
type Memory struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	Type      string `json:"type"`
}

type memoriesResponse struct {
	Memories []Memory `json:"memories"`
	Total    int      `json:"total"`
}

// Memories fetches GET /v1/memory.
func (c *Client) Memories(ctx context.Context) ([]Memory, error) {
	var r memoriesResponse
	if err := c.get(ctx, "/v1/memory", &r); err != nil {
		return nil, err
	}
	return r.Memories, nil
}

// ─── Pilot — live SSE stream ─────────────────────────────────────────────────

// PilotStream opens GET /events and calls onChunk for each SSE event.
// It runs until ctx is cancelled or the stream closes.
func (c *Client) PilotStream(ctx context.Context, clientID string, onChunk func(SSEChunk)) error {
	url := c.base + "/events?client_id=" + clientID

	// Use a client with no timeout for the long-lived SSE connection.
	httpClient := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gateway returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return parseSSE(resp.Body, onChunk)
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gateway %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(b)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		rb, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gateway %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(rb)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
