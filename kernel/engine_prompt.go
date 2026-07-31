package kernel

import (
	"fmt"
	"fxos/market"
	"fxos/provider/nofx"
	"fxos/provider/vergex"
	"fxos/store"
	"strings"
	"time"
)

// ============================================================================
// Prompt Building - System Prompt
// ============================================================================

// BuildSystemPrompt builds System Prompt according to strategy configuration
func (e *StrategyEngine) BuildSystemPrompt(accountEquity float64, variant string) string {
	var sb strings.Builder
	riskControl := e.config.RiskControl
	promptSections := e.config.PromptSections
	// System prompts are intentionally English-only. UI copy can be localized,
	// but the model contract should stay language-stable for an international
	// open-source project and for reproducible trading behavior.
	lang := LangEnglish
	singleSymbol, primarySymbol := e.singleSymbolInfo()

	if e.usesVergexSignalPrompt() {
		return e.buildVergexSystemPrompt(accountEquity, variant, lang, singleSymbol, primarySymbol)
	}

	// 0. Data Dictionary & Schema (ensure AI understands all fields)
	sb.WriteString(GetSchemaPrompt(lang))
	sb.WriteString("\n\n")
	sb.WriteString("---\n\n")

	// 1. Role definition (editable; falls back to a generic intro).
	roleDefinition := userPromptSection(promptSections.RoleDefinition)
	if roleDefinition != "" {
		sb.WriteString(roleDefinition)
		sb.WriteString("\n\n")
	} else {
		if e.usesHyperliquidNativeUniverse() {
			sb.WriteString("# You are a professional Hyperliquid USDC multi-asset trading AI\n\n")
			sb.WriteString("Your task is to make trading decisions based on the provided market data.\n\n")
		} else {
			sb.WriteString("# You are a professional crypto perpetual futures trading AI\n\n")
			sb.WriteString("Your task is to make trading decisions based on the provided market data.\n\n")
		}
	}

	// 2. Trading mode variant
	writeModeVariant(&sb, variant)

	// 3. Hard constraints (risk control).
	//
	// `singleSymbol` is true for strategies that deliberately trade just one
	// instrument (the quick-create flow, single-asset templates). For those,
	// the "BTC/ETH vs Altcoin" two-tier categorization is irrelevant and
	// actively misleading — we surface a single position-value limit instead.
	btcEthPosValueRatio := riskControl.BTCETHMaxPositionValueRatio
	if btcEthPosValueRatio <= 0 {
		btcEthPosValueRatio = 5.0
	}
	altcoinPosValueRatio := riskControl.AltcoinMaxPositionValueRatio
	if altcoinPosValueRatio <= 0 {
		altcoinPosValueRatio = 1.0
	}

	writeHardConstraints(&sb, accountEquity, riskControl, btcEthPosValueRatio, altcoinPosValueRatio, singleSymbol, primarySymbol)

	// 3b. Correlation risk (single-symbol strategies skip this)
	if !singleSymbol {
		sb.WriteString("## Correlation Risk\n\n")
		sb.WriteString("- BTC and ETH are highly correlated (~0.85). Never hold same-direction positions in both simultaneously.\n")
		sb.WriteString("- If you want exposure to both, treat them as one position and split the notional.\n")
		sb.WriteString("- Other high-correlation pairs: SOL/AVAX, DOGE/SHIB. Check correlation before opening a second position.\n")
		sb.WriteString("- Opening too many positions in the same cycle = overexposure. See Anti-Patterns for the limit. Prefer 1-2 high-conviction trades.\n\n")
	}

	// 3c. Total risk budget
	sb.WriteString("## Total Risk Budget\n\n")
	sb.WriteString(fmt.Sprintf("- Maximum total margin usage across ALL positions: ≤%.0f%% (backend enforced)\n", riskControl.MaxMarginUsage*100))
	sb.WriteString("- When multiple positions are open, reduce each position's size proportionally.\n")
	sb.WriteString("- Example: 3 positions open → each uses ≤33%% of available margin.\n")
	sb.WriteString("- If total margin is already near the limit, only accept trades with confidence ≥ 85.\n\n")

	// 4. Trading frequency (editable)
	tradingFrequency := userPromptSection(promptSections.TradingFrequency)
	if tradingFrequency != "" {
		sb.WriteString(tradingFrequency)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# ⏱️ Trading Frequency Awareness\n\n")
		sb.WriteString("- Excellent traders: 2-4 trades/day ≈ 0.1-0.2 trades/hour\n")
		sb.WriteString("- >2 trades/hour = overtrading\n")
		sb.WriteString("- Single position hold time ≥ 45-90 minutes\n")
		sb.WriteString("If you find yourself trading every cycle → standards too low; if closing positions < 45 minutes → too impulsive.\n\n")
	}

	// 5. Entry standards (editable)
	entryStandards := userPromptSection(promptSections.EntryStandards)
	if entryStandards != "" {
		sb.WriteString(entryStandards)
		sb.WriteString("\n\nYou have the following indicator data:\n")
		e.writeAvailableIndicators(&sb)
		sb.WriteString(fmt.Sprintf("\n**Confidence ≥ %d** required to open positions.\n\n", riskControl.MinConfidence))
	} else {
		sb.WriteString("# 🎯 Entry Standards (Strict)\n\n")
		sb.WriteString("Only open positions when multiple signals resonate. You have:\n")
		e.writeAvailableIndicators(&sb)
		sb.WriteString(fmt.Sprintf("\nFeel free to use any effective analysis method, but **confidence ≥ %d** is required to open positions; avoid low-quality behaviors such as single-indicator entries, contradictory signals, sideways chop, or re-entering immediately after a close.\n\n", riskControl.MinConfidence))
	}

	// 6. Decision process (editable)
	decisionProcess := userPromptSection(promptSections.DecisionProcess)
	if decisionProcess != "" {
		sb.WriteString(decisionProcess)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# 📋 Decision Process\n\n")
		sb.WriteString("1. **REGIME**: Use the pre-computed Regime hint from the user prompt. If it says \"unknown\", classify yourself using the criteria below.\n")
		sb.WriteString("   - Trending: price above EMA20 > EMA50, ADX > 25, breakout patterns\n")
		sb.WriteString("   - Ranging: Bollinger width < 3%, EMA20 ≈ EMA50, price oscillating around mean\n")
		sb.WriteString("   - High-vol: ATR > 1.5x 20-period average, large candles, rapid moves\n")
		sb.WriteString("   - Low-vol: ATR < 0.7x 20-period average, tight ranges, compression\n")
		sb.WriteString("2. **RISK CHECK**: Total margin usage vs limit? Room to add risk?\n")
		sb.WriteString("3. **POSITION MANAGEMENT**: Any open positions need stop/take-profit adjustment?\n")
		sb.WriteString("4. **CANDIDATE SCAN**: Which symbols show the strongest setup for THIS regime?\n")
		sb.WriteString("5. **CONFLUENCE**: Do multiple timeframes + indicators agree?\n")
		sb.WriteString("6. **R/R CHECK**: Expected move to target ≥ 3x the risk cost (fees + stop distance)?\n")
		sb.WriteString("7. **DECISION**: Output JSON\n")
		sb.WriteString("- If all margin is used or Max Drawdown > 20%, skip to step 3 (POSITION MANAGEMENT) and output hold/wait.\n\n")
	}

	// 6b. Trading discipline — shared logic across all prompt paths.
	// Anti-Patterns, Time Stop, Order Handling, Position Management.
	// Canonical text lives in writeCommonDiscipline so both the generic
	// and vergex paths present identical constraints to the model.
	writeCommonDiscipline(&sb, riskControl)

	// 6c. Confidence calibration guidance
	sb.WriteString("## Confidence Calibration\n\n")
	sb.WriteString(fmt.Sprintf("- 90-100: All signals align (trend + momentum + volume + catalyst), high-conviction setup\n"))
	sb.WriteString(fmt.Sprintf("- %d-89: Most signals agree, 1-2 minor conflicts, solid setup\n", riskControl.MinConfidence))
	sb.WriteString("- 60-77: Mixed signals, wait for more confirmation\n")
	sb.WriteString("- Below 60: No edge detected, output `wait`\n\n")

	// 6e. Quick Reference —浓缩关键规则防 rule fatigue
	sb.WriteString("## Quick Reference (remember these)\n\n")
	sb.WriteString(fmt.Sprintf("Max positions: %d | Margin limit: %.0f%% | Min confidence: %d | SL/TP required for opens | No revenge trading | No flipping same cycle\n\n", riskControl.MaxPositions, riskControl.MaxMarginUsage*100, riskControl.MinConfidence))

	// 7. Output format — schema spec stays in English (this is a parser
	//    contract; reasoning copy is localized below).
	writeOutputFormat(&sb, accountEquity, btcEthPosValueRatio, riskControl, singleSymbol, primarySymbol)

	// 8. Custom Prompt.
	//
	// For single-symbol Hyperliquid XYZ assets (US equities, commodities,
	// forex), we replace any stored CustomPrompt with a built-in English
	// stock-trader template. This serves two purposes:
	//   1. The auto-generated CustomPrompt from the quick-create flow used
	//      to be Chinese (matching UI language), which produced an
	//      incoherent mixed-language final prompt that confused the LLM.
	//   2. It guarantees a stock-specific, US-equity-tuned briefing
	//      regardless of when the strategy was first created.
	customPrompt := userPromptSection(e.config.CustomPrompt)
	if singleSymbol && market.IsXyzDexAsset(primarySymbol) {
		customPrompt = buildXYZStockCustomPrompt(primarySymbol)
	}

	if customPrompt != "" {
		sb.WriteString("# 📌 Personalized Trading Strategy\n\n")
		sb.WriteString(customPrompt)
		sb.WriteString("\n\n")
		sb.WriteString("Note: the above personalized strategy supplements the basic rules but must not override core risk controls (max positions, margin limits, stop-loss requirements).\n")
	}

	return sb.String()
}

func (e *StrategyEngine) usesVergexSignalPrompt() bool {
	if e == nil || e.config == nil {
		return false
	}
	coinSource := e.config.CoinSource
	sourceType := strings.ToLower(strings.TrimSpace(coinSource.SourceType))
	return sourceType == "vergex_signal" ||
		sourceType == "claw402" ||
		sourceType == "claw402_vergex" ||
		coinSource.VergexMarketType != "" ||
		coinSource.VergexChain != "" ||
		coinSource.VergexLimit > 0
}

