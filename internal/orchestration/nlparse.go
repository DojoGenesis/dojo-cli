package orchestration

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/DojoGenesis/cli/internal/client"
)

// ParseTaskToDAG decomposes a natural language task into an ExecutionPlan.
// This is a client-side heuristic parser — no LLM involved.
func ParseTaskToDAG(task string) client.ExecutionPlan {
	steps := splitSteps(task)
	if len(steps) == 0 {
		// Single step — wrap the whole task
		steps = []rawStep{{text: task, parallel: false}}
	}

	dag := make([]client.ToolInvocation, 0, len(steps))
	var lastSeqID string

	for i, step := range steps {
		id := fmt.Sprintf("step-%d", i+1)
		tool := inferTool(step.text)

		inv := client.ToolInvocation{
			ID:       id,
			ToolName: tool,
			Input:    map[string]any{"task": strings.TrimSpace(step.text)},
		}

		if i > 0 {
			if step.parallel && lastSeqID != "" {
				// Parallel steps depend on the last sequential step
				inv.DependsOn = []string{lastSeqID}
			} else {
				// Sequential: depends on immediately preceding step
				inv.DependsOn = []string{fmt.Sprintf("step-%d", i)}
				inv.Input["context"] = fmt.Sprintf("{{step-%d.output}}", i)
			}
		}

		if !step.parallel {
			lastSeqID = id
		}

		dag = append(dag, inv)
	}

	return client.ExecutionPlan{
		ID:   fmt.Sprintf("dag-%d", time.Now().UnixNano()),
		Name: "NL-DAG: " + truncateStr(task, 60),
		DAG:  dag,
	}
}

type rawStep struct {
	text     string
	parallel bool // true if this step runs in parallel with the previous
}

// splitSteps breaks a task into ordered steps.
func splitSteps(task string) []rawStep {
	var steps []rawStep

	// Split on sentence boundaries
	sentences := splitSentences(task)

	for i, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}

		parallel := false
		if i > 0 {
			lower := strings.ToLower(sent)
			// Check for parallel markers at the start
			for _, marker := range []string{"and ", "also ", "while ", "simultaneously ", "in parallel ", "at the same time "} {
				if strings.HasPrefix(lower, marker) {
					parallel = true
					sent = strings.TrimSpace(sent[len(marker):])
					break
				}
			}
			// Check for sequential markers and strip them
			for _, marker := range []string{"then ", "after that ", "next ", "finally ", "lastly "} {
				lower = strings.ToLower(sent)
				if strings.HasPrefix(lower, marker) {
					sent = strings.TrimSpace(sent[len(marker):])
					break
				}
			}
		}

		// Capitalize first letter
		if len(sent) > 0 {
			runes := []rune(sent)
			runes[0] = unicode.ToUpper(runes[0])
			sent = string(runes)
		}

		steps = append(steps, rawStep{text: sent, parallel: parallel})
	}

	return steps
}

// splitSentences splits text on ". ", "; ", "\n", and explicit conjunctions.
func splitSentences(text string) []string {
	// Split the original text on multiple delimiters, preserving case.
	var parts []string
	current := text

	for _, delim := range []string{". ", "; ", "\n", ". Then ", ". Next "} {
		var newParts []string
		for _, seg := range append(parts, current) {
			if len(parts) == 0 && seg == current {
				// First pass: split current
				split := splitOnDelim(current, delim)
				newParts = append(newParts, split...)
				current = ""
			} else if seg != "" {
				sub := splitOnDelim(seg, delim)
				newParts = append(newParts, sub...)
			}
		}
		if len(newParts) > 0 {
			parts = newParts
			current = ""
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	// Now split remaining parts on conjunction words (case-insensitive)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Split on " then " and " and then "
		subParts := splitOnConjunctions(p)
		result = append(result, subParts...)
	}

	// Filter empty
	var final []string
	for _, r := range result {
		r = strings.TrimSpace(r)
		if r != "" {
			final = append(final, r)
		}
	}
	return final
}

// splitOnDelim splits a string on the given delimiter, returning all non-empty parts.
func splitOnDelim(s, delim string) []string {
	if delim == "" {
		return []string{s}
	}
	parts := strings.Split(s, delim)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// splitOnConjunctions splits a sentence on " then " and " and then " (case-insensitive).
func splitOnConjunctions(text string) []string {
	conjunctions := []string{" and then ", " then "}
	var parts []string
	remaining := text

	for {
		found := false
		for _, conj := range conjunctions {
			lower := strings.ToLower(remaining)
			idx := strings.Index(lower, conj)
			if idx >= 0 {
				parts = append(parts, remaining[:idx])
				remaining = remaining[idx+len(conj):]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	parts = append(parts, remaining)
	return parts
}

// inferTool maps task description to a tool name.
func inferTool(task string) string {
	lower := strings.ToLower(task)

	toolMap := []struct {
		keywords []string
		tool     string
	}{
		{[]string{"search", "find", "look up", "google", "query"}, "web_search"},
		{[]string{"read", "open", "fetch", "load"}, "file_read"},
		{[]string{"write", "create", "generate", "save"}, "file_write"},
		{[]string{"analyze", "examine", "review", "inspect", "audit"}, "analyze"},
		{[]string{"summarize", "recap", "digest", "condense", "tldr"}, "summarize"},
		{[]string{"compare", "contrast", "diff", "versus"}, "compare"},
		{[]string{"test", "verify", "check", "validate", "assert"}, "test"},
		{[]string{"build", "compile", "make"}, "build"},
		{[]string{"deploy", "publish", "release", "ship"}, "deploy"},
		{[]string{"install", "setup", "configure"}, "install"},
		{[]string{"delete", "remove", "clean", "purge"}, "cleanup"},
		{[]string{"transform", "convert", "format", "parse"}, "transform"},
	}

	for _, entry := range toolMap {
		for _, kw := range entry.keywords {
			if containsWord(lower, kw) {
				return entry.tool
			}
		}
	}

	return "execute" // fallback — gateway routes to appropriate handler
}

// containsWord returns true if text contains kw as a whole word (or multi-word phrase).
// For multi-word keywords (e.g. "look up"), a simple substring match is sufficient.
func containsWord(text, kw string) bool {
	idx := strings.Index(text, kw)
	if idx < 0 {
		return false
	}
	// For single-word keywords, require word boundaries.
	if !strings.Contains(kw, " ") {
		before := idx == 0 || !isAlpha(rune(text[idx-1]))
		after := idx+len(kw) >= len(text) || !isAlpha(rune(text[idx+len(kw)]))
		return before && after
	}
	return true
}

// isAlpha returns true for ASCII letters.
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
