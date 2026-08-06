package kucoin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"fxos/logger"
	"fxos/trader/types"
)

// KuCoinTrader implements GridTrader natively (limit orders, single-order
// cancel, order book), so the GridTraderAdapter stop-order fallback is
// never used for KuCoin.
var _ types.GridTrader = (*KuCoinTrader)(nil)

// PlaceLimitOrder places a native limit order on KuCoin Futures. KuCoin
// uses one-way position mode; the order side is taken from req.Side. This
// replaces the GridTraderAdapter's stop-order emulation.
func (t *KuCoinTrader) PlaceLimitOrder(req *types.LimitOrderRequest) (*types.LimitOrderResult, error) {
	// Set leverage if specified
	if req.Leverage > 0 {
		if err := t.SetLeverage(req.Symbol, req.Leverage); err != nil {
			logger.Infof("⚠️ Failed to set leverage: %v", err)
		}
	}

	kcSymbol := t.convertSymbol(req.Symbol)

	// Convert quantity to lots
	lots, err := t.quantityToLots(req.Symbol, req.Quantity)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate lots: %w", err)
	}

	side := "buy"
	if req.Side == "SELL" {
		side = "sell"
	}

	body := map[string]interface{}{
		"clientOid":  fmt.Sprintf("nfx%d", time.Now().UnixNano()),
		"symbol":     kcSymbol,
		"side":       side,
		"type":       "limit",
		"size":       lots,
		"price":      strconv.FormatFloat(req.Price, 'f', -1, 64),
		"marginMode": "CROSS", // Use cross margin mode
	}
	if req.PostOnly {
		body["postOnly"] = true
	}
	if req.ReduceOnly {
		body["reduceOnly"] = true
	}

	data, err := t.doRequest("POST", kucoinOrderPath, body)
	if err != nil {
		return nil, fmt.Errorf("failed to place limit order: %w", err)
	}

	var result struct {
		OrderId string `json:"orderId"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse order response: %w", err)
	}

	logger.Infof("✓ KuCoin limit order placed: %s side=%s lots=%d price=%s orderId=%s",
		req.Symbol, side, lots, body["price"], result.OrderId)

	return &types.LimitOrderResult{
		OrderID:      result.OrderId,
		ClientID:     req.ClientID,
		Symbol:       req.Symbol,
		Side:         req.Side,
		PositionSide: req.PositionSide,
		Price:        req.Price,
		Quantity:     req.Quantity,
		Status:       "NEW",
	}, nil
}

// CancelOrder cancels a single order by its KuCoin order ID.
func (t *KuCoinTrader) CancelOrder(symbol, orderID string) error {
	path := fmt.Sprintf("%s/%s", kucoinOrderPath, orderID)
	_, err := t.doRequest("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("failed to cancel order %s: %w", orderID, err)
	}
	logger.Infof("✓ KuCoin cancelled order: %s", orderID)
	return nil
}

// GetOrderBook returns the current order book as [price, size] pairs,
// best levels first.
func (t *KuCoinTrader) GetOrderBook(symbol string, depth int) (bids, asks [][]float64, err error) {
	if depth <= 0 {
		depth = 20
	}
	path := fmt.Sprintf("/api/v1/market/orderbook/level2?symbol=%s&limit=%d", t.convertSymbol(symbol), depth)

	data, err := t.doRequest("GET", path, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get order book: %w", err)
	}

	var resp struct {
		Data struct {
			Bids [][]string `json:"bids"`
			Asks [][]string `json:"asks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse order book: %w", err)
	}

	parseLevels := func(levels [][]string) [][]float64 {
		var out [][]float64
		for _, lvl := range levels {
			if len(lvl) < 2 {
				continue
			}
			price, perr := strconv.ParseFloat(lvl[0], 64)
			size, serr := strconv.ParseFloat(lvl[1], 64)
			if perr != nil || serr != nil {
				continue
			}
			out = append(out, []float64{price, size})
		}
		return out
	}

	return parseLevels(resp.Data.Bids), parseLevels(resp.Data.Asks), nil
}
