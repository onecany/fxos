package kernel

import (
	"testing"

	"fxos/trader/market"
)

// TestOIFilterSkipsLowRealOI locks the liquidity filter: a REAL OI value below
// the 15M USD threshold must skip the coin (existing behaviour, preserved).
func TestOIFilterSkipsLowRealOI(t *testing.T) {
	data := &market.Data{
		CurrentPrice: 1.0,
		OpenInterest: &market.OIData{Latest: 5_000_000}, // 5M * $1 = 5M USD < 15M
	}
	if !shouldSkipCoinByOIFilter("DOGEUSDT", data, nil) {
		t.Fatal("expected skip for real OI below threshold")
	}
}

// TestOIFilterKeepsHighRealOI locks the opposite: real OI above the threshold
// must pass.
func TestOIFilterKeepsHighRealOI(t *testing.T) {
	data := &market.Data{
		CurrentPrice: 100.0,
		OpenInterest: &market.OIData{Latest: 500_000}, // 500K * $100 = 50M USD > 15M
	}
	if shouldSkipCoinByOIFilter("BTCUSDT", data, nil) {
		t.Fatal("expected pass for real OI above threshold")
	}
}

// TestOIFilterKeepsMissingOI is the regression lock for the geo-blocked OI
// source bug: when OI data is unavailable (nil or zero), the coin MUST NOT be
// skipped. Before the fix, a geo-blocked Binance fapi produced OI=0 and the
// filter silently emptied the candidate universe, so prompts were built with
// zero candidate coins.
func TestOIFilterKeepsMissingOI(t *testing.T) {
	cases := []struct {
		name string
		data *market.Data
	}{
		{"nil OI", &market.Data{CurrentPrice: 100.0}},
		{"zero OI", &market.Data{CurrentPrice: 100.0, OpenInterest: &market.OIData{Latest: 0}}},
		{"nil OI nil price", &market.Data{}},
	}
	for _, tc := range cases {
		if shouldSkipCoinByOIFilter("BTCUSDT", tc.data, nil) {
			t.Fatalf("%s: missing OI data must not skip the coin", tc.name)
		}
	}
}

// TestOIFilterKeepsPositionsAndXYZ locks the exemptions: existing positions and
// xyz dex assets (no perp OI) are never skipped, even with low OI.
func TestOIFilterKeepsPositionsAndXYZ(t *testing.T) {
	data := &market.Data{
		CurrentPrice: 1.0,
		OpenInterest: &market.OIData{Latest: 5_000_000}, // below threshold
	}
	if shouldSkipCoinByOIFilter("BTCUSDT", data, map[string]bool{"BTCUSDT": true}) {
		t.Fatal("existing position must never be skipped by OI filter")
	}
	if shouldSkipCoinByOIFilter("xyz:TSLA", data, nil) {
		t.Fatal("xyz dex asset must never be skipped by OI filter")
	}
}

// TestOIFilterNilDataNeverSkips guards the nil-data edge: a missing market
// snapshot is not liquidity evidence.
func TestOIFilterNilDataNeverSkips(t *testing.T) {
	if shouldSkipCoinByOIFilter("BTCUSDT", nil, nil) {
		t.Fatal("nil data must never be skipped by OI filter")
	}
}
