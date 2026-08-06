package kucoin

import (
	"encoding/json"
	"fmt"
	"fxos/logger"
	"fxos/store"
	"fxos/trader/syncloop"
	"fxos/trader/types"
	"sort"
	"strings"
	"time"
)

// KuCoinTrade represents a trade record from KuCoin fill history
type KuCoinTrade struct {
	Symbol      string
	TradeID     string
	OrderID     string
	Side        string // buy or sell
	FillPrice   float64
	FillQty     float64 // In base currency (e.g., ETH), not lots
	Fee         float64
	FeeAsset    string
	ExecTime    time.Time
	ProfitLoss  float64
	OrderAction string // open_long, open_short, close_long, close_short
}

// GetTrades retrieves trade/fill records from KuCoin
func (t *KuCoinTrader) GetTrades(startTime time.Time, limit int) ([]KuCoinTrade, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100 // KuCoin max limit
	}

	// Build query path
	path := fmt.Sprintf("%s?pageSize=%d", kucoinFillsPath, limit)
	if !startTime.IsZero() {
		path += fmt.Sprintf("&startAt=%d", startTime.UnixMilli())
	}

	data, err := t.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade history: %w", err)
	}

	var response struct {
		CurrentPage int `json:"currentPage"`
		PageSize    int `json:"pageSize"`
		TotalNum    int `json:"totalNum"`
		TotalPage   int `json:"totalPage"`
		Items       []struct {
			Symbol      string `json:"symbol"`
			TradeId     string `json:"tradeId"`
			OrderId     string `json:"orderId"`
			Side        string `json:"side"`
			Price       string `json:"price"`
			Size        int64  `json:"size"`
			Value       string `json:"value"`       // Trade value in quote currency
			Fee         string `json:"fee"`         // Total fee
			FeeRate     string `json:"feeRate"`     // Fee rate
			FeeCurrency string `json:"feeCurrency"` // Fee currency (USDT)
			OpenFeePay  string `json:"openFeePay"`  // Fee for opening (>0 means opening trade)
			CloseFeePay string `json:"closeFeePay"` // Fee for closing (>0 means closing trade)
			TradeTime   int64  `json:"tradeTime"`   // Nanoseconds
			MarginMode  string `json:"marginMode"`  // CROSS or ISOLATED
			OrderType   string `json:"orderType"`   // market, limit
		} `json:"items"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse trade history: %w", err)
	}

	logger.Infof("📥 Received %d trades from KuCoin", len(response.Items))

	result := make([]KuCoinTrade, 0, len(response.Items))

	for _, trade := range response.Items {
		// Parse numeric values from strings
		var fillPrice, fee, openFeePay, closeFeePay float64
		fmt.Sscanf(trade.Price, "%f", &fillPrice)
		fmt.Sscanf(trade.Fee, "%f", &fee)
		fmt.Sscanf(trade.OpenFeePay, "%f", &openFeePay)
		fmt.Sscanf(trade.CloseFeePay, "%f", &closeFeePay)

		// Get multiplier from contract info
		symbol := t.convertSymbolBack(trade.Symbol)
		var multiplier float64
		contract, err := t.getContract(symbol)
		if err == nil && contract != nil {
			multiplier = contract.Multiplier
		} else {
			// Default multipliers based on symbol
			if strings.Contains(symbol, "BTC") {
				multiplier = 0.001
			} else {
				multiplier = 0.01 // Default for altcoins
			}
		}

		// Convert lots to actual quantity
		absSize := trade.Size
		if absSize < 0 {
			absSize = -absSize
		}
		fillQty := float64(absSize) * multiplier

		// Determine side and order action
		// KuCoin uses openFeePay/closeFeePay to indicate if trade is opening or closing
		side := strings.ToUpper(trade.Side) // BUY or SELL
		isClosing := closeFeePay > 0

		var orderAction string
		if trade.Side == "buy" {
			if isClosing {
				// Buying to close short
				orderAction = types.ActionCloseShort
			} else {
				// Buying to open long
				orderAction = types.ActionOpenLong
			}
		} else { // sell
			if isClosing {
				// Selling to close long
				orderAction = types.ActionCloseLong
			} else {
				// Selling to open short
				orderAction = types.ActionOpenShort
			}
		}

		// Trade time is in nanoseconds
		execTime := time.Unix(0, trade.TradeTime)

		result = append(result, KuCoinTrade{
			Symbol:      symbol,
			TradeID:     trade.TradeId,
			OrderID:     trade.OrderId,
			Side:        side,
			FillPrice:   fillPrice,
			FillQty:     fillQty,
			Fee:         fee,
			FeeAsset:    trade.FeeCurrency,
			ExecTime:    execTime,
			ProfitLoss:  0, // KuCoin fills API doesn't return PnL per trade
			OrderAction: orderAction,
		})
	}

	// Sort by execution time (oldest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ExecTime.Before(result[j].ExecTime)
	})

	return result, nil
}

// GetRecentTrades retrieves recent trades (faster, no pagination)
func (t *KuCoinTrader) GetRecentTrades() ([]KuCoinTrade, error) {
	data, err := t.doRequest("GET", kucoinRecentFillsPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent trades: %w", err)
	}

	var trades []struct {
		Symbol      string `json:"symbol"`
		TradeId     string `json:"tradeId"`
		OrderId     string `json:"orderId"`
		Side        string `json:"side"`
		Price       string `json:"price"`
		Size        int64  `json:"size"`
		Fee         string `json:"fee"`
		FeeCurrency string `json:"feeCurrency"`
		OpenFeePay  string `json:"openFeePay"`
		CloseFeePay string `json:"closeFeePay"`
		TradeTime   int64  `json:"tradeTime"`
	}

	if err := json.Unmarshal(data, &trades); err != nil {
		return nil, fmt.Errorf("failed to parse recent trades: %w", err)
	}

	result := make([]KuCoinTrade, 0, len(trades))

	for _, trade := range trades {
		var fillPrice, fee, openFeePay, closeFeePay float64
		fmt.Sscanf(trade.Price, "%f", &fillPrice)
		fmt.Sscanf(trade.Fee, "%f", &fee)
		fmt.Sscanf(trade.OpenFeePay, "%f", &openFeePay)
		fmt.Sscanf(trade.CloseFeePay, "%f", &closeFeePay)

		// Get multiplier from contract info
		symbol := t.convertSymbolBack(trade.Symbol)
		var multiplier float64
		contract, err := t.getContract(symbol)
		if err == nil && contract != nil {
			multiplier = contract.Multiplier
		} else {
			if strings.Contains(symbol, "BTC") {
				multiplier = 0.001
			} else {
				multiplier = 0.01
			}
		}

		absSize := trade.Size
		if absSize < 0 {
			absSize = -absSize
		}
		fillQty := float64(absSize) * multiplier

		side := strings.ToUpper(trade.Side)
		isClosing := closeFeePay > 0

		var orderAction string
		if trade.Side == "buy" {
			if isClosing {
				orderAction = types.ActionCloseShort
			} else {
				orderAction = types.ActionOpenLong
			}
		} else {
			if isClosing {
				orderAction = types.ActionCloseLong
			} else {
				orderAction = types.ActionOpenShort
			}
		}

		execTime := time.Unix(0, trade.TradeTime)

		result = append(result, KuCoinTrade{
			Symbol:      symbol,
			TradeID:     trade.TradeId,
			OrderID:     trade.OrderId,
			Side:        side,
			FillPrice:   fillPrice,
			FillQty:     fillQty,
			Fee:         fee,
			FeeAsset:    trade.FeeCurrency,
			ExecTime:    execTime,
			ProfitLoss:  0,
			OrderAction: orderAction,
		})
	}

	return result, nil
}

// ToTradeRecord converts KuCoinTrade to types.TradeRecord
func (t *KuCoinTrade) ToTradeRecord() types.TradeRecord {
	// Determine position side from order action
	positionSide := "LONG"
	if strings.Contains(t.OrderAction, types.SideShort) {
		positionSide = "SHORT"
	}

	return types.TradeRecord{
		TradeID:      t.TradeID,
		Symbol:       t.Symbol,
		Side:         t.Side,
		PositionSide: positionSide,
		OrderAction:  t.OrderAction,
		Price:        t.FillPrice,
		Quantity:     t.FillQty,
		RealizedPnL:  t.ProfitLoss,
		Fee:          t.Fee,
		Time:         t.ExecTime,
	}
}

// SyncOrdersFromKuCoin syncs KuCoin exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("kucoin")
func (t *KuCoinTrader) SyncOrdersFromKuCoin(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	// Sync cursor: memory + DB recovery; falls back to a 24h lookback on first sync
	nowMs := time.Now().UTC().UnixMilli()
	startTime := time.UnixMilli(t.syncCursor.GetOrInit(exchangeID, st, nowMs))

	logger.Infof("🔄 Syncing KuCoin trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 100)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from KuCoin", len(trades))

	// Convert to unified TradeRecord format (ToTradeRecord infers the
	// position side from the order action) and persist via the shared
	// sync engine (sort ASC, dedup, symbol normalization, order/fill/
	// position records). KuCoin returns one-way-mode fills; the position
	// side is inferred per trade.
	records := make([]types.TradeRecord, 0, len(trades))
	for _, trade := range trades {
		records = append(records, trade.ToTradeRecord())
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

	logger.Infof("✅ KuCoin order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// StartOrderSync starts background order sync task for KuCoin
func (t *KuCoinTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "KuCoin", func() error {
		return t.SyncOrdersFromKuCoin(traderID, exchangeID, exchangeType, st)
	})
}
