package orchestration

import (
	"testing"
)

func TestMatchTemplate_Research(t *testing.T) {
	tmpl := MatchTemplate("research MCP protocol and summarize")
	if tmpl == nil {
		t.Fatal("expected a template match, got nil")
	}
	if tmpl.Name != "research-and-summarize" {
		t.Fatalf("expected template %q, got %q", "research-and-summarize", tmpl.Name)
	}
}

func TestMatchTemplate_Compare(t *testing.T) {
	tmpl := MatchTemplate("compare Go vs Rust for CLI tools")
	if tmpl == nil {
		t.Fatal("expected a template match, got nil")
	}
	if tmpl.Name != "search-and-compare" {
		t.Fatalf("expected template %q, got %q", "search-and-compare", tmpl.Name)
	}
}

func TestMatchTemplate_MultiStep(t *testing.T) {
	tmpl := MatchTemplate("search for X then compile results then publish")
	if tmpl == nil {
		t.Fatal("expected a template match, got nil")
	}
	if tmpl.Name != "multi-step" {
		t.Fatalf("expected template %q, got %q", "multi-step", tmpl.Name)
	}
}

func TestMatchTemplate_NoMatch(t *testing.T) {
	tmpl := MatchTemplate("hello world")
	if tmpl != nil {
		t.Fatalf("expected nil, got template %q", tmpl.Name)
	}
}

func TestBuild_Research(t *testing.T) {
	tmpl := MatchTemplate("research MCP protocol and summarize")
	if tmpl == nil {
		t.Fatal("expected template match")
	}
	plan := tmpl.Build("research MCP protocol and summarize")

	if len(plan.DAG) != 2 {
		t.Fatalf("expected 2 DAG nodes, got %d", len(plan.DAG))
	}
	if plan.DAG[0].ID != "step-1" {
		t.Errorf("expected step-1, got %s", plan.DAG[0].ID)
	}
	if plan.DAG[1].ID != "step-2" {
		t.Errorf("expected step-2, got %s", plan.DAG[1].ID)
	}
	if len(plan.DAG[0].DependsOn) != 0 {
		t.Errorf("step-1 should have no dependencies, got %v", plan.DAG[0].DependsOn)
	}
	if len(plan.DAG[1].DependsOn) != 1 || plan.DAG[1].DependsOn[0] != "step-1" {
		t.Errorf("step-2 should depend on step-1, got %v", plan.DAG[1].DependsOn)
	}
}

func TestBuild_Compare(t *testing.T) {
	tmpl := MatchTemplate("compare Go vs Rust for CLI tools")
	if tmpl == nil {
		t.Fatal("expected template match")
	}
	plan := tmpl.Build("compare Go vs Rust for CLI tools")

	if len(plan.DAG) != 3 {
		t.Fatalf("expected 3 DAG nodes, got %d", len(plan.DAG))
	}
	// step-1 and step-2 are parallel (no deps)
	if len(plan.DAG[0].DependsOn) != 0 {
		t.Errorf("step-1 should have no dependencies, got %v", plan.DAG[0].DependsOn)
	}
	if len(plan.DAG[1].DependsOn) != 0 {
		t.Errorf("step-2 should have no dependencies, got %v", plan.DAG[1].DependsOn)
	}
	// step-3 depends on both
	if len(plan.DAG[2].DependsOn) != 2 {
		t.Fatalf("step-3 should depend on 2 nodes, got %d", len(plan.DAG[2].DependsOn))
	}
	deps := map[string]bool{}
	for _, d := range plan.DAG[2].DependsOn {
		deps[d] = true
	}
	if !deps["step-1"] || !deps["step-2"] {
		t.Errorf("step-3 should depend on step-1 and step-2, got %v", plan.DAG[2].DependsOn)
	}
}

func TestBuild_MultiStep(t *testing.T) {
	tmpl := MatchTemplate("A then B then C")
	if tmpl == nil {
		t.Fatal("expected template match")
	}
	plan := tmpl.Build("A then B then C")

	if len(plan.DAG) != 3 {
		t.Fatalf("expected 3 DAG nodes, got %d", len(plan.DAG))
	}
	// Linear chain: step-1 -> step-2 -> step-3
	if len(plan.DAG[0].DependsOn) != 0 {
		t.Errorf("step-1 should have no dependencies, got %v", plan.DAG[0].DependsOn)
	}
	if len(plan.DAG[1].DependsOn) != 1 || plan.DAG[1].DependsOn[0] != "step-1" {
		t.Errorf("step-2 should depend on step-1, got %v", plan.DAG[1].DependsOn)
	}
	if len(plan.DAG[2].DependsOn) != 1 || plan.DAG[2].DependsOn[0] != "step-2" {
		t.Errorf("step-3 should depend on step-2, got %v", plan.DAG[2].DependsOn)
	}
}
