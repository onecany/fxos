package gate

import (
	"fmt"
	"fxos/logger"
	"fxos/market"
	"fxos/store"
	"fxos/trader/syncloop"
	"fxos/trader/types"
	"strconv"
	"strings"
	"time"

	"github.com/antihax/optional"
	"github.com/gateio/gateapi-go/v6"
)

// GateTrade represents a trade record from Gate fill history
type GateTrade struct {
	Symbol      string
	TradeID     string
	OrderID     string
	Side        string // buy or sell
	FillPrice   float64
	FillQty     float64 // In base currency (e.g., ETH), not contracts
	Fee         float64
	FeeAsset    string
	ExecTime    time.Time
	ProfitLoss  float64
	OrderType   string
	OrderAction string // open_long, open_short, close_long, close_short
}

// GetTrades retrieves trade/fill records from Gate
func (t *GateTrader) GetTrades(startTime time.Time, limit int) ([]GateTrade, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100 // Gate max limit
	}

	opts := &gateapi.GetMyTradesOpts{
		Limit: optional.NewInt32(int32(limit)),
	}

	// Get trades from Gate API
	trades, _, err := t.client.FuturesApi.GetMyTrades(t.ctx, "usdt", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade history: %w", err)
	}

	logger.Infof("📥 Received %d trades from Gate", len(trades))

	result := make([]GateTrade, 0, len(trades))

	for _, trade := range trades {
		// Filter by start time
		createTime := int64(trade.CreateTime)
		if createTime < startTime.Unix() {
			continue
		}

		fillPrice, err := strconv.ParseFloat(trade.Price, 64)
		if err != nil || fillPrice == 0 {
			logger.Infof("⚠️  Gate trade %d: fillPrice parse issue - raw='%s' parsed=%.8f err=%v",
				trade.Id, trade.Price, fillPrice, err)
		}

		// Get quanto_multiplier for this contract to convert size to base currency
		quantoMultiplier := 1.0
		contract, err := t.getContract(trade.Contract)
		if err == nil && contract != nil {
			qm, _ := strconv.ParseFloat(contract.QuantoMultiplier, 64)
			if qm > 0 {
				quantoMultiplier = qm
			}
		}

		// Convert contract size to actual quantity
		absSize := trade.Size
		if absSize < 0 {
			absSize = -absSize
		}
		fillQty := float64(absSize) * quantoMultiplier

		// Determine side and order action based on size and close_size
		// Gate close_size field determines if trade is opening or closing:
		// close_size=0 && size>0: Open long
		// close_size=0 && size<0: Open short
		// close_size>0 && size>0: Close short (and possibly open long if size > close_size)
		// close_size<0 && size<0: Close long (and possibly open short if |size| > |close_size|)
		side := "BUY"
		orderAction := types.ActionOpenLong

		if trade.Size > 0 {
			side = "BUY"
			if trade.CloseSize > 0 {
				// Closing short position
				orderAction = types.ActionCloseShort
			} else {
				// Opening long position
				orderAction = types.ActionOpenLong
			}
		} else if trade.Size < 0 {
			side = "SELL"
			if trade.CloseSize < 0 {
				// Closing long position
				orderAction = types.ActionCloseLong
			} else {
				// Opening short position
				orderAction = types.ActionOpenShort
			}
		}

		// Calculate fee (Gate returns fee as negative value)
		fee, _ := strconv.ParseFloat(trade.Fee, 64)
		if fee < 0 {
			fee = -fee
		}

		// For closed positions, estimate PnL (Gate doesn't directly provide it in trade record)
		pnl := 0.0
		if strings.Contains(orderAction, "close") {
			// PnL would need to be calculated from position history
			// For now, we leave it as 0 and let position builder handle it
		}

		gateTrade := GateTrade{
			Symbol:      trade.Contract,
			TradeID:     fmt.Sprintf("%d", trade.Id),
			OrderID:     trade.OrderId,
			Side:        side,
			FillPrice:   fillPrice,
			FillQty:     fillQty,
			Fee:         fee,
			FeeAsset:    "USDT",
			ExecTime:    time.Unix(createTime, 0).UTC(),
			ProfitLoss:  pnl,
			OrderType:   "MARKET",
			OrderAction: orderAction,
		}

		result = append(result, gateTrade)
	}

	return result, nil
}

