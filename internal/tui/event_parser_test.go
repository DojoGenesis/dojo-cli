package tui

import (
	"strings"
	"testing"

	"github.com/DojoGenesis/dojo-cli/internal/client"
)

// ─── ParseSSEEvent — Core Events ────────────────────────────────────────────

func TestParseSSEEvent_IntentClassified(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "intent_classified",
		Data:  `{"intent":"CodeGeneration","confidence":0.92}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "CodeGeneration")
	assertContains(t, pe.Summary, "92%")
}

func TestParseSSEEvent_ProviderSelected(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "provider_selected",
		Data:  `{"provider":"anthropic","model":"claude-sonnet-4-6"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "anthropic")
	assertContains(t, pe.Summary, "claude-sonnet-4-6")
}

func TestParseSSEEvent_ToolInvoked(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "tool_invoked",
		Data:  `{"tool":"mcp_file_read"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "mcp_file_read")
}

func TestParseSSEEvent_ToolCompleted_Success(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "tool_completed",
		Data:  `{"tool":"mcp_file_read","duration_ms":42,"status":"success"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeveritySuccess)
	assertContains(t, pe.Summary, "mcp_file_read")
	assertContains(t, pe.Summary, "42ms")
}

func TestParseSSEEvent_ToolCompleted_Failed(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "tool_completed",
		Data:  `{"tool":"execute","status":"failed"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityError)
	assertContains(t, pe.Summary, "FAILED")
}

func TestParseSSEEvent_ToolCompleted_ErrorField(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "tool_completed",
		Data:  `{"tool":"execute","error":"timeout"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityError)
	assertContains(t, pe.Summary, "FAILED")
}

func TestParseSSEEvent_Thinking(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "thinking",
		Data:  `{"message":"analyzing the code structure"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "analyzing the code structure")
}

func TestParseSSEEvent_Thinking_LongMessage(t *testing.T) {
	long := strings.Repeat("a", 200)
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "thinking",
		Data:  `{"message":"` + long + `"}`,
	})
	// Summary should be truncated to 80 chars + ellipsis prefix.
	if len(pe.Summary) > 85 {
		t.Errorf("thinking summary not truncated: len=%d", len(pe.Summary))
	}
}

func TestParseSSEEvent_ResponseChunk(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "response_chunk",
		Data:  `{"content":"Hello, how can I help?"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "22 chars")
}

func TestParseSSEEvent_MemoryRetrieved(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "memory_retrieved",
		Data:  `{"memories_found":5}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "5 items")
}

func TestParseSSEEvent_Complete(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "complete",
		Data:  `{"tokens_in":4200,"tokens_out":1800}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeveritySuccess)
	assertContains(t, pe.Summary, "4200")
	assertContains(t, pe.Summary, "1800")
}

