package kernel

import (
	"fmt"
	"fxos/logger"
	"fxos/trader/market"
	"fxos/trader/types"
	ktypes "fxos/kernel/types"
)

// ============================================================================
// ktypes.Decision Validation
// ============================================================================

func validateDecisions(decisions []ktypes.Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, priceMap map[string]float64, posSymbols map[string]bool, minRiskRewardRatio float64) error {
	for i := range decisions {
		if err := validateDecision(&decisions[i], accountEquity, btcEthLeverage, altcoinLeverage, btcEthPosRatio, altcoinPosRatio, priceMap, posSymbols, minRiskRewardRatio); err != nil {
			return fmt.Errorf("decision #%d validation failed: %w", i+1, err)
		}
	}
	return nil
}

func validateDecision(d *ktypes.Decision, accountEquity float64, btcEthLeverage, altcoinLeverage int, btcEthPosRatio, altcoinPosRatio float64, priceMap map[string]float64, posSymbols map[string]bool, minRiskRewardRatio float64) error {
	validActions := map[string]bool{
		types.ActionOpenLong:   true,
		types.ActionOpenShort:  true,
		types.ActionCloseLong:  true,
		types.ActionCloseShort: true,
		types.ActionHold:       true,
		types.ActionWait:       true,
		types.ActionModify:     true,
	}

	if !validActions[d.Action] {
		return fmt.Errorf("invalid action: %s", d.Action)
	}

	// Validate close actions: symbol must exist in current positions
	if (d.Action == types.ActionCloseLong || d.Action == types.ActionCloseShort) && posSymbols != nil {
		if !posSymbols[d.Symbol] {
			return fmt.Errorf("cannot close %s: symbol not in current positions", d.Symbol)
		}
	}

	if d.Action == types.ActionOpenLong || d.Action == types.ActionOpenShort {
		// Asset tiering for validation:
		//   - BTC/ETH crypto perps use the BTC/ETH tier (typically 5x equity).
		//   - Hyperliquid XYZ assets (US equities, commodities, forex) are
		//     also treated as the higher tier — they are not crypto altcoins
		//     and the user's quick-trade flow shows them at the higher cap,
		//     so the validator must match.
		//   - Everything else is altcoin (1x equity by default).
		maxLeverage := altcoinLeverage
		posRatio := altcoinPosRatio
		maxPositionValue := accountEquity * posRatio
		isMajor := d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" || market.IsXyzDexAsset(d.Symbol)
		if isMajor {
			maxLeverage = btcEthLeverage
			posRatio = btcEthPosRatio
			maxPositionValue = accountEquity * posRatio
		}

		if d.Leverage <= 0 {
			return fmt.Errorf("leverage must be greater than 0: %d", d.Leverage)
		}
		if d.Leverage > maxLeverage {
			logger.Infof("⚠️  [Leverage Fallback] %s leverage exceeded (%dx > %dx), auto-adjusting to limit %dx",
				d.Symbol, d.Leverage, maxLeverage, maxLeverage)
			d.Leverage = maxLeverage
		}
		if d.PositionSizeUSD <= 0 {
			return fmt.Errorf("position size must be greater than 0: %.2f", d.PositionSizeUSD)
		}

		const minPositionSizeGeneral = 12.0
		const minPositionSizeBTCETH = 60.0

		if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			if d.PositionSizeUSD < minPositionSizeBTCETH {
				return fmt.Errorf("%s opening amount too small (%.2f USDT), must be ≥%.2f USDT", d.Symbol, d.PositionSizeUSD, minPositionSizeBTCETH)
			}
		} else {
			if d.PositionSizeUSD < minPositionSizeGeneral {
				return fmt.Errorf("opening amount too small (%.2f USDT), must be ≥%.2f USDT", d.PositionSizeUSD, minPositionSizeGeneral)
			}
		}

		tolerance := maxPositionValue * 0.01
		if d.PositionSizeUSD > maxPositionValue+tolerance {
			switch {
			case d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT":
				return fmt.Errorf("BTC/ETH single coin position value cannot exceed %.0f USDT (%.1fx account equity), actual: %.0f", maxPositionValue, posRatio, d.PositionSizeUSD)
			case market.IsXyzDexAsset(d.Symbol):
				return fmt.Errorf("%s position value cannot exceed %.0f USDT (%.1fx account equity), actual: %.0f", d.Symbol, maxPositionValue, posRatio, d.PositionSizeUSD)
			default:
				return fmt.Errorf("altcoin single coin position value cannot exceed %.0f USDT (%.1fx account equity), actual: %.0f", maxPositionValue, posRatio, d.PositionSizeUSD)
			}
		}
		if d.StopLoss <= 0 || d.TakeProfit <= 0 {
			return fmt.Errorf("stop loss and take profit must be greater than 0")
		}

		if d.Action == types.ActionOpenLong {
			if d.StopLoss >= d.TakeProfit {
				return fmt.Errorf("for long positions, stop loss price must be less than take profit price")
			}
		} else {
			if d.StopLoss <= d.TakeProfit {
				return fmt.Errorf("for short positions, stop loss price must be greater than take profit price")
			}
		}

		// Use current market price as entry approximation (most accurate).
		// If price unavailable, skip R/R validation — we cannot verify it
		// without knowing the actual entry price.
		var entryPrice float64
		if priceMap != nil {
			entryPrice = priceMap[d.Symbol]
		}
		if entryPrice > 0 {
			var riskPercent, rewardPercent, riskRewardRatio float64
			if d.Action == types.ActionOpenLong {
				riskPercent = (entryPrice - d.StopLoss) / entryPrice * 100
				rewardPercent = (d.TakeProfit - entryPrice) / entryPrice * 100
				if riskPercent > 0 {
					riskRewardRatio = rewardPercent / riskPercent
				}
			} else {
				riskPercent = (d.StopLoss - entryPrice) / entryPrice * 100
				rewardPercent = (entryPrice - d.TakeProfit) / entryPrice * 100
				if riskPercent > 0 {
					riskRewardRatio = rewardPercent / riskPercent
				}
			}

			if riskRewardRatio < minRiskRewardRatio {
				return fmt.Errorf("risk/reward ratio too low (%.2f:1), must be ≥%.1f:1 [risk: %.2f%% reward: %.2f%%] [stop loss: %.2f take profit: %.2f]",
					riskRewardRatio, minRiskRewardRatio, riskPercent, rewardPercent, d.StopLoss, d.TakeProfit)
			}
		}
	}

	return nil
}
