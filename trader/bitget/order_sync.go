package bitget

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

// BitgetTrade represents a trade record from Bitget fill history
type BitgetTrade struct {
	Symbol      string
	TradeID     string
	OrderID     string
	Side        string // buy or sell
	FillPrice   float64
	FillQty     float64
	Fee         float64
	FeeAsset    string
	ExecTime    time.Time
	ProfitLoss  float64
	OrderType   string
	OrderAction string // open_long, open_short, close_long, close_short
}

// GetTrades retrieves trade/fill records from Bitget
func (t *BitgetTrader) GetTrades(startTime time.Time, limit int) ([]BitgetTrade, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100 // Bitget max limit is 100
	}

	params := map[string]interface{}{
		"productType": "USDT-FUTURES",
		"startTime":   fmt.Sprintf("%d", startTime.UnixMilli()),
		"limit":       fmt.Sprintf("%d", limit),
	}

	data, err := t.doRequest("GET", "/api/v2/mix/order/fill-history", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get fill history: %w", err)
	}

	// Bitget fill structure - supports both one-way and hedge mode
	type BitgetFill struct {
		TradeID    string `json:"tradeId"`
		Symbol     string `json:"symbol"`
		OrderID    string `json:"orderId"`
		Side       string `json:"side"`       // buy, sell
		Price      string `json:"price"`      // Fill price
		BaseVolume string `json:"baseVolume"` // Fill size in base currency
		Profit     string `json:"profit"`     // Realized PnL
		CTime      string `json:"cTime"`      // Fill time (ms)
		TradeSide  string `json:"tradeSide"`  // one-way: buy_single/sell_single, hedge: open/close
		FeeDetail  []struct {
			FeeCoin  string `json:"feeCoin"`
			TotalFee string `json:"totalFee"`
		} `json:"feeDetail"`
	}

	// Try parsing as wrapped response first (fillList field)
	var wrappedResp struct {
		FillList []BitgetFill `json:"fillList"`
	}

	// Try direct array format (Bitget V2 API returns data as direct array)
	var directFills []BitgetFill

	// Try wrapped format first
	if err := json.Unmarshal(data, &wrappedResp); err == nil && len(wrappedResp.FillList) > 0 {
		logger.Infof("🔍 Bitget: parsed as wrapped format, fillList count: %d", len(wrappedResp.FillList))
		directFills = wrappedResp.FillList
	} else {
		// Try direct array format
		if err := json.Unmarshal(data, &directFills); err != nil {
			logger.Infof("⚠️ Bitget fill-history parse failed, raw: %s", string(data))
			return nil, fmt.Errorf("failed to parse fills: %w", err)
		}
		logger.Infof("🔍 Bitget: parsed as direct array, fills count: %d", len(directFills))
	}

	trades := make([]BitgetTrade, 0, len(directFills))

	for _, fill := range directFills {
		fillPrice, _ := strconv.ParseFloat(fill.Price, 64)
		fillQty, _ := strconv.ParseFloat(fill.BaseVolume, 64)
		profit, _ := strconv.ParseFloat(fill.Profit, 64)
		cTime, _ := strconv.ParseInt(fill.CTime, 10, 64)

		// Extract fee from feeDetail array (Bitget V2 API)
		var fee float64
		var feeAsset string
		if len(fill.FeeDetail) > 0 {
			fee, _ = strconv.ParseFloat(fill.FeeDetail[0].TotalFee, 64)
			feeAsset = fill.FeeDetail[0].FeeCoin
		}

		// Determine order action based on side and tradeSide
		// Bitget one-way mode: buy_single (open long), sell_single (close long)
		// Bitget hedge mode: open + buy = open_long, close + sell = close_long
		orderAction := types.ActionOpenLong
		side := strings.ToLower(fill.Side)
		tradeSide := strings.ToLower(fill.TradeSide)

		// One-way position mode (buy_single/sell_single)
		if tradeSide == "buy_single" {
			orderAction = types.ActionOpenLong
		} else if tradeSide == "sell_single" {
			orderAction = types.ActionCloseLong
		} else if tradeSide == "open" {
			// Hedge mode: open
			if side == "buy" {
				orderAction = types.ActionOpenLong
			} else {
				orderAction = types.ActionOpenShort
			}
		} else if tradeSide == "close" {
			// Hedge mode: close
			if side == "sell" {
				orderAction = types.ActionCloseLong
			} else {
				orderAction = types.ActionCloseShort
			}
		}

		trade := BitgetTrade{
			Symbol:      fill.Symbol,
			TradeID:     fill.TradeID,
			OrderID:     fill.OrderID,
			Side:        fill.Side,
			FillPrice:   fillPrice,
			FillQty:     fillQty,
			Fee:         -fee, // Bitget returns negative fee, convert to positive
			FeeAsset:    feeAsset,
			ExecTime:    time.UnixMilli(cTime).UTC(),
			ProfitLoss:  profit,
			OrderType:   "MARKET",
			OrderAction: orderAction,
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// SyncOrdersFromBitget syncs Bitget exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("bitget")
func (t *BitgetTrader) SyncOrdersFromBitget(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	// Sync cursor: memory + DB recovery; falls back to a 24h lookback on first sync
	nowMs := time.Now().UTC().UnixMilli()
	startTime := time.UnixMilli(t.syncCursor.GetOrInit(exchangeID, st, nowMs))

	logger.Infof("🔄 Syncing Bitget trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 100)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from Bitget", len(trades))

	// Convert exchange trades to the shared TradeRecord shape and persist via
	// the shared sync engine (sort ASC, dedup, symbol normalization,
	// order/fill/position records). Bitget keeps one-way position mode
	// (PositionSide "BOTH"); the order action is parsed per trade in
	// GetTrades and carried over as-is. The per-fill fee currency
	// (feeDetail[0].feeCoin) is carried through CommissionAssetFunc.
	feeAssetByTrade := make(map[string]string, len(trades))
	records := make([]types.TradeRecord, 0, len(trades))
	for _, trade := range trades {
		feeAssetByTrade[trade.TradeID] = trade.FeeAsset
		records = append(records, types.TradeRecord{
			TradeID:      trade.TradeID,
			Symbol:       trade.Symbol,
			Side:         trade.Side,
			PositionSide: "BOTH", // Bitget uses one-way position mode
			OrderAction:  trade.OrderAction,
			OrderType:    trade.OrderType,
			Price:        trade.FillPrice,
			Quantity:     trade.FillQty,
			RealizedPnL:  trade.ProfitLoss,
			Fee:          trade.Fee,
			Time:         trade.ExecTime,
		})
	}

	syncedCount, skippedCount := syncloop.PersistTrades(st, records, syncloop.PersistOptions{
		TraderID:               traderID,
		ExchangeID:             exchangeID,
		ExchangeType:           exchangeType,
		PositionSideFallback:   "BOTH",
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

	logger.Infof("✅ Bitget order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// StartOrderSync starts background order sync task for Bitget
func (t *BitgetTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "Bitget", func() error {
		return t.SyncOrdersFromBitget(traderID, exchangeID, exchangeType, st)
	})
}
