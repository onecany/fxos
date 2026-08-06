package hyperliquid

import (
	"fmt"
	"fxos/logger"
	"fxos/market"
	"fxos/store"
	"fxos/trader/syncloop"
	"time"
)

// SyncOrdersFromHyperliquid syncs Hyperliquid exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("hyperliquid")
func (t *HyperliquidTrader) SyncOrdersFromHyperliquid(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	startTime := time.Now().Add(-24 * time.Hour)

	logger.Infof("🔄 Syncing Hyperliquid trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 1000)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from Hyperliquid", len(trades))

	positionStore := st.Position()

	// Persist trades via the shared sync engine (sort ASC, dedup, symbol
	// normalization, order/fill/position records). Hyperliquid parses the
	// order action from the exchange Dir field, so no action mapper is
	// needed; position side is inferred from the action for the position
	// builder (fill records now carry LONG/SHORT instead of BOTH).
	syncedCount, _ := syncloop.PersistTrades(st, trades, syncloop.PersistOptions{
		TraderID:               traderID,
		ExchangeID:             exchangeID,
		ExchangeType:           exchangeType,
		PositionSideFallback:   "LONG",
		DefaultCommissionAsset: "USDT",
	})

	logger.Infof("✅ Order sync completed: %d new trades synced", syncedCount)

	// Reconcile local OPEN rows against the exchange's live book. Without
	// this, any missed/unmatched fill leaves a zombie OPEN row that swallows
	// every later close as a "partial close" — its realized PnL then never
	// reaches the closed-trade statistics. Scoped by exchange account so rows
	// left by prior autopilot incarnations are healed too.
	if err := t.reconcilePositions(exchangeID, positionStore); err != nil {
		logger.Infof("⚠️ Position reconcile skipped: %v", err)
	}

	return nil
}

// reconcilePositions builds the live (symbol, side) → quantity map from the
// exchange (core perps + xyz dex) and lets the store close/trim any local
// OPEN rows on this exchange account the exchange no longer backs.
func (t *HyperliquidTrader) reconcilePositions(exchangeID string, positionStore *store.PositionStore) error {
	livePositions, err := t.GetPositions()
	if err != nil {
		return fmt.Errorf("failed to get live positions: %w", err)
	}

	liveQty := make(map[string]float64, len(livePositions))
	for _, pos := range livePositions {
		symbol := pos.Symbol
		side := pos.Side
		qty := pos.Quantity
		if symbol == "" || qty <= 0 {
			continue
		}
		liveQty[store.LivePositionKey(market.Normalize(symbol), side)] += qty
	}

	_, err = positionStore.ReconcileOpenPositionsWithLive(exchangeID, liveQty)
	return err
}

// StartOrderSync starts background order sync task
func (t *HyperliquidTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "Hyperliquid", func() error {
		return t.SyncOrdersFromHyperliquid(traderID, exchangeID, exchangeType, st)
	})
}
