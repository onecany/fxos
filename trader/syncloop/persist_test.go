package syncloop

import (
	"testing"
	"time"

	"fxos/store"
	"fxos/trader/types"
)

func TestSyncCursorDefaultsWithoutStore(t *testing.T) {
	c := NewSyncCursor()
	nowMs := time.Now().UTC().UnixMilli()
	got := c.GetOrInit("acc-1", nil, nowMs)

	// First sync goes back 24h
	want := nowMs - 24*time.Hour.Milliseconds()
	if got < want-1000 || got > want+1000 {
		t.Fatalf("GetOrInit(nil store) = %d, want ~%d", got, want)
	}

	// Second call hits the memory cache (same value)
	got2 := c.GetOrInit("acc-1", nil, nowMs)
	if got2 != got {
		t.Fatalf("second GetOrInit = %d, want cached %d", got2, got)
	}
}

func TestSyncCursorAdvance(t *testing.T) {
	c := NewSyncCursor()
	nowMs := time.Now().UTC().UnixMilli()
	_ = c.GetOrInit("acc-1", nil, nowMs)

	latest := nowMs - 5*time.Minute.Milliseconds() // latest trade time
	c.Advance("acc-1", latest)

	got := c.GetOrInit("acc-1", nil, nowMs)
	if got != latest {
		t.Fatalf("after Advance = %d, want %d", got, latest)
	}

	// Different exchange accounts keep independent cursors
	got2 := c.GetOrInit("acc-2", nil, nowMs)
	if got2 == latest {
		t.Fatalf("acc-2 cursor leaked acc-1 value: %d", got2)
	}
}

func TestPersistTradesDedup(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	now := time.Now().UTC()
	trades := []types.TradeRecord{
		{
			TradeID:      "trade-1",
			Symbol:       "BTCUSDT",
			Side:         "BUY",
			PositionSide: "",
			OrderAction:  "open_long",
			Price:        50000,
			Quantity:     0.1,
			Fee:          0.5,
			RealizedPnL:  0,
			Time:         now.Add(-2 * time.Minute),
		},
		{
			TradeID:      "trade-2",
			Symbol:       "BTCUSDT",
			Side:         "SELL",
			PositionSide: "LONG",
			OrderAction:  "close_long",
			Price:        51000,
			Quantity:     0.1,
			Fee:          0.51,
			RealizedPnL:  100,
			Time:         now.Add(-1 * time.Minute),
		},
	}

	opts := PersistOptions{
		TraderID:               "trader-1",
		ExchangeID:             "acc-1",
		ExchangeType:           "test",
		PositionSideFallback:   "LONG",
		SideNormalize:          true,
		DefaultCommissionAsset: "USDT",
	}

	synced, skipped := PersistTrades(st, trades, opts)
	if synced != 2 {
		t.Fatalf("first persist: synced=%d, want 2", synced)
	}
	if skipped != 0 {
		t.Fatalf("first persist: skipped=%d, want 0", skipped)
	}

	// Second run must dedup everything
	synced2, skipped2 := PersistTrades(st, trades, opts)
	if synced2 != 0 {
		t.Fatalf("second persist: synced=%d, want 0 (all deduped)", synced2)
	}
	if skipped2 != 2 {
		t.Fatalf("second persist: skipped=%d, want 2", skipped2)
	}

	// Position records were built (open then close -> no open position left)
	openPositions, err := st.Position().GetOpenPositions("trader-1")
	if err != nil {
		t.Fatalf("list open positions: %v", err)
	}
	if len(openPositions) != 0 {
		t.Fatalf("expected 0 open positions after open+close, got %d", len(openPositions))
	}
}
