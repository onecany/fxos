package store

import "testing"

func TestDefaultStrategyDoesNotEnableNofxOSData(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	assertHyperMainDefault(t, cfg)
	ind := cfg.Indicators
	if ind.NofxOSAPIKey != "" {
		t.Fatalf("default should not include a NofxOS API key for exchange-direct strategies")
	}
	if ind.EnableQuantData || ind.EnableQuantOI || ind.EnableQuantNetflow || ind.EnableOIRanking || ind.EnableNetFlowRanking || ind.EnablePriceRanking {
		t.Fatalf("default strategy must not enable NofxOS datasets: %+v", ind)
	}
	if !ind.EnableRawKlines {
		t.Fatalf("raw Hyperliquid klines must stay enabled")
	}
}

func TestHyperMainDefaultSurvivesClampAndNormalize(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	cfg.CoinSource.UseAI500 = true
	cfg.ClampLimits()
	assertHyperMainDefault(t, cfg)
	if cfg.CoinSource.UseAI500 {
		t.Fatalf("hyper_main strategy must clear stale AI500 flag: %+v", cfg.CoinSource)
	}
}

func TestEmptyCoinSourceInfersHyperMainNotAI500(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	cfg.CoinSource = CoinSourceConfig{}
	cfg.NormalizeProductSchema()
	assertHyperMainDefault(t, cfg)
}

func assertHyperMainDefault(t *testing.T, cfg StrategyConfig) {
	t.Helper()
	if cfg.CoinSource.SourceType != "hyper_main" || !cfg.CoinSource.UseHyperMain || cfg.CoinSource.HyperMainLimit != 30 {
		t.Fatalf("coin source = %+v, want Hyperliquid main markets top 30 (exchange-direct default)", cfg.CoinSource)
	}
	// The exchange-direct default must not carry vergex/claw402 fields, or
	// the kernel would route it onto the vergex prompt path.
	if cfg.CoinSource.VergexLimit != 0 || cfg.CoinSource.VergexMarketType != "" || cfg.CoinSource.VergexChain != "" || cfg.CoinSource.VergexLiqBand != "" {
		t.Fatalf("exchange-direct default must not set vergex fields: %+v", cfg.CoinSource)
	}
}
