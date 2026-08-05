package trader

import (
	"fmt"
	"fxos/kernel"
	"fxos/logger"
	"fxos/market"
	"fxos/store"
	"strings"
	"time"
)

// startDrawdownMonitor starts drawdown monitoring
func (at *AutoTrader) startDrawdownMonitor() {
	at.monitorWg.Add(1)
	go func() {
		defer at.monitorWg.Done()

		ticker := time.NewTicker(1 * time.Minute) // Check every minute
		defer ticker.Stop()

		logger.Info("📊 Started position drawdown monitoring (check every minute)")

		for {
			select {
			case <-ticker.C:
				at.checkPositionDrawdown()
			case <-at.stopMonitorCh:
				logger.Info("⏹ Stopped position drawdown monitoring")
				return
			}
		}
	}()
}

// checkPositionDrawdown checks position drawdown situation
func (at *AutoTrader) checkPositionDrawdown() {
	// Get current positions
	positions, err := at.trader.GetPositions()
	if err != nil {
		logger.Infof("❌ Drawdown monitoring: failed to get positions: %v", err)
		return
	}

	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity // Short position quantity is negative, convert to positive
		}

		// Guard: skip if entry price is zero (prevents division by zero panic)
		if entryPrice <= 0 {
			logger.Warnf("⚠️ Drawdown monitoring: %s %s has zero entry price, skipping", symbol, side)
			continue
		}

		// Calculate current P&L percentage
		leverage := 10 // Default value
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}

		var currentPnLPct float64
		if side == "long" {
			currentPnLPct = ((markPrice - entryPrice) / entryPrice) * float64(leverage) * 100
		} else {
			currentPnLPct = ((entryPrice - markPrice) / entryPrice) * float64(leverage) * 100
		}

		// Construct unique position identifier (distinguish long/short)
		posKey := symbol + "_" + side

		// Get historical peak profit for this position
		at.peakPnLCacheMutex.RLock()
		peakPnLPct, exists := at.peakPnLCache[posKey]
		at.peakPnLCacheMutex.RUnlock()

		if !exists {
			// If no historical peak record, use current P&L as initial value
			peakPnLPct = currentPnLPct
			at.UpdatePeakPnL(symbol, side, currentPnLPct)
		} else {
			// Update peak cache
			at.UpdatePeakPnL(symbol, side, currentPnLPct)
		}

		// Calculate drawdown (magnitude of decline from peak)
		var drawdownPct float64
		if peakPnLPct > 0 && currentPnLPct < peakPnLPct {
			drawdownPct = ((peakPnLPct - currentPnLPct) / peakPnLPct) * 100
		}

		// Check close position condition: profit > 5% and drawdown >= 40%
		if currentPnLPct > 5.0 && drawdownPct >= 40.0 {
			logger.Infof("🚨 Drawdown close position condition triggered: %s %s | Current profit: %.2f%% | Peak profit: %.2f%% | Drawdown: %.2f%%",
				symbol, side, currentPnLPct, peakPnLPct, drawdownPct)

			// Execute close position
			if err := at.emergencyClosePosition(symbol, side); err != nil {
				logger.Infof("❌ Drawdown close position failed (%s %s): %v", symbol, side, err)
			} else {
				logger.Infof("✅ Drawdown close position succeeded: %s %s", symbol, side)
				// Clear cache for this position after closing
				at.ClearPeakPnLCache(symbol, side)
				at.ClearBreakevenStopCache(symbol, side)
			}
		} else if currentPnLPct > 5.0 {
			// Record situations close to close position condition (for debugging)
			logger.Infof("📊 Drawdown monitoring: %s %s | Profit: %.2f%% | Peak: %.2f%% | Drawdown: %.2f%%",
				symbol, side, currentPnLPct, peakPnLPct, drawdownPct)
		}

		// Dynamic stop-loss: breakeven stop and trailing stop
		at.adjustStopLoss(symbol, side, entryPrice, markPrice, quantity, currentPnLPct, peakPnLPct, leverage)
	}
}

