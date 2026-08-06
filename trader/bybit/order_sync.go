package bybit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"fxos/httpclient"
	"fxos/logger"
	"fxos/store"
	"fxos/trader/syncloop"
	"fxos/trader/types"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// BybitTrade represents a trade record from Bybit execution list
type BybitTrade struct {
	Symbol      string
	OrderID     string
	ExecID      string
	Side        string // Buy or Sell
	ExecPrice   float64
	ExecQty     float64
	ExecFee     float64
	ExecTime    time.Time
	IsMaker     bool
	OrderType   string
	ClosedSize  float64 // For close orders
	ClosedPnL   float64
	OrderAction string // open_long, open_short, close_long, close_short
}

// GetTrades retrieves trade/execution records from Bybit
func (t *BybitTrader) GetTrades(startTime time.Time, limit int) ([]BybitTrade, error) {
	return t.getTradesViaHTTP(startTime, limit)
}

// getTradesViaHTTP makes direct HTTP call to Bybit API for execution list
func (t *BybitTrader) getTradesViaHTTP(startTime time.Time, limit int) ([]BybitTrade, error) {
	// Build query string
	queryParams := fmt.Sprintf("category=linear&startTime=%d&limit=%d", startTime.UnixMilli(), limit)
	url := "https://api.bybit.com/v5/execution/list?" + queryParams

	// Generate timestamp
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	recvWindow := "5000"

	// Build signature payload: timestamp + api_key + recv_window + queryString
	signPayload := timestamp + t.apiKey + recvWindow + queryParams

	// Generate HMAC-SHA256 signature
	h := hmac.New(sha256.New, []byte(t.secretKey))
	h.Write([]byte(signPayload))
	signature := hex.EncodeToString(h.Sum(nil))

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Bybit V5 API headers
	req.Header.Set("X-BAPI-API-KEY", t.apiKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-SIGN-TYPE", "2")
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)
	req.Header.Set("Content-Type", "application/json")

	client := httpclient.New(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Bybit API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List []map[string]interface{} `json:"list"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.RetCode != 0 {
		return nil, fmt.Errorf("Bybit API error: %s", result.RetMsg)
	}

	return t.parseTradesResult(result.Result.List)
}

// parseTradesResult parses the execution list result from Bybit API
func (t *BybitTrader) parseTradesResult(list []map[string]interface{}) ([]BybitTrade, error) {
	var trades []BybitTrade

	for _, item := range list {
		symbol, _ := item["symbol"].(string)
		orderID, _ := item["orderId"].(string)
		execID, _ := item["execId"].(string)
		side, _ := item["side"].(string)
		orderType, _ := item["orderType"].(string)
		isMaker, _ := item["isMaker"].(bool)

		execPriceStr, _ := item["execPrice"].(string)
		execQtyStr, _ := item["execQty"].(string)
		execFeeStr, _ := item["execFee"].(string)
		closedSizeStr, _ := item["closedSize"].(string)
		closedPnlStr, _ := item["closedPnl"].(string)
		execTimeStr, _ := item["execTime"].(string)

		execPrice, _ := strconv.ParseFloat(execPriceStr, 64)
		execQty, _ := strconv.ParseFloat(execQtyStr, 64)
		execFee, _ := strconv.ParseFloat(execFeeStr, 64)
		closedSize, _ := strconv.ParseFloat(closedSizeStr, 64)
		closedPnl, _ := strconv.ParseFloat(closedPnlStr, 64)
		execTimeMs, _ := strconv.ParseInt(execTimeStr, 10, 64)
		execTime := time.UnixMilli(execTimeMs).UTC()

		// Determine order action based on side and closedSize
		// If closedSize > 0, it's a close trade
		// Side: Buy = long direction, Sell = short direction
		orderAction := types.ActionOpenLong
		if closedSize > 0 {
			// This is a close trade
			if strings.ToLower(side) == "sell" {
				orderAction = types.ActionCloseLong // Selling to close a long
			} else {
				orderAction = types.ActionCloseShort // Buying to close a short
			}
		} else {
			// This is an open trade
			if strings.ToLower(side) == "buy" {
				orderAction = types.ActionOpenLong
			} else {
				orderAction = types.ActionOpenShort
			}
		}

		trade := BybitTrade{
			Symbol:      symbol,
			OrderID:     orderID,
			ExecID:      execID,
			Side:        side,
			ExecPrice:   execPrice,
			ExecQty:     execQty,
			ExecFee:     execFee,
			ExecTime:    execTime,
			IsMaker:     isMaker,
			OrderType:   orderType,
			ClosedSize:  closedSize,
			ClosedPnL:   closedPnl,
			OrderAction: orderAction,
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

// SyncOrdersFromBybit syncs Bybit exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("bybit")
func (t *BybitTrader) SyncOrdersFromBybit(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	startTime := time.Now().Add(-24 * time.Hour)

	logger.Infof("🔄 Syncing Bybit trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 1000)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from Bybit", len(trades))

	// Convert exchange trades to the shared TradeRecord shape and persist via
	// the shared sync engine (sort ASC, dedup, symbol normalization,
	// order/fill/position records). Bybit keeps one-way position mode
	// (PositionSide "BOTH"); the order action is parsed per trade in
	// parseTradesResult and carried over as-is.
	records := make([]types.TradeRecord, 0, len(trades))
	for _, trade := range trades {
		records = append(records, types.TradeRecord{
			TradeID:      trade.ExecID,
			Symbol:       trade.Symbol,
			Side:         trade.Side,
			PositionSide: "BOTH", // Bybit uses one-way position mode
			OrderAction:  trade.OrderAction,
			OrderType:    trade.OrderType,
			IsMaker:      trade.IsMaker,
			Price:        trade.ExecPrice,
			Quantity:     trade.ExecQty,
			RealizedPnL:  trade.ClosedPnL,
			Fee:          trade.ExecFee,
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
	})

	logger.Infof("✅ Bybit order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// StartOrderSync starts background order sync task for Bybit
func (t *BybitTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "Bybit", func() error {
		return t.SyncOrdersFromBybit(traderID, exchangeID, exchangeType, st)
	})
}
