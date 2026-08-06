package trader

import (
	ktypes "fxos/kernel/types"
	"fxos/trader/types"
	"strings"
	"testing"
	"time"
)

func throttleContext(symbol, side string, heldFor time.Duration, pnlPct float64) *ktypes.Context {
	return &ktypes.Context{
		Positions: []ktypes.PositionInfo{
			{
				Symbol:           symbol,
				Side:             side,
				UnrealizedPnLPct: pnlPct,
				UpdateTime:       time.Now().Add(-heldFor).UnixMilli(),
			},
		},
	}
}

func TestTradeThrottleBlocksEarlyNoiseClose(t *testing.T) {
	at := &AutoTrader{}
	ctx := throttleContext("xyz:INTC", types.SideLong, 20*time.Minute, -0.3)

	reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionCloseLong}, ctx, 0)
	if !strings.Contains(reason, "min AI-managed hold") {
		t.Fatalf("expected early close to be blocked by min hold, got %q", reason)
	}
}

func TestTradeThrottleAllowsEarlyHardStop(t *testing.T) {
	at := &AutoTrader{}
	ctx := throttleContext("xyz:INTC", types.SideLong, 20*time.Minute, -3.0)

	reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionCloseLong}, ctx, 0)
	if reason != "" {
		t.Fatalf("expected hard stop close to pass, got %q", reason)
	}
}

func TestTradeThrottleBlocksFlatCloseInsideNoiseWindow(t *testing.T) {
	at := &AutoTrader{}
	ctx := throttleContext("xyz:INTC", types.SideLong, 60*time.Minute, 0.4)

	reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionCloseLong}, ctx, 0)
	if !strings.Contains(reason, "noise band") {
		t.Fatalf("expected flat close to be blocked inside noise window, got %q", reason)
	}
}

func TestTradeThrottleAllowsConfirmedLossAfterMinimumHold(t *testing.T) {
	at := &AutoTrader{}
	ctx := throttleContext("xyz:INTC", types.SideLong, 60*time.Minute, -1.2)

	reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionCloseLong}, ctx, 0)
	if reason != "" {
		t.Fatalf("expected confirmed loss after min hold to pass, got %q", reason)
	}
}

func TestTradeThrottleAllowsLongShortPairInCycle(t *testing.T) {
	at := &AutoTrader{}
	ctx := &ktypes.Context{}

	// One open already queued this cycle (e.g. the long) — the second open
	// (the short) must still be allowed so a directional pair can open.
	reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionOpenShort}, ctx, 1)
	if reason != "" {
		t.Fatalf("expected the second (short) open in cycle to be allowed, got %q", reason)
	}
}

func TestTradeThrottleBlocksOpensOverCycleCap(t *testing.T) {
	at := &AutoTrader{}
	ctx := &ktypes.Context{}

	// under the 6-per-cycle cap, a further open is allowed
	if reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionOpenLong}, ctx, 5); reason != "" {
		t.Fatalf("expected open within the 6-per-cycle cap to be allowed, got %q", reason)
	}
	// at the cap, the next open is blocked
	if reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionOpenLong}, ctx, 6); !strings.Contains(reason, "6 new position") {
		t.Fatalf("expected open beyond the 6-per-cycle cap to be blocked, got %q", reason)
	}
}

func TestTradeThrottleBlocksOpeningAgainstExistingPosition(t *testing.T) {
	at := &AutoTrader{}
	ctx := throttleContext("xyz:INTC", types.SideLong, 2*time.Hour, 1.0)

	reason := at.tradeThrottleReason(ktypes.Decision{Symbol: "xyz:INTC", Action: types.ActionOpenShort}, ctx, 0)
	if !strings.Contains(reason, "already has an open") {
		t.Fatalf("expected opposite open to be blocked when position exists, got %q", reason)
	}
}
