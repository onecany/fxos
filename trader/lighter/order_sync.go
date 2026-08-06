package lighter

import (
	"fmt"
	"fxos/logger"
	"fxos/store"
	"fxos/trader/syncloop"
	tradertypes "fxos/trader/types"
	"strings"
	"time"
)

// SyncOrdersFromLighter syncs Lighter exchange trade history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("lighter")
func (t *LighterTraderV2) SyncOrdersFromLighter(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	// Sync cursor: memory + DB recovery; falls back to a 24h lookback on first sync
	nowMs := time.Now().UTC().UnixMilli()
	startTime := time.UnixMilli(t.syncCursor.GetOrInit(exchangeID, st, nowMs))

	logger.Infof("🔄 Syncing Lighter trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records (same as other exchanges)
	trades, err := t.GetTrades(startTime, 100)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from Lighter", len(trades))

	// Persist via the shared sync engine (sort ASC, dedup, symbol
	// normalization, order/fill/position records). Lighter's GetTrades
	// already returns types.TradeRecord with OrderAction and PositionSide
	// filled; the action fallback below preserves the old empty-action
	// inference (open_long/open_short from the trade side).
	syncedCount, skippedCount := syncloop.PersistTrades(st, trades, syncloop.PersistOptions{
		TraderID:               traderID,
		ExchangeID:             exchangeID,
		ExchangeType:           exchangeType,
		PositionSideFallback:   "LONG",
		SideNormalize:          true,
		DefaultCommissionAsset: "USDT",
		DetermineOrderAction: func(trade tradertypes.TradeRecord) string {
			if trade.OrderAction != "" {
				return trade.OrderAction
			}
			// Fallback if OrderAction is empty (shouldn't happen with updated GetTrades)
			if strings.ToUpper(trade.Side) == "BUY" {
				return tradertypes.ActionOpenLong
			}
			return tradertypes.ActionOpenShort
		},
	})

	// Advance the sync cursor to the latest processed trade (only after a
	// fully successful sync so failures retry from the same position).
	if len(trades) > 0 {
		t.syncCursor.Advance(exchangeID, trades[len(trades)-1].Time.UTC().UnixMilli())
	}

	logger.Infof("✅ Order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// StartOrderSync starts background order sync task
func (t *LighterTraderV2) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "Lighter", func() error {
		err := t.SyncOrdersFromLighter(traderID, exchangeID, exchangeType, st)
		// A 404 just means the account has no fills yet — treat as a
		// successful empty sync to avoid log spam and pointless backoff.
		if err != nil && strings.Contains(err.Error(), "status 404") {
			return nil
		}
		return err
	})
}
