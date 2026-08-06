package okxdata

import (
	"sort"
	"strconv"
	"time"

	"fxos/provider/nofx"
)

// PriceRanking builds a nofx.PriceRankingData from OKX SWAP tickers
// (exchange-direct). Gainers/losers are computed from open24h → last.
// Duration is informational; OKX tickers carry a single 24h window.
func PriceRanking(duration string, limit int) (*nofx.PriceRankingData, error) {
	if duration == "" {
		duration = "24h"
	}
	if limit <= 0 {
		limit = 10
	}
	ts, err := tickers()
	if err != nil {
		return nil, err
	}

	type delta struct {
		symbol   string
		price    float64
		deltaPct float64
	}
	items := make([]delta, 0, len(ts))
	for _, t := range ts {
		last, err1 := strconv.ParseFloat(t.Last, 64)
		open, err2 := strconv.ParseFloat(t.Open24h, 64)
		if err1 != nil || err2 != nil || open <= 0 || last <= 0 {
			continue
		}
		items = append(items, delta{
			symbol:   baseSymbol(t.InstID),
			price:    last,
			deltaPct: (last - open) / open,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].deltaPct > items[j].deltaPct })

	top := make([]nofx.PriceRankingItem, 0, limit)
	low := make([]nofx.PriceRankingItem, 0, limit)
	for i := 0; i < len(items) && (len(top) < limit || len(low) < limit); i++ {
		item := items[i]
		if len(top) < limit {
			top = append(top, nofx.PriceRankingItem{
				Symbol:     item.symbol,
				Price:      item.price,
				PriceDelta: item.deltaPct,
			})
		}
		j := len(items) - 1 - i
		if j >= 0 && j != i && len(low) < limit {
			lowItem := items[j]
			low = append(low, nofx.PriceRankingItem{
				Symbol:     lowItem.symbol,
				Price:      lowItem.price,
				PriceDelta: lowItem.deltaPct,
			})
		}
	}

	return &nofx.PriceRankingData{
		Durations: map[string]*nofx.PriceRankingDuration{
			duration: {Top: top, Low: low},
		},
		FetchedAt: time.Now(),
	}, nil
}
