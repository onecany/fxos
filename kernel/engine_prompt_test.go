package kernel

import (
	"strings"
	"testing"

	"fxos/store"
	"fxos/trader/types"
)

func TestBuildSystemPromptUsesVergexClaw402Prompt(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "vergex_signal"
	cfg.CoinSource.VergexLimit = 5
	cfg.PromptSections.RoleDefinition = "# You are a professional Hyperliquid USDC multi-asset trading AI"
	cfg.CustomPrompt = "Long only, no shorts."

	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(30, "balanced")

	// An operator-edited role_definition REPLACES the built-in Claw402 role
	// instead of stacking a second conflicting identity — the model must see
	// exactly one "you are" statement. The trading-universe boundary survives
	// as an explicit system constraint.
	if strings.Contains(prompt, "FXOS Claw402 auto-trader") {
		t.Fatalf("prompt must not contain the built-in Claw402 role when role_definition is edited:\n%s", prompt)
	}
	if !strings.Contains(prompt, "System constraint (never override): trade only the Hyperliquid instruments returned by this cycle's Claw402.ai/Vergex board") {
		t.Fatalf("prompt should keep the trading-universe system constraint after a custom role:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Claw402.ai Signal Ranking") || !strings.Contains(prompt, "Signal Lab") || !strings.Contains(prompt, "Cost/Liquidation Heatmap") {
		t.Fatalf("prompt is missing Claw402/Vergex detail data guidance:\n%s", prompt)
	}
	if !strings.Contains(prompt, types.ActionOpenShort) {
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

// TestVergexEditedRoleReplacesBuiltInRole locks the fix for the role
// overlap: when the operator edits role_definition, the built-in Claw402
// role must NOT appear anywhere (two conflicting "you are" statements
// corrupt the model's judgment). The trading-universe boundary is a system
// safety invariant and must survive the replacement as an explicit
// non-overridable constraint.
func TestVergexEditedRoleReplacesBuiltInRole(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "vergex_signal"
	customRole := "# 你是只做高置信度突破的趋势交易员"
	cfg.PromptSections.RoleDefinition = customRole
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")

	if strings.Contains(prompt, "FXOS Claw402 auto-trader") {
		t.Fatalf("built-in role must be replaced by edited role, not duplicated:\n%s", prompt)
	}
	if !strings.Contains(prompt, customRole) {
		t.Fatalf("edited role definition missing from prompt:\n%s", prompt)
	}
	if !strings.Contains(prompt, "# Role Definition\n\n"+customRole) {
		t.Fatalf("edited role must render under the Role Definition header:\n%s", prompt)
	}
	if !strings.Contains(prompt, "System constraint (never override): trade only the Hyperliquid instruments returned by this cycle's Claw402.ai/Vergex board") {
		t.Fatalf("trading-universe system constraint must survive a custom role:\n%s", prompt)
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
		types.ActionOpenShort,
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

// TestBalancedVariantWritesModeBlock locks the fix for the missing balanced
// mode: production callers pass "balanced" as the default variant, and the
// prompt MUST include a mode description so the model knows its posture.
func TestBalancedVariantWritesModeBlock(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.CoinSource.SourceType = "static"
	cfg.CoinSource.StaticCoins = []string{"BTCUSDT"}
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")
	if !strings.Contains(prompt, "## Mode: Balanced") {
		t.Fatalf("balanced variant must produce a Mode: Balanced block:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Open only when multiple signals resonate across timeframes") {
		t.Fatalf("balanced mode must include signal resonance guidance:\n%s", prompt)
	}
}

// TestVergexPathHasTotalRiskBudget locks the new Total Risk Budget section
// in the vergex prompt path: multi-position risk allocation was previously
// only surfaced in the generic path, leaving vergex traders without explicit
// proportional-sizing guidance.
func TestVergexPathHasTotalRiskBudget(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.CoinSource.SourceType = "vergex_signal"
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")
	if !strings.Contains(prompt, "## Total Risk Budget") {
		t.Fatalf("vergex prompt must include Total Risk Budget section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "reduce each position's size proportionally") {
		t.Fatalf("vergex prompt must include proportional sizing rule:\n%s", prompt)
	}
}

// TestVergexPathHasFundingRateCrowding locks the funding-rate crowding
// interpretation section in the vergex path — it uses a compact schema
// without the full Data Dictionary, so the crowding signal must be
// injected explicitly.
func TestVergexPathHasFundingRateCrowding(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.CoinSource.SourceType = "vergex_signal"
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")
	if !strings.Contains(prompt, "## Funding Rate Crowding") {
		t.Fatalf("vergex prompt must include Funding Rate Crowding section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "FR at extreme levels") {
		t.Fatalf("vergex funding rate section must cover extreme crowding:\n%s", prompt)
	}
}

// TestSharedDisciplineAppearsInBothPaths locks the refactored
// writeCommonDiscipline: rules that were previously duplicated (and
// diverging) across the generic and vergex paths must now appear in
// both and include the previously-missing canonical text.
func TestSharedDisciplineAppearsInBothPaths(t *testing.T) {
	// Generic path
	cfgGen := store.GetDefaultStrategyConfig("en")
	cfgGen.CoinSource.SourceType = "static"
	cfgGen.CoinSource.StaticCoins = []string{"BTCUSDT"}
	engineGen := NewStrategyEngine(&cfgGen)
	promptGen := engineGen.BuildSystemPrompt(1000, "balanced")

	// Vergex path
	cfgVx := store.GetDefaultStrategyConfig("en")
	cfgVx.CoinSource.SourceType = "vergex_signal"
	engineVx := NewStrategyEngine(&cfgVx)
	promptVx := engineVx.BuildSystemPrompt(1000, "balanced")

	// Rules that the vergex path was previously missing (divergence):
	shared := []string{
		"Confusing realized and unrealized PnL",
		"Ignoring OI changes",
		"Never hold a losing position overnight",
		"weekend gap risk",
	}
	for _, phrase := range shared {
		if !strings.Contains(promptVx, phrase) {
			t.Fatalf("vergex path is STILL missing shared-discipline rule %q after refactor:\n%s", phrase, promptVx)
		}
		if !strings.Contains(promptGen, phrase) {
			t.Fatalf("generic path lost shared-discipline rule %q after refactor:\n%s", phrase, promptGen)
		}
	}
}

// TestVergexPromptNoLongerMissingOvernightRule is a targeted regression
// test: before the shared-discipline refactor, the vergex Time Stop
// section was missing the overnight-hold prohibition. After the refactor,
// both paths share identical Time Stop text including that rule.
func TestVergexPromptNoLongerMissingOvernightRule(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.CoinSource.SourceType = "vergex_signal"
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")

	// These were absent from the vergex path before the refactor.
	if !strings.Contains(prompt, "Never hold a losing position overnight") {
		t.Fatalf("vergex path should now include overnight-hold rule:\n%s", prompt)
	}
	// Vergex Anti-Patterns was missing flips, PnL confusion, OI, leverage warnings.
	if !strings.Contains(prompt, "Flipping from long to short") {
		t.Fatalf("vergex path should include direction-flip anti-pattern:\n%s", prompt)
	}
}

// TestLintPromptCleanDefault verifies that the built prompt from default
// config passes lint with zero issues.
func TestLintPromptCleanDefault(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("en")
	cfg.CoinSource.SourceType = "vergex_signal"
	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(1000, "balanced")
	issues := LintPrompt(prompt)
	if len(issues) > 0 {
		t.Fatalf("default vergex prompt should be lint-clean:\n  %s", strings.Join(issues, "\n  "))
	}
}

// TestLintPromptDetectsDuplicatedRole verifies the lint catches a duplicated
// Claw402 role statement.
func TestLintPromptDetectsDuplicatedRole(t *testing.T) {
	prompt := "---\n# You are the FXOS Claw402 auto-trader\n\nstuff\n\n# You are the FXOS Claw402 auto-trader\n\nmore\n"
	issues := LintPrompt(prompt)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "Claw402 role") {
			found = true
		}
	}
	if !found {
		t.Fatalf("lint should detect duplicated Claw402 role, got: %v", issues)
	}
}

// TestLintPromptFlagsZeroRisk verifies the lint catches risk_usd: 0.
func TestLintPromptFlagsZeroRisk(t *testing.T) {
	prompt := `---\n## Anti-Patterns (DO NOT)\n\nstuff\n## Time Stop\n\nstuff\n## Order Handling\n\nstuff\n## Position Management Rules\n\n{"risk_usd": 0}\n`
	issues := LintPrompt(prompt)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "risk_usd") {
			found = true
		}
	}
	if !found {
		t.Fatalf("lint should flag risk_usd: 0, got: %v", issues)
	}
}