// emergencyClosePosition emergency close position function
func (at *AutoTrader) emergencyClosePosition(symbol, side string) error {
	switch side {
	case "long":
		order, err := at.trader.CloseLong(symbol, 0) // 0 = close all
		if err != nil {
			return err
		}
		logger.Infof("✅ Emergency close long position succeeded, order ID: %v", order["orderId"])
	case "short":
		order, err := at.trader.CloseShort(symbol, 0) // 0 = close all
		if err != nil {
			return err
		}
		logger.Infof("✅ Emergency close short position succeeded, order ID: %v", order["orderId"])
	default:
		return fmt.Errorf("unknown position direction: %s", side)
	}

	return nil
}

// GetPeakPnLCache gets peak profit cache
func (at *AutoTrader) GetPeakPnLCache() map[string]float64 {
	at.peakPnLCacheMutex.RLock()
	defer at.peakPnLCacheMutex.RUnlock()

	// Return a copy of the cache
	cache := make(map[string]float64)
	for k, v := range at.peakPnLCache {
		cache[k] = v
	}
	return cache
}

// UpdatePeakPnL updates peak profit cache
func (at *AutoTrader) UpdatePeakPnL(symbol, side string, currentPnLPct float64) {
	at.peakPnLCacheMutex.Lock()
	defer at.peakPnLCacheMutex.Unlock()

	posKey := symbol + "_" + side
	if peak, exists := at.peakPnLCache[posKey]; exists {
		// Update peak (if long, take larger value; if short, currentPnLPct is negative, also compare)
		if currentPnLPct > peak {
			at.peakPnLCache[posKey] = currentPnLPct
		}
	} else {
		// First time recording
		at.peakPnLCache[posKey] = currentPnLPct
	}
}

// ClearPeakPnLCache clears peak cache for specified position
func (at *AutoTrader) ClearPeakPnLCache(symbol, side string) {
	at.peakPnLCacheMutex.Lock()
	defer at.peakPnLCacheMutex.Unlock()

	posKey := symbol + "_" + side
	delete(at.peakPnLCache, posKey)
}

// ClearBreakevenStopCache clears breakeven stop cache for specified position
func (at *AutoTrader) ClearBreakevenStopCache(symbol, side string) {
	at.breakevenStopCacheMu.Lock()
	defer at.breakevenStopCacheMu.Unlock()

	posKey := symbol + "_" + side
	delete(at.breakevenStopCache, posKey)
}

// ============================================================================
// Dynamic Stop-Loss (Breakeven + Trailing)
// ============================================================================

// adjustStopLoss implements dynamic stop-loss management:
// 1. Breakeven stop: when profit >= 1x risk, move SL to entry price
// 2. Trailing stop: when profit >= 2x risk, trail SL at 50% from peak
func (at *AutoTrader) adjustStopLoss(symbol, side string, entryPrice, markPrice, quantity, currentPnLPct, peakPnLPct float64, leverage int) {
	if entryPrice <= 0 || quantity <= 0 || leverage <= 0 {
		return
	}

	// Determine profit thresholds based on asset type
	// BTC/ETH: tighter thresholds (more liquid, less volatile)
	// Altcoins: wider thresholds (more volatile)
	var breakevenThreshold float64 // profit % to trigger breakeven stop
	var trailingThreshold float64  // profit % to trigger trailing stop
	var trailingDrawdown float64   // max drawdown from peak before trailing kicks in

	if isBTCETH(symbol) {
		breakevenThreshold = 1.5 // 1.5% profit = ~1x risk for typical 1.5% stop
		trailingThreshold = 3.0  // 3% profit = ~2x risk
		trailingDrawdown = 50.0  // trail at 50% from peak
	} else {
		breakevenThreshold = 2.5 // 2.5% profit for altcoins (higher vol)
		trailingThreshold = 5.0  // 5% profit for altcoins
		trailingDrawdown = 50.0  // trail at 50% from peak
	}

	posKey := symbol + "_" + side

	// --- Trailing Stop: when profit >= trailingThreshold, trail SL at trailingDrawdown% from peak ---
	if currentPnLPct >= trailingThreshold && peakPnLPct > currentPnLPct {
		// Calculate trailing stop price
		var trailStopPrice float64
		lev := float64(leverage)
		if side == "long" {
			// For long: trail below peak
			peakPrice := entryPrice * (1 + peakPnLPct/100/lev)
			trailStopPrice = peakPrice * (1 - trailingDrawdown/100*peakPnLPct/100/lev)
			// Ensure stop is above entry (at least breakeven)
			if trailStopPrice < entryPrice {
				trailStopPrice = entryPrice
			}
		} else {
			// For short: trail above trough
			peakPrice := entryPrice * (1 - peakPnLPct/100/lev)
			trailStopPrice = peakPrice * (1 + trailingDrawdown/100*peakPnLPct/100/lev)
			// Ensure stop is below entry (at least breakeven)
			if trailStopPrice > entryPrice {
				trailStopPrice = entryPrice
			}
		}

		if trailStopPrice > 0 {
			at.applyTrailingStop(symbol, side, quantity, trailStopPrice, "trailing")
		}
		return
	}

	// --- Breakeven Stop: when profit >= breakevenThreshold, move SL to entry ---
	at.breakevenStopCacheMu.RLock()
	alreadyApplied := at.breakevenStopCache[posKey]
	at.breakevenStopCacheMu.RUnlock()

	if currentPnLPct >= breakevenThreshold && !alreadyApplied {
		logger.Infof("🔒 Breakeven stop triggered: %s %s | Profit: %.2f%% >= %.2f%% threshold",
			symbol, side, currentPnLPct, breakevenThreshold)

		at.applyTrailingStop(symbol, side, quantity, entryPrice, "breakeven")

		at.breakevenStopCacheMu.Lock()
		at.breakevenStopCache[posKey] = true
		at.breakevenStopCacheMu.Unlock()
	}
}

