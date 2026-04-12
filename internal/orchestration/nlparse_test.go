package orchestration

import (
	"fmt"
	"strings"
	"testing"
)

// TestParseTaskToDAG_SingleStep — single sentence produces 1-node DAG with no dependencies.
func TestParseTaskToDAG_SingleStep(t *testing.T) {
	plan := ParseTaskToDAG("Search for information about Go generics")

	if len(plan.DAG) != 1 {
		t.Fatalf("expected 1 node, got %d", len(plan.DAG))
	}
	node := plan.DAG[0]
	if node.ID != "step-1" {
		t.Errorf("expected id 'step-1', got %q", node.ID)
	}
	if len(node.DependsOn) != 0 {
		t.Errorf("expected no dependencies for first node, got %v", node.DependsOn)
	}
	if !strings.HasPrefix(plan.Name, "NL-DAG: ") {
		t.Errorf("expected plan name to start with 'NL-DAG: ', got %q", plan.Name)
	}
	if plan.ID == "" {
		t.Error("expected non-empty plan ID")
	}
}

// TestParseTaskToDAG_Sequential — "X then Y then Z" produces 3 nodes in a chain.
func TestParseTaskToDAG_Sequential(t *testing.T) {
	plan := ParseTaskToDAG("Search for the topic then analyze the results then summarize findings")

	if len(plan.DAG) != 3 {
		t.Fatalf("expected 3 nodes, got %d: %v", len(plan.DAG), plan.DAG)
	}

	// step-1: no dependencies
	if len(plan.DAG[0].DependsOn) != 0 {
		t.Errorf("step-1 should have no dependencies, got %v", plan.DAG[0].DependsOn)
	}

	// step-2: depends on step-1
	if len(plan.DAG[1].DependsOn) != 1 || plan.DAG[1].DependsOn[0] != "step-1" {
		t.Errorf("step-2 should depend on step-1, got %v", plan.DAG[1].DependsOn)
	}

	// step-3: depends on step-2
	if len(plan.DAG[2].DependsOn) != 1 || plan.DAG[2].DependsOn[0] != "step-2" {
		t.Errorf("step-3 should depend on step-2, got %v", plan.DAG[2].DependsOn)
	}

	// Verify sequential nodes have context references
	if _, ok := plan.DAG[1].Input["context"]; !ok {
		t.Error("step-2 should have a context reference in input")
	}
}

// TestParseTaskToDAG_Parallel — parallel markers produce fan-out from a common predecessor.
func TestParseTaskToDAG_Parallel(t *testing.T) {
	plan := ParseTaskToDAG("Search for topic A. And search for topic B")

	if len(plan.DAG) != 2 {
		t.Fatalf("expected 2 nodes, got %d: %v", len(plan.DAG), plan.DAG)
	}

	// step-1: no dependencies
	if len(plan.DAG[0].DependsOn) != 0 {
		t.Errorf("step-1 should have no dependencies, got %v", plan.DAG[0].DependsOn)
	}

	// step-2: parallel — should depend on lastSeqID (step-1)
	if len(plan.DAG[1].DependsOn) != 1 || plan.DAG[1].DependsOn[0] != "step-1" {
		t.Errorf("step-2 (parallel) should depend on step-1 (last sequential), got %v", plan.DAG[1].DependsOn)
	}

	// Parallel node should NOT have context reference (it's not sequential)
	if _, ok := plan.DAG[1].Input["context"]; ok {
		t.Error("parallel step-2 should NOT have a sequential context reference")
	}
}

