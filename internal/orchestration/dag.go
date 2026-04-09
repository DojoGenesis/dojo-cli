package orchestration

import (
	"fmt"
	"strings"
	"time"

	"github.com/DojoGenesis/dojo-cli/internal/client"
)

// Template is a named DAG pattern.
type Template struct {
	Name        string
	Description string
	Match       func(task string) bool
	Build       func(task string) client.ExecutionPlan
}

// BuiltinTemplates returns pre-defined DAG templates.
func BuiltinTemplates() []Template {
	return []Template{
		researchAndSummarize(),
		analyzeAndReport(),
		searchAndCompare(),
		multiStep(),
	}
}

// MatchTemplate finds the first matching template, or nil.
func MatchTemplate(task string) *Template {
	lower := strings.ToLower(task)
	for _, t := range BuiltinTemplates() {
		if t.Match(lower) {
			return &t
		}
	}
	return nil
}

func planID() string {
	return fmt.Sprintf("plan-%d", time.Now().UnixNano())
}

func researchAndSummarize() Template {
	return Template{
		Name:        "research-and-summarize",
		Description: "Search the web for a topic and summarize findings",
		Match: func(task string) bool {
			return (strings.Contains(task, "research") || strings.Contains(task, "look up")) &&
				(strings.Contains(task, "summar") || strings.Contains(task, "write up"))
		},
		Build: func(task string) client.ExecutionPlan {
			// Extract the topic (strip common prefixes)
			topic := task
			for _, prefix := range []string{"research ", "look up "} {
				lower := strings.ToLower(task)
				if idx := strings.Index(lower, prefix); idx >= 0 {
					topic = task[idx+len(prefix):]
					break
				}
			}
			// Strip "and summarize" / "and write up" suffix
			for _, suffix := range []string{" and summarize", " and write up", " then summarize"} {
				if idx := strings.Index(strings.ToLower(topic), suffix); idx >= 0 {
					topic = topic[:idx]
				}
			}
			return client.ExecutionPlan{
				ID:   planID(),
				Name: "Research and summarize: " + topic,
				DAG: []client.ToolInvocation{
					{ID: "step-1", ToolName: "web_search", Input: map[string]any{"query": topic}},
					{ID: "step-2", ToolName: "summarize", Input: map[string]any{"text": "{{step-1.output}}"}, DependsOn: []string{"step-1"}},
				},
			}
		},
	}
}

func analyzeAndReport() Template {
	return Template{
		Name:        "analyze-and-report",
		Description: "Search, analyze, and produce a report",
		Match: func(task string) bool {
			return (strings.Contains(task, "analy") || strings.Contains(task, "examin")) &&
				strings.Contains(task, "report")
		},
		Build: func(task string) client.ExecutionPlan {
			topic := extractTopic(task, []string{"analyze ", "examine "})
			return client.ExecutionPlan{
				ID:   planID(),
				Name: "Analyze and report: " + topic,
				DAG: []client.ToolInvocation{
					{ID: "step-1", ToolName: "web_search", Input: map[string]any{"query": topic}},
					{ID: "step-2", ToolName: "analyze", Input: map[string]any{"text": "{{step-1.output}}"}, DependsOn: []string{"step-1"}},
					{ID: "step-3", ToolName: "report", Input: map[string]any{"text": "{{step-2.output}}"}, DependsOn: []string{"step-2"}},
				},
			}
		},
	}
}

func searchAndCompare() Template {
	return Template{
		Name:        "search-and-compare",
		Description: "Search for two topics and compare them",
		Match: func(task string) bool {
			return strings.Contains(task, "compare") &&
				(strings.Contains(task, " and ") || strings.Contains(task, " vs ") || strings.Contains(task, " versus "))
		},
		Build: func(task string) client.ExecutionPlan {
			// Split on "and", "vs", "versus" to get two topics
			var a, b string
			lower := strings.ToLower(task)
			for _, sep := range []string{" vs ", " versus ", " and "} {
				if idx := strings.Index(lower, sep); idx > 0 {
					a = strings.TrimSpace(task[:idx])
					b = strings.TrimSpace(task[idx+len(sep):])
					break
				}
			}
			// Strip "compare" prefix from a
			for _, p := range []string{"compare "} {
				if strings.HasPrefix(strings.ToLower(a), p) {
					a = a[len(p):]
				}
			}
			if a == "" {
				a = "topic A"
			}
			if b == "" {
				b = "topic B"
			}
			return client.ExecutionPlan{
				ID:   planID(),
				Name: fmt.Sprintf("Compare: %s vs %s", a, b),
				DAG: []client.ToolInvocation{
					{ID: "step-1", ToolName: "web_search", Input: map[string]any{"query": a}},
					{ID: "step-2", ToolName: "web_search", Input: map[string]any{"query": b}},
					{ID: "step-3", ToolName: "compare", Input: map[string]any{"a": "{{step-1.output}}", "b": "{{step-2.output}}"}, DependsOn: []string{"step-1", "step-2"}},
				},
			}
		},
	}
}

func multiStep() Template {
	return Template{
		Name:        "multi-step",
		Description: "Sequential steps separated by 'then'",
		Match: func(task string) bool {
			return strings.Contains(task, " then ")
		},
		Build: func(task string) client.ExecutionPlan {
			parts := strings.Split(task, " then ")
			dag := make([]client.ToolInvocation, 0, len(parts))
			for i, part := range parts {
				step := client.ToolInvocation{
					ID:       fmt.Sprintf("step-%d", i+1),
					ToolName: "execute",
					Input:    map[string]any{"task": strings.TrimSpace(part)},
				}
				if i > 0 {
					step.DependsOn = []string{fmt.Sprintf("step-%d", i)}
					step.Input["context"] = fmt.Sprintf("{{step-%d.output}}", i)
				}
				dag = append(dag, step)
			}
			return client.ExecutionPlan{
				ID:   planID(),
				Name: "Multi-step: " + truncateStr(task, 50),
				DAG:  dag,
			}
		},
	}
}

func extractTopic(task string, prefixes []string) string {
	lower := strings.ToLower(task)
	for _, p := range prefixes {
		if idx := strings.Index(lower, p); idx >= 0 {
			return strings.TrimSpace(task[idx+len(p):])
		}
	}
	return task
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "\u2026"
}