// applyTrailingStop cancels existing SL and places new SL at the given price
func (at *AutoTrader) applyTrailingStop(symbol, side string, quantity, stopPrice float64, reason string) {
	// Cancel existing stop-loss orders
	if err := at.trader.CancelStopLossOrders(symbol); err != nil {
		logger.Infof("⚠️ Failed to cancel SL orders for %s: %v", symbol, err)
		return
	}

	// Place new stop-loss at adjusted price
	positionSide := "LONG"
	if side == "short" {
		positionSide = "SHORT"
	}

	if err := at.trader.SetStopLoss(symbol, positionSide, quantity, stopPrice); err != nil {
		logger.Infof("⚠️ Failed to set %s stop for %s at %.4f: %v", reason, symbol, stopPrice, err)
		return
	}

	logger.Infof("✅ %s stop applied: %s %s | New SL: %.4f", reason, symbol, side, stopPrice)
}

// getLeverage extracts leverage from position data (returns default 10 if unavailable)
func (at *AutoTrader) getLeverage(pos map[string]interface{}) int {
	if lev, ok := pos["leverage"].(float64); ok && lev > 0 {
		return int(lev)
	}
	return 10
}

// ============================================================================
// Risk Control Helpers
// ============================================================================

// isBTCETH checks if a symbol is BTC or ETH
func isBTCETH(symbol string) bool {
	symbol = strings.ToUpper(symbol)
	return strings.HasPrefix(symbol, "BTC") || strings.HasPrefix(symbol, "ETH")
}

// isMajorAsset returns true for assets that should use the BTC/ETH higher
// position-value tier rather than the altcoin (1x equity) tier. This covers
// BTC/ETH crypto perps AND Hyperliquid XYZ assets (US equities, commodities,
// forex) — none of which are "altcoins" and all of which deserve the higher
// per-position cap so the AI can actually take meaningful positions.
func isMajorAsset(symbol string) bool {
	if isBTCETH(symbol) {
		return true
	}
	return market.IsXyzDexAsset(symbol)
}

