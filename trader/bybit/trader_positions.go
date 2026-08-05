package bybit

import (
	"context"
	"fmt"
	"fxos/logger"
	"fxos/trader/types"
	"strconv"
	"strings"
	"time"
)

// GetPositions retrieves all positions
func (t *BybitTrader) GetPositions() ([]types.Position, error) {
	// Check cache
	t.positionsCacheMutex.RLock()
	if t.cachedPositions != nil && time.Since(t.positionsCacheTime) < t.cacheDuration {
		positions := t.cachedPositions
		t.positionsCacheMutex.RUnlock()
		return positions, nil
	}
	t.positionsCacheMutex.RUnlock()

	// Call API
	params := map[string]interface{}{
		"category":   "linear",
		"settleCoin": "USDT",
	}

	result, err := t.client.NewUtaBybitServiceWithParams(params).GetPositionList(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Bybit positions: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("Bybit API error: %s", result.RetMsg)
	}

	resultData, ok := result.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Bybit positions return format error")
	}

	list, _ := resultData["list"].([]interface{})

	var positions []types.Position

	for _, item := range list {
		pos, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		symbol, _ := pos["symbol"].(string)

		sizeStr, _ := pos["size"].(string)
		size, _ := strconv.ParseFloat(sizeStr, 64)

		// Skip empty positions
		if size == 0 {
			continue
		}

		entryPriceStr, _ := pos["avgPrice"].(string)
		entryPrice, _ := strconv.ParseFloat(entryPriceStr, 64)

		unrealisedPnlStr, _ := pos["unrealisedPnl"].(string)
		unrealisedPnl, _ := strconv.ParseFloat(unrealisedPnlStr, 64)

		leverageStr, _ := pos["leverage"].(string)
		leverage, _ := strconv.ParseFloat(leverageStr, 64)

		// Mark price
		markPriceStr, _ := pos["markPrice"].(string)
		markPrice, _ := strconv.ParseFloat(markPriceStr, 64)

		// Liquidation price
		liqPriceStr, _ := pos["liqPrice"].(string)
		liqPrice, _ := strconv.ParseFloat(liqPriceStr, 64)

		// Position created time (milliseconds timestamp)
		createdTimeStr, _ := pos["createdTime"].(string)
		createdTime, _ := strconv.ParseInt(createdTimeStr, 10, 64)

		positionSide, _ := pos["side"].(string) // Buy = long, Sell = short

		// Log raw position data for debugging
		logger.Infof("[Bybit] GetPositions raw: symbol=%v, side=%s, size=%v", symbol, positionSide, sizeStr)

		// Convert to unified format (use lowercase for consistency with other exchanges)
		// Bybit returns "Buy" for long, "Sell" for short
		side := "long"
		positionSideLower := strings.ToLower(positionSide)
		if positionSideLower == "sell" {
			side = "short"
		}

		logger.Infof("[Bybit] GetPositions converted: symbol=%v, rawSide=%s -> side=%s", symbol, positionSide, side)

		position := types.Position{
			Symbol:           symbol,
			Side:             side,
			Quantity:         size, // Always positive (short positions normalized to positive)
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			UnrealizedPnL:    unrealisedPnl,
			Leverage:         int(leverage),
			LiquidationPrice: liqPrice,
			CreatedTime:      createdTime, // Position open time (ms)
		}

		positions = append(positions, position)
	}

	// Update cache
	t.positionsCacheMutex.Lock()
	t.cachedPositions = positions
	t.positionsCacheTime = time.Now()
	t.positionsCacheMutex.Unlock()

	return positions, nil
}
