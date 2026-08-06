package okxdata

import "testing"

// TestBaseSymbol locks the instId → bare base conversion used to key
// rankings and quant snapshots (BTC-USDT-SWAP → BTC).
func TestBaseSymbol(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"BTC-USDT-SWAP", "BTC"},
		{"ETH-USDT-SWAP", "ETH"},
		{"BTC-USD-SWAP", "BTC"},
		{"XRP-USDC-SWAP", "XRP"},
		{"SOL-USDT-SWAP", "SOL"},
	}
	for _, tc := range cases {
		if got := baseSymbol(tc.in); got != tc.want {
			t.Errorf("baseSymbol(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestStringsHasSuffix locks the local suffix helper.
func TestStringsHasSuffix(t *testing.T) {
	if !stringsHasSuffix("BTC-USDT-SWAP", "-USDT-SWAP") {
		t.Error("should match USDT-SWAP suffix")
	}
	if stringsHasSuffix("BTC-USD-SWAP", "-USDT-SWAP") {
		t.Error("must not match USDT-SWAP against USD-SWAP")
	}
}