func (e *StrategyEngine) buildVergexSystemPrompt(accountEquity float64, variant string, lang Language, singleSymbol bool, primarySymbol string) string {
	var sb strings.Builder
	riskControl := e.config.RiskControl

	writeVergexSchemaPrompt(&sb)
	sb.WriteString("\n\n---\n\n")

	// Role identity: the built-in Claw402 role, OR the operator's edited
	// role_definition — never both. Two conflicting "you are" statements
	// (built-in auto-trader + custom role) make the model contradict itself.
	// The trading-universe boundary is a system safety invariant, so it is
	// re-stated after any custom role instead of being silently replaced.
	sections := e.config.PromptSections
	defaultSections := store.GetDefaultStrategyConfig(e.config.Language).PromptSections
	normalizeForDedup := func(s string) string {
		s = strings.ToLower(s)
		s = strings.Map(func(r rune) rune {
			switch {
			case r >= 'a' && r <= 'z':
				return r
			case r >= '0' && r <= '9':
				return r
			default:
				return ' '
			}
		}, s)
		return strings.Join(strings.Fields(s), " ")
	}
	roleUnedited := func(body string) bool {
		body = strings.TrimSpace(body)
		return body == "" || normalizeForDedup(body) == normalizeForDedup(defaultSections.RoleDefinition)
	}
	if roleUnedited(sections.RoleDefinition) {
		sb.WriteString("# You are the FXOS Claw402 auto-trader\n\n")
		sb.WriteString("Trade only Hyperliquid instruments returned by this cycle's Claw402.ai/Vergex board. You may trade only the current candidate symbols and existing positions; never invent tickers or rotate outside the provided universe.\n\n")
	} else {
		sb.WriteString("# Role Definition\n\n")
		sb.WriteString(strings.TrimSpace(sections.RoleDefinition))
		sb.WriteString("\n\n")
		sb.WriteString("System constraint (never override): trade only the Hyperliquid instruments returned by this cycle's Claw402.ai/Vergex board and existing positions; never invent tickers or rotate outside the provided universe.\n\n")
	}

	sb.WriteString("# Decision Data Priority\n\n")
	sb.WriteString("1. Claw402.ai Signal Ranking: candidate pool, rank, direction and category.\n")
	sb.WriteString("2. Claw402.ai Signal Lab: trend, momentum, event/model confirmation; this is the core pre-entry confirmation source.\n")
	sb.WriteString("3. Claw402.ai Cost/Liquidation Heatmap: crowded liquidation/cost zones, stop placement and target zones.\n")
	sb.WriteString("4. Raw OHLCV candles: entry timing, trend structure, volatility and risk/reward validation.\n\n")
	sb.WriteString("# Trading Rules\n\n")
	sb.WriteString("- Manage existing positions before opening new ones.\n")
	sb.WriteString("- Open only when Signal Lab, heatmap and raw candles broadly agree; wait when key data is missing or contradictory.\n")
	sb.WriteString("- Ranking alone is not an entry reason; it only defines the candidate pool.\n")
	sb.WriteString("- Every symbol in Candidate Coins is part of the allowed trading universe; missing detail can lower confidence or trigger waiting, but does not make the symbol non-tradable.\n")
	sb.WriteString("- If Signal Lab or heatmap is absent from that symbol's Vergex Claw402 Signals, state it in reasoning; if it is present, never claim the symbol lacks that data.\n")
	sb.WriteString("- Avoid churn: unless stopping out or taking a strong profit, hold new positions for at least 60 minutes; avoid flat/noise closes until roughly 90 minutes; after closing a symbol, wait 90 minutes before re-entry; open at most 1 new position per hour.\n")
	sb.WriteString("- Fees are the main edge killer: a round trip costs roughly 0.1%% of notional (about 1%% of margin at 10x). Only take setups whose expected move to target is at least 3x that cost; fewer, higher-conviction, longer-hold trades beat frequent scalps.\n")
	sb.WriteString("- Stops must sit beyond invalidation; targets should prefer heatmap resistance/liquidation zones or valid risk/reward levels.\n")
	sb.WriteString("- Correlation: avoid same-direction positions in highly correlated instruments (e.g. BTC+ETH). If multiple candidates are correlated, pick the strongest one.\n\n")

	// Funding rate crowding signal — vergex path needs this explicitly
	// because it uses a compact schema without the full Data Dictionary.
	// Crowding is a high-signal entry/exit filter for perps: when the
	// crowd piles onto one side, the opposite trade's quality rises.
	sb.WriteString("## Funding Rate Crowding\n\n")
	sb.WriteString("- FR significantly above its own recent average → that side is crowded. Contrarian (opposite-direction) setups become higher quality.\n")
	sb.WriteString("- FR significantly below its own recent average → opposite side crowded. Trend continuation setups higher quality.\n")
	sb.WriteString("- FR near zero or at its average → balanced positioning. No crowding signal; rely on other indicators.\n")
	sb.WriteString("- FR at extreme levels (>5× recent average) → highly crowded. Expect mean reversion or squeeze. High risk of sudden reversal.\n\n")

	// Total Risk Budget — multi-position capital allocation guardrails.
	sb.WriteString("## Total Risk Budget\n\n")
	sb.WriteString(fmt.Sprintf("- Maximum total margin usage across ALL positions: ≤%.0f%% (backend enforced)\n", riskControl.MaxMarginUsage*100))
	sb.WriteString("- When multiple positions are open, reduce each position's size proportionally.\n")
	sb.WriteString(fmt.Sprintf("- Example: %d positions open → each uses ≤%.0f%% of available margin.\n", riskControl.MaxPositions, 100.0/float64(riskControl.MaxPositions)))
	sb.WriteString("- If total margin is already near the limit, only accept trades with confidence ≥ 85.\n")
	sb.WriteString("- Multi-position risk: never allocate >50%% of available margin to a single symbol when multiple candidates are tradeable.\n\n")

	// Trading discipline — shared canonical rules, same text as
	// the generic prompt path (writeCommonDiscipline).
	writeCommonDiscipline(&sb, riskControl)

	writeModeVariant(&sb, variant)

	altcoinPosValueRatio := riskControl.AltcoinMaxPositionValueRatio
	if altcoinPosValueRatio <= 0 {
		altcoinPosValueRatio = 1.0
	}
	writeVergexHardConstraints(&sb, accountEquity, riskControl, altcoinPosValueRatio)
	writeVergexOutputFormat(&sb, accountEquity, riskControl, altcoinPosValueRatio, singleSymbol, primarySymbol)

	// User-edited prompt sections (Prompt Studio). Only sections the operator
	// actually changed are injected — the default vergex config ships the same
	// English Claw402 copy in PromptSections, so injecting it verbatim would
	// duplicate the built-in sections above. Compare against the language
	// default with punctuation/whitespace/case normalization: older stored
	// strategies may carry near-identical default copy with a trailing '!' or
	// different line breaks (e.g. "# ... auto-trader!" vs "# ... auto-trader"),
	// which must not count as an operator edit. RoleDefinition is handled at
	// the top of the prompt (built-in role OR edited role, never both), so it
	// is deliberately skipped here.
	writeUserSection := func(title, body, defaultBody string) {
		body = strings.TrimSpace(body)
		if body == "" || normalizeForDedup(body) == normalizeForDedup(defaultBody) {
			return
		}
		sb.WriteString("# " + title + "\n\n")
		sb.WriteString(body)
		sb.WriteString("\n\n")
	}
	writeUserSection("Trading Frequency", sections.TradingFrequency, defaultSections.TradingFrequency)
	writeUserSection("Entry Standards", sections.EntryStandards, defaultSections.EntryStandards)
	writeUserSection("Decision Process", sections.DecisionProcess, defaultSections.DecisionProcess)

	customPrompt := vergexCustomPromptSection(e.config.CustomPrompt)
	if customPrompt != "" {
		sb.WriteString("# User Preference\n\n")
		sb.WriteString(customPrompt)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// vergexCustomPromptSection returns the user's custom prompt for the vergex
// path, dropping legacy directional overrides ("long only" era) that would
// contradict the data-driven direction rule baked into this prompt.
func vergexCustomPromptSection(section string) string {
	trimmed := strings.TrimSpace(section)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	legacyDirectives := []string{
		"long only",
		"long-only",
		"do not short",
		"no shorts",
		"must open a long",
		"short only",
		"short-only",
	}
	for _, directive := range legacyDirectives {
		if strings.Contains(lower, directive) {
			return ""
		}
	}
	return trimmed
}

// userPromptSection returns the user's explicitly edited prompt section
// verbatim (trimmed). User-authored sections are deliberately NOT filtered by
// language: an operator who writes Chinese instructions expects them to reach
// the model. Only the built-in sections (schema, risk, output format) stay
// English for contract stability. Legacy direction filters still apply to the
// vergex custom-prompt path via vergexCustomPromptSection.
func userPromptSection(section string) string {
	return strings.TrimSpace(section)
}

func writeVergexSchemaPrompt(sb *strings.Builder) {
	sb.WriteString("# Claw402.ai TradeFi Data Guide\n\n")
	sb.WriteString("- Equity: total account value including unrealized PnL, in USDT.\n")
	sb.WriteString("- Balance: available balance for new positions, in USDT.\n")
	sb.WriteString("- Margin: current margin usage; higher means more risk.\n")
	sb.WriteString("- Position: current holdings with side, entry, leverage, unrealized PnL and liquidation price.\n")
	sb.WriteString("- Claw402 Ranking: tradable candidate pool, rank, direction and category for this cycle.\n")
	sb.WriteString("- Signal Lab: per-symbol Claw402 deep signal used to confirm trend and quality.\n")
	sb.WriteString("- Cost/Liquidation Heatmap: cost and liquidation clusters used for stops, targets and crowding risk.\n")
	sb.WriteString("- Raw OHLCV Kline: raw candles used for trend structure, entry timing and risk/reward.\n")
}

func writeVergexHardConstraints(sb *strings.Builder, accountEquity float64, riskControl store.RiskControlConfig, tradeFiPositionValueRatio float64) {
	maxPositionValue := accountEquity * tradeFiPositionValueRatio
	sb.WriteString("# Hard Risk Constraints\n\n")
	sb.WriteString("## Backend enforced\n")
	sb.WriteString(fmt.Sprintf("- Max positions: %d Claw402 candidate instruments at the same time\n", riskControl.MaxPositions))
	sb.WriteString(fmt.Sprintf("- Max notional per position: %.0f USDT (= equity %.0f × %.1fx)\n", maxPositionValue, accountEquity, tradeFiPositionValueRatio))
	sb.WriteString(fmt.Sprintf("- Max margin usage: ≤%.0f%%\n", riskControl.MaxMarginUsage*100))
	sb.WriteString(fmt.Sprintf("- Min order size: ≥%.0f USDT\n\n", riskControl.MinPositionSize))
	sb.WriteString("## AI guided\n")
	sb.WriteString(fmt.Sprintf("- Leverage: use up to %dx; reduce when ATR is above 1.5x average or during high-volatility regimes\n", riskControl.AltcoinMaxLeverage))
	sb.WriteString(fmt.Sprintf("- Risk/reward: ≥1:%.1f\n", riskControl.MinRiskRewardRatio))
	sb.WriteString(fmt.Sprintf("- Min confidence to open: ≥%d\n\n", riskControl.MinConfidence))
	sb.WriteString("# Position Sizing\n\n")
	sb.WriteString("Calculate `position_size_usd` from your confidence and the max notional per position:\n")
	sb.WriteString("- High confidence (≥85): use 80-100%% of the max notional.\n")
	sb.WriteString("- Medium confidence (70-84): use 50-80%% of the max notional.\n")
	sb.WriteString("- Low confidence (60-69): use 30-50%% of the max notional.\n")
	sb.WriteString("- If the setup is not strong enough for even 30%%, output `wait`.\n")
	sb.WriteString("- Do not use available_balance directly as position_size_usd.\n\n")

	sb.WriteString("## Confidence Calibration\n\n")
	sb.WriteString("- 90-100: All signals align (Signal Lab + heatmap + candles agree), high-conviction setup\n")
	sb.WriteString(fmt.Sprintf("- %d-89: Most signals agree, 1-2 minor conflicts, solid setup\n", riskControl.MinConfidence))
	sb.WriteString("- 60-77: Mixed signals, wait for more confirmation\n")
	sb.WriteString("- Below 60: No edge detected, output `wait`\n\n")
}

func writeVergexOutputFormat(sb *strings.Builder, accountEquity float64, riskControl store.RiskControlConfig, tradeFiPositionValueRatio float64, singleSymbol bool, primarySymbol string) {
	exampleSymbol := "xyz:NVDA"
	secondSymbol := "xyz:AAPL"
	if singleSymbol && strings.TrimSpace(primarySymbol) != "" {
		exampleSymbol = primarySymbol
		secondSymbol = primarySymbol
	}
	positionSize := accountEquity * tradeFiPositionValueRatio
	leverage := riskControl.AltcoinMaxLeverage
	if leverage <= 0 {
		leverage = 1
	}

	sb.WriteString("# Output Format (Strictly Follow)\n\n")
	sb.WriteString("Use XML tags <reasoning> and <decision> to separate concise analysis from the decision JSON.\n\n")
	sb.WriteString("Direction must be data-driven: use `open_long` for confirmed upside structures and `open_short` for confirmed downside structures; never default to long-only or short-only behavior.\n\n")
	if !singleSymbol {
		sb.WriteString("Evaluate both directions every cycle, but enter a side only when its own signals independently justify it. Never open a position just to balance the book — an unbalanced book beats a forced trade.\n\n")
	}
	sb.WriteString("<reasoning>\n")
	sb.WriteString("Briefly state whether Claw402 ranking, Signal Lab, heatmap and candles agree; if data is missing or conflicting, explain why you wait.\n")
	sb.WriteString("</reasoning>\n\n")
	sb.WriteString("<decision>\n")
	sb.WriteString("// NOTE: Example prices below are FORMAT ILLUSTRATIONS only.\n")
	sb.WriteString("// Replace with actual current market prices in your decisions.\n")
	sb.WriteString("```json\n[\n")
	// Use realistic SL/TP examples (never 0) so AI learns the correct pattern.
	// Prices are template values — the model must calculate real SL/TP from data.
	exampleSL1 := 97000.0
	exampleTP1 := 103000.0
	exampleSL2 := 3600.0
	exampleTP2 := 3300.0
	exampleRisk := accountEquity * 0.01 // 1% of equity as example risk
	if exampleRisk < 10 {
		exampleRisk = 10
	}
	if singleSymbol {
		sb.WriteString(fmt.Sprintf("  {\"symbol\": \"%s\", \"action\": \"open_short\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": %.0f, \"take_profit\": %.0f, \"confidence\": 85, \"risk_usd\": %.0f}\n", exampleSymbol, leverage, positionSize, exampleSL1, exampleTP1, exampleRisk))
	} else {
		sb.WriteString(fmt.Sprintf("  {\"symbol\": \"%s\", \"action\": \"open_long\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": %.0f, \"take_profit\": %.0f, \"confidence\": 85, \"risk_usd\": %.0f},\n", exampleSymbol, leverage, positionSize, exampleSL1, exampleTP1, exampleRisk))
		sb.WriteString(fmt.Sprintf("  {\"symbol\": \"%s\", \"action\": \"open_short\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": %.0f, \"take_profit\": %.0f, \"confidence\": 85, \"risk_usd\": %.0f}\n", secondSymbol, leverage, positionSize, exampleSL2, exampleTP2, exampleRisk))
	}
	sb.WriteString("]\n```\n")
	sb.WriteString("</decision>\n\n")

	sb.WriteString("## Field Requirements\n\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait | modify\n")
	sb.WriteString(fmt.Sprintf("- `confidence`: 0-100; recommended ≥ %d to open\n", riskControl.MinConfidence))
	sb.WriteString("- Required when opening: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd\n")
	sb.WriteString("- Required when modifying: stop_loss and/or take_profit (at least one). Use `modify` to adjust SL/TP on existing positions.\n")
	sb.WriteString("- All numeric values must be calculated numbers, not formulas.\n")
	if singleSymbol {
		sb.WriteString(fmt.Sprintf("- This strategy trades only `%s`; JSON symbol must match it exactly.\n", exampleSymbol))
	} else {
		sb.WriteString("- JSON symbols must exactly match current candidates or existing positions; keep `xyz:` on XYZ instruments, and do not add `xyz:` or `USDT` to core crypto symbols.\n")
	}
	sb.WriteString("\n")
}

// buildXYZStockCustomPrompt returns the canonical English directional stock
// briefing the agent uses for single-symbol Hyperliquid USDC perpetuals on
// the XYZ board. Symbol is inlined for LLM grounding so it never confuses the
// trading instrument.
func buildXYZStockCustomPrompt(symbol string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Trade ONLY the Hyperliquid USDC perpetual %s (US equity / xyz board).\n\n", symbol))
	sb.WriteString("Core stance: DIRECTIONAL, SIGNAL-DRIVEN. You may open long or short; never force a trade when Signal Lab, liquidation structure and candles disagree.\n\n")

	sb.WriteString("## Flat-Account Rule\n")
	sb.WriteString("If `Current Positions` is None / empty, evaluate both directions from scratch.\n")
	sb.WriteString("- Use `open_long` only when upside continuation or bullish reversal is confirmed.\n")
	sb.WriteString("- Use `open_short` only when downside continuation or bearish reversal is confirmed.\n")
	sb.WriteString("- Use `wait` when neither side meets the minimum confidence and risk/reward threshold.\n")
	sb.WriteString("- Do not raise confidence just to force an order; confidence must reflect the evidence.\n\n")

	sb.WriteString("## Long Entry Conditions\n")
	sb.WriteString("- Break of the prior session/intraday high on rising volume.\n")
	sb.WriteString("- Pullback to a clearly held intraday support (prior swing low, VWAP, EMA20/50) with a bullish reaction bar.\n")
	sb.WriteString("- Sector tape strength (broad US-equity bid, sympathy with peers in the same theme).\n")
	sb.WriteString("- Confirmed catalyst: earnings beat, guide up, sector rotation, macro tailwind.\n\n")

	sb.WriteString("## Short Entry Conditions\n")
	sb.WriteString("- Breakdown below intraday support or value area with expanding volume.\n")
	sb.WriteString("- Failed breakout, lower high, or bearish rejection at resistance.\n")
	sb.WriteString("- Signal Lab / liquidation structure shows downside fuel, trapped longs, or weak support below.\n")
	sb.WriteString("- Negative catalyst: earnings miss, guide down, sector weakness, macro headwind.\n\n")

	sb.WriteString("## Risk Guardrails (non-negotiable)\n")
	sb.WriteString("- Per-trade stop-loss: 1.5-3% from entry. ALWAYS set a numeric `stop_loss`.\n")
	sb.WriteString("- Take-profit: target at least R/R 2:1; set a numeric `take_profit`.\n")
	sb.WriteString("- Per-trade notional: <= 25% of account equity (probing 10-15%, full 20-25%).\n")
	sb.WriteString("- Leverage: 2-3x default, never above 5x. Never go all-in.\n")
	sb.WriteString("- Do not flip directly from long to short or short to long in the same cycle. Manage or close the open position first.\n\n")

	sb.WriteString("## Position Management\n")
	sb.WriteString("- Trail stop to breakeven once +1R, take partial profits at +2R if momentum stalls.\n")
	sb.WriteString("- Cut quickly if price breaks the stop or the catalyst thesis fails.\n")
	sb.WriteString("- Holding past 45 minutes is fine; flipping in/out every cycle is not.\n\n")

	sb.WriteString("## Discipline\n")
	sb.WriteString(fmt.Sprintf("- Single-symbol mandate: never rotate into another ticker. The decision JSON `symbol` MUST be exactly \"%s\".\n", symbol))
	sb.WriteString("- Before every decision: check current price vs prior pivot, volume vs 5m/1h average, and the broader US-equity tape.\n")
	sb.WriteString("- If positions are open, prioritize managing them over piling on new ones.")
	return sb.String()
}

// singleSymbolInfo returns (true, "ARM-USDC") for static-coin strategies that
// trade exactly one instrument. Multi-symbol strategies return (false, "").
// The flag is used to drop crypto-specific "BTC/ETH vs Altcoin" labeling and
// to put the actual trading symbol into the JSON example.
func (e *StrategyEngine) singleSymbolInfo() (bool, string) {
	coinSource := e.config.CoinSource
	if (coinSource.SourceType == "static" || coinSource.SourceType == "vergex_signal") && len(coinSource.StaticCoins) == 1 {
		return true, strings.ToUpper(strings.TrimSpace(coinSource.StaticCoins[0]))
	}
	return false, ""
}

func writeModeVariant(sb *strings.Builder, variant string) {
	switch strings.ToLower(strings.TrimSpace(variant)) {
	case "aggressive":
		sb.WriteString("## Mode: Aggressive\n- Prioritize capturing trend breakouts; may scale in when confidence ≥ 70\n- Allow larger positions, but must strictly set stop-loss and explain the risk-reward ratio\n\n")
	case "conservative":
		sb.WriteString("## Mode: Conservative\n- Open positions only when multiple signals resonate\n- Prioritize capital preservation; pause for multiple periods after consecutive losses\n\n")
	case "scalping":
		sb.WriteString("## Mode: Scalping\n- Focus on short-term momentum, smaller profit targets but require quick action\n- If price doesn't move as expected within two bars, immediately reduce position or stop-loss\n\n")
	case "balanced", "":
		sb.WriteString("## Mode: Balanced\n- Open only when multiple signals resonate across timeframes\n- Standard position sizing: confidence-based, no systematic tilt toward aggression or caution\n- Prioritize capital preservation in unclear regimes; pause after consecutive losses\n\n")
	}
}

// writeCommonDiscipline writes trading discipline rules shared by both the
// generic and vergex prompt paths (Anti-Patterns, Time Stop, Order Handling,
// Position Management). The canonical text is the union of both paths so
// the model sees identical constraints regardless of strategy type.
func writeCommonDiscipline(sb *strings.Builder, riskControl store.RiskControlConfig) {
	sb.WriteString("## Anti-Patterns (DO NOT)\n\n")
	sb.WriteString("- Opening a position because \"it looks cheap\" — no data, no trade.\n")
	sb.WriteString("- Setting stop_loss = 0 — will be rejected by backend.\n")
	sb.WriteString("- confidence = 99 on every trade — overconfident, not credible.\n")
	sb.WriteString(fmt.Sprintf("- Opening more than %d positions in the same cycle — overexposure.\n", (riskControl.MaxPositions+1)/2))
	sb.WriteString("- Closing a position < 15 minutes after opening — churning.\n")
	sb.WriteString("- Flipping from long to short (or vice versa) in the same cycle on the same symbol. To flip direction: first close, then open in the NEXT cycle.\n")
	sb.WriteString("- Confusing realized and unrealized PnL — realized is already in balance, don't double count.\n")
	sb.WriteString("- Ignoring leverage impact — 1% price move with 3x leverage = ~3% PnL, not 1%.\n")
	sb.WriteString("- Not watching Peak PnL — when current PnL nears Peak PnL, consider taking profit.\n")
	sb.WriteString("- Ignoring OI changes — use OI to validate trend authenticity; OI up + price up = strong.\n")
	sb.WriteString("- Revenge trading: never increase position size after a loss to \"make it back.\" Size down after consecutive losses.\n")
	sb.WriteString("- Anchoring: do not bias your analysis based on your entry price. Evaluate each decision from current market state, not from your PnL.\n")
	sb.WriteString("- Opening new positions when Max Drawdown > 20% — reduce risk first, trade later.\n\n")
	sb.WriteString("## Time Stop\n\n")
	sb.WriteString("- If a position has not moved in your favor after 2 hours, reassess the thesis.\n")
	sb.WriteString("- If holding > 4 hours with PnL < 0, consider closing to free up capital.\n")
	sb.WriteString("- Never hold a losing position overnight unless the fundamental thesis is intact.\n")
	sb.WriteString("- For xyz equity instruments: close or reduce before Friday US market close — weekend gap risk is real.\n")
	sb.WriteString("- Funding rate accrues 3x on Friday for crypto perps; factor this into hold/exit decisions near week-end.\n\n")
	sb.WriteString("## Order Handling\n\n")
	sb.WriteString("- If an order is partially filled, the backend handles the rest automatically. Do not re-submit the remaining quantity.\n")
	sb.WriteString("- If an order fails entirely, the error is logged. Do not retry the same order in the same cycle.\n\n")
	sb.WriteString("## Position Management Rules\n\n")
	sb.WriteString("- Scale-in: Only add to winning positions, max 2 additions. Price must be above average cost by at least 0.5x ATR(14) (minimum 1% for stable coins, 2% for volatile altcoins).\n")
	sb.WriteString("- Scale-out: Close 33% at +3% unrealized, 50% at +5%, 100% at +8%. Let winners run, lock profits incrementally.\n")
	sb.WriteString("  - In strong trending regimes, consider extending targets by 1.5x (e.g., +12% instead of +8%).\n")
	sb.WriteString("- Trailing stop: Close position when unrealized PnL pulls back 30% from peak (e.g., peak +5%, close at +3.5%).\n")
	sb.WriteString("- Never average down a losing position.\n")
	sb.WriteString("- Emergency exit: Close immediately if price gaps through your stop-loss level or if a major adverse event occurs (flash crash, exchange outage, regulatory news). Do not wait for scale-out targets.\n\n")

	// Multi-Timeframe Analysis Framework — tells the model how to use the
	// multiple timeframe K-line data it receives in the user prompt.
	sb.WriteString("## Multi-Timeframe Analysis\n\n")
	sb.WriteString("- Long-term TF (1h-4h): determine the PRIMARY trend direction. Trade with it, not against it.\n")
	sb.WriteString("- Medium-term TF (15m-30m): find price structure — support/resistance, patterns, consolidation zones.\n")
	sb.WriteString("- Short-term TF (1m-5m): time the entry. Look for pullbacks into structure in the primary trend direction.\n")
	sb.WriteString("- All TFs aligned in same direction → high-confidence setup (confidence ≥85).\n")
	sb.WriteString("- Long-term trend up but medium-term pulling back → wait for entry signal at structure support.\n")
	sb.WriteString("- TFs conflicting (e.g. 1h down, 15m up) → lower confidence (≤70) or wait. Counter-trend trades need at least 3 confirmations.\n")
	sb.WriteString("- Use the current candle's close + volume to validate; opening ranges and session overlap volumes add weight.\n\n")

	// Position Sizing by Market Regime — the user prompt includes a Regime
	// hint (TRENDING_UP/DOWN, RANGING, HIGH_VOL, LOW_VOL); adjust sizing.
	sb.WriteString("## Position Sizing by Market Regime\n\n")
	sb.WriteString("- TRENDING (directional): full-size positions per confidence; may scale into winners.\n")
	sb.WriteString("- RANGING: half-size positions; prefer range-bound setups, take profits at band extremes.\n")
	sb.WriteString("- HIGH_VOL (large 1h moves, wide BB, elevated ATR): reduce size to 30-50%, widen stops to 1.5-2× ATR.\n")
	sb.WriteString("- LOW_VOL (compression, tight ranges): minimal probing positions (10-20%) or wait. Tight ranges often precede explosive moves.\n")
	sb.WriteString("- TRANSITIONAL / UNKNOWN: standard sizing, but require stronger confluence before opening.\n")
	sb.WriteString("- Max Drawdown >20%% or consecutive losses ≥3: pause new positions for 2-3 cycles. Only manage/close existing.\n")
	sb.WriteString("- Account equity declining (equity trend ↓): reduce position count and size; capital preservation first.\n")
	sb.WriteString("- Account equity rising (equity trend ↑): may gradually scale up to standard sizing over 3+ winning cycles.\n\n")
}

func writeHardConstraints(sb *strings.Builder, accountEquity float64, riskControl store.RiskControlConfig, btcEthPosValueRatio, altcoinPosValueRatio float64, singleSymbol bool, primarySymbol string) {
	sb.WriteString("# Hard Constraints (Risk Control)\n\n")
	sb.WriteString("## CODE ENFORCED (backend validation, cannot be bypassed):\n")
	sb.WriteString(fmt.Sprintf("- Max Positions: %d instruments simultaneously\n", riskControl.MaxPositions))

	if singleSymbol {
		// One symbol — pick the higher of the two configured ratios so the
		// limit isn't accidentally clamped to the altcoin cap for a stock.
		ratio := altcoinPosValueRatio
		if btcEthPosValueRatio > ratio {
			ratio = btcEthPosValueRatio
		}
		maxVal := accountEquity * ratio
		symLabel := primarySymbol
		sb.WriteString(fmt.Sprintf("- Position Value Limit (%s): max %.0f USDT (= equity %.0f × %.1fx)\n", symLabel, maxVal, accountEquity, ratio))
	} else {
		sb.WriteString(fmt.Sprintf("- Position Value Limit (Altcoin/Stock): max %.0f USDT (= equity %.0f × %.1fx)\n", accountEquity*altcoinPosValueRatio, accountEquity, altcoinPosValueRatio))
		sb.WriteString(fmt.Sprintf("- Position Value Limit (BTC/ETH): max %.0f USDT (= equity %.0f × %.1fx)\n", accountEquity*btcEthPosValueRatio, accountEquity, btcEthPosValueRatio))
	}

	sb.WriteString(fmt.Sprintf("- Max Margin Usage: ≤%.0f%%\n", riskControl.MaxMarginUsage*100))
	sb.WriteString(fmt.Sprintf("- Min Position Size: ≥%.0f USDT\n\n", riskControl.MinPositionSize))
	sb.WriteString("## AI GUIDED (recommended):\n")

	if singleSymbol {
		lev := riskControl.AltcoinMaxLeverage
		if riskControl.BTCETHMaxLeverage > lev {
			lev = riskControl.BTCETHMaxLeverage
		}
		sb.WriteString(fmt.Sprintf("- Trading Leverage (%s): max %dx\n", primarySymbol, lev))
	} else {
		sb.WriteString(fmt.Sprintf("- Trading Leverage: Altcoin/Stock max %dx | BTC/ETH max %dx\n", riskControl.AltcoinMaxLeverage, riskControl.BTCETHMaxLeverage))
	}
	sb.WriteString(fmt.Sprintf("- Risk-Reward Ratio: ≥1:%.1f (take_profit / stop_loss)\n", riskControl.MinRiskRewardRatio))
	sb.WriteString(fmt.Sprintf("- Min Confidence: ≥%d to open position\n", riskControl.MinConfidence))
	sb.WriteString("- Stop-loss: 1-2x ATR(14) from entry, capped at 3%% for BTC/ETH, 5%% for altcoins. Never place stop inside liquidation clusters.\n\n")

	// Position sizing guidance
	exampleRatio := btcEthPosValueRatio
	if singleSymbol {
		exampleRatio = altcoinPosValueRatio
		if btcEthPosValueRatio > exampleRatio {
			exampleRatio = btcEthPosValueRatio
		}
	}
	sb.WriteString("## Position Sizing Guidance\n")
	sb.WriteString("Calculate `position_size_usd` from your confidence and the Position Value Limits above:\n")
	sb.WriteString("- High confidence (≥85): use 80-100%% of the position value limit\n")
	sb.WriteString("- Medium confidence (70-84): use 50-80%% of the position value limit\n")
	sb.WriteString("- Low confidence (60-69): use 30-50%% of the position value limit\n")
	sb.WriteString(fmt.Sprintf("- Example: equity %.0f × %.1fx = max %.0f USDT\n", accountEquity, exampleRatio, accountEquity*exampleRatio))
	sb.WriteString("- **DO NOT** just use available_balance as position_size_usd. Use the Position Value Limit!\n\n")
}

func writeOutputFormat(sb *strings.Builder, accountEquity, btcEthPosValueRatio float64, riskControl store.RiskControlConfig, singleSymbol bool, primarySymbol string) {
	// Output format schema MUST stay English/structural; parser depends on it.
	sb.WriteString("# Output Format (Strictly Follow)\n\n")
	sb.WriteString("**Must use XML tags <reasoning> and <decision> to separate chain of thought and decision JSON, avoiding parsing errors**\n\n")
	sb.WriteString("## Format Requirements\n\n")
	sb.WriteString("<reasoning>\n")
	sb.WriteString("Your chain of thought analysis...\n- Briefly analyze your thinking process\n")
	sb.WriteString("</reasoning>\n\n")
	sb.WriteString("<decision>\n")
	sb.WriteString("Step 2: JSON decision array\n\n")
	sb.WriteString("```json\n[\n")

	// Build a JSON example using the actual trading symbol when the strategy
	// is single-symbol. Falls back to the legacy BTC/ETH two-line example
	// only for multi-symbol strategies that genuinely have BTC/ETH on tap.
	if singleSymbol {
		lev := riskControl.AltcoinMaxLeverage
		if riskControl.BTCETHMaxLeverage > lev {
			lev = riskControl.BTCETHMaxLeverage
		}
		ratio := btcEthPosValueRatio // already chosen as the larger above when single-symbol
		size := accountEquity * ratio
		exampleRisk := size * 0.01 // 1% of position value as example risk
		if exampleRisk < 10 {
			exampleRisk = 10
		}
		sb.WriteString(fmt.Sprintf("  {\"symbol\": \"%s\", \"action\": \"open_long\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": 97000, \"take_profit\": 103000, \"confidence\": 85, \"risk_usd\": %.0f},\n", primarySymbol, lev, size, exampleRisk))
		sb.WriteString(fmt.Sprintf("  {\"symbol\": \"%s\", \"action\": \"wait\"}\n", primarySymbol))
	} else {
		examplePositionSize := accountEquity * btcEthPosValueRatio
		sb.WriteString(fmt.Sprintf("  {\"symbol\": \"BTCUSDT\", \"action\": \"open_short\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": 97000, \"take_profit\": 91000, \"confidence\": 85, \"risk_usd\": 300},\n",
			riskControl.BTCETHMaxLeverage, examplePositionSize))
		sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\"},\n")
		sb.WriteString("  {\"symbol\": \"SOLUSDT\", \"action\": \"modify\", \"stop_loss\": 130, \"take_profit\": 180}\n")
	}
	sb.WriteString("]\n```\n")
	sb.WriteString("</decision>\n\n")

	sb.WriteString("## Field Description\n\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait | modify\n")
	sb.WriteString("  - `hold`: keep existing position unchanged — use when thesis intact, position profitable or neutral, no SL/TP adjustment needed\n")
	sb.WriteString("  - `wait`: skip this cycle, do not open new positions — use when no edge detected, signals conflicting, or risk budget exceeded\n")
	sb.WriteString("  - `close`: exit position — use when thesis broken, time stop triggered, or risk/reward no longer favorable\n")
	sb.WriteString(fmt.Sprintf("- `confidence`: 0-100 (opening recommended ≥ %d)\n", riskControl.MinConfidence))
	sb.WriteString("- Required when opening: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd\n")
	sb.WriteString("- Required when modifying: stop_loss and/or take_profit (at least one). Use `modify` to adjust SL/TP on existing positions.\n")
	sb.WriteString("- Example modify: {\"symbol\": \"BTCUSDT\", \"action\": \"modify\", \"stop_loss\": 95000, \"take_profit\": 105000}\n")
	sb.WriteString("- **IMPORTANT**: all numeric values must be calculated numbers, NOT formulas/expressions (e.g. use `27.776`, not `3000 * 0.01`)\n")
	sb.WriteString("- Output `[]` (empty array) only when you cannot make any decision at all. Prefer `wait` over empty array.\n")
	if singleSymbol {
		sb.WriteString(fmt.Sprintf("- **This strategy trades only %s.** The JSON `symbol` MUST match `%s` exactly — do not add USDT/USDC suffix variants.\n", primarySymbol, primarySymbol))
	}
	sb.WriteString("\n")
}

func (e *StrategyEngine) writeAvailableIndicators(sb *strings.Builder) {
	indicators := e.config.Indicators
	kline := indicators.Klines

	sb.WriteString(fmt.Sprintf("- %s price series", kline.PrimaryTimeframe))
	if kline.EnableMultiTimeframe {
		sb.WriteString(fmt.Sprintf(" + %s K-line series\n", kline.LongerTimeframe))
	} else {
		sb.WriteString("\n")
	}

	if indicators.EnableEMA {
		sb.WriteString("- EMA indicators")
		if len(indicators.EMAPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.EMAPeriods))
		}
		sb.WriteString("\n")
	}
	if indicators.EnableMACD {
		sb.WriteString("- MACD indicators\n")
	}
	if indicators.EnableRSI {
		sb.WriteString("- RSI indicators")
		if len(indicators.RSIPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.RSIPeriods))
		}
		sb.WriteString("\n")
	}
	if indicators.EnableATR {
		sb.WriteString("- ATR indicators")
		if len(indicators.ATRPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.ATRPeriods))
		}
		sb.WriteString("\n")
	}
	if indicators.EnableBOLL {
		sb.WriteString("- Bollinger Bands (BOLL) - Upper/Middle/Lower bands")
		if len(indicators.BOLLPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.BOLLPeriods))
		}
		sb.WriteString("\n")
	}
	if indicators.EnableVolume {
		sb.WriteString("- Volume data\n")
	}
	if indicators.EnableOI {
		sb.WriteString("- Open Interest (OI) data\n")
	}
	if indicators.EnableFundingRate {
		sb.WriteString("- Funding rate\n")
	}
	if len(e.config.CoinSource.StaticCoins) > 0 || e.config.CoinSource.UseAI500 || e.config.CoinSource.UseOITop {
		sb.WriteString("- AI500 / OI_Top filter tags (if available)\n")
	}
	if indicators.EnableQuantData {
		sb.WriteString("- Quantitative data (institutional/retail fund flow, position changes, multi-period price changes)\n")
	}
}

