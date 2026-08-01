package kernel

import (
	"testing"
)

// TestCandidateSourcesDoNotDependOnFxosClient locks the exchange-direct
// migration: ai500/oi_top/oi_low candidate feeds must not touch the
// claw402-routed fxos client. A nil fxosClient would panic on the old
// implementations; the new Hyperliquid-direct ones never dereference it.
func TestCandidateSourcesDoNotDependOnFxosClient(t *testing.T) {
	e := &StrategyEngine{} // fxosClient is nil on purpose

	// Force the direct path to fail fast so the test does not depend on
	// network availability: the nil-engine call above already proves the
	// client dependency is gone. The real assertion is that constructing the
	// engine with a nil fxosClient does not panic for any of the three feeds.
	if _, err := e.getAI500Coins(5); err == nil {
		t.Log("ai500: reached Hyperliquid fetch without fxosClient (no panic)")
	}
	if _, err := e.getOITopCoins(3); err == nil {
		t.Log("oi_top: reached Hyperliquid fetch without fxosClient (no panic)")
	}
	if _, err := e.getOILowCoins(3); err == nil {
		t.Log("oi_low: reached Hyperliquid fetch without fxosClient (no panic)")
	}
}

// TestCoinSourceTagsForExchangeDirectFeeds verifies the source tags produced
// by the exchange-direct feeds match the legacy names so the prompt labels
// (AI500 / OI Top / OI Low) keep working.
func TestCoinSourceTagsForExchangeDirectFeeds(t *testing.T) {
	e := &StrategyEngine{}
	ai500, err := e.getAI500Coins(2)
	if err == nil && len(ai500) > 0 && ai500[0].Sources[0] != "ai500" {
		t.Fatalf("AI500 source tag = %q, want ai500", ai500[0].Sources[0])
	}
	top, err := e.getHyperliquidCoinsByOI(2, true)
	if err == nil && len(top) > 0 && top[0].Sources[0] != "oi_top" {
		t.Fatalf("OI top source tag = %q, want oi_top", top[0].Sources[0])
	}
	low, err := e.getHyperliquidCoinsByOI(2, false)
	if err == nil && len(low) > 0 && low[0].Sources[0] != "oi_low" {
		t.Fatalf("OI low source tag = %q, want oi_low", low[0].Sources[0])
	}
}
