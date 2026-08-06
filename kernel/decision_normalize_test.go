package kernel

import (
	"strings"
	"testing"

	"fxos/store"
)

// TestExtractDecisionsNormalizesPlaceholderSymbol locks the defensive
// normalization: a model that copies the prompt's "..." wait example
// verbatim (or emits an empty symbol) must be coerced to an explicit
// full-portfolio wait — never passed through as a fake ticker.
func TestExtractDecisionsNormalizesPlaceholderSymbol(t *testing.T) {
	raw := `<decision>` + "\n```json\n" +
		`[
  {"symbol": "...", "action": "wait"},
  {"symbol": "BTCUSDT", "action": "wait"}
]` + "\n```\n</decision>"

	decisions, err := extractDecisions(raw)
	if err != nil {
		t.Fatalf("extractDecisions: %v", err)
	}
	if len(decisions) != 2 {
		t.Fatalf("got %d decisions, want 2", len(decisions))
	}
	if decisions[0].Symbol != "ALL" {
		t.Errorf("placeholder '...' symbol must normalize to ALL, got %q", decisions[0].Symbol)
	}
	if decisions[1].Symbol != "BTCUSDT" {
		t.Errorf("real symbol must pass through unchanged, got %q", decisions[1].Symbol)
	}
}

// TestExtractDecisionsNormalizesEmptySymbol covers the empty-symbol variant.
func TestExtractDecisionsNormalizesEmptySymbol(t *testing.T) {
	raw := `<decision>` + "\n```json\n" + `[{"symbol": "", "action": "wait"}]` + "\n```\n</decision>"
	decisions, err := extractDecisions(raw)
	if err != nil {
		t.Fatalf("extractDecisions: %v", err)
	}
	if len(decisions) != 1 || decisions[0].Symbol != "ALL" {
		t.Fatalf("empty symbol must normalize to ALL, got %+v", decisions)
	}
}

// TestWaitInstructionHasNoPlaceholder locks the prompt fix: the
// "nothing qualifies" guidance must not show a copyable "..." placeholder
// that models echo back as a literal symbol. It should instruct a real
// candidate symbol with wait action.
func TestWaitInstructionHasNoPlaceholder(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.CoinSource.SourceType = "static"
	cfg.CoinSource.StaticCoins = []string{"BTCUSDT"}
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")

	if strings.Contains(prompt, `{"symbol": "..."`) {
		t.Fatalf("prompt must not contain the copyable '...' placeholder:\n%s", prompt)
	}
	if !strings.Contains(prompt, "do NOT invent a placeholder symbol") {
		t.Fatalf("prompt must forbid placeholder symbols:\n%s", prompt)
	}
}