// toTradeRecord converts a GateTrade to the unified TradeRecord format.
// The symbol keeps its underscore form here (BTC_USDT); the shared engine
// runs market.Normalize which strips "_" (equivalent to the old explicit
// strings.ReplaceAll before Normalize). Position side is inferred from the
// order action, matching the value the position builder used to receive.
func (g GateTrade) toTradeRecord() types.TradeRecord {
	// Determine position side from order action (Gate uses one-way mode)
	positionSide := "LONG"
	if strings.Contains(g.OrderAction, types.SideShort) {
		positionSide = "SHORT"
	}

	return types.TradeRecord{
		TradeID:      g.TradeID,
		Symbol:       g.Symbol,
		Side:         g.Side,
		PositionSide: positionSide,
		OrderAction:  g.OrderAction,
		OrderType:    g.OrderType,
		Price:        g.FillPrice,
		Quantity:     g.FillQty,
		Fee:          g.Fee,
		RealizedPnL:  g.ProfitLoss,
		Time:         g.ExecTime,
	}
}

// retryClosePositionUpdates is Gate-specific compensation logic: when a
// close trade's order already exists but its position update previously
// failed, retry the position update. The shared sync engine skips existing
// orders entirely, so this retry stays exchange-specific.
func (t *GateTrader) retryClosePositionUpdates(traderID string, exchangeID string, exchangeType string, st *store.Store, trades []GateTrade) {
	orderStore := st.Order()
	posBuilder := store.NewPositionBuilder(st.Position())

	for _, trade := range trades {
		if !strings.HasPrefix(trade.OrderAction, "close_") || trade.FillPrice <= 0 {
			continue
		}

		// Check if trade already exists (use exchangeID which is UUID, not exchange type)
		existing, err := orderStore.GetOrderByExchangeID(exchangeID, trade.TradeID)
		if err != nil || existing == nil {
			continue
		}

		// Normalize symbol (Gate uses BTC_USDT, normalize to BTCUSDT)
		symbol := market.Normalize(strings.ReplaceAll(trade.Symbol, "_", ""))

		// Determine position side from order action
		positionSide := "LONG"
		if strings.Contains(trade.OrderAction, types.SideShort) {
			positionSide = "SHORT"
		}

		if err := posBuilder.ProcessTrade(
			traderID, exchangeID, exchangeType,
			symbol, positionSide, trade.OrderAction,
			trade.FillQty, trade.FillPrice, trade.Fee, trade.ProfitLoss,
			trade.ExecTime.UTC().UnixMilli(), trade.TradeID,
		); err != nil {
			logger.Infof("  ⚠️ Retry position update for existing trade %s failed: %v", trade.TradeID, err)
		}
	}
}

// SyncOrdersFromGate syncs Gate exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("gate")
func (t *GateTrader) SyncOrdersFromGate(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	// Sync cursor: memory + DB recovery; falls back to a 24h lookback on first sync
	nowMs := time.Now().UTC().UnixMilli()
	startTime := time.UnixMilli(t.syncCursor.GetOrInit(exchangeID, st, nowMs))

	logger.Infof("🔄 Syncing Gate trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 100)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from Gate", len(trades))

	// Gate-specific: retry position updates for close trades whose order
	// already exists but whose position update previously failed.
	t.retryClosePositionUpdates(traderID, exchangeID, exchangeType, st, trades)

	// Convert to unified TradeRecord format and persist via the shared
	// sync engine (sort ASC, dedup, symbol normalization, order/fill/
	// position records). Gate returns one-way-mode fills; the position
	// side is inferred per trade (see toTradeRecord).
	records := make([]types.TradeRecord, 0, len(trades))
	for _, trade := range trades {
		records = append(records, trade.toTradeRecord())
	}

	syncedCount, skippedCount := syncloop.PersistTrades(st, records, syncloop.PersistOptions{
		TraderID:               traderID,
		ExchangeID:             exchangeID,
		ExchangeType:           exchangeType,
		PositionSideFallback:   "LONG",
		SideNormalize:          true,
		DefaultCommissionAsset: "USDT",
	})

	// Advance the sync cursor to the latest processed trade (only after a
	// fully successful sync so failures retry from the same position).
	if len(trades) > 0 {
		t.syncCursor.Advance(exchangeID, trades[len(trades)-1].ExecTime.UTC().UnixMilli())
	}

	logger.Infof("✅ Gate order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// StartOrderSync starts background order sync task for Gate
func (t *GateTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "Gate", func() error {
		return t.SyncOrdersFromGate(traderID, exchangeID, exchangeType, st)
	})
}