// ============================================================================
// Prompt Building - User Prompt
// ============================================================================

// BuildUserPrompt builds User Prompt based on strategy configuration
func (e *StrategyEngine) BuildUserPrompt(ctx *Context) string {
	var sb strings.Builder

	// System status
	sb.WriteString(fmt.Sprintf("Time: %s | Period: #%d | Runtime: %d minutes\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes))

	// BTC market
	if btcData, hasBTC := ctx.MarketDataMap["BTCUSDT"]; hasBTC {
		sb.WriteString(fmt.Sprintf("BTC: %.2f (1h: %+.2f%%, 4h: %+.2f%%) | MACD: %.4f | RSI: %.2f\n",
			btcData.CurrentPrice, btcData.PriceChange1h, btcData.PriceChange4h,
			btcData.CurrentMACD, btcData.CurrentRSI7))

		// Pre-computed regime hint (saves AI from inferring from raw candles)
		regime := computeRegimeHint(btcData)
		sb.WriteString(fmt.Sprintf("Regime: %s\n", regime))

		// Trading session context
		session := computeSessionCtx()
		sb.WriteString(fmt.Sprintf("Session: %s\n\n", session))
	} else {
		sb.WriteString("\n")
	}

	// Account information (with limits so AI knows headroom)
	maxMarginPct := e.config.RiskControl.MaxMarginUsage * 100
	maxPos := e.config.RiskControl.MaxPositions
	// Compute equity trend from PnL percentage
	equityTrend := ""
	if ctx.Account.TotalPnLPct > 0 {
		equityTrend = fmt.Sprintf(" (↑ %+.1f%%)", ctx.Account.TotalPnLPct)
	} else if ctx.Account.TotalPnLPct < 0 {
		equityTrend = fmt.Sprintf(" (↓ %+.1f%%)", ctx.Account.TotalPnLPct)
	}
	sb.WriteString(fmt.Sprintf("Account: Equity %.2f%s | Balance %.2f (%.1f%%) | Margin %.1f%% (limit %.0f%%) | Positions %d (max %d)\n\n",
		ctx.Account.TotalEquity, equityTrend,
		ctx.Account.AvailableBalance,
		(ctx.Account.AvailableBalance/ctx.Account.TotalEquity)*100,
		ctx.Account.MarginUsedPct, maxMarginPct,
		ctx.Account.PositionCount, maxPos))

	// Recently completed orders (placed before positions to ensure visibility)
	if len(ctx.RecentOrders) > 0 {
		sb.WriteString("## Recent Completed Trades\n")
		var totalWin, totalLoss, longWins, longLosses, shortWins, shortLosses int
		var totalPnL float64
		var consecutiveLosses int
		for i, order := range ctx.RecentOrders {
			resultStr := "Profit"
			if order.RealizedPnL < 0 {
				resultStr = "Loss"
				totalLoss++
				if order.Side == "long" {
					longLosses++
				} else {
					shortLosses++
				}
			} else {
				totalWin++
				if order.Side == "long" {
					longWins++
				} else {
					shortWins++
				}
			}
			totalPnL += order.RealizedPnL
			sb.WriteString(fmt.Sprintf("%d. %s %s | Entry %.4f Exit %.4f | %s: %+.2f USDT (%+.2f%%) | %s→%s (%s)\n",
				i+1, order.Symbol, order.Side,
				order.EntryPrice, order.ExitPrice,
				resultStr, order.RealizedPnL, order.PnLPct,
				order.EntryTime, order.ExitTime, order.HoldDuration))
		}
		// Pattern summary
		n := totalWin + totalLoss
		if n > 0 {
			sb.WriteString(fmt.Sprintf("\nPattern: %dW/%dL (%.0f%% win) | Net: %+.2f USDT | Long: %dW/%dL | Short: %dW/%dL\n",
				totalWin, totalLoss, float64(totalWin)/float64(n)*100, totalPnL,
				longWins, longLosses, shortWins, shortLosses))
		}
		// Detect consecutive losses (most recent trades first)
		if len(ctx.RecentOrders) > 0 {
			for _, order := range ctx.RecentOrders {
				if order.RealizedPnL < 0 {
					consecutiveLosses++
				} else {
					break
				}
			}
			if consecutiveLosses >= 2 {
				sb.WriteString(fmt.Sprintf("⚠️ Consecutive losses: %d (caution: consider reducing position size)\n", consecutiveLosses))
			}
		}
		sb.WriteString("\n")
	}

	// Historical trading statistics (helps AI understand past performance)
	if ctx.TradingStats != nil && ctx.TradingStats.TotalTrades > 0 {
		// Win/Loss ratio
		var winLossRatio float64
		if ctx.TradingStats.AvgLoss > 0 {
			winLossRatio = ctx.TradingStats.AvgWin / ctx.TradingStats.AvgLoss
		}

		sb.WriteString("## Historical Trading Statistics\n")
		sb.WriteString(fmt.Sprintf("Total Trades: %d | Profit Factor: %.2f | Sharpe: %.2f | Win/Loss Ratio: %.2f\n",
			ctx.TradingStats.TotalTrades,
			ctx.TradingStats.ProfitFactor,
			ctx.TradingStats.SharpeRatio,
			winLossRatio))
		sb.WriteString(fmt.Sprintf("Total PnL: %+.2f USDT | Avg Win: +%.2f | Avg Loss: -%.2f | Max Drawdown: %.1f%%\n",
			ctx.TradingStats.TotalPnL,
			ctx.TradingStats.AvgWin,
			ctx.TradingStats.AvgLoss,
			ctx.TradingStats.MaxDrawdownPct))

		// Performance hints based on profit factor, sharpe, and drawdown
		if ctx.TradingStats.ProfitFactor >= 1.5 && ctx.TradingStats.SharpeRatio >= 1 {
			sb.WriteString("Performance: GOOD - maintain current strategy\n")
		} else if ctx.TradingStats.ProfitFactor < 1 {
			sb.WriteString("Performance: NEEDS IMPROVEMENT - improve win/loss ratio, optimize TP/SL\n")
		} else if ctx.TradingStats.MaxDrawdownPct > 30 {
			sb.WriteString("Performance: HIGH RISK - reduce position size, control drawdown\n")
		} else {
			sb.WriteString("Performance: NORMAL - room for optimization\n")
		}
		sb.WriteString("\n")
	}

	// Decision Quality Feedback (helps AI learn from recent patterns)
	if len(ctx.RecentOrders) >= 3 {
		sb.WriteString("## Decision Quality Feedback\n")

		// Recent win rate (last 10 trades for better statistical significance)
		recentN := 10
		if len(ctx.RecentOrders) < recentN {
			recentN = len(ctx.RecentOrders)
		}
		recentWins := 0
		var recentPnL float64
		var longWins, longTrades, shortWins, shortTrades int
		var longPnL, shortPnL float64
		for i := 0; i < recentN; i++ {
			order := ctx.RecentOrders[i]
			recentPnL += order.RealizedPnL
			if order.RealizedPnL > 0 {
				recentWins++
			}
			if order.Side == "long" {
				longTrades++
				longPnL += order.RealizedPnL
				if order.RealizedPnL > 0 {
					longWins++
				}
			} else {
				shortTrades++
				shortPnL += order.RealizedPnL
				if order.RealizedPnL > 0 {
					shortWins++
				}
			}
		}
		sb.WriteString(fmt.Sprintf("Last %d trades: %dW/%dL (%.0f%% win) | Net: %+.2f USDT\n",
			recentN, recentWins, recentN-recentWins,
			float64(recentWins)/float64(recentN)*100, recentPnL))
		if longTrades > 0 {
			sb.WriteString(fmt.Sprintf("  Long:  %dW/%dL (%.0f%%) | Net: %+.2f USDT\n",
				longWins, longTrades-longWins,
				float64(longWins)/float64(longTrades)*100, longPnL))
		}
		if shortTrades > 0 {
			sb.WriteString(fmt.Sprintf("  Short: %dW/%dL (%.0f%%) | Net: %+.2f USDT\n",
				shortWins, shortTrades-shortWins,
				float64(shortWins)/float64(shortTrades)*100, shortPnL))
		}
		// Direction skew warning: if one side is significantly worse, flag it.
		if longTrades >= 3 && shortTrades >= 3 {
			longWR := float64(longWins) / float64(longTrades) * 100
			shortWR := float64(shortWins) / float64(shortTrades) * 100
			if longWR-shortWR > 25 {
				sb.WriteString("⚠️ Short entries significantly underperforming — review short-only criteria\n")
			} else if shortWR-longWR > 25 {
				sb.WriteString("⚠️ Long entries significantly underperforming — review long-only criteria\n")
			}
		}
		if recentN < 10 {
			sb.WriteString("(sample size small, interpret with caution)\n")
		}

		// Common error detection
		if recentWins < recentN/2 {
			sb.WriteString("⚠️ Win rate below 50% on recent trades — consider tightening entry criteria\n")
		}
		if recentPnL < 0 {
			sb.WriteString("⚠️ Recent net PnL negative — reduce position size until performance recovers\n")
		}
		sb.WriteString("\n")
	}

	// Position information
	if len(ctx.Positions) > 0 {
		sb.WriteString("## Current Positions\n")
		var totalLongValue, totalShortValue, closestLiqBuffer float64
		closestLiqSymbol := ""
		for i, pos := range ctx.Positions {
			sb.WriteString(e.formatPositionInfo(i+1, pos, ctx))
			posValue := pos.Quantity * pos.MarkPrice
			if posValue < 0 {
				posValue = -posValue
			}
			if pos.Side == "long" {
				totalLongValue += posValue
			} else {
				totalShortValue += posValue
			}
			// Track closest to liquidation
			if pos.LiquidationPrice > 0 && pos.MarkPrice > 0 {
				var buffer float64
				if pos.Side == "long" {
					buffer = (pos.MarkPrice - pos.LiquidationPrice) / pos.MarkPrice * 100
				} else {
					buffer = (pos.LiquidationPrice - pos.MarkPrice) / pos.MarkPrice * 100
				}
				if closestLiqBuffer == 0 || buffer < closestLiqBuffer {
					closestLiqBuffer = buffer
					closestLiqSymbol = pos.Symbol
				}
			}
		}
		// Portfolio Risk Summary
		totalExposure := totalLongValue + totalShortValue
		equity := ctx.Account.TotalEquity
		if equity > 0 {
			netDir := "Neutral"
			if totalLongValue > totalShortValue*1.1 {
				netDir = "Net Long"
			} else if totalShortValue > totalLongValue*1.1 {
				netDir = "Net Short"
			}
			sb.WriteString(fmt.Sprintf("## Portfolio Risk Summary\n"))
			sb.WriteString(fmt.Sprintf("Exposure: %.1fx equity | Long: %.1fx | Short: %.1fx | Direction: %s\n",
				totalExposure/equity, totalLongValue/equity, totalShortValue/equity, netDir))
			if closestLiqSymbol != "" && closestLiqBuffer < 20 {
				sb.WriteString(fmt.Sprintf("⚠️ Closest to liquidation: %s (%.1f%% buffer)\n", closestLiqSymbol, closestLiqBuffer))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("Current Positions: None\n\n")
	}

	// Candidate coins (exclude coins already in positions to avoid duplicate data)
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		// Normalize symbol to handle both "ETH" and "ETHUSDT" formats
		normalizedSymbol := market.Normalize(pos.Symbol)
		positionSymbols[normalizedSymbol] = true
	}

	sb.WriteString(fmt.Sprintf("## Candidate Coins (%d coins)\n\n", len(ctx.MarketDataMap)))
	displayedCount := 0
	const compactThreshold = 5 // After this many candidates, compact mode: only recent 5 candles
	for _, coin := range ctx.CandidateCoins {
		// Skip if this coin is already a position (data already shown in positions section)
		normalizedCoinSymbol := market.Normalize(coin.Symbol)
		if positionSymbols[normalizedCoinSymbol] {
			continue
		}

		marketData, hasData := ctx.MarketDataMap[coin.Symbol]
		if !hasData {
			continue
		}
		displayedCount++

		// Compact mode: after 8 candidates, only provide summary to save tokens
		compact := displayedCount > compactThreshold

		sourceTags := e.formatCoinSourceTag(coin.Sources)
		sb.WriteString(fmt.Sprintf("### %d. %s%s\n\n", displayedCount, coin.Symbol, sourceTags))
		sb.WriteString(e.formatMarketData(marketData, compact))

		if !compact {
			if ctx.QuantDataMap != nil {
				if quantData, hasQuant := ctx.QuantDataMap[coin.Symbol]; hasQuant {
					sb.WriteString(e.formatQuantData(quantData))
				}
			}
			if ctx.VergexDataMap != nil {
				if vergexData, hasVergex := ctx.VergexDataMap[coin.Symbol]; hasVergex {
					sb.WriteString(e.formatVergexData(vergexData))
				}
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Get language for market data formatting
	fxosLang := nofx.LangEnglish
	if e.GetLanguage() == LangChinese {
		fxosLang = nofx.LangChinese
	}

	// OI Ranking data (market-wide open interest changes)
	if ctx.OIRankingData != nil {
		sb.WriteString(nofx.FormatOIRankingForAI(ctx.OIRankingData, fxosLang))
	}

	// NetFlow Ranking data (market-wide fund flow)
	if ctx.NetFlowRankingData != nil {
		sb.WriteString(nofx.FormatNetFlowRankingForAI(ctx.NetFlowRankingData, fxosLang))
	}

	// Price Ranking data (market-wide gainers/losers)
	if ctx.PriceRankingData != nil {
		sb.WriteString(nofx.FormatPriceRankingForAI(ctx.PriceRankingData, fxosLang))
	}

	return sb.String()
}

func (e *StrategyEngine) formatPositionInfo(index int, pos PositionInfo, ctx *Context) string {
	var sb strings.Builder

	holdingDuration := ""
	if pos.UpdateTime > 0 {
		durationMs := time.Now().UnixMilli() - pos.UpdateTime
		durationMin := durationMs / (1000 * 60)
		if durationMin < 60 {
			holdingDuration = fmt.Sprintf(" | Holding Duration %d min", durationMin)
		} else {
			durationHour := durationMin / 60
			durationMinRemainder := durationMin % 60
			holdingDuration = fmt.Sprintf(" | Holding Duration %dh %dm", durationHour, durationMinRemainder)
		}
	}

	positionValue := pos.Quantity * pos.MarkPrice
	if positionValue < 0 {
		positionValue = -positionValue
	}

	sb.WriteString(fmt.Sprintf("%d. %s %s | Entry %.4f Current %.4f | Qty %.4f | Position Value %.2f USDT | PnL%+.2f%% | PnL Amount%+.2f USDT | Peak PnL%.2f%% | Leverage %dx | Margin %.0f | Liq Price %.4f%s\n\n",
		index, pos.Symbol, strings.ToUpper(pos.Side),
		pos.EntryPrice, pos.MarkPrice, pos.Quantity, positionValue, pos.UnrealizedPnLPct, pos.UnrealizedPnL, pos.PeakPnLPct,
		pos.Leverage, pos.MarginUsed, pos.LiquidationPrice, holdingDuration))

	if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
		sb.WriteString(e.formatMarketData(marketData, false))

		if ctx.QuantDataMap != nil {
			if quantData, hasQuant := ctx.QuantDataMap[pos.Symbol]; hasQuant {
				sb.WriteString(e.formatQuantData(quantData))
			}
		}
		if ctx.VergexDataMap != nil {
			if vergexData, hasVergex := ctx.VergexDataMap[pos.Symbol]; hasVergex {
				sb.WriteString(e.formatVergexData(vergexData))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (e *StrategyEngine) formatCoinSourceTag(sources []string) string {
	if len(sources) > 1 {
		// Multiple signal source combination
		hasAI500 := false
		hasOITop := false
		hasOILow := false
		hasHyperAll := false
		hasHyperMain := false
		for _, s := range sources {
			switch s {
			case "ai500":
				hasAI500 = true
			case "oi_top":
				hasOITop = true
			case "oi_low":
				hasOILow = true
			case "hyper_all":
				hasHyperAll = true
			case "hyper_main":
				hasHyperMain = true
			}
		}
		if hasAI500 && hasOITop {
			return " (AI500+OI_Top dual signal)"
		}
		if hasAI500 && hasOILow {
			return " (AI500+OI_Low dual signal)"
		}
		if hasOITop && hasOILow {
			return " (OI_Top+OI_Low)"
		}
		if hasHyperMain && hasAI500 {
			return " (HyperMain+AI500)"
		}
		if hasHyperAll || hasHyperMain {
			return " (Hyperliquid)"
		}
		return " (Multiple sources)"
	} else if len(sources) == 1 {
		switch sources[0] {
		case "ai500":
			return " (AI500)"
		case "oi_top":
			return " (OI_Top OI increase)"
		case "oi_low":
			return " (OI_Low OI decrease)"
		case "static":
			return " (Manual selection)"
		case "hyper_all":
			return " (Hyperliquid All)"
		case "hyper_main":
			return " (Hyperliquid Top20)"
		case "vergex_signal":
			return " (Vergex Signal)"
		}
		if strings.HasPrefix(sources[0], "hyper_rank") {
			return " (Hyperliquid Dynamic Rank)"
		}
	}
	return ""
}

func (e *StrategyEngine) formatVergexData(data *vergex.MarketAnalysis) string {
	if data == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\nVergex Claw402 Signals:\n")
	sb.WriteString(vergex.FormatAnalysisForAI(data))
	return sb.String()
}

// ============================================================================
// Market Data Formatting
// ============================================================================

func (e *StrategyEngine) formatMarketData(data *market.Data, compact bool) string {
	var sb strings.Builder
	indicators := e.config.Indicators

	// Clearly label the coin symbol
	sb.WriteString(fmt.Sprintf("=== %s Market Data ===\n\n", data.Symbol))
	sb.WriteString(fmt.Sprintf("current_price = %.4f", data.CurrentPrice))

	if indicators.EnableEMA {
		sb.WriteString(fmt.Sprintf(", current_ema20 = %.3f", data.CurrentEMA20))
	}

	if indicators.EnableMACD {
		sb.WriteString(fmt.Sprintf(", current_macd = %.3f", data.CurrentMACD))
	}

	if indicators.EnableRSI {
		sb.WriteString(fmt.Sprintf(", current_rsi7 = %.3f", data.CurrentRSI7))
	}

	sb.WriteString("\n\n")

	// Compact mode: only show price summary + OI/FR + last 5 candles.
	if compact {
		sb.WriteString("(compact: last 5 candles only — preceding data omitted to save tokens)\n\n")
		if indicators.EnableOI || indicators.EnableFundingRate {
			sb.WriteString(fmt.Sprintf("Additional data for %s:\n\n", data.Symbol))
			if indicators.EnableOI && data.OpenInterest != nil {
				sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f Average: %.2f\n\n",
					data.OpenInterest.Latest, data.OpenInterest.Average))
			}
			if indicators.EnableFundingRate {
				sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))
			}
		}
		if len(data.TimeframeData) > 0 {
			timeframeOrder := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w"}
			for _, tf := range timeframeOrder {
				if tfData, ok := data.TimeframeData[tf]; ok {
					sb.WriteString(fmt.Sprintf("=== %s Timeframe (last 5, oldest → latest) ===\n\n", strings.ToUpper(tf)))
					e.formatTimeframeSeriesData(&sb, tfData, indicators, true)
				}
			}
		}
		return sb.String()
	}

	if indicators.EnableOI || indicators.EnableFundingRate {
		sb.WriteString(fmt.Sprintf("Additional data for %s:\n\n", data.Symbol))

		if indicators.EnableOI && data.OpenInterest != nil {
			sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f Average: %.2f\n\n",
				data.OpenInterest.Latest, data.OpenInterest.Average))
		}

		if indicators.EnableFundingRate {
			sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))
		}
	}

	if len(data.TimeframeData) > 0 {
		timeframeOrder := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w"}
		for _, tf := range timeframeOrder {
			if tfData, ok := data.TimeframeData[tf]; ok {
				sb.WriteString(fmt.Sprintf("=== %s Timeframe (oldest → latest) ===\n\n", strings.ToUpper(tf)))
				e.formatTimeframeSeriesData(&sb, tfData, indicators, false)
			}
		}
	} else {
		// Compatible with old data format
		if data.IntradaySeries != nil {
			klineConfig := indicators.Klines
			sb.WriteString(fmt.Sprintf("Intraday series (%s intervals, oldest → latest):\n\n", klineConfig.PrimaryTimeframe))

			if len(data.IntradaySeries.MidPrices) > 0 {
				sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
			}

			if indicators.EnableEMA && len(data.IntradaySeries.EMA20Values) > 0 {
				sb.WriteString(fmt.Sprintf("EMA indicators (20-period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
			}

			if indicators.EnableMACD && len(data.IntradaySeries.MACDValues) > 0 {
				sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
			}

			if indicators.EnableRSI {
				if len(data.IntradaySeries.RSI7Values) > 0 {
					sb.WriteString(fmt.Sprintf("RSI indicators (7-Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
				}
				if len(data.IntradaySeries.RSI14Values) > 0 {
					sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
				}
			}

			if indicators.EnableVolume && len(data.IntradaySeries.Volume) > 0 {
				sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.IntradaySeries.Volume)))
			}

			if indicators.EnableATR {
				sb.WriteString(fmt.Sprintf("3m ATR (14-period): %.3f\n\n", data.IntradaySeries.ATR14))
			}
		}

		if data.LongerTermContext != nil && indicators.Klines.EnableMultiTimeframe {
			sb.WriteString(fmt.Sprintf("Longer-term context (%s timeframe):\n\n", indicators.Klines.LongerTimeframe))

			if indicators.EnableEMA {
				sb.WriteString(fmt.Sprintf("20-Period EMA: %.3f vs. 50-Period EMA: %.3f\n\n",
					data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))
			}

			if indicators.EnableATR {
				sb.WriteString(fmt.Sprintf("3-Period ATR: %.3f vs. 14-Period ATR: %.3f\n\n",
					data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))
			}

			if indicators.EnableVolume {
				sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
					data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))
			}

			if indicators.EnableMACD && len(data.LongerTermContext.MACDValues) > 0 {
				sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
			}

			if indicators.EnableRSI && len(data.LongerTermContext.RSI14Values) > 0 {
				sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
			}
		}
	}

	return sb.String()
}

