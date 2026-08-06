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

// TestOKXMapInterval locks the interval-name → OKX bar-unit mapping used by
// the kline fetcher (1m/3m/5m/... lowercase → 1m/3m/5m/1H/4H/1D uppercase).
func TestOKXMapInterval(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1m", "1m"},
		{"3m", "3m"},
		{"5m", "5m"},
		{"15m", "15m"},
		{"30m", "30m"},
		{"1h", "1H"},
		{"2h", "2H"},
		{"4h", "4H"},
		{"6h", "6H"},
		{"12h", "12H"},
		{"1d", "1D"},
		{"1M", ""},  // monthly unsupported
		{"1w", ""},  // weekly unsupported
		{"", ""},    // empty
		{"xyz", ""}, // garbage
	}
	for _, tc := range cases {
		if got := okxMapInterval(tc.in); got != tc.want {
			t.Errorf("okxMapInterval(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestOKXBarMillis locks bar-duration mapping used for CloseTime derivation.
func TestOKXBarMillis(t *testing.T) {
	if got := okxBarMillis("1m"); got != 60_000 {
		t.Errorf("1m = %d, want 60000", got)
	}
	if got := okxBarMillis("4H"); got != 14_400_000 {
		t.Errorf("4H = %d, want 14400000", got)
	}
	if got := okxBarMillis("1D"); got != 86_400_000 {
		t.Errorf("1D = %d, want 86400000", got)
	}
}
