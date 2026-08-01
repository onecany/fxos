package hyperliquid

import (
	"context"
	"net/http"
	"testing"
)

func TestFetchPerpDexCoinsParsesOpenInterest(t *testing.T) {
	withStubbedPerpDexFetch(t, func(ctx context.Context, client *http.Client, dex string) ([]CoinInfo, error) {
		return []CoinInfo{
			{Symbol: "BTC", Volume24h: 1e9, OpenInterest: 5e8},
			{Symbol: "ETH", Volume24h: 8e8, OpenInterest: 3e8},
			{Symbol: "SOL", Volume24h: 2e8, OpenInterest: 1e7},
		}, nil
	})

	coins, err := GetPerpDexCoins(context.Background(), "")
	if err != nil {
		t.Fatalf("GetPerpDexCoins: %v", err)
	}
	if len(coins) != 3 {
		t.Fatalf("got %d coins, want 3", len(coins))
	}
	// OpenInterest must survive the round-trip through the cache copy.
	if coins[0].Symbol != "BTC" || coins[0].OpenInterest != 5e8 {
		t.Fatalf("BTC OI = %v (symbol %s), want 5e8", coins[0].OpenInterest, coins[0].Symbol)
	}
	if coins[2].Symbol != "SOL" || coins[2].OpenInterest != 1e7 {
		t.Fatalf("SOL OI = %v, want 1e7", coins[2].OpenInterest)
	}
}