// TestParseTaskToDAG_Mixed — "X then Y and Z" produces a chain with a parallel fan-out.
func TestParseTaskToDAG_Mixed(t *testing.T) {
	// step1 sequential, step2 sequential, step3 parallel with step2's predecessor
	plan := ParseTaskToDAG("Search for the topic then analyze the results. And also summarize the data")

	if len(plan.DAG) < 3 {
		t.Fatalf("expected at least 3 nodes, got %d: %v", len(plan.DAG), plan.DAG)
	}

	// step-1: no dependencies
	if len(plan.DAG[0].DependsOn) != 0 {
		t.Errorf("step-1 should have no dependencies, got %v", plan.DAG[0].DependsOn)
	}

	// step-2: depends on step-1 (sequential)
	if len(plan.DAG[1].DependsOn) == 0 {
		t.Errorf("step-2 should have a dependency, got none")
	}

	// step-3: parallel — depends on last sequential node (step-2), not step-2's deps
	step3 := plan.DAG[2]
	if len(step3.DependsOn) == 0 {
		t.Errorf("step-3 (parallel) should depend on last sequential step, got none")
	}
}

// TestInferTool — keyword → tool mapping for each category.
func TestInferTool(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"search for Go tutorials", "web_search"},
		{"find the best practices", "web_search"},
		{"look up the documentation", "web_search"},
		{"read the config file", "file_read"},
		{"open the settings", "file_read"},
		{"fetch the remote data", "file_read"},
		{"write the report to disk", "file_write"},
		{"create a new document", "file_write"},
		{"generate the result", "file_write"},
		{"analyze the code quality", "analyze"},
		{"examine the performance metrics", "analyze"},
		{"review the pull request", "analyze"},
		{"summarize the findings", "summarize"},
		{"recap the meeting notes", "summarize"},
		{"compare the two approaches", "compare"},
		{"contrast the alternatives", "compare"},
		{"test the new feature", "test"},
		{"verify the output", "test"},
		{"validate the schema", "test"},
		{"build the binary", "build"},
		{"compile the source", "build"},
		{"deploy the application", "deploy"},
		{"publish the release", "deploy"},
		{"install the dependencies", "install"},
		{"configure the server", "install"},
		{"delete the old files", "cleanup"},
		{"remove the temporary data", "cleanup"},
		{"transform the JSON", "transform"},
		{"convert the format", "transform"},
		{"do something completely unknown", "execute"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := inferTool(tc.input)
			if got != tc.expected {
				t.Errorf("inferTool(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestSplitSteps — verify sentence splitting handles ". ", "; ", conjunctions.
func TestSplitSteps(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		minSteps int
	}{
		{
			name:     "period separator",
			input:    "Do task A. Do task B",
			minSteps: 2,
		},
		{
			name:     "semicolon separator",
			input:    "Do task A; Do task B; Do task C",
			minSteps: 3,
		},
		{
			name:     "then conjunction",
			input:    "Do task A then do task B",
			minSteps: 2,
		},
		{
			name:     "and then conjunction",
			input:    "Do task A and then do task B",
			minSteps: 2,
		},
		{
			name:     "newline separator",
			input:    "Do task A\nDo task B",
			minSteps: 2,
		},
		{
			name:     "single step",
			input:    "Do a single thing",
			minSteps: 1,
		},
		{
			name:     "parallel marker",
			input:    "Search for A. And search for B",
			minSteps: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			steps := splitSteps(tc.input)
			if len(steps) < tc.minSteps {
				t.Errorf("splitSteps(%q) = %d steps, want at least %d", tc.input, len(steps), tc.minSteps)
			}
			// All steps should have non-empty text
			for i, s := range steps {
				if strings.TrimSpace(s.text) == "" {
					t.Errorf("step %d has empty text", i)
				}
			}
		})
	}
}

// TestParseTaskToDAG_IDFormat — verifies DAG IDs and plan name format.
func TestParseTaskToDAG_IDFormat(t *testing.T) {
	plan := ParseTaskToDAG("Build the project then deploy it")

	for i, node := range plan.DAG {
		expectedID := fmt.Sprintf("step-%d", i+1)
		if node.ID != expectedID {
			t.Errorf("node %d: expected ID %q, got %q", i, expectedID, node.ID)
		}
		if _, ok := node.Input["task"]; !ok {
			t.Errorf("node %d: missing 'task' key in input", i)
		}
	}
}