// enforcePositionValueRatio checks and enforces position value ratio limits (CODE ENFORCED)
// Returns the adjusted position size (capped if necessary) and whether the position was capped
// positionSizeUSD: the original position size in USD
// equity: the account equity
// symbol: the trading symbol
func (at *AutoTrader) enforcePositionValueRatio(positionSizeUSD float64, equity float64, symbol string) (float64, bool) {
	if at.config.StrategyConfig == nil {
		return positionSizeUSD, false
	}

	riskControl := at.config.StrategyConfig.RiskControl

	// Get the appropriate position value ratio limit. BTC/ETH AND Hyperliquid
	// XYZ assets (US stocks etc.) use the higher tier; pure altcoins use the
	// lower tier.
	var maxPositionValueRatio float64
	if isMajorAsset(symbol) {
		maxPositionValueRatio = riskControl.BTCETHMaxPositionValueRatio
		if maxPositionValueRatio <= 0 {
			maxPositionValueRatio = 5.0 // Default: 5x for BTC/ETH and XYZ assets
		}
	} else {
		maxPositionValueRatio = riskControl.AltcoinMaxPositionValueRatio
		if maxPositionValueRatio <= 0 {
			maxPositionValueRatio = 1.0 // Default: 1x for altcoins
		}
	}

	// Calculate max allowed position value = equity × ratio
	maxPositionValue := equity * maxPositionValueRatio

	// Check if position size exceeds limit
	if positionSizeUSD > maxPositionValue {
		logger.Infof("  ⚠️ [RISK CONTROL] Position %.2f USDT exceeds limit (equity %.2f × %.1fx = %.2f USDT max for %s), capping",
			positionSizeUSD, equity, maxPositionValueRatio, maxPositionValue, symbol)
		return maxPositionValue, true
	}

	return positionSizeUSD, false
}

func (at *AutoTrader) applyAutopilotFullSizeOpen(decision *kernel.Decision, equity float64) {
	if at == nil || decision == nil || at.config.StrategyConfig == nil || equity <= 0 {
		return
	}

	cfg := at.config.StrategyConfig
	if cfg.CoinSource.SourceType != "vergex_signal" {
		return
	}

	riskControl := cfg.RiskControl
	leverage := riskControl.AltcoinMaxLeverage
	positionValueRatio := riskControl.AltcoinMaxPositionValueRatio
	if isMajorAsset(decision.Symbol) {
		leverage = riskControl.BTCETHMaxLeverage
		positionValueRatio = riskControl.BTCETHMaxPositionValueRatio
	}
	if leverage < store.MinLeverage {
		leverage = store.MinLeverage
	}
	if leverage > store.MaxAltLeverage {
		leverage = store.MaxAltLeverage
	}
	if positionValueRatio <= 0 {
		positionValueRatio = 1.0
	}

	fullPositionSize := equity * positionValueRatio
	if fullPositionSize <= 0 {
		return
	}

	if decision.Leverage != leverage || decision.PositionSizeUSD != fullPositionSize {
		logger.Infof("  📏 [AUTOPILOT] Full-size open enforced for %s: leverage %dx → %dx, notional %.2f → %.2f USDT",
			decision.Symbol, decision.Leverage, leverage, decision.PositionSizeUSD, fullPositionSize)
	}
	decision.Leverage = leverage
	decision.PositionSizeUSD = fullPositionSize
}

// enforceMinPositionSize checks minimum position size (CODE ENFORCED)
func (at *AutoTrader) enforceMinPositionSize(positionSizeUSD float64) error {
	if at.config.StrategyConfig == nil {
		return nil
	}

	minSize := at.config.StrategyConfig.RiskControl.MinPositionSize
	if minSize <= 0 {
		minSize = 12 // Default: 12 USDT
	}

	if positionSizeUSD < minSize {
		return fmt.Errorf("❌ [RISK CONTROL] Position %.2f USDT below minimum (%.2f USDT)", positionSizeUSD, minSize)
	}
	return nil
}

// enforceMaxPositions checks maximum positions count (CODE ENFORCED)
func (at *AutoTrader) enforceMaxPositions(currentPositionCount int) error {
	if at.config.StrategyConfig == nil {
		return nil
	}

	maxPositions := at.config.StrategyConfig.RiskControl.MaxPositions
	if maxPositions <= 0 {
		maxPositions = 3 // Default: 3 positions
	}

	if currentPositionCount >= maxPositions {
		return fmt.Errorf("❌ [RISK CONTROL] Already at max positions (%d/%d)", currentPositionCount, maxPositions)
	}
	return nil
}

// getSideFromAction converts order action to side (BUY/SELL)
func getSideFromAction(action string) string {
	switch action {
	case "open_long", "close_short":
		return "BUY"
	case "open_short", "close_long":
		return "SELL"
	default:
		return "BUY"
	}
}