func (e *StrategyEngine) formatTimeframeSeriesData(sb *strings.Builder, data *market.TimeframeSeriesData, indicators store.IndicatorConfig, compact bool) {
	maxItems := len(data.Klines)
	if compact && maxItems > 5 {
		maxItems = 5
	}
	if len(data.Klines) > 0 {
		klines := data.Klines
		if compact && len(klines) > 5 {
			klines = klines[len(klines)-5:]
		}
		if compact {
			sb.WriteString(fmt.Sprintf("(showing last %d of %d candles)\n", len(klines), len(data.Klines)))
		}
		sb.WriteString("Time(UTC)      Open      High      Low       Close     Volume\n")
		for i, k := range klines {
			t := time.Unix(k.Time/1000, 0).UTC()
			timeStr := t.Format("01-02 15:04")
			marker := ""
			if i == len(klines)-1 {
				marker = "  <- current"
			}
			sb.WriteString(fmt.Sprintf("%-14s %-9.4f %-9.4f %-9.4f %-9.4f %-12.2f%s\n",
				timeStr, k.Open, k.High, k.Low, k.Close, k.Volume, marker))
		}
		sb.WriteString("\n")
	} else if len(data.MidPrices) > 0 {
		prices := data.MidPrices
		if compact && len(prices) > 5 {
			prices = prices[len(prices)-5:]
		}
		sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(prices)))
		if indicators.EnableVolume && len(data.Volume) > 0 {
			vols := data.Volume
			if compact && len(vols) > 5 {
				vols = vols[len(vols)-5:]
			}
			sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(vols)))
		}
	}

	if indicators.EnableEMA {
		if len(data.EMA20Values) > 0 {
			vals := data.EMA20Values
			if compact && len(vals) > 5 {
				vals = vals[len(vals)-5:]
			}
			sb.WriteString(fmt.Sprintf("EMA20: %s\n", formatFloatSlice(vals)))
		}
		if len(data.EMA50Values) > 0 {
			vals := data.EMA50Values
			if compact && len(vals) > 5 {
				vals = vals[len(vals)-5:]
			}
			sb.WriteString(fmt.Sprintf("EMA50: %s\n", formatFloatSlice(vals)))
		}
	}

	if indicators.EnableMACD && len(data.MACDValues) > 0 {
		vals := data.MACDValues
		if compact && len(vals) > 5 {
			vals = vals[len(vals)-5:]
		}
		sb.WriteString(fmt.Sprintf("MACD: %s\n", formatFloatSlice(vals)))
	}

	if indicators.EnableRSI {
		if len(data.RSI7Values) > 0 {
			vals := data.RSI7Values
			if compact && len(vals) > 5 {
				vals = vals[len(vals)-5:]
			}
			sb.WriteString(fmt.Sprintf("RSI7: %s\n", formatFloatSlice(vals)))
		}
		if len(data.RSI14Values) > 0 {
			vals := data.RSI14Values
			if compact && len(vals) > 5 {
				vals = vals[len(vals)-5:]
			}
			sb.WriteString(fmt.Sprintf("RSI14: %s\n", formatFloatSlice(vals)))
		}
	}

	if indicators.EnableATR && data.ATR14 > 0 {
		sb.WriteString(fmt.Sprintf("ATR14: %.4f\n", data.ATR14))
	}

	if indicators.EnableBOLL && len(data.BOLLUpper) > 0 {
		sb.WriteString(fmt.Sprintf("BOLL Upper: %s\n", formatFloatSlice(data.BOLLUpper)))
		sb.WriteString(fmt.Sprintf("BOLL Middle: %s\n", formatFloatSlice(data.BOLLMiddle)))
		sb.WriteString(fmt.Sprintf("BOLL Lower: %s\n", formatFloatSlice(data.BOLLLower)))
	}

	sb.WriteString("\n")
}

