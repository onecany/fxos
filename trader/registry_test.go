package trader

import "testing"

// TestRegistryHasAllExchanges locks the registry contract: every supported
// exchange must be registered, otherwise CreateTrader fails at runtime
// instead of compile time. Adding a new exchange adapter requires a
// RegisterExchange call in registry.go init().
func TestRegistryHasAllExchanges(t *testing.T) {
	expected := []string{"binance", "bybit", "okx", "bitget", "gate", "kucoin", "hyperliquid", "aster", "lighter", "indodax"}
	if len(exchangeRegistry) < len(expected) {
		t.Fatalf("registry has %d exchanges, expected at least %d", len(exchangeRegistry), len(expected))
	}
	for _, name := range expected {
		if _, ok := exchangeRegistry[name]; !ok {
			t.Errorf("exchange %q not registered in exchangeRegistry", name)
		}
	}
}

// TestCreateTraderUnknownExchange verifies that unknown exchange names
// surface an error instead of falling through a switch to a zero trader.
func TestCreateTraderUnknownExchange(t *testing.T) {
	trader, err := CreateTrader("not_a_real_exchange", AutoTraderConfig{}, "user")
	if err == nil {
		t.Fatal("expected error for unknown exchange")
	}
	if trader != nil {
		t.Fatalf("expected nil trader for unknown exchange, got %v", trader)
	}
}
