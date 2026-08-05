package hyperliquid

import (
	"fmt"
	"fxos/logger"
	"fxos/trader/types"
	"strconv"
	"strings"
)

// GetPositions gets all positions (including xyz dex positions)
func (t *HyperliquidTrader) GetPositions() ([]types.Position, error) {
	// Get account status
	accountState, err := t.exchange.Info().UserState(t.ctx, t.walletAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	var result []types.Position

	// Iterate through all perp positions
	for _, assetPos := range accountState.AssetPositions {
		position := assetPos.Position

		// Position amount (string type)
		posAmt, _ := strconv.ParseFloat(position.Szi, 64)

		if posAmt == 0 {
			continue // Skip positions with zero amount
		}

		// Normalize symbol format (Hyperliquid uses "BTC", we convert to "BTCUSDT")
		symbol := position.Coin + "USDT"

		// Position amount and direction
		side := "long"
		if posAmt < 0 {
			side = "short"
			posAmt = -posAmt // Convert to positive number
		}

		// Price information (EntryPx and LiquidationPx are pointer types)
		var entryPrice, liquidationPx float64
		if position.EntryPx != nil {
			entryPrice, _ = strconv.ParseFloat(*position.EntryPx, 64)
		}
		if position.LiquidationPx != nil {
			liquidationPx, _ = strconv.ParseFloat(*position.LiquidationPx, 64)
		}

		positionValue, _ := strconv.ParseFloat(position.PositionValue, 64)
		unrealizedPnl, _ := strconv.ParseFloat(position.UnrealizedPnl, 64)

		// Calculate mark price (positionValue / abs(posAmt))
		var markPrice float64
		if posAmt != 0 {
			markPrice = positionValue / posAmt
		}

		result = append(result, types.Position{
			Symbol:           symbol,
			Side:             side,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			Quantity:         posAmt,
			UnrealizedPnL:    unrealizedPnl,
			Leverage:         position.Leverage.Value,
			LiquidationPrice: liquidationPx,
		})
	}

	// Also get xyz dex positions (stocks, forex, commodities)
	_, _, xyzPositions, err := t.getXYZDexBalance()
	if err != nil {
		// xyz dex query failed - log warning but don't fail
		logger.Infof("⚠️  Failed to get xyz dex positions: %v", err)
	} else {
		for _, pos := range xyzPositions {
			posAmt, _ := strconv.ParseFloat(pos.Position.Szi, 64)
			if posAmt == 0 {
				continue
			}

			// xyz dex positions - the API returns coin names with xyz: prefix (e.g., "xyz:SILVER")
			// Only add prefix if not already present
			symbol := pos.Position.Coin
			if !strings.HasPrefix(symbol, "xyz:") {
				symbol = "xyz:" + symbol
			}

			side := "long"
			if posAmt < 0 {
				side = "short"
				posAmt = -posAmt
			}

			// Parse price information
			var entryPrice, liquidationPx float64
			if pos.Position.EntryPx != nil {
				entryPrice, _ = strconv.ParseFloat(*pos.Position.EntryPx, 64)
			}
			if pos.Position.LiquidationPx != nil {
				liquidationPx, _ = strconv.ParseFloat(*pos.Position.LiquidationPx, 64)
			}

			positionValue, _ := strconv.ParseFloat(pos.Position.PositionValue, 64)
			unrealizedPnl, _ := strconv.ParseFloat(pos.Position.UnrealizedPnl, 64)

			// Calculate mark price from position value
			var markPrice float64
			if posAmt != 0 {
				markPrice = positionValue / posAmt
			}

			// Get leverage (default to 1 if not available)
			leverage := pos.Position.Leverage.Value
			if leverage == 0 {
				leverage = 1
			}

			result = append(result, types.Position{
				Symbol:           symbol,
				Side:             side,
				EntryPrice:       entryPrice,
				MarkPrice:        markPrice,
				Quantity:         posAmt,
				UnrealizedPnL:    unrealizedPnl,
				Leverage:         leverage,
				LiquidationPrice: liquidationPx,
			})
		}
	}

	return result, nil
}

// SetMarginMode sets margin mode (set together with SetLeverage)
func (t *HyperliquidTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	// Hyperliquid's margin mode is set in SetLeverage, only record here
	t.isCrossMargin = isCrossMargin
	marginModeStr := "cross margin"
	if !isCrossMargin {
		marginModeStr = "isolated margin"
	}
	logger.Infof("  ✓ %s will use %s mode", symbol, marginModeStr)
	return nil
}

// SetLeverage sets leverage
func (t *HyperliquidTrader) SetLeverage(symbol string, leverage int) error {
	// Hyperliquid symbol format (remove USDT suffix)
	coin := convertSymbolToHyperliquid(symbol)

	// Call UpdateLeverage (leverage int, name string, isCross bool)
	// Third parameter: true=cross margin mode, false=isolated margin mode
	_, err := t.exchange.UpdateLeverage(t.ctx, leverage, coin, t.isCrossMargin)
	if err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	logger.Infof("  ✓ %s leverage switched to %dx", symbol, leverage)
	return nil
}
