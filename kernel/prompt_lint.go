package kernel

import (
	"fmt"
	"strings"
)

// LintPrompt runs structural checks on a generated system prompt and returns
// human-readable issues. It does NOT validate trading logic or strategy
// correctness — only prompt-engineering hygiene (duplicate headings,
// missing sections, common anti-patterns).
//
// Returns nil if no issues found.
func LintPrompt(prompt string) []string {
	var issues []string

	// --- Duplicate heading detection ---
	headings := []string{
		"## Anti-Patterns (DO NOT)",
		"## Time Stop",
		"## Order Handling",
		"## Position Management",
		"## Total Risk Budget",
		"## Funding Rate Crowding",
		"## Multi-Timeframe Analysis",
		"## Position Sizing by Market Regime",
		"## Confidence Calibration",
		"## Hard Risk Constraints",
		"# Hard Constraints (Risk Control)",
		"# Output Format (Strictly Follow)",
		"# Role Definition",
	}
	for _, h := range headings {
		count := strings.Count(prompt, h)
		if count > 1 {
			issues = append(issues, fmt.Sprintf("Duplicate heading %q appears %d times", h, count))
		}
	}

	// --- Duplicate identity (two role statements) ---
	if strings.Count(prompt, "FXOS Claw402 auto-trader") > 1 {
		issues = append(issues, "Built-in Claw402 role appears more than once — potential duplicate identity")
	}

	// --- Missing shared discipline headings ---
	for _, h := range []string{"## Anti-Patterns (DO NOT)", "## Time Stop", "## Order Handling", "## Position Management Rules"} {
		if !strings.Contains(prompt, h) {
			issues = append(issues, fmt.Sprintf("Missing shared-discipline heading %q", h))
		}
	}

	// --- Zero risk example contaminant ---
	if strings.Contains(prompt, `"risk_usd": 0`) || strings.Contains(prompt, `"risk_usd":0`) {
		issues = append(issues, "Prompt contains risk_usd: 0 example — may induce zero-risk outputs")
	}

	// --- Hardcoded price anchors (example values far from rational) ---
	// Only flag if the example comment is also missing (i.e. truly unguarded).
	if (strings.Contains(prompt, `"stop_loss": 97000`) || strings.Contains(prompt, `"take_profit": 103000`)) &&
		!strings.Contains(prompt, "FORMAT ILLUSTRATIONS only") {
		issues = append(issues, "Hardcoded example prices present without format-illustration disclaimer")
	}

	// --- Missing mode description ---
	// "balanced" is the production default; absence of any Mode: line = silent default.
	if !strings.Contains(prompt, "## Mode:") {
		issues = append(issues, "No mode description found — balanced/default variant is silent (add '## Mode: Balanced')")
	}

	return issues
}