func (e *StrategyEngine) formatQuantData(data *QuantData) string {
	if data == nil {
		return ""
	}

	indicators := e.config.Indicators
	if !indicators.EnableQuantOI && !indicators.EnableQuantNetflow {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 %s Quantitative Data:\n", data.Symbol))

	if len(data.PriceChange) > 0 {
		sb.WriteString("Price Change: ")
		timeframes := []string{"5m", "15m", "1h", "4h", "12h", "24h"}
		parts := []string{}
		for _, tf := range timeframes {
			if v, ok := data.PriceChange[tf]; ok {
				parts = append(parts, fmt.Sprintf("%s: %+.4f%%", tf, v*100))
			}
		}
		sb.WriteString(strings.Join(parts, " | "))
		sb.WriteString("\n")
	}

	if indicators.EnableQuantNetflow && data.Netflow != nil {
		sb.WriteString("Fund Flow (Netflow):\n")
		timeframes := []string{"5m", "15m", "1h", "4h", "12h", "24h"}

		if data.Netflow.Institution != nil {
			if data.Netflow.Institution.Future != nil && len(data.Netflow.Institution.Future) > 0 {
				sb.WriteString("  Institutional Futures:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Institution.Future[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
			if data.Netflow.Institution.Spot != nil && len(data.Netflow.Institution.Spot) > 0 {
				sb.WriteString("  Institutional Spot:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Institution.Spot[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
		}

		if data.Netflow.Personal != nil {
			if data.Netflow.Personal.Future != nil && len(data.Netflow.Personal.Future) > 0 {
				sb.WriteString("  Retail Futures:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Personal.Future[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
			if data.Netflow.Personal.Spot != nil && len(data.Netflow.Personal.Spot) > 0 {
				sb.WriteString("  Retail Spot:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Personal.Spot[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
		}
	}

	if indicators.EnableQuantOI && len(data.OI) > 0 {
		for exchange, oiData := range data.OI {
			if len(oiData.Delta) > 0 {
				sb.WriteString(fmt.Sprintf("Open Interest (%s):\n", exchange))
				for _, tf := range []string{"5m", "15m", "1h", "4h", "12h", "24h"} {
					if d, ok := oiData.Delta[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %+.4f%% (%s)\n", tf, d.OIDeltaPercent, formatFlowValue(d.OIDeltaValue)))
					}
				}
			}
		}
	}

	// Synthesize Order Flow Bias from OI and netflow data
	if indicators.EnableQuantOI || indicators.EnableQuantNetflow {
		oiBias := ""
		flowBias := ""

		// OI bias: check 1h delta
		if data.OI != nil {
			for _, oiData := range data.OI {
				if d, ok := oiData.Delta["1h"]; ok {
					if d.OIDeltaPercent > 2.0 {
						oiBias = "OI↑"
					} else if d.OIDeltaPercent < -2.0 {
						oiBias = "OI↓"
					}
				}
				break // check first exchange only
			}
		}

		// Netflow bias: check 1h institutional
		if data.Netflow != nil && data.Netflow.Institution != nil && data.Netflow.Institution.Future != nil {
			if v, ok := data.Netflow.Institution.Future["1h"]; ok {
				if v > 1e6 {
					flowBias = "inst_inflow"
				} else if v < -1e6 {
					flowBias = "inst_outflow"
				}
			}
		}

		if oiBias != "" || flowBias != "" {
			parts := []string{}
			if oiBias != "" {
				parts = append(parts, oiBias)
			}
			if flowBias != "" {
				parts = append(parts, flowBias)
			}
			sb.WriteString(fmt.Sprintf("Order Flow Bias: %s\n", strings.Join(parts, " + ")))
		}
	}

	return sb.String()
}

func formatFlowValue(v float64) string {
	sign := ""
	if v >= 0 {
		sign = "+"
	}
	absV := v
	if absV < 0 {
		absV = -absV
	}
	if absV >= 1e9 {
		return fmt.Sprintf("%s%.2fB", sign, v/1e9)
	} else if absV >= 1e6 {
		return fmt.Sprintf("%s%.2fM", sign, v/1e6)
	} else if absV >= 1e3 {
		return fmt.Sprintf("%s%.2fK", sign, v/1e3)
	}
	return fmt.Sprintf("%s%.2f", sign, v)
}

func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%.4f", v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// computeRegimeHint derives a simple regime label from BTC market data.
// This saves the AI from inferring regime from raw candles every cycle.
// Uses price changes, RSI, MACD, Bollinger Band width, and ATR context.
func computeRegimeHint(btc *market.Data) string {
	if btc == nil {
		return "unknown"
	}

	// Extract BB width and ATR from timeframe data if available
	bbWidthPct := 0.0
	atrRatio := 0.0
	if btc.TimeframeData != nil {
		// Try 1h first, then 5m
		for _, tf := range []string{"1h", "5m"} {
			if tfData, ok := btc.TimeframeData[tf]; ok {
				if len(tfData.BOLLUpper) > 0 && len(tfData.BOLLMiddle) > 0 && tfData.BOLLMiddle[len(tfData.BOLLMiddle)-1] > 0 {
					upper := tfData.BOLLUpper[len(tfData.BOLLUpper)-1]
					lower := tfData.BOLLLower[len(tfData.BOLLLower)-1]
					middle := tfData.BOLLMiddle[len(tfData.BOLLMiddle)-1]
					bbWidthPct = (upper - lower) / middle * 100
				}
				if tfData.ATR14 > 0 && btc.CurrentPrice > 0 {
					atrRatio = tfData.ATR14 / btc.CurrentPrice * 100
				}
				break
			}
		}
	}

	// High volatility: large 1h move OR wide Bollinger Bands OR high ATR
	if btc.PriceChange1h > 3.0 || btc.PriceChange1h < -3.0 {
		return "HIGH_VOL (large 1h move)"
	}
	if bbWidthPct > 5.0 {
		return fmt.Sprintf("HIGH_VOL (BB width %.1f%%)", bbWidthPct)
	}
	if atrRatio > 3.0 {
		return fmt.Sprintf("HIGH_VOL (ATR %.1f%% of price)", atrRatio)
	}

	// Low volatility: tight BB + small moves
	if bbWidthPct > 0 && bbWidthPct < 2.0 {
		return fmt.Sprintf("LOW_VOL (BB width %.1f%%, compression)", bbWidthPct)
	}
	if btc.PriceChange1h > -0.5 && btc.PriceChange1h < 0.5 &&
		btc.PriceChange4h > -1.0 && btc.PriceChange4h < 1.0 {
		return "LOW_VOL (tight range)"
	}

	// Trending: RSI directional + MACD confirmation
	if btc.CurrentRSI7 > 60 && btc.CurrentMACD > 0 {
		return "TRENDING_UP (RSI >60, MACD positive)"
	}
	if btc.CurrentRSI7 < 40 && btc.CurrentMACD < 0 {
		return "TRENDING_DOWN (RSI <40, MACD negative)"
	}

	// Ranging: RSI neutral, small moves
	if btc.CurrentRSI7 > 35 && btc.CurrentRSI7 < 65 {
		return "RANGING (RSI neutral, mixed signals)"
	}

	return "TRANSITIONAL"
}

// computeSessionCtx returns the current trading session based on UTC hour.
func computeSessionCtx() string {
	hour := time.Now().UTC().Hour()

	switch {
	case hour >= 0 && hour < 7:
		return "Asian (00-07 UTC) — lower liquidity, thinner books"
	case hour >= 7 && hour < 12:
		return "European (07-12 UTC) — building volume, key data releases"
	case hour >= 12 && hour < 16:
		return "US-Europe Overlap (12-16 UTC) — peak liquidity, highest volume"
	case hour >= 16 && hour < 21:
		return "US Afternoon (16-21 UTC) — US-driven, winding down"
	default:
		return "Off-hours (21-00 UTC) — thin liquidity, gap risk"
	}
}
