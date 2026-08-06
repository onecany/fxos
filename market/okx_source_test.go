package market

import "testing"

// TestOKXBaseSymbol locks the OKX SWAP instId base conversion: the legacy
// USDT-perp symbol must map to the bare base used by OKX (BTCUSDT →
// BTC-USDT-SWAP) so the open-interest and funding-rate calls hit the right
// instrument.
func TestOKXBaseSymbol(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"BTCUSDT", "BTC"},
		{"btcusdt", "BTC"},
		{"SOLUSDT", "SOL"},
		{"ETH-USDT-SWAP", "ETH-USDT-SWAP"}, // already OKX-formatted: no suffix strip
		{"XRPUSDC", "XRP"},
		{"DOGE", "DOGE"}, // bare base passes through
		{" BTCUSDT ", "BTC"},
	}
	for _, tc := range cases {
		if got := okxBaseSymbol(tc.in); got != tc.want {
			t.Errorf("okxBaseSymbol(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
