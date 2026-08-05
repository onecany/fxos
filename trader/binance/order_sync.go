package binance

import (
	"fmt"
	"fxos/logger"
	"fxos/store"
	"fxos/trader/syncloop"
	"fxos/trader/types"
	"strings"
	"time"
)

// SyncOrdersFromBinance syncs Binance Futures trade history to local database
// Uses COMMISSION detection + fromId for efficient incremental sync
func (t *FuturesTrader) SyncOrdersFromBinance(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	orderStore := st.Order()

	// Get last sync time (Unix ms) - shared cursor (memory + DB recovery)
	nowMs := time.Now().UTC().UnixMilli()
	lastSyncTimeMs := t.syncCursor.GetOrInit(exchangeID, st, nowMs)

	logger.Infof("🔄 Syncing Binance trades from: %s (UTC) [ms: %d, now: %d]",
		time.UnixMilli(lastSyncTimeMs).UTC().Format("2006-01-02 15:04:05"), lastSyncTimeMs, nowMs)

	// Step 1: Get max trade IDs from local DB for incremental sync
	maxTradeIDs, err := orderStore.GetMaxTradeIDsByExchange(exchangeID)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get max trade IDs: %v, will use time-based query", err)
		maxTradeIDs = make(map[string]int64)
	}

	// Step 2: Detect symbols to sync using multiple methods
	// COMMISSION detection may miss trades (VIP users, BNB discount, 0-fee trades)
	symbolMap := make(map[string]bool)
	lastSyncTime := time.UnixMilli(lastSyncTimeMs) // Convert to time.Time for API calls

	// Method 1: COMMISSION income detection
	commissionSymbols, err := t.GetCommissionSymbols(lastSyncTime)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get commission symbols: %v", err)
	} else {
		logger.Infof("  📋 COMMISSION symbols found: %d - %v", len(commissionSymbols), commissionSymbols)
		for _, s := range commissionSymbols {
			symbolMap[s] = true
		}
	}

	// Method 2: Always include active positions (catches trades that COMMISSION missed)
	positionSymbols := t.getPositionSymbols()
	logger.Infof("  📋 Position symbols found: %d - %v", len(positionSymbols), positionSymbols)
	for _, s := range positionSymbols {
		symbolMap[s] = true
	}

	// Method 3: Include symbols from recent fills in DB (in case some were partially synced)
	recentSymbols, _ := orderStore.GetRecentFillSymbolsByExchange(exchangeID, lastSyncTimeMs)
	logger.Infof("  📋 Recent fill symbols found: %d - %v", len(recentSymbols), recentSymbols)
	for _, s := range recentSymbols {
		symbolMap[s] = true
	}

	// Method 4: ALWAYS query REALIZED_PNL income to find symbols with closed trades
	// This catches trades that COMMISSION missed (VIP users, BNB fee discount)
	// IMPORTANT: Must run always, not just when symbolMap is empty,
	// because a position might be fully closed (no active position) but have PnL
	pnlSymbols, err := t.GetPnLSymbols(lastSyncTime)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get PnL symbols: %v", err)
	} else {
		logger.Infof("  📋 REALIZED_PNL symbols found: %d - %v", len(pnlSymbols), pnlSymbols)
		for _, s := range pnlSymbols {
			symbolMap[s] = true
		}
	}

	var changedSymbols []string
	for s := range symbolMap {
		changedSymbols = append(changedSymbols, s)
	}

	if len(changedSymbols) == 0 {
		logger.Infof("📭 No symbols with new trades to sync")
		// DON'T update lastSyncTime to current time here!
		// Keep using the last actual trade time from DB to avoid creating gaps
		// The lastSyncTimeMs from DB already has +1000ms buffer added
		return nil
	}

	logger.Infof("📊 Found %d symbols with new trades: %v", len(changedSymbols), changedSymbols)

	// Step 3: Query trades for changed symbols using fromId (incremental) or time-based (new symbols)
	var allTrades []types.TradeRecord
	var failedSymbols []string
	apiCalls := 0
	for _, symbol := range changedSymbols {
		var trades []types.TradeRecord
		var queryErr error

		if lastID, ok := maxTradeIDs[symbol]; ok && lastID > 0 {
			// Incremental sync: query from last known trade ID
			trades, queryErr = t.GetTradesForSymbolFromID(symbol, lastID+1, 500)
		} else {
			// New symbol or first sync: query by time
			trades, queryErr = t.GetTradesForSymbol(symbol, lastSyncTime, 500)
		}
		apiCalls++

		if queryErr != nil {
			logger.Infof("  ⚠️ Failed to get trades for %s: %v", symbol, queryErr)
			failedSymbols = append(failedSymbols, symbol)
			continue
		}
		allTrades = append(allTrades, trades...)
	}

	logger.Infof("📥 Received %d trades from Binance (%d API calls)", len(allTrades), apiCalls)

	if len(allTrades) == 0 {
		// No trades returned, but symbols were detected - might be false positive from COMMISSION/PnL detection
		// Don't update lastSyncTime, keep using DB value
		if len(failedSymbols) > 0 {
			logger.Infof("  ⚠️ %d symbols failed: %v", len(failedSymbols), failedSymbols)
		}
		return nil
	}

	// Persist trades via the shared sync engine (sort ASC, dedup, symbol
	// normalization, order/fill/position records). Action mapping stays
	// exchange-specific via DetermineOrderAction.
	syncedCount, skippedCount := syncloop.PersistTrades(st, allTrades, syncloop.PersistOptions{
		TraderID:               traderID,
		ExchangeID:             exchangeID,
		ExchangeType:           exchangeType,
		PositionSideFallback:   "LONG",
		SideNormalize:          true,
		DefaultCommissionAsset: "USDT",
		DetermineOrderAction: func(trade types.TradeRecord) string {
			return t.determineOrderAction(trade.Side, trade.PositionSide, trade.RealizedPnL)
		},
		IsMaker: false,
	})

	// Update lastSyncTime to the LATEST trade time (not current time!)
	// This ensures next sync starts from where we left off, not from "now"
	// allTrades is already sorted by time ASC, so last element is the latest
	if len(failedSymbols) == 0 {
		latestTradeTimeMs := allTrades[len(allTrades)-1].Time.UTC().UnixMilli()
		t.syncCursor.Advance(exchangeID, latestTradeTimeMs)
	} else {
		logger.Infof("  ⚠️ %d symbols failed, not updating lastSyncTime to retry next time: %v", len(failedSymbols), failedSymbols)
	}

	logger.Infof("✅ Binance order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// getPositionSymbols returns list of symbols that have active positions
// Used as fallback when COMMISSION detection fails
func (t *FuturesTrader) getPositionSymbols() []string {
	positions, err := t.GetPositions()
	if err != nil {
		return nil
	}

	var symbols []string
	for _, pos := range positions {
		if pos.Symbol != "" {
			symbols = append(symbols, pos.Symbol)
		}
	}
	return symbols
}

// determineOrderAction determines the order action based on trade data
func (t *FuturesTrader) determineOrderAction(side, positionSide string, realizedPnL float64) string {
	side = strings.ToUpper(side)
	positionSide = strings.ToUpper(positionSide)

	// If there's realized PnL, it's likely a close trade
	isClose := realizedPnL != 0

	if positionSide == "LONG" || positionSide == "" {
		if side == "BUY" {
			if isClose {
				return types.ActionCloseShort // Buying to close short
			}
			return types.ActionOpenLong
		} else {
			if isClose {
				return types.ActionCloseLong // Selling to close long
			}
			return types.ActionOpenShort
		}
	} else if positionSide == "SHORT" {
		if side == "SELL" {
			if isClose {
				return types.ActionCloseLong
			}
			return types.ActionOpenShort
		} else {
			if isClose {
				return types.ActionCloseShort
			}
			return types.ActionOpenLong
		}
	}

	// Default fallback
	if side == "BUY" {
		return types.ActionOpenLong
	}
	return types.ActionOpenShort
}

// StartOrderSync starts background order sync task for Binance
func (t *FuturesTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	// Run first sync immediately
	go func() {
		logger.Infof("🔄 Running initial Binance order sync...")
		if err := t.SyncOrdersFromBinance(traderID, exchangeID, exchangeType, st); err != nil {
			logger.Infof("⚠️  Initial Binance order sync failed: %v", err)
		}
	}()

	// Then run periodically
	syncloop.Run(stop, interval, "Binance", func() error {
		return t.SyncOrdersFromBinance(traderID, exchangeID, exchangeType, st)
	})
}
