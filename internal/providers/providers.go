package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// KnownProvider represents a provider we know about statically.
type KnownProvider struct {
	ID          string       // "anthropic", "openai", "local"
	DisplayName string       // "Anthropic", "OpenAI", "Local"
	EnvKey      string       // env var name for the API key, e.g. "ANTHROPIC_API_KEY"
	Models      []KnownModel
}

// KnownModel is a statically-known model within a provider.
type KnownModel struct {
	ID          string // "claude-opus-4-6", "gpt-4o", etc.
	DisplayName string // "Claude Opus 4.6", "GPT-4o", etc.
	Notes       string // "fast", "most capable", etc.
}

// Catalog is the static list of known providers and their models.
var Catalog []KnownProvider

func init() {
	Catalog = []KnownProvider{
		{
			ID:          "anthropic",
			DisplayName: "Anthropic",
			EnvKey:      "ANTHROPIC_API_KEY",
			Models: []KnownModel{
				{ID: "claude-opus-4-6", DisplayName: "Claude Opus 4.6", Notes: "most capable"},
				{ID: "claude-sonnet-4-6", DisplayName: "Claude Sonnet 4.6", Notes: "fast + capable"},
				{ID: "claude-haiku-4-5", DisplayName: "Claude Haiku 4.5", Notes: "fastest, lightweight"},
			},
		},
		{
			ID:          "openai",
			DisplayName: "OpenAI",
			EnvKey:      "OPENAI_API_KEY",
			Models: []KnownModel{
				{ID: "gpt-4o", DisplayName: "GPT-4o", Notes: "multimodal flagship"},
				{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Notes: "fast, cost-effective"},
				{ID: "o3", DisplayName: "o3", Notes: "advanced reasoning"},
				{ID: "o4-mini", DisplayName: "o4-mini", Notes: "fast reasoning"},
			},
		},
		{
			ID:          "kimi",
			DisplayName: "Kimi (Moonshot)",
			EnvKey:      "KIMI_API_KEY",
			Models: []KnownModel{
				{ID: "kimi-k2.5", DisplayName: "Kimi K2.5", Notes: "most capable"},
				{ID: "kimi-k2", DisplayName: "Kimi K2", Notes: "balanced"},
				{ID: "moonshot-v1-128k", DisplayName: "Moonshot v1 128K", Notes: "long context"},
				{ID: "moonshot-v1-32k", DisplayName: "Moonshot v1 32K", Notes: "balanced"},
				{ID: "moonshot-v1-8k", DisplayName: "Moonshot v1 8K", Notes: "fast"},
			},
		},
		{
			ID:          "local",
			DisplayName: "Local",
			EnvKey:      "",
			Models: []KnownModel{
				{ID: "llama3", DisplayName: "Llama 3", Notes: "open-source, local"},
				{ID: "mistral", DisplayName: "Mistral", Notes: "open-source, local"},
			},
		},
	}
}

// InferProvider returns the provider ID for a given model ID.
// It first checks the static Catalog for an exact model ID match, then
// falls back to common model name prefix matching. Returns "" if unrecognised.
func InferProvider(model string) string {
	lower := strings.ToLower(model)

	// Exact match in catalog
	for _, p := range Catalog {
		for _, m := range p.Models {
			if strings.ToLower(m.ID) == lower {
				return p.ID
			}
		}
	}

	// Prefix-based inference (mirrors gateway selectProviderWithRouting)
	prefixes := []struct{ prefix, provider string }{
		{"claude-", "anthropic"},
		{"gpt-", "openai"},
		{"o1-", "openai"},
		{"o3", "openai"},
		{"o4-", "openai"},
		{"chatgpt-", "openai"},
		{"gemini-", "google"},
		{"moonshot-", "kimi"},
		{"kimi-", "kimi"},
		{"llama-", "local"},
		{"mistral-", "local"},
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p.prefix) {
			return p.provider
		}
	}

	return ""
}

// APIKeys holds API keys discovered from environment variables.
type APIKeys struct {
	AnthropicKey string // ANTHROPIC_API_KEY
	OpenAIKey    string // OPENAI_API_KEY
	KimiKey      string // KIMI_API_KEY
}

// LoadAPIKeys reads API key environment variables.
func LoadAPIKeys() APIKeys {
	return APIKeys{
		AnthropicKey: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		KimiKey:      os.Getenv("KIMI_API_KEY"),
	}
}

// HasDirectAccess returns true if at least one direct API key is configured.
func (k APIKeys) HasDirectAccess() bool {
	return k.AnthropicKey != "" || k.OpenAIKey != "" || k.KimiKey != ""
}

// KeyForProvider returns the API key for the given provider ID, or empty string.
func (k APIKeys) KeyForProvider(provider string) string {
	switch provider {
	case "anthropic":
		return k.AnthropicKey
	case "openai":
		return k.OpenAIKey
	case "kimi":
		return k.KimiKey
	default:
		return ""
	}
}

