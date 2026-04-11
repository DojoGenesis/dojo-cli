package skills

import (
	"testing"

	"github.com/DojoGenesis/cli/internal/client"
)

// TestClusterCategory verifies that representative skills from the live corpus
// resolve to the expected category.  Each case uses the real skill name and a
// short excerpt of its actual description so the test stays grounded.
func TestClusterCategory(t *testing.T) {
	cases := []struct {
		name        string
		description string
		want        string
	}{
		// ── Agents ────────────────────────────────────────────────────────────
		{"agent-dispatch-playbook", "Produces a dispatch plan — isolation model, agent count, sequencing, and model routing", "agents"},
		{"maestro-orchestration", "Conductor agent pattern that decomposes complex tasks, dispatches specialist sub-agents", "agents"},
		{"parallel-agents", "Launch multiple agents in parallel with file-based coordination and context isolation", "agents"},
		{"multi-agent", "Multi-agent collaboration plugin that spawns N parallel subagents competing on tasks", "agents"},

		// ── Engineering ───────────────────────────────────────────────────────
		{"code-review", "Review code for quality, maintainability, and correctness. Use when reviewing pull requests", "engineering"},
		{"focused-fix", "Use when the user asks to fix, debug, or make a specific feature/module/area work end-to-end", "engineering"},
		{"tech-debt", "Scan, prioritize, and report technical debt across the codebase", "engineering"},
		{"tdd-guide", "Test-driven development skill for writing unit tests, generating test fixtures", "engineering"},
		{"refactor", "Code refactoring workflow — analyze, plan, implement, review, validate", "engineering"},
		{"monorepo-navigator", "Navigate and optimize monorepos with cross-package impact analysis, selective builds", "engineering"},

		// ── Security ──────────────────────────────────────────────────────────
		{"ai-security", "Use when assessing AI/ML systems for prompt injection, jailbreak vulnerabilities", "security"},
		{"semgrep-rule-creator", "Creates custom Semgrep rules for detecting security vulnerabilities, bug patterns", "security"},
		{"insecure-defaults", "Detects fail-open insecure defaults — hardcoded secrets, weak auth, permissive settings", "security"},
		{"red-team", "Use when planning or executing authorized red team engagements, attack path analysis", "security"},
		{"zeroize-audit", "Detects missing zeroization of sensitive data in source code", "security"},
		{"constant-time-analysis", "Detects timing side-channel vulnerabilities in cryptographic code", "security"},

		// ── Blockchain ────────────────────────────────────────────────────────
		{"solana-vulnerability-scanner", "Scans Solana programs for 6 critical vulnerabilities including arbitrary CPI", "blockchain"},
		{"algorand-vulnerability-scanner", "Scans Algorand smart contracts for 11 common vulnerabilities", "blockchain"},
		{"cairo-vulnerability-scanner", "Scans Cairo/StarkNet smart contracts for 6 critical vulnerabilities", "blockchain"},
		{"cosmos-vulnerability-scanner", "Scans Cosmos SDK blockchain modules and CosmWasm contracts for consensus-critical issues", "blockchain"},

		// ── Cloud ─────────────────────────────────────────────────────────────
		{"aws-cdk-development", "AWS Cloud Development Kit (CDK) expert for building cloud infrastructure with TypeScript", "cloud"},
		{"terraform-patterns", "Terraform infrastructure-as-code agent skill for IaC development", "cloud"},
		{"helm-chart-builder", "Helm chart development agent skill for Kubernetes deployment packaging", "cloud"},
		{"azure-cloud-architect", "Design Azure architectures for startups and enterprises", "cloud"},

		// ── AI/ML ─────────────────────────────────────────────────────────────
		{"rag-architect", "Use when the user asks to design RAG pipelines, optimize retrieval strategies", "ai-ml"},
		{"llm-cost-optimizer", "Use when you need to reduce LLM API spend, control token usage, route between models", "ai-ml"},
		{"senior-ml-engineer", "ML engineering skill for productionizing models, building MLOps pipelines", "ai-ml"},
		{"senior-computer-vision", "Computer vision engineering skill for object detection, image segmentation", "ai-ml"},

		// ── Math ──────────────────────────────────────────────────────────────
		{"hilbert-spaces", "Problem-solving strategies for hilbert spaces in functional analysis", "math"},
		{"lebesgue-measure", "Problem-solving strategies for lebesgue measure in measure theory", "math"},
		{"eigenvalues", "Problem-solving strategies for eigenvalues in linear algebra", "math"},
		{"contour-integrals", "Problem-solving strategies for contour integrals in complex analysis", "math"},
		{"banach-spaces", "Problem-solving strategies for banach spaces in functional analysis", "math"},

		// ── Marketing ─────────────────────────────────────────────────────────
		{"seo-audit", "When the user wants to audit, review, or diagnose SEO issues on their site", "marketing"},
		{"content-strategist", "Builds content engines that rank, convert, and compound through SEO and conversion", "marketing"},
		{"landing-page-generator", "Generates high-converting landing pages as complete Next.js/React components", "marketing"},
		{"ad-creative", "When the user needs to generate, iterate, or scale ad creative for paid advertising", "marketing"},
		{"programmatic-seo", "When the user wants to create SEO-driven pages at scale using templates", "marketing"},

		// ── Sales ─────────────────────────────────────────────────────────────
		{"cold-email", "When the user wants to write, improve, or build a sequence of B2B cold outreach emails", "sales"},
		{"sales-engineer", "Analyzes RFP/RFI responses for coverage gaps, builds competitive feature comparisons", "sales"},
		{"revenue-operations", "Analyzes sales pipeline health, revenue forecasting accuracy, and go-to-market efficiency", "sales"},

		// ── Product ───────────────────────────────────────────────────────────
		{"agile-product-owner", "Agile product ownership for backlog management and sprint execution", "product"},
		{"sprint-plan", "Sprint planning shortcut for goal and capacity planning", "product"},
		{"product-manager", "Ships outcomes, not features. Writes specs engineers actually read. Prioritizes ruthlessly", "product"},
		{"prd", "Quick PRD generation command for feature or problem definition", "product"},

		// ── Design ────────────────────────────────────────────────────────────
		{"figma-to-code", "Extracts design tokens, component structure, and layout from a Figma file", "design"},
		{"a11y-audit", "Accessibility audit skill for scanning, fixing, and verifying WCAG 2.2 Level A", "design"},
		{"ui-design-system", "UI design system toolkit for Senior UI Designer including design token generation", "design"},

		// ── Data ──────────────────────────────────────────────────────────────
		{"senior-data-engineer", "Data engineering skill for building scalable data pipelines, ETL/ELT systems", "data"},
		{"snowflake-development", "Use when writing Snowflake SQL, building data pipelines with Dynamic Tables", "data"},
		{"statistical-analyst", "Run hypothesis tests, analyze A/B experiment results, calculate sample sizes", "data"},
		{"data-quality-auditor", "Audit datasets for completeness, consistency, accuracy, and validity", "data"},

		// ── Finance ───────────────────────────────────────────────────────────
		{"saas-metrics-coach", "SaaS financial health advisor. Use when a user shares revenue or customer numbers", "finance"},
		{"financial-analyst", "Performs financial ratio analysis, DCF valuation, budget variance analysis", "finance"},
		{"cfo-advisor", "Financial leadership for startups and scaling companies. Financial modeling, unit economics", "finance"},
		{"saas-health", "Calculate SaaS health metrics (ARR, MRR, churn, CAC, LTV, NRR)", "finance"},

		// ── Strategy ──────────────────────────────────────────────────────────
		{"ceo-advisor", "Executive leadership guidance for strategic decision-making, organizational development", "strategy"},
		{"chief-of-staff", "C-suite orchestration layer. Routes founder questions to the right advisor role", "strategy"},
		{"competitive-intel", "Systematic competitor tracking that feeds CMO positioning and CRO battlecards", "strategy"},
		{"board-meeting", "Multi-agent board meeting protocol for strategic decisions", "strategy"},

		// ── Operations ────────────────────────────────────────────────────────
		{"runbook-generator", "Generate operational runbooks from a service name with deployment, incident response", "operations"},
		{"incident-commander", "Manage production incidents from detection through postmortem with severity classification", "operations"},
		{"postmortem", "Honest Analysis of What Went Wrong — structured post-incident review", "operations"},

		// ── Compliance ────────────────────────────────────────────────────────
		{"gdpr-dsgvo-expert", "GDPR and German DSGVO compliance automation. Scans codebases for privacy risks", "compliance"},
		{"soc2-compliance", "Use when the user asks to prepare for SOC 2 audits, map Trust Service Criteria", "compliance"},
		{"iso-13485", "ISO 13485 Quality Management System implementation for medical device companies", "compliance"},
		{"fda-consultant-specialist", "FDA regulatory consultant for medical device companies. Provides 510(k)/PMA guidance", "compliance"},

		// ── Research ──────────────────────────────────────────────────────────
		{"web-research", "Produces a structured Research Summary document with findings, sources, attributions", "research"},
		{"research-synthesis", "Produces a unified Synthesis Document organized by themes with cross-source evidence", "research"},
		{"perplexity-search", "AI-powered web search, research, and reasoning via Perplexity", "research"},

		// ── Writing ───────────────────────────────────────────────────────────
		{"content-humanizer", "Makes AI-generated content sound genuinely human — not just cleaned up", "writing"},
		{"internal-comms", "A set of resources to help write all kinds of internal communications", "writing"},
		{"documentation-audit", "Produces a committed audit log enumerating documentation drift — inaccuracies", "writing"},
		{"changelog", "Generate changelogs from git history and validate conventional commits", "writing"},

		// ── Memory ────────────────────────────────────────────────────────────
		{"memory-garden", "Writes a structured memory entry to the garden — daily note, curated seed", "memory"},
		{"compression-ritual", "Produces markdown memory artifacts — conversation summaries, seed files", "memory"},
		{"seed-extraction", "Produces a seed file — a YAML-fronted markdown document capturing a reusable pattern", "memory"},
		{"continuity-ledger", "Maintains a running decision log and context state across agent sessions", "memory"},

		// ── Skill System ──────────────────────────────────────────────────────
		{"skill-creation", "Produces a complete SKILL.md packaged as a .skill file ready for CAS distribution", "skill-system"},
		{"normalize-community-skill", "Produces an enriched SKILL.md with all six Dojo SkillRegistry.IsValid() fields", "skill-system"},
		{"skill-audit-upgrade", "Produces a graded audit report and upgraded SKILL.md files by assessing skill quality", "skill-system"},

		// ── MCP ───────────────────────────────────────────────────────────────
		{"mcp-builder", "Guide for creating high-quality MCP (Model Context Protocol) servers", "mcp"},
		{"mcp-server-builder", "Scaffolds production-ready MCP servers from OpenAPI specs with schema validation", "mcp"},
		{"fastmcp-client-cli", "Query and invoke tools on MCP servers using fastmcp list and fastmcp call", "mcp"},

		// ── People ────────────────────────────────────────────────────────────
		{"chro-advisor", "People leadership for scaling companies. Hiring strategy, compensation design", "people"},
		{"interview-system-designer", "Design interview processes, create structured evaluation rubrics for hiring", "people"},
		{"culture-architect", "Build, measure, and evolve company culture as operational behavior", "people"},

		// ── General (no clear category) ───────────────────────────────────────
		{"ask-questions-if-underspecified", "Clarify requirements before implementing. Use when serious doubts arise", "general"},
		{"behuman", "Use when the user wants more human-like AI responses — less robotic, less list-heavy", "general"},
		{"let-fate-decide", "Draws 4 Tarot cards to inject entropy into planning when prompts are vague", "general"},
	}

	for _, tc := range cases {
		got := ClusterCategory(tc.name, tc.description)
		if got != tc.want {
			t.Errorf("ClusterCategory(%q, ...) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestEnrichCategories checks that skills with empty categories get enriched
// and skills that already have categories are left untouched.
func TestEnrichCategories(t *testing.T) {
	input := []client.Skill{
		{Name: "code-review", Description: "Review code for quality, correctness", Category: ""},
		{Name: "seo-audit", Description: "Audit SEO issues on a site", Category: ""},
		{Name: "already-set", Description: "Some skill", Category: "custom"},
		{Name: "marketing-ops", Description: "Central router for the marketing skill ecosystem", Category: "general"},
	}

	got := EnrichCategories(input)

	if got[0].Category != "engineering" {
		t.Errorf("code-review: got category %q, want engineering", got[0].Category)
	}
	if got[1].Category != "marketing" {
		t.Errorf("seo-audit: got category %q, want marketing", got[1].Category)
	}
	// Pre-set non-general category must be preserved.
	if got[2].Category != "custom" {
		t.Errorf("already-set: category should be preserved, got %q", got[2].Category)
	}
	// "general" should be re-clustered.
	if got[3].Category == "general" {
		t.Errorf("marketing-ops: category should not remain 'general', got %q", got[3].Category)
	}
}

// TestCategoryNames ensures every taxonomy entry is exposed.
func TestCategoryNames(t *testing.T) {
	names := CategoryNames()
	if len(names) == 0 {
		t.Fatal("CategoryNames() returned empty slice")
	}
	// Spot-check a few expected category names.
	want := map[string]bool{
		"engineering": false, "security": false, "marketing": false,
		"agents": false, "blockchain": false, "math": false,
	}
	for _, n := range names {
		if _, ok := want[n]; ok {
			want[n] = true
		}
	}
	for cat, found := range want {
		if !found {
			t.Errorf("expected category %q in CategoryNames(), not found", cat)
		}
	}
}
