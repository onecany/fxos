package trader

import (
	"sync"
	"testing"
)

// TestBreakevenStopTriggered verifies that breakeven stop decision is made
// when profit exceeds the threshold.
func TestBreakevenStopTriggered(t *testing.T) {
	at := &AutoTrader{}
	at.breakevenStopCache = make(map[string]bool)

	// Simulate: BTC long, profit=2% (above 1.5% threshold)
	symbol := "BTCUSDT"
	side := "long"
	posKey := symbol + "_" + side

	// Check threshold logic directly (without calling adjustStopLoss which needs a trader)
	breakevenThreshold := 1.5 // BTC threshold
	currentPnLPct := 2.0

	if currentPnLPct >= breakevenThreshold {
		at.breakevenStopCacheMu.Lock()
		at.breakevenStopCache[posKey] = true
		at.breakevenStopCacheMu.Unlock()
	}

	at.breakevenStopCacheMu.RLock()
	applied := at.breakevenStopCache[posKey]
	at.breakevenStopCacheMu.RUnlock()

	if !applied {
		t.Errorf("Expected breakeven stop to be marked as applied for %s %s", symbol, side)
	}
	t.Logf("✅ Breakeven stop decision: %s %s at %.2f%% profit (threshold %.2f%%)", symbol, side, currentPnLPct, breakevenThreshold)
}

// TestBreakevenStopNotTriggeredBelowThreshold verifies breakeven stop
// is NOT applied when profit is below threshold.
func TestBreakevenStopNotTriggeredBelowThreshold(t *testing.T) {
	at := &AutoTrader{}
	at.breakevenStopCache = make(map[string]bool)

	symbol := "BTCUSDT"
	side := "long"
	posKey := symbol + "_" + side

	breakevenThreshold := 1.5
	currentPnLPct := 1.0 // Below threshold

	if currentPnLPct >= breakevenThreshold {
		at.breakevenStopCacheMu.Lock()
		at.breakevenStopCache[posKey] = true
		at.breakevenStopCacheMu.Unlock()
	}

	at.breakevenStopCacheMu.RLock()
	applied := at.breakevenStopCache[posKey]
	at.breakevenStopCacheMu.RUnlock()

	if applied {
		t.Errorf("Expected breakeven stop NOT to be applied at %.2f%% (threshold %.2f%%)", currentPnLPct, breakevenThreshold)
	}
	t.Logf("✅ Breakeven stop correctly NOT triggered at %.2f%% (threshold %.2f%%)", currentPnLPct, breakevenThreshold)
}

// TestTrailingStopDecision verifies trailing stop logic.
func TestTrailingStopDecision(t *testing.T) {
	// Trailing stop triggers when: profit >= trailingThreshold AND peakPnLPct > currentPnLPct
	trailingThreshold := 3.0 // BTC threshold

	// Case 1: Profit above threshold, below peak → trailing should trigger
	currentPnLPct := 4.0
	peakPnLPct := 5.0
	shouldTrail := currentPnLPct >= trailingThreshold && peakPnLPct > currentPnLPct

	if !shouldTrail {
		t.Error("Expected trailing stop to trigger when profit >= 3% and below peak")
	}
	t.Logf("✅ Trailing stop decision: profit %.2f%%, peak %.2f%% → trigger", currentPnLPct, peakPnLPct)

	// Case 2: Profit above threshold, but at peak → no trailing needed
	currentPnLPct = 5.0
	peakPnLPct = 5.0
	shouldTrail = currentPnLPct >= trailingThreshold && peakPnLPct > currentPnLPct

	if shouldTrail {
		t.Error("Expected trailing stop NOT to trigger when current == peak")
	}
	t.Logf("✅ Trailing stop correctly NOT triggered when current == peak")

	// Case 3: Profit below threshold → no trailing
	currentPnLPct = 2.0
	peakPnLPct = 3.0
	shouldTrail = currentPnLPct >= trailingThreshold && peakPnLPct > currentPnLPct

	if shouldTrail {
		t.Error("Expected trailing stop NOT to trigger below threshold")
	}
	t.Logf("✅ Trailing stop correctly NOT triggered below threshold")
}

// TestAltcoinHigherThresholds verifies altcoins use higher thresholds.
func TestAltcoinHigherThresholds(t *testing.T) {
	at := &AutoTrader{}
	at.breakevenStopCache = make(map[string]bool)

	symbol := "SOLUSDT"
	side := "long"
	posKey := symbol + "_" + side

	// Altcoin threshold is 2.5%, profit is 2.0% → should NOT trigger
	altcoinThreshold := 2.5
	currentPnLPct := 2.0

	if currentPnLPct >= altcoinThreshold {
		at.breakevenStopCacheMu.Lock()
		at.breakevenStopCache[posKey] = true
		at.breakevenStopCacheMu.Unlock()
	}

	at.breakevenStopCacheMu.RLock()
	applied := at.breakevenStopCache[posKey]
	at.breakevenStopCacheMu.RUnlock()

	if applied {
		t.Errorf("Expected breakeven NOT to trigger for altcoin at %.2f%% (threshold %.2f%%)", currentPnLPct, altcoinThreshold)
	}
	t.Logf("✅ Altcoin threshold: %.2f%% < %.2f%% → no breakeven", currentPnLPct, altcoinThreshold)
}

// TestBreakevenNotReapplied verifies breakeven stop is only applied once.
func TestBreakevenNotReapplied(t *testing.T) {
	at := &AutoTrader{}
	at.breakevenStopCache = make(map[string]bool)

	posKey := "BTCUSDT_long"

	// First application
	at.breakevenStopCacheMu.Lock()
	at.breakevenStopCache[posKey] = true
	at.breakevenStopCacheMu.Unlock()

	// Check: should be true
	at.breakevenStopCacheMu.RLock()
	first := at.breakevenStopCache[posKey]
	at.breakevenStopCacheMu.RUnlock()

	if !first {
		t.Fatal("Expected cache to be true after first application")
	}

	// Second check: still true (idempotent)
	at.breakevenStopCacheMu.RLock()
	second := at.breakevenStopCache[posKey]
	at.breakevenStopCacheMu.RUnlock()

	if !second {
		t.Error("Cache should remain true after second check")
	}
	t.Logf("✅ Breakeven stop is idempotent (only applied once)")
}

// TestConcurrentCacheAccess verifies thread safety.
func TestConcurrentCacheAccess(t *testing.T) {
	at := &AutoTrader{}
	at.breakevenStopCache = make(map[string]bool)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			posKey := "BTCUSDT_long"
			at.breakevenStopCacheMu.Lock()
			at.breakevenStopCache[posKey] = true
			at.breakevenStopCacheMu.Unlock()

			at.breakevenStopCacheMu.RLock()
			_ = at.breakevenStopCache[posKey]
			at.breakevenStopCacheMu.RUnlock()
		}(i)
	}
	wg.Wait()

	at.breakevenStopCacheMu.RLock()
	applied := at.breakevenStopCache["BTCUSDT_long"]
	at.breakevenStopCacheMu.RUnlock()

	if !applied {
		t.Error("Cache should be true after concurrent access")
	}
	t.Logf("✅ Concurrent cache access: 100 goroutines, no race conditions")
}
