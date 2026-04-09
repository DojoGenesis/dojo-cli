package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ─── parseSSE ────────────────────────────────────────────────────────────────

func TestParseSSE_BasicChunks(t *testing.T) {
	input := "data: hello\ndata: world\n"
	r := strings.NewReader(input)

	var got []SSEChunk
	err := parseSSE(r, func(c SSEChunk) { got = append(got, c) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}
	if got[0].Data != "hello" {
		t.Errorf("expected 'hello', got %q", got[0].Data)
	}
	if got[1].Data != "world" {
		t.Errorf("expected 'world', got %q", got[1].Data)
	}
}

func TestParseSSE_DoneTerminator(t *testing.T) {
	// [DONE] should stop processing without calling onChunk.
	input := "data: first\ndata: [DONE]\ndata: should_not_appear\n"
	r := strings.NewReader(input)

	var got []SSEChunk
	err := parseSSE(r, func(c SSEChunk) { got = append(got, c) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 chunk (before [DONE]), got %d", len(got))
	}
	if got[0].Data != "first" {
		t.Errorf("expected 'first', got %q", got[0].Data)
	}
}

func TestParseSSE_EventPrefix(t *testing.T) {
	// event: lines should be captured in SSEChunk.Event and cleared after data.
	input := "event: ping\ndata: payload\n\nevent: pong\ndata: other\n"
	r := strings.NewReader(input)

	var got []SSEChunk
	err := parseSSE(r, func(c SSEChunk) { got = append(got, c) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}
	if got[0].Event != "ping" {
		t.Errorf("expected event 'ping', got %q", got[0].Event)
	}
	if got[0].Data != "payload" {
		t.Errorf("expected data 'payload', got %q", got[0].Data)
	}
	if got[1].Event != "pong" {
		t.Errorf("expected event 'pong', got %q", got[1].Event)
	}
}

func TestParseSSE_EmptyData(t *testing.T) {
	// Empty stream should produce no chunks and no error.
	r := strings.NewReader("")
	var got []SSEChunk
	err := parseSSE(r, func(c SSEChunk) { got = append(got, c) })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 chunks, got %d", len(got))
	}
}

func TestParseSSE_DataWithLeadingSpace(t *testing.T) {
	// "data: " prefix includes a space; TrimSpace should handle " value" -> "value".
	input := "data:  trimmed \n"
	r := strings.NewReader(input)

	var got []SSEChunk
	if err := parseSSE(r, func(c SSEChunk) { got = append(got, c) }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Data != "trimmed" {
		t.Errorf("expected trimmed data 'trimmed', got %v", got)
	}
}

// ─── extractText — tested via SSEChunk directly (same logic as repl.extractText) ──

// extractTextLocal mirrors the logic in repl.extractText so we can test JSON
// unwrap behaviour without depending on the repl package (which imports readline).
func extractTextLocal(data string) string {
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err == nil {
		// OpenAI delta format
		if choices, ok := m["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if delta, ok := choice["delta"].(map[string]any); ok {
					if content, ok := delta["content"].(string); ok {
						return content
					}
				}
				if text, ok := choice["text"].(string); ok {
					return text
				}
			}
		}
		for _, key := range []string{"text", "content", "message", "response"} {
			if v, ok := m[key].(string); ok {
				return v
			}
		}
		return ""
	}
	return data
}

func TestExtractText_PlainText(t *testing.T) {
	got := extractTextLocal("hello world")
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestExtractText_OpenAIDelta(t *testing.T) {
	data := `{"choices":[{"delta":{"content":"hello"}}]}`
	got := extractTextLocal(data)
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestExtractText_SimpleText(t *testing.T) {
	data := `{"text":"hello"}`
	got := extractTextLocal(data)
	if got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestExtractText_SimpleContent(t *testing.T) {
	data := `{"content":"world"}`
	got := extractTextLocal(data)
	if got != "world" {
		t.Errorf("expected 'world', got %q", got)
	}
}

func TestExtractText_Done(t *testing.T) {
	got := extractTextLocal("[DONE]")
	if got != "" {
		t.Errorf("expected empty string for [DONE], got %q", got)
	}
}

func TestExtractText_Empty(t *testing.T) {
	got := extractTextLocal("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// ─── Health ──────────────────────────────────────────────────────────────────

func TestHealth_MockServer(t *testing.T) {
	want := HealthResponse{
		Status:            "ok",
		Version:           "1.2.3",
		Timestamp:         "2026-04-09T00:00:00Z",
		Providers:         map[string]string{"claude": "healthy"},
		Dependencies:      map[string]string{"db": "healthy"},
		UptimeSeconds:     42,
		RequestsProcessed: 100,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "5s")
	got, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() returned error: %v", err)
	}
	if got.Status != want.Status {
		t.Errorf("Status: got %q, want %q", got.Status, want.Status)
	}
	if got.Version != want.Version {
		t.Errorf("Version: got %q, want %q", got.Version, want.Version)
	}
	if got.UptimeSeconds != want.UptimeSeconds {
		t.Errorf("UptimeSeconds: got %d, want %d", got.UptimeSeconds, want.UptimeSeconds)
	}
	if got.RequestsProcessed != want.RequestsProcessed {
		t.Errorf("RequestsProcessed: got %d, want %d", got.RequestsProcessed, want.RequestsProcessed)
	}
	if got.Providers["claude"] != "healthy" {
		t.Errorf("Providers[claude]: got %q, want %q", got.Providers["claude"], "healthy")
	}
}

func TestHealth_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "5s")
	_, err := c.Health(context.Background())
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}

// ─── Seeds ───────────────────────────────────────────────────────────────────

func TestSeeds_MockServer(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	wantSeed := Seed{
		ID:          "seed-1",
		Name:        "test seed",
		Description: "a test seed",
		Trigger:     "testing",
		Content:     "seed content",
		UsageCount:  5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	envelope := seedsEnvelope{
		Success: true,
		Count:   1,
		Seeds:   []Seed{wantSeed},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/seeds" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "5s")
	seeds, err := c.Seeds(context.Background())
	if err != nil {
		t.Fatalf("Seeds() returned error: %v", err)
	}
	if len(seeds) != 1 {
		t.Fatalf("expected 1 seed, got %d", len(seeds))
	}
	if seeds[0].ID != wantSeed.ID {
		t.Errorf("ID: got %q, want %q", seeds[0].ID, wantSeed.ID)
	}
	if seeds[0].Name != wantSeed.Name {
		t.Errorf("Name: got %q, want %q", seeds[0].Name, wantSeed.Name)
	}
	if seeds[0].UsageCount != wantSeed.UsageCount {
		t.Errorf("UsageCount: got %d, want %d", seeds[0].UsageCount, wantSeed.UsageCount)
	}
}

func TestSeeds_EmptyEnvelope(t *testing.T) {
	envelope := seedsEnvelope{Success: true, Count: 0, Seeds: nil}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope)
	}))
	defer srv.Close()

	c := New(srv.URL, "", "5s")
	seeds, err := c.Seeds(context.Background())
	if err != nil {
		t.Fatalf("Seeds() returned error: %v", err)
	}
	if len(seeds) != 0 {
		t.Errorf("expected 0 seeds, got %d", len(seeds))
	}
}

func TestNew_BadTimeout_Defaults(t *testing.T) {
	// A bad timeout string should fall back to 60s without panicking.
	c := New("http://localhost:9999", "", "not-a-duration")
	if c == nil {
		t.Fatal("New() returned nil")
	}
	if c.http.Timeout != 60*time.Second {
		t.Errorf("expected default 60s timeout, got %v", c.http.Timeout)
	}
}
