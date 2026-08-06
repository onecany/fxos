package aster

import (
	"fmt"
	"fxos/logger"
	"fxos/store"
	"fxos/trader/syncloop"
	"fxos/trader/types"
	"strings"
	"time"
)

// SyncOrdersFromAster syncs Aster exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
// exchangeID: Exchange account UUID (from exchanges.id)
// exchangeType: Exchange type ("aster")
func (t *AsterTrader) SyncOrdersFromAster(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	startTime := time.Now().Add(-24 * time.Hour)

	logger.Infof("🔄 Syncing Aster trades from: %s", startTime.Format(time.RFC3339))

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 500)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	logger.Infof("📥 Received %d trades from Aster", len(trades))

	// Persist via the shared sync engine (sort ASC, dedup, symbol
	// normalization, order/fill/position records). Aster's GetTrades
	// already returns types.TradeRecord; the order action is derived
	// exchange-specifically from side/positionSide/realized PnL via
	// DetermineOrderAction (Aster uses one-way position mode).
	syncedCount, skippedCount := syncloop.PersistTrades(st, trades, syncloop.PersistOptions{
		TraderID:               traderID,
		ExchangeID:             exchangeID,
		ExchangeType:           exchangeType,
		PositionSideFallback:   "LONG",
		SideNormalize:          true,
		DefaultCommissionAsset: "USDT",
		DetermineOrderAction: func(trade types.TradeRecord) string {
			return deriveAsterOrderAction(trade.Side, trade.PositionSide, trade.RealizedPnL)
		},
	})

	logger.Infof("✅ Aster order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// deriveAsterOrderAction determines order action from trade details
// Aster uses one-way position mode (BOTH), so we infer from:
// - Side: BUY or SELL
// - RealizedPnL: non-zero means closing trade
func deriveAsterOrderAction(side, positionSide string, realizedPnL float64) string {
	side = strings.ToUpper(side)
	positionSide = strings.ToUpper(positionSide)

	// Check if this is a closing trade (has realized PnL)
	isClose := realizedPnL != 0

	if positionSide == "LONG" {
		if isClose {
			return types.ActionCloseLong
		}
		return types.ActionOpenLong
	} else if positionSide == "SHORT" {
		if isClose {
			return types.ActionCloseShort
		}
		return types.ActionOpenShort
	} else {
		// BOTH mode - infer from side and PnL
		if side == "BUY" {
			if isClose {
				return types.ActionCloseShort // Buying to close short
			}
			return types.ActionOpenLong // Buying to open long
		} else {
			if isClose {
				return types.ActionCloseLong // Selling to close long
			}
			return types.ActionOpenShort // Selling to open short
		}
	}
}

// StartOrderSync starts background order sync task for Aster
func (t *AsterTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration, stop <-chan struct{}) {
	syncloop.Run(stop, interval, "Aster", func() error {
		return t.SyncOrdersFromAster(traderID, exchangeID, exchangeType, st)
	})
}