func TestParseSSEEvent_Error(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "error",
		Data:  `{"error_code":"RATE_LIMIT","error":"too many requests"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityError)
	assertContains(t, pe.Summary, "RATE_LIMIT")
	assertContains(t, pe.Summary, "too many requests")
}

// ─── ParseSSEEvent — Trace Events ───────────────────────────────────────────

func TestParseSSEEvent_TraceSpanStart(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "trace_span_start",
		Data:  `{"name":"intent_classification"}`,
	})
	assertCategory(t, pe, CategoryTrace)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "intent_classification")
	assertContains(t, pe.Summary, "started")
}

func TestParseSSEEvent_TraceSpanEnd_Success(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "trace_span_end",
		Data:  `{"name":"intent_classification","duration_ms":12,"status":"ok"}`,
	})
	assertCategory(t, pe, CategoryTrace)
	assertSeverity(t, pe, SeveritySuccess)
	assertContains(t, pe.Summary, "12ms")
}

func TestParseSSEEvent_TraceSpanEnd_Error(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "trace_span_end",
		Data:  `{"name":"provider_call","duration_ms":5000,"status":"error"}`,
	})
	assertCategory(t, pe, CategoryTrace)
	assertSeverity(t, pe, SeverityError)
}

// ─── ParseSSEEvent — Artifact Events ────────────────────────────────────────

func TestParseSSEEvent_ArtifactCreated(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "artifact_created",
		Data:  `{"artifact_name":"event_parser.go","artifact_type":"go"}`,
	})
	assertCategory(t, pe, CategoryArtifact)
	assertSeverity(t, pe, SeveritySuccess)
	assertContains(t, pe.Summary, "event_parser.go")
	assertContains(t, pe.Summary, "go")
}

func TestParseSSEEvent_ArtifactUpdated(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "artifact_updated",
		Data:  `{"artifact_name":"pilot.go","version":"2"}`,
	})
	assertCategory(t, pe, CategoryArtifact)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "pilot.go")
	assertContains(t, pe.Summary, "v2")
}

func TestParseSSEEvent_ProjectSwitched(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "project_switched",
		Data:  `{"project_name":"dojo-cli"}`,
	})
	assertCategory(t, pe, CategoryArtifact)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "dojo-cli")
}

func TestParseSSEEvent_DiagramRendered(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "diagram_rendered",
		Data:  `{"diagram_type":"mermaid","format":"svg"}`,
	})
	assertCategory(t, pe, CategoryArtifact)
	assertSeverity(t, pe, SeveritySuccess)
	assertContains(t, pe.Summary, "mermaid")
	assertContains(t, pe.Summary, "svg")
}

func TestParseSSEEvent_PatchIntent(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "patch_intent",
		Data:  `{"operation":"insert","description":"Add ParsedEvent struct to event_parser.go"}`,
	})
	assertCategory(t, pe, CategoryArtifact)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "insert")
}

func TestParseSSEEvent_PatchIntent_LongDescription(t *testing.T) {
	long := strings.Repeat("x", 200)
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "patch_intent",
		Data:  `{"operation":"replace","description":"` + long + `"}`,
	})
	// Description should be truncated to 60 chars.
	if len(pe.Summary) > 80 {
		t.Errorf("patch_intent description not truncated: len=%d", len(pe.Summary))
	}
}

// ─── ParseSSEEvent — Orchestration Events ───────────────────────────────────

func TestParseSSEEvent_OrchPlanCreated(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "orchestration_plan_created",
		Data:  `{"node_count":5,"estimated_cost":0.0234}`,
	})
	assertCategory(t, pe, CategoryOrchestration)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "5 nodes")
	assertContains(t, pe.Summary, "$0.0234")
}

func TestParseSSEEvent_OrchNodeStart(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "orchestration_node_start",
		Data:  `{"tool_name":"scout"}`,
	})
	assertCategory(t, pe, CategoryOrchestration)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "scout")
}

func TestParseSSEEvent_OrchNodeEnd_Success(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "orchestration_node_end",
		Data:  `{"tool_name":"scout","duration_ms":350,"status":"success"}`,
	})
	assertCategory(t, pe, CategoryOrchestration)
	assertSeverity(t, pe, SeveritySuccess)
	assertContains(t, pe.Summary, "scout")
	assertContains(t, pe.Summary, "350ms")
}

func TestParseSSEEvent_OrchNodeEnd_Failed(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "orchestration_node_end",
		Data:  `{"tool_name":"reflect","status":"failed","error":"timeout"}`,
	})
	assertCategory(t, pe, CategoryOrchestration)
	assertSeverity(t, pe, SeverityError)
	assertContains(t, pe.Summary, "FAILED")
}

func TestParseSSEEvent_OrchReplanning(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "orchestration_replanning",
		Data:  `{"reason":"node failed, replanning"}`,
	})
	assertCategory(t, pe, CategoryOrchestration)
	assertSeverity(t, pe, SeverityWarning)
	assertContains(t, pe.Summary, "replanning")
}

func TestParseSSEEvent_OrchComplete(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "orchestration_complete",
		Data:  `{"success_nodes":4,"total_nodes":5,"duration_ms":12000}`,
	})
	assertCategory(t, pe, CategoryOrchestration)
	assertSeverity(t, pe, SeveritySuccess)
	assertContains(t, pe.Summary, "4/5")
	assertContains(t, pe.Summary, "12000ms")
}

func TestParseSSEEvent_OrchFailed(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "orchestration_failed",
		Data:  `{"reason":"circuit breaker tripped"}`,
	})
	assertCategory(t, pe, CategoryOrchestration)
	assertSeverity(t, pe, SeverityError)
	assertContains(t, pe.Summary, "circuit breaker")
}

// ─── ParseSSEEvent — Edge Cases ─────────────────────────────────────────────

func TestParseSSEEvent_UnknownEvent(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "custom_event",
		Data:  `{"foo":"bar"}`,
	})
	assertCategory(t, pe, CategoryCore)
	assertSeverity(t, pe, SeverityInfo)
	assertContains(t, pe.Summary, "custom_event")
}

func TestParseSSEEvent_UnknownEvent_NoEventType(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "",
		Data:  `some raw text`,
	})
	assertContains(t, pe.Summary, "some raw text")
}

func TestParseSSEEvent_InvalidJSON(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "intent_classified",
		Data:  `not json`,
	})
	// Should not panic, should fall through to default summary.
	if pe.EventType != "intent_classified" {
		t.Errorf("expected event type preserved, got %q", pe.EventType)
	}
}

func TestParseSSEEvent_EmptyData(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "complete",
		Data:  "",
	})
	if pe.EventType != "complete" {
		t.Errorf("expected event type complete, got %q", pe.EventType)
	}
}

func TestParseSSEEvent_LongRawData(t *testing.T) {
	long := `{"data":"` + strings.Repeat("x", 200) + `"}`
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "unknown_type",
		Data:  long,
	})
	// Summary for unknown events should be truncated to ~80 chars.
	if len(pe.Summary) > 100 {
		t.Errorf("unknown event summary not truncated: len=%d", len(pe.Summary))
	}
}

func TestParseSSEEvent_TimestampFormat(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "complete",
		Data:  `{}`,
	})
	// Time should be HH:MM:SS format.
	if len(pe.Time) != 8 || pe.Time[2] != ':' || pe.Time[5] != ':' {
		t.Errorf("time format not HH:MM:SS: got %q", pe.Time)
	}
}

func TestParseSSEEvent_RawDataPreserved(t *testing.T) {
	raw := `{"intent":"Debugging","confidence":0.85}`
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "intent_classified",
		Data:  raw,
	})
	if pe.RawData != raw {
		t.Errorf("RawData not preserved: got %q", pe.RawData)
	}
}

func TestParseSSEEvent_ParsedMapPopulated(t *testing.T) {
	pe := ParseSSEEvent(client.SSEChunk{
		Event: "provider_selected",
		Data:  `{"provider":"openai","model":"gpt-4"}`,
	})
	if pe.Parsed["provider"] != "openai" {
		t.Errorf("Parsed[provider] = %v, want openai", pe.Parsed["provider"])
	}
}

// ─── JSON Helper Tests ──────────────────────────────────────────────────────

func TestGetStr(t *testing.T) {
	m := map[string]any{"key": "value", "num": 42.0, "nil": nil}

	if got := getStr(m, "key"); got != "value" {
		t.Errorf("getStr(key) = %q, want %q", got, "value")
	}
	if got := getStr(m, "num"); got != "42" {
		t.Errorf("getStr(num) = %q, want %q", got, "42")
	}
	if got := getStr(m, "missing"); got != "" {
		t.Errorf("getStr(missing) = %q, want empty", got)
	}
}

func TestGetFloat(t *testing.T) {
	m := map[string]any{"val": 3.14, "str": "hello", "nil": nil}

	if got := getFloat(m, "val"); got != 3.14 {
		t.Errorf("getFloat(val) = %f, want 3.14", got)
	}
	if got := getFloat(m, "str"); got != 0 {
		t.Errorf("getFloat(str) = %f, want 0", got)
	}
	if got := getFloat(m, "missing"); got != 0 {
		t.Errorf("getFloat(missing) = %f, want 0", got)
	}
}

func TestIsTruthy(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want bool
	}{
		{"true bool", true, true},
		{"false bool", false, false},
		{"non-empty string", "yes", true},
		{"empty string", "", false},
		{"non-zero float", 1.0, true},
		{"zero float", 0.0, false},
		{"nil", nil, false},
		{"slice (default true)", []string{}, true},
	}
	for _, tc := range tests {
		m := map[string]any{"k": tc.val}
		if got := isTruthy(m, "k"); got != tc.want {
			t.Errorf("isTruthy(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
	// Missing key.
	if got := isTruthy(map[string]any{}, "missing"); got != false {
		t.Errorf("isTruthy(missing) = %v, want false", got)
	}
}

// ─── Table-Driven Coverage ──────────────────────────────────────────────────

func TestParseSSEEvent_AllEventTypes(t *testing.T) {
	// Verify every known event type produces the correct category and severity.
	tests := []struct {
		event    string
		data     string
		category EventCategory
		severity EventSeverity
	}{
		{"intent_classified", `{"intent":"Planning","confidence":0.8}`, CategoryCore, SeverityInfo},
		{"provider_selected", `{"provider":"anthropic","model":"sonnet"}`, CategoryCore, SeverityInfo},
		{"tool_invoked", `{"tool":"search"}`, CategoryCore, SeverityInfo},
		{"tool_completed", `{"tool":"search","duration_ms":10}`, CategoryCore, SeveritySuccess},
		{"thinking", `{"message":"hmm"}`, CategoryCore, SeverityInfo},
		{"response_chunk", `{"content":"hi"}`, CategoryCore, SeverityInfo},
		{"memory_retrieved", `{"memories_found":3}`, CategoryCore, SeverityInfo},
		{"complete", `{"tokens_in":100,"tokens_out":50}`, CategoryCore, SeveritySuccess},
		{"error", `{"error_code":"E1","error":"bad"}`, CategoryCore, SeverityError},
		{"trace_span_start", `{"name":"s1"}`, CategoryTrace, SeverityInfo},
		{"trace_span_end", `{"name":"s1","duration_ms":5,"status":"ok"}`, CategoryTrace, SeveritySuccess},
		{"artifact_created", `{"artifact_name":"f","artifact_type":"go"}`, CategoryArtifact, SeveritySuccess},
		{"artifact_updated", `{"artifact_name":"f","version":"1"}`, CategoryArtifact, SeverityInfo},
		{"project_switched", `{"project_name":"p"}`, CategoryArtifact, SeverityInfo},
		{"diagram_rendered", `{"diagram_type":"m","format":"svg"}`, CategoryArtifact, SeveritySuccess},
		{"patch_intent", `{"operation":"o","description":"d"}`, CategoryArtifact, SeverityInfo},
		{"orchestration_plan_created", `{"node_count":3,"estimated_cost":0.01}`, CategoryOrchestration, SeverityInfo},
		{"orchestration_node_start", `{"tool_name":"t"}`, CategoryOrchestration, SeverityInfo},
		{"orchestration_node_end", `{"tool_name":"t","duration_ms":1}`, CategoryOrchestration, SeveritySuccess},
		{"orchestration_replanning", `{"reason":"r"}`, CategoryOrchestration, SeverityWarning},
		{"orchestration_complete", `{"success_nodes":2,"total_nodes":2,"duration_ms":100}`, CategoryOrchestration, SeveritySuccess},
		{"orchestration_failed", `{"reason":"r"}`, CategoryOrchestration, SeverityError},
	}
	for _, tc := range tests {
		t.Run(tc.event, func(t *testing.T) {
			pe := ParseSSEEvent(client.SSEChunk{Event: tc.event, Data: tc.data})
			assertCategory(t, pe, tc.category)
			assertSeverity(t, pe, tc.severity)
			if pe.Summary == "" {
				t.Error("expected non-empty Summary")
			}
		})
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func assertCategory(t *testing.T, pe ParsedEvent, want EventCategory) {
	t.Helper()
	if pe.Category != want {
		t.Errorf("Category = %d, want %d", pe.Category, want)
	}
}

func assertSeverity(t *testing.T, pe ParsedEvent, want EventSeverity) {
	t.Helper()
	if pe.Severity != want {
		t.Errorf("Severity = %d, want %d", pe.Severity, want)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected %q to contain %q", s, substr)
	}
}
