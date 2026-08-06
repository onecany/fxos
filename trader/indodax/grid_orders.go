package indodax

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"fxos/logger"
	"fxos/trader/types"
)

// IndodaxTrader implements GridTrader natively (limit orders, single-order
// cancel, order book), so the GridTraderAdapter stop-order fallback is
// never used for Indodax.
var _ types.GridTrader = (*IndodaxTrader)(nil)

// PlaceLimitOrder places a native limit order on Indodax spot. Indodax
// prices are in IDR (integer); the order rests on the book until filled.
// This replaces the GridTraderAdapter's stop-order emulation.
func (t *IndodaxTrader) PlaceLimitOrder(req *types.LimitOrderRequest) (*types.LimitOrderResult, error) {
	t.clearCache()

	pair := t.convertSymbol(req.Symbol)
	coin := t.getCoinFromSymbol(req.Symbol)

	side := "buy"
	if req.Side == "SELL" {
		side = "sell"
	}

	params := url.Values{}
	params.Set("method", "trade")
	params.Set("pair", pair)
	params.Set("type", side)
	params.Set("price", strconv.FormatFloat(req.Price, 'f', 0, 64))
	params.Set(coin, strconv.FormatFloat(req.Quantity, 'f', 8, 64))
	params.Set("order_type", "limit")

	data, err := t.doPrivateRequest(params)
	if err != nil {
		return nil, fmt.Errorf("failed to place limit order: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse trade response: %w", err)
	}

	logger.Infof("[Indodax] Limit order placed: %s side=%s qty=%.8f price=%.0f",
		req.Symbol, side, req.Quantity, req.Price)

	status := "NEW"
	if s, ok := result["status"].(string); ok && s == "success" {
		status = "NEW"
	} else if id, ok := result["order_id"]; ok && id != nil {
		status = "NEW"
	}

	return &types.LimitOrderResult{
		OrderID:      fmt.Sprintf("%v", result["order_id"]),
		ClientID:     req.ClientID,
		Symbol:       req.Symbol,
		Side:         req.Side,
		PositionSide: req.PositionSide,
		Price:        req.Price,
		Quantity:     req.Quantity,
		Status:       status,
	}, nil
}

// CancelOrder cancels a single order by its Indodax order ID. Indodax's
// cancelOrder method requires the order side, so it is looked up from the
// open-orders list first.
func (t *IndodaxTrader) CancelOrder(symbol, orderID string) error {
	t.clearCache()

	pair := t.convertSymbol(symbol)

	// Look up the order type (buy/sell) from open orders
	orderType, err := t.lookupOpenOrderType(pair, orderID)
	if err != nil {
		return fmt.Errorf("failed to look up order %s: %w", orderID, err)
	}
	if orderType == "" {
		return fmt.Errorf("order %s not found among open orders", orderID)
	}

	params := url.Values{}
	params.Set("method", "cancelOrder")
	params.Set("pair", pair)
	params.Set("order_id", orderID)
	params.Set("type", orderType)

	if _, err := t.doPrivateRequest(params); err != nil {
		return fmt.Errorf("failed to cancel order %s: %w", orderID, err)
	}

	logger.Infof("[Indodax] Cancelled order: %s (%s)", orderID, orderType)
	return nil
}

// lookupOpenOrderType finds the side (buy/sell) of an open order by ID.
func (t *IndodaxTrader) lookupOpenOrderType(pair, orderID string) (string, error) {
	params := url.Values{}
	params.Set("method", "openOrders")
	params.Set("pair", pair)

	data, err := t.doPrivateRequest(params)
	if err != nil {
		return "", fmt.Errorf("failed to get open orders: %w", err)
	}

	var result struct {
		Orders []struct {
			OrderID json.Number `json:"order_id"`
			Type    string      `json:"type"`
		} `json:"orders"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to parse open orders: %w", err)
	}

	for _, order := range result.Orders {
		if order.OrderID.String() == orderID {
			return order.Type, nil
		}
	}
	return "", nil
}

// GetOrderBook returns the current spot order book as [price, size] pairs,
// best levels first (buy = bids, sell = asks).
func (t *IndodaxTrader) GetOrderBook(symbol string, depth int) (bids, asks [][]float64, err error) {
	if depth <= 0 {
		depth = 20
	}
	pairID := strings.ToLower(strings.ReplaceAll(t.convertSymbol(symbol), "_", ""))

	data, err := t.doPublicRequest("/depth/" + pairID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get order book: %w", err)
	}

	var resp struct {
		Buy  [][]string `json:"buy"`
		Sell [][]string `json:"sell"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse order book: %w", err)
	}

	parseLevels := func(levels [][]string) [][]float64 {
		var out [][]float64
		for i, lvl := range levels {
			if i >= depth || len(lvl) < 2 {
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

	return parseLevels(resp.Buy), parseLevels(resp.Sell), nil
}
