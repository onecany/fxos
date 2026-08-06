package okx

import (
	"encoding/json"
	"fmt"
	"fxos/logger"
	"fxos/store"
	"fxos/trader/syncloop"
	"fxos/trader/types"
	"strconv"
	"strings"
	"time"
)

// OKXTrade represents a trade record from OKX fills history
type OKXTrade struct {
	InstID      string
	Symbol      string
	TradeID     string
	OrderID     string
	Side        string // buy or sell
	PosSide     string // long or short
	FillPrice   float64
	FillQty     float64 // In contracts
	FillQtyBase float64 // In base asset (BTC, ETH, etc)
	Fee         float64
	FeeAsset    string
	ExecTime    time.Time
	IsMaker     bool
	OrderType   string
	OrderAction string // open_long, open_short, close_long, close_short
}

// GetTrades retrieves trade/fill records from OKX
func (t *OKXTrader) GetTrades(startTime time.Time, limit int) ([]OKXTrade, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100 // OKX max limit is 100
	}

	// Build query path
	// OKX fills-history endpoint for historical fills
	path := fmt.Sprintf("/api/v5/trade/fills-history?instType=SWAP&limit=%d", limit)
	if !startTime.IsZero() {
		path += fmt.Sprintf("&begin=%d", startTime.UnixMilli())
	}

	data, err := t.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get fills history: %w", err)
	}

	var fills []struct {
		InstID   string `json:"instId"`   // e.g., "BTC-USDT-SWAP"
		TradeID  string `json:"tradeId"`  // Trade ID
		OrdID    string `json:"ordId"`    // Order ID
		BillID   string `json:"billId"`   // Bill ID
		Side     string `json:"side"`     // buy or sell
		PosSide  string `json:"posSide"`  // long, short, or net
		FillPx   string `json:"fillPx"`   // Fill price
		FillSz   string `json:"fillSz"`   // Fill size (contracts)
		Fee      string `json:"fee"`      // Fee (negative for cost)
		FeeCcy   string `json:"feeCcy"`   // Fee currency
		Ts       string `json:"ts"`       // Trade timestamp (ms)
		ExecType string `json:"execType"` // T: taker, M: maker
		Tag      string `json:"tag"`      // Order tag
	}

	if err := json.Unmarshal(data, &fills); err != nil {
		return nil, fmt.Errorf("failed to parse fills: %w", err)
	}

	trades := make([]OKXTrade, 0, len(fills))

	for _, fill := range fills {
		fillPrice, _ := strconv.ParseFloat(fill.FillPx, 64)
		fillSz, _ := strconv.ParseFloat(fill.FillSz, 64)
		fee, _ := strconv.ParseFloat(fill.Fee, 64)
		ts, _ := strconv.ParseInt(fill.Ts, 10, 64)

		// Convert symbol: BTC-USDT-SWAP -> BTCUSDT
		symbol := t.convertSymbolBack(fill.InstID)

		// Convert contract count to base asset quantity
		fillQtyBase := fillSz
		inst, err := t.getInstrument(symbol)
		if err == nil && inst.CtVal > 0 {
			fillQtyBase = fillSz * inst.CtVal
		}

		// Determine order action based on side and posSide
		// OKX uses dual position mode:
		// - buy + long = open long
		// - sell + long = close long
		// - sell + short = open short
		// - buy + short = close short
		orderAction := types.ActionOpenLong
		posSide := strings.ToLower(fill.PosSide)
		side := strings.ToLower(fill.Side)

		if posSide == types.SideLong {
			if side == "buy" {
				orderAction = types.ActionOpenLong
			} else {
				orderAction = types.ActionCloseLong
			}
		} else if posSide == types.SideShort {
			if side == "sell" {
				orderAction = types.ActionOpenShort
			} else {
				orderAction = types.ActionCloseShort
			}
		} else {
			// One-way mode (net position)
			if side == "buy" {
				orderAction = types.ActionOpenLong
			} else {
				orderAction = types.ActionOpenShort
			}
		}

		trade := OKXTrade{
			InstID:      fill.InstID,
			Symbol:      symbol,
			TradeID:     fill.TradeID,
			OrderID:     fill.OrdID,
			Side:        fill.Side,
			PosSide:     fill.PosSide,
			FillPrice:   fillPrice,
			FillQty:     fillSz,
			FillQtyBase: fillQtyBase,
			Fee:         -fee, // OKX returns negative fee
			FeeAsset:    fill.FeeCcy,
			ExecTime:    time.UnixMilli(ts).UTC(),
			IsMaker:     fill.ExecType == "M",
			OrderType:   "MARKET",
			OrderAction: orderAction,
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// SyncOrdersFromOKX syncs OKX exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("okx")
func (t *OKXTrader) SyncOrdersFromOKX(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	// Sync cursor: memory + DB recovery; falls back to a 24h lookback on first sync
	nowMs := time.Now().UTC().UnixMilli()
	startTime := time.UnixMilli(t.syncCursor.GetOrInit(exchangeID, st, nowMs))

	logger.Infof("🔄 Syncing OKX trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 100)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from OKX", len(trades))

	// Convert exchange trades to the shared TradeRecord shape and persist via
	// the shared sync engine (sort ASC, dedup, symbol normalization,
	// order/fill/position records). OKX fills report quantity in base asset
	// terms (contracts * contract value) and carry no per-trade PnL; the
	// position side is inferred from the order action (matches the original
	// inference logic). The per-fill fee currency (feeCcy) is carried
	// through CommissionAssetFunc.
	feeAssetByTrade := make(map[string]string, len(trades))
	records := make([]types.TradeRecord, 0, len(trades))
	for _, trade := range trades {
		feeAssetByTrade[trade.TradeID] = trade.FeeAsset
		records = append(records, types.TradeRecord{
			TradeID:      trade.TradeID,
			Symbol:       trade.Symbol,
			Side:         trade.Side,
			PositionSide: "", // inferred from the order action by the shared engine
			OrderAction:  trade.OrderAction,
			OrderType:    trade.OrderType,
			Price:        trade.FillPrice,
			Quantity:     trade.FillQtyBase,
			RealizedPnL:  0, // OKX fills don't include PnL per trade
			Fee:          trade.Fee,
			Time:         trade.ExecTime,
		})
	}

	syncedCount, skippedCount := syncloop.PersistTrades(st, records, syncloop.PersistOptions{
		TraderID:               traderID,
		ExchangeID:             exchangeID,
		ExchangeType:           exchangeType,
		PositionSideFallback:   "LONG",
		SideNormalize:          true,
		DefaultCommissionAsset: "USDT",
		CommissionAssetFunc: func(trade types.TradeRecord) string {
			return feeAssetByTrade[trade.TradeID]
		},
	})

	// Advance the sync cursor to the latest processed trade (only after a
	// fully successful sync so failures retry from the same position).
	if len(trades) > 0 {
		t.syncCursor.Advance(exchangeID, trades[len(trades)-1].ExecTime.UTC().UnixMilli())
	}

	logger.Infof("✅ OKX order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// StartOrderSync starts background order sync task for OKX
func (t *OKXTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "OKX", func() error {
		return t.SyncOrdersFromOKX(traderID, exchangeID, exchangeType, st)
	})
}
