package gate

import (
	"fmt"
	"strconv"

	"fxos/logger"
	"fxos/trader/types"

	"github.com/antihax/optional"
	"github.com/gateio/gateapi-go/v6"
)

// GateTrader implements GridTrader natively (limit orders, single-order
// cancel, order book), so the GridTraderAdapter stop-order fallback is
// never used for Gate.
var _ types.GridTrader = (*GateTrader)(nil)

// PlaceLimitOrder places a native limit order on Gate Futures. Gate uses
// one-way position mode: positive size = long/buy, negative size =
// short/sell. This replaces the GridTraderAdapter's stop-order emulation.
func (t *GateTrader) PlaceLimitOrder(req *types.LimitOrderRequest) (*types.LimitOrderResult, error) {
	symbol := t.convertSymbol(req.Symbol)

	// Set leverage if specified
	if req.Leverage > 0 {
		if err := t.SetLeverage(symbol, req.Leverage); err != nil {
			logger.Warnf("  [Gate] Failed to set leverage: %v", err)
		}
	}

	// Get contract info for size calculation (each contract = quanto_multiplier base currency)
	contract, err := t.getContract(symbol)
	if err != nil {
		return nil, err
	}
	quantoMultiplier, _ := strconv.ParseFloat(contract.QuantoMultiplier, 64)
	size := int64(req.Quantity / quantoMultiplier)
	if size <= 0 {
		size = 1
	}
	if req.Side == "SELL" {
		size = -size
	}

	// Maker-only orders use "poc" (post-only), otherwise keep the order
	// resting as "gtc".
	tif := "gtc"
	if req.PostOnly {
		tif = "poc"
	}

	order := gateapi.FuturesOrder{
		Contract:   symbol,
		Size:       size,
		Price:      strconv.FormatFloat(req.Price, 'f', -1, 64),
		Tif:        tif,
		Text:       "t-fxos-grid",
		ReduceOnly: req.ReduceOnly,
	}

	logger.Infof("  [Gate] PlaceLimitOrder: symbol=%s, size=%d, price=%s, tif=%s", symbol, size, order.Price, tif)

	result, _, err := t.client.FuturesApi.CreateFuturesOrder(t.ctx, "usdt", order, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to place limit order: %w", err)
	}

	t.clearCache()

	return &types.LimitOrderResult{
		OrderID:      strconv.FormatInt(result.Id, 10),
		ClientID:     req.ClientID,
		Symbol:       req.Symbol,
		Side:         req.Side,
		PositionSide: req.PositionSide,
		Price:        req.Price,
		Quantity:     req.Quantity,
		Status:       "NEW",
	}, nil
}

// CancelOrder cancels a single order by its Gate order ID.
func (t *GateTrader) CancelOrder(symbol, orderID string) error {
	_, _, err := t.client.FuturesApi.CancelFuturesOrder(t.ctx, "usdt", orderID, nil)
	if err != nil {
		return fmt.Errorf("failed to cancel order %s: %w", orderID, err)
	}
	logger.Infof("  [Gate] Cancelled order: %s", orderID)
	return nil
}

// GetOrderBook returns the current futures order book as [price, size]
// pairs, best levels first.
func (t *GateTrader) GetOrderBook(symbol string, depth int) (bids, asks [][]float64, err error) {
	if depth <= 0 {
		depth = 20
	}
	book, _, err := t.client.FuturesApi.ListFuturesOrderBook(t.ctx, "usdt", t.convertSymbol(symbol), &gateapi.ListFuturesOrderBookOpts{
		Limit: optional.NewInt32(int32(depth)),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get order book: %w", err)
	}

	for _, item := range book.Bids {
		price, perr := strconv.ParseFloat(item.P, 64)
		if perr != nil {
			continue
		}
		bids = append(bids, []float64{price, float64(item.S)})
	}
	for _, item := range book.Asks {
		price, perr := strconv.ParseFloat(item.P, 64)
		if perr != nil {
			continue
		}
		asks = append(asks, []float64{price, float64(item.S)})
	}
	return bids, asks, nil
}
