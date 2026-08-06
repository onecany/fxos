package syncloop

import (
	"sort"
	"strings"

	"fxos/logger"
	"fxos/market"
	"fxos/store"
	"fxos/trader/types"
)

// PersistOptions controls how trades are written to the store. The fields
// capture the per-exchange differences that used to be copy-pasted across
// nine order_sync.go files (dedup check, symbol normalization, order-action
// mapping, position-side inference, fill metadata).
type PersistOptions struct {
	TraderID     string
	ExchangeID   string
	ExchangeType string

	// PositionSideFallback is used when a trade has no usable position
	// side: it is inferred from the order action ("open_long" -> LONG).
	// Pass "BOTH" to keep one-way-mode semantics as-is.
	PositionSideFallback string

	// SideNormalize uppercases the trade side before persisting
	// (binance returns lowercase; hyperliquid already uppercases).
	SideNormalize bool

	// DefaultCommissionAsset for fill records when the trade does not
	// carry an asset (and CommissionAssetFunc is nil).
	DefaultCommissionAsset string

	// CommissionAssetFunc returns the fill commission asset per trade;
	// overrides DefaultCommissionAsset when set (bitget/gate/kucoin
	// report a per-trade fee asset).
	CommissionAssetFunc func(trade types.TradeRecord) string

	// DetermineOrderAction maps a trade to an order action; when nil the
	// trade's own OrderAction field is used (hyperliquid parses it from
	// the exchange Dir field).
	DetermineOrderAction func(trade types.TradeRecord) string
}

// PersistTrades dedups, normalizes and writes trades to the order, fill
// and position stores. Trades are sorted by time ASC (oldest first) for
// correct position building, mirroring the per-exchange logic it replaces.
// Returns the number of new and skipped (already-existing) trades.
func PersistTrades(st *store.Store, trades []types.TradeRecord, opts PersistOptions) (synced, skipped int) {
	if st == nil || len(trades) == 0 {
		return 0, 0
	}

	// Sort trades by time ASC (oldest first) for proper position building
	sort.Slice(trades, func(i, j int) bool {
		return trades[i].Time.UnixMilli() < trades[j].Time.UnixMilli()
	})

	orderStore := st.Order()
	positionStore := st.Position()
	posBuilder := store.NewPositionBuilder(positionStore)

	for _, trade := range trades {
		// Check if trade already exists (use exchangeID which is UUID)
		existing, err := orderStore.GetOrderByExchangeID(opts.ExchangeID, trade.TradeID)
		if err == nil && existing != nil {
			skipped++
			continue // Order already exists, skip
		}

		// Normalize symbol
		symbol := market.Normalize(trade.Symbol)

		// Determine order action (exchange-specific mapping or trade's own)
		orderAction := trade.OrderAction
		if opts.DetermineOrderAction != nil {
			orderAction = opts.DetermineOrderAction(trade)
		}

		// Determine position side for the position builder
		positionSide := trade.PositionSide
		if positionSide == "" || positionSide == "BOTH" {
			if opts.PositionSideFallback == "BOTH" {
				positionSide = "BOTH"
			} else if strings.Contains(orderAction, types.SideLong) {
				positionSide = "LONG"
			} else {
				positionSide = "SHORT"
			}
		}

		// Normalize side
		side := trade.Side
		if opts.SideNormalize {
			side = strings.ToUpper(side)
		}

		// Create order record - use Unix milliseconds UTC
		tradeTimeMs := trade.Time.UTC().UnixMilli()
		orderType := trade.OrderType
		if orderType == "" {
			orderType = "MARKET"
		}
		orderRecord := &store.TraderOrder{
			TraderID:        opts.TraderID,
			ExchangeID:      opts.ExchangeID,
			ExchangeType:    opts.ExchangeType,
			ExchangeOrderID: trade.TradeID,
			Symbol:          symbol,
			Side:            side,
			PositionSide:    positionSide,
			Type:            orderType,
			OrderAction:     orderAction,
			Quantity:        trade.Quantity,
			Price:           trade.Price,
			Status:          "FILLED",
			FilledQuantity:  trade.Quantity,
			AvgFillPrice:    trade.Price,
			Commission:      trade.Fee,
			FilledAt:        tradeTimeMs,
			CreatedAt:       tradeTimeMs,
			UpdatedAt:       tradeTimeMs,
		}

		// Insert order record
		if err := orderStore.CreateOrder(orderRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync trade %s: %v", trade.TradeID, err)
			continue
		}

		// Create fill record
		commissionAsset := opts.DefaultCommissionAsset
		if opts.CommissionAssetFunc != nil {
			commissionAsset = opts.CommissionAssetFunc(trade)
		}
		if commissionAsset == "" {
			commissionAsset = "USDT"
		}
		fillRecord := &store.TraderFill{
			TraderID:        opts.TraderID,
			ExchangeID:      opts.ExchangeID,
			ExchangeType:    opts.ExchangeType,
			OrderID:         orderRecord.ID,
			ExchangeOrderID: trade.TradeID,
			ExchangeTradeID: trade.TradeID,
			Symbol:          symbol,
			Side:            side,
			Price:           trade.Price,
			Quantity:        trade.Quantity,
			QuoteQuantity:   trade.Price * trade.Quantity,
			Commission:      trade.Fee,
			CommissionAsset: commissionAsset,
			RealizedPnL:     trade.RealizedPnL,
			IsMaker:         trade.IsMaker,
			CreatedAt:       tradeTimeMs,
		}

		if err := orderStore.CreateFill(fillRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync fill for trade %s: %v", trade.TradeID, err)
		}

		// Create/update position record using PositionBuilder
		if err := posBuilder.ProcessTrade(
			opts.TraderID, opts.ExchangeID, opts.ExchangeType,
			symbol, positionSide, orderAction,
			trade.Quantity, trade.Price, trade.Fee, trade.RealizedPnL,
			tradeTimeMs, trade.TradeID,
		); err != nil {
			logger.Infof("  ⚠️ Failed to sync position for trade %s: %v", trade.TradeID, err)
		}

		synced++
		logger.Infof("  ✅ Synced trade: %s %s %s qty=%.6f price=%.6f pnl=%.2f fee=%.6f action=%s time=%s(UTC)",
			trade.TradeID, symbol, side, trade.Quantity, trade.Price, trade.RealizedPnL, trade.Fee, orderAction,
			trade.Time.UTC().Format("01-02 15:04:05"))
	}

	logger.Infof("✅ Order sync persisted: %d new trades synced, %d skipped (already exist)", synced, skipped)
	return synced, skipped
}
