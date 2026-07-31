package kernel

import (
	"strings"
	"testing"

	"fxos/store"
)

func TestBuildSystemPromptUsesVergexClaw402Prompt(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "vergex_signal"
	cfg.CoinSource.VergexLimit = 5
	cfg.PromptSections.RoleDefinition = "# You are a professional Hyperliquid USDC multi-asset trading AI"
	cfg.CustomPrompt = "Long only, no shorts."

	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(30, "balanced")

	if !strings.Contains(prompt, "FXOS Claw402 auto-trader") {
		t.Fatalf("prompt did not use the Claw402/Vergex TradeFi role:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Claw402.ai Signal Ranking") || !strings.Contains(prompt, "Signal Lab") || !strings.Contains(prompt, "Cost/Liquidation Heatmap") {
		t.Fatalf("prompt is missing Claw402/Vergex detail data guidance:\n%s", prompt)
	}
	if !strings.Contains(prompt, "open_short") {
		t.Fatalf("prompt should explicitly allow short entries:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Direction must be data-driven") {
		t.Fatalf("prompt should explain that direction is data-driven, not long-only:\n%s", prompt)
	}
	if !strings.Contains(prompt, "use up to 20x") {
		t.Fatalf("prompt should allow up to 20x leverage for Claw402 opens:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Position Sizing") || !strings.Contains(prompt, "confidence") {
		t.Fatalf("prompt should have confidence-based position sizing for Claw402:\n%s", prompt)
	}
	// The user-edited RoleDefinition is passed through verbatim (Prompt Studio
	// contract: explicit user sections reach the model, even in Chinese UI
	// configurations). The built-in sections must stay English-only.
	if !strings.Contains(prompt, "# Role Definition\n\n# You are a professional Hyperliquid USDC multi-asset trading AI") {
		t.Fatalf("prompt should include the user-edited role definition verbatim:\n%s", prompt)
	}
	// Legacy directional overrides in the custom prompt are still filtered.
	if strings.Contains(prompt, "Long only") {
		t.Fatalf("prompt should drop legacy 'Long only' directive from custom prompt:\n%s", prompt)
	}
	legacyPhrases := []string{
		"Altcoin",
		"BTC/ETH",
		"LONG-ONLY",
		"Do not short",
		"MUST open a long",
	}
	for _, phrase := range legacyPhrases {
		if strings.Contains(prompt, phrase) {
			t.Fatalf("prompt still contains legacy phrase %q:\n%s", phrase, prompt)
		}
	}
}

func TestBuildSystemPromptPassesThroughUserChineseSections(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.CoinSource.SourceType = "static"
	cfg.CoinSource.StaticCoins = []string{"BTCUSDT", "ETHUSDT"}
	cfg.CoinSource.VergexLimit = 0
	cfg.CoinSource.VergexMarketType = ""
	cfg.CoinSource.VergexChain = ""
	cfg.PromptSections.RoleDefinition = "# 你是一个专业的加密货币交易AI"
	cfg.PromptSections.TradingFrequency = "# High-frequency trading\nTrade every minute."
	cfg.PromptSections.EntryStandards = "# Entry\nOpen positions freely."
	cfg.PromptSections.DecisionProcess = "# Decision\nOutput directly."
	cfg.CustomPrompt = "中文偏好应当进入系统提示词"

	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(30, "balanced")

	// User-edited sections pass through verbatim, Chinese included.
	required := []string{
		"# 你是一个专业的加密货币交易AI",
		"# High-frequency trading",
		"# Entry",
		"# Decision",
		"中文偏好应当进入系统提示词",
	}
	for _, phrase := range required {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("prompt missing user-edited section %q:\n%s", phrase, prompt)
		}
	}
	// Built-in non-editable sections (schema, risk, anti-patterns) still render
	// in English — only the four editable sections are replaced by user input.
	for _, phrase := range []string{
		"Data Dictionary & Trading Rules",
		"Hard Constraints (Risk Control)",
		"Anti-Patterns",
	} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("English built-in prompt missing %q:\n%s", phrase, prompt)
		}
	}
}

// TestDefaultVergexPromptDoesNotDuplicateBuiltInSections locks the vergex
// prompt path: the default config ships English Claw402 copy in
// PromptSections, which is identical to the built-in vergex sections. Only
// operator-edited sections may be injected — unchanged defaults must stay
// silent or the prompt duplicates its own role definition.
func TestDefaultVergexPromptDoesNotDuplicateBuiltInSections(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "vergex_signal"
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")
	if count := strings.Count(prompt, "FXOS Claw402 auto-trader"); count != 1 {
		t.Fatalf("built-in role must appear exactly once, got %d:\n%s", count, prompt)
	}
	if strings.Contains(prompt, "# Role Definition") {
		t.Fatalf("unchanged default sections must not be injected into vergex prompt:\n%s", prompt)
	}
}

// TestVergexEditedSectionsStillInjected locks the Prompt Studio contract for
// the vergex path: sections the operator actually changed (Chinese included)
// must reach the model even though unchanged defaults are suppressed.
func TestVergexEditedSectionsStillInjected(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "vergex_signal"
	cfg.PromptSections.RoleDefinition = "# 自定义角色:严格趋势交易者"
	cfg.PromptSections.EntryStandards = "只交易清晰趋势"
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")
	for _, phrase := range []string{"自定义角色:严格趋势交易者", "只交易清晰趋势"} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("edited section %q missing from vergex prompt:\n%s", phrase, prompt)
		}
	}
	if strings.Contains(prompt, "# Role Definition\n\n# You are the FXOS Claw402 auto-trader") {
		t.Fatalf("default role copy must not be injected for edited config:\n%s", prompt)
	}
}

// TestVergexDedupToleratesLegacyDefaultCopy locks the normalization guard:
// older strategies may store near-identical default copy with a trailing '!'
// (e.g. "# ... auto-trader!" vs the current "# ... auto-trader"). Such copy is
// NOT an operator edit and must not be injected a second time.
func TestVergexDedupToleratesLegacyDefaultCopy(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "vergex_signal"
	// Simulate the legacy stored role: default copy plus a trailing '!'.
	cfg.PromptSections.RoleDefinition = cfg.PromptSections.RoleDefinition + "!"
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")
	if count := strings.Count(prompt, "FXOS Claw402 auto-trader"); count != 1 {
		t.Fatalf("legacy default copy must not be injected twice, got %d:\n%s", count, prompt)
	}
	if strings.Contains(prompt, "# Role Definition") {
		t.Fatalf("legacy default copy must not be injected as user section:\n%s", prompt)
	}
}

func TestBuildSystemPromptDoesNotForceLongOnlyForSingleXYZ(t *testing.T) {
	prompt := buildXYZStockCustomPrompt("XYZ:INTC")

	required := []string{
		"DIRECTIONAL, SIGNAL-DRIVEN",
		"You may open long or short",
		"open_short",
	}
	for _, phrase := range required {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("single XYZ prompt missing %q:\n%s", phrase, prompt)
		}
	}

	forbidden := []string{
		"LONG-ONLY",
		"Do not short",
		"MUST open a long",
		"Probing > waiting",
	}
	for _, phrase := range forbidden {
		if strings.Contains(prompt, phrase) {
			t.Fatalf("single XYZ prompt still contains forced-long phrase %q:\n%s", phrase, prompt)
		}
	}
}

func containsCJK(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