// DirectChatRequest is a provider-agnostic chat request for direct API calls.
type DirectChatRequest struct {
	Provider  string // "anthropic" or "openai"
	Model     string // model ID
	Messages  []DirectMessage
	MaxTokens int
	APIKey    string
}

// DirectMessage is a single message in a chat request.
type DirectMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// DirectChatResponse is the parsed response from a direct API call.
type DirectChatResponse struct {
	Content string
	Model   string
	Usage   DirectUsage
}

// DirectUsage holds token usage information.
type DirectUsage struct {
	InputTokens  int
	OutputTokens int
}

// Chat sends a direct request to Anthropic or OpenAI (not via gateway).
// Uses net/http only — no external SDK.
func Chat(ctx context.Context, req DirectChatRequest) (*DirectChatResponse, error) {
	switch req.Provider {
	case "anthropic":
		return chatAnthropic(ctx, req)
	case "openai":
		return chatOpenAICompatible(ctx, req, "https://api.openai.com/v1/chat/completions")
	case "kimi":
		return chatOpenAICompatible(ctx, req, "https://api.moonshot.cn/v1/chat/completions")
	default:
		return nil, fmt.Errorf("unsupported provider %q: must be \"anthropic\", \"openai\", or \"kimi\"", req.Provider)
	}
}

// ─── Anthropic ────────────────────────────────────────────────────────────────

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	Messages  []anthropicMessage  `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func chatAnthropic(ctx context.Context, req DirectChatRequest) (*DirectChatResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	msgs := make([]anthropicMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = anthropicMessage{Role: m.Role, Content: m.Content}
	}

	body := anthropicRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		Messages:  msgs,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", req.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("anthropic: unexpected status %d: %v", resp.StatusCode, errBody)
	}

	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("anthropic: decode response: %w", err)
	}

	var text string
	if len(ar.Content) > 0 {
		text = ar.Content[0].Text
	}

	return &DirectChatResponse{
		Content: text,
		Model:   ar.Model,
		Usage: DirectUsage{
			InputTokens:  ar.Usage.InputTokens,
			OutputTokens: ar.Usage.OutputTokens,
		},
	}, nil
}

// ─── OpenAI ───────────────────────────────────────────────────────────────────

type openAIRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// chatOpenAICompatible works with any OpenAI-compatible API (OpenAI, Kimi/Moonshot, etc.)
func chatOpenAICompatible(ctx context.Context, req DirectChatRequest, endpoint string) (*DirectChatResponse, error) {
	msgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}

	body := openAIRequest{
		Model:     req.Model,
		Messages:  msgs,
		MaxTokens: req.MaxTokens,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", req.Provider, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		endpoint,
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", req.Provider, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: http request: %w", req.Provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("%s: unexpected status %d: %v", req.Provider, resp.StatusCode, errBody)
	}

	var or openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("%s: decode response: %w", req.Provider, err)
	}

	var text string
	if len(or.Choices) > 0 {
		text = or.Choices[0].Message.Content
	}

	return &DirectChatResponse{
		Content: text,
		Model:   or.Model,
		Usage: DirectUsage{
			InputTokens:  or.Usage.PromptTokens,
			OutputTokens: or.Usage.CompletionTokens,
		},
	}, nil
}

// ─── Display ──────────────────────────────────────────────────────────────────

// FormatProviderTable returns a formatted string showing all known providers
// and their models, with a checkmark if an API key is available.
func FormatProviderTable(keys APIKeys) string {
	var sb strings.Builder

	for _, p := range Catalog {
		// Determine key status
		var keyVal string
		if p.EnvKey == "" {
			keyVal = "n/a"
		} else {
			envVal := os.Getenv(p.EnvKey)
			if envVal != "" {
				keyVal = "set"
			} else {
				keyVal = "not set"
			}
		}

		if p.EnvKey == "" {
			fmt.Fprintf(&sb, "%s\n", p.DisplayName)
		} else {
			fmt.Fprintf(&sb, "%s [key: %s]\n", p.DisplayName, keyVal)
		}

		// Calculate max model ID width for alignment
		maxIDLen := 0
		for _, m := range p.Models {
			if len(m.ID) > maxIDLen {
				maxIDLen = len(m.ID)
			}
		}

		for _, m := range p.Models {
			if m.Notes != "" {
				padding := strings.Repeat(" ", maxIDLen-len(m.ID)+2)
				fmt.Fprintf(&sb, "  %s%s%s\n", m.ID, padding, m.Notes)
			} else {
				fmt.Fprintf(&sb, "  %s\n", m.ID)
			}
		}
	}

	return sb.String()
}
